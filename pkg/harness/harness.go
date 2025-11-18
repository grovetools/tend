package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-core/pkg/tmux"
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/project"
)

// Context carries state through a scenario execution
type Context struct {
	// Core paths
	RootDir     string // Root temporary directory for the scenario
	ProjectRoot string // Root of the project being tested
	GroveBinary string // Path to the grove binary under test
	TestID      string // Unique ID for this test run

	// Sandboxed home environment
	homeDir   string
	configDir string
	dataDir   string
	cacheDir  string

	// State management
	dirs              map[string]string      // Named directories created during test
	values            map[string]interface{} // Generic key-value store for step communication
	UseRealDeps       map[string]bool        // Map of dependencies to use real binaries for
	mockOverrides     map[string]string      // Map of command name to its mock binary path
	shellPaneID       string                 // Tmux pane ID for the shell
	editorPaneID      string                 // Tmux pane ID for the editor
	currentEditorFile string                 // The file currently open in the editor pane

	// UI for displaying command output
	ui *UI
}

// Step represents a single action in a scenario
type Step struct {
	Name        string
	Description string
	Func        func(ctx *Context) error
	File        string
	Line        int
}

// Scenario is a collection of steps defining a test
type Scenario struct {
	Name         string
	Description  string
	Tags         []string
	Steps        []Step
	LocalOnly    bool // Skips in CI by default
	ExplicitOnly bool // Skips in "run all"
	File         string
	Line         int
}

// Options for harness execution
type Options struct {
	Interactive     bool          // Enable interactive mode
	Verbose         bool          // Enable verbose output
	VeryVerbose     bool          // Enable very verbose output (includes command details)
	NoCleanup       bool          // Keep temp dirs for debugging
	ContinueOnError bool          // Continue batch execution on error
	Timeout         time.Duration // Global timeout for scenarios
	GroveBinary     string        // Path to Grove binary (optional)
	RootDir         string        // Root directory for tests (optional)
	MonitorDocker   bool          // Show live Docker container updates
	DockerFilter    string        // Filter for Docker containers (e.g., "name=grove")
	TmuxSplit       bool          // Split tmux window and cd to test directory
	Nvim            bool          // Start nvim in the new tmux split
	UseRealDeps     []string      // List of dependencies to use real binaries for
}

// Harness runs scenarios
type Harness struct {
	opts Options
}

// Result represents the outcome of a scenario run
type Result struct {
	ScenarioName string
	Success      bool
	FailedStep   string
	Error        error
	Duration     time.Duration
	StartTime    time.Time
	EndTime      time.Time
	StepResults  []StepResult // Added for detailed reporting
}

// StepResult represents the outcome of a single step
type StepResult struct {
	Name      string
	Success   bool
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}


// New creates a new test harness
func New(opts Options) *Harness {
	return &Harness{
		opts: opts,
	}
}

// Run executes a scenario and returns the result
func (h *Harness) Run(ctx context.Context, scenario *Scenario) (*Result, error) {
	start := time.Now()
	result := &Result{
		ScenarioName: scenario.Name,
		Success:      true,
		StartTime:    start,
	}

	// Setup phase
	ui := NewUI(h.opts.Interactive, h.opts.Verbose || h.opts.VeryVerbose, h.opts.VeryVerbose)
	ui.ScenarioStart(scenario.Name, scenario.Description)
	
	// Enable Docker monitoring if requested
	if h.opts.MonitorDocker {
		filter := h.opts.DockerFilter
		if filter == "" {
			filter = "name=grove"
		}
		ui.EnableMonitoring(filter)
		defer ui.DisableMonitoring()
		
		// Show initial container state
		if h.opts.Verbose {
			ui.ShowDockerStatus()
		}
	}

	// Create temp directory manager
	tempMgr, err := h.createTempManager(scenario.Name)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("creating temp directory: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	// Setup sandboxed home directory structure
	sandboxedHome := filepath.Join(tempMgr.BaseDir(), "home")
	sandboxedConfig := filepath.Join(sandboxedHome, ".config")
	sandboxedData := filepath.Join(sandboxedHome, ".local", "share")
	sandboxedCache := filepath.Join(sandboxedHome, ".cache")

	for _, dir := range []string{sandboxedHome, sandboxedConfig, sandboxedData, sandboxedCache} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("creating sandboxed home directory structure: %w", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, result.Error
		}
	}

	// Declare testCtx variable here so defer can reference it
	var testCtx *Context
	
	// Setup cleanup
	if !h.opts.NoCleanup {
		defer func() {
			ui.Cleanup()
			// Clean up any TUI sessions if testCtx was initialized
			if testCtx != nil {
				if sessionName := testCtx.GetString("active_tui_session_name"); sessionName != "" {
					tmuxClient, err := tmux.NewClient()
					if err == nil {
						_ = tmuxClient.KillSession(context.Background(), sessionName)
					}
				}
			}
			if cleanErr := tempMgr.Cleanup(); cleanErr != nil {
				ui.Error("Cleanup failed", cleanErr)
			}
		}()
	} else {
		ui.Info("Cleanup disabled", fmt.Sprintf("Test files preserved in: %s", tempMgr.BaseDir()))
	}

	// Initialize context
	groveBinary := h.resolveBinary()
	if h.opts.Verbose || h.opts.VeryVerbose {
		ui.Info("Grove binary", groveBinary)
	}
	// Generate a unique test ID based on the temp directory
	testID := filepath.Base(tempMgr.BaseDir())
	
	// Populate the map for real dependencies
	realDepsMap := make(map[string]bool)
	if len(h.opts.UseRealDeps) > 0 {
		if len(h.opts.UseRealDeps) == 1 && h.opts.UseRealDeps[0] == "all" {
			// A special value to swap all swappable mocks
			realDepsMap["all"] = true
		} else {
			for _, dep := range h.opts.UseRealDeps {
				realDepsMap[dep] = true
			}
		}
	}

	testCtx = &Context{
		RootDir:       tempMgr.BaseDir(),
		ProjectRoot:   h.opts.RootDir, // Pass project root from harness options
		GroveBinary:   groveBinary,
		TestID:        testID,
		homeDir:       sandboxedHome,
		configDir:     sandboxedConfig,
		dataDir:       sandboxedData,
		cacheDir:      sandboxedCache,
		dirs:          make(map[string]string),
		values:        make(map[string]interface{}),
		UseRealDeps:   realDepsMap,
		mockOverrides: make(map[string]string),
		ui:            ui,
	}
	
	// Set the test ID in the UI for container filtering
	ui.SetTestID(testID)

	// Handle tmux split if requested
	if h.opts.TmuxSplit || h.opts.Nvim {
		if err := h.setupDebugPanes(testCtx, ui, scenario); err != nil {
			// This is a non-fatal error for the test itself, so we just show a warning.
			ui.Error("tmux pane setup failed", err)
		}
	}

	// Execute steps
	var stepResults []StepResult
	for i, step := range scenario.Steps {
		select {
		case <-ctx.Done():
			result.Success = false
			result.Error = fmt.Errorf("scenario cancelled: %w", ctx.Err())
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.StepResults = stepResults
			return result, result.Error
		default:
		}

		// Jump to step definition in editor if in debug mode
		if h.opts.Nvim && testCtx.editorPaneID != "" && step.Line > 0 {
			tmuxClient, err := tmux.NewClient()
			if err == nil {
				// Check if the step is in a different file and switch if needed
				if step.File != testCtx.currentEditorFile {
					tmuxClient.SendKeys(context.Background(), testCtx.editorPaneID, fmt.Sprintf(":e %s", step.File), "C-m")
					testCtx.currentEditorFile = step.File
					time.Sleep(100 * time.Millisecond) // Give nvim time to open the file
				}
				// Now, jump to the line
				jumpCmd := fmt.Sprintf(":%d", step.Line)
				tmuxClient.SendKeys(context.Background(), testCtx.editorPaneID, jumpCmd, "C-m")
			}
		}

		ui.StepStart(i+1, len(scenario.Steps), step.Name)

		// Interactive pause
		if h.opts.Interactive {
			// First, display the TUI state if a session is active
			if sessionName := testCtx.GetString("active_tui_session_name"); sessionName != "" {
				tmuxClient, err := tmux.NewClient()
				if err == nil {
					if content, err := tmuxClient.CapturePane(context.Background(), sessionName); err == nil {
						ui.RenderTUICapture(content)
					}
				}
			}

			action := ui.WaitForUser()
			switch action {
			case "quit":
				result.Success = false
				result.Error = fmt.Errorf("user cancelled at step: %s", step.Name)
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				result.StepResults = stepResults
				return result, result.Error
			case "attach":
				if sessionName := testCtx.GetString("active_tui_session_name"); sessionName != "" {
					ui.Info("Attach", fmt.Sprintf("Attaching to tmux session '%s'. Detach with 'Ctrl-b d' to continue test.", sessionName))
					
					// Temporarily suspend the runner to attach
					cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					_ = cmd.Run() // This will block until user detaches
				} else {
					ui.Error("Attach failed", fmt.Errorf("no active TUI session found"))
				}
			case "continue":
				// Do nothing, proceed to step execution
			}
		}

		// Execute step
		stepResult := h.executeStep(ctx, testCtx, step, ui)
		stepResults = append(stepResults, stepResult)

		if stepResult.Error != nil {
			ui.StepFailed(step.Name, stepResult.Error, stepResult.Duration)
			result.Success = false
			result.FailedStep = step.Name
			result.Error = &StepError{
				StepName: step.Name,
				Err:      stepResult.Error,
			}
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.StepResults = stepResults
			return result, result.Error
		}

		ui.StepSuccess(step.Name, stepResult.Duration)
	}

	// Success
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.StepResults = stepResults
	ui.ScenarioSuccess(scenario.Name, result.Duration)

	return result, nil
}

// RunAll executes multiple scenarios in sequence
func (h *Harness) RunAll(ctx context.Context, scenarios []*Scenario) ([]*Result, error) {
	results := make([]*Result, 0, len(scenarios))
	ui := NewUI(h.opts.Interactive, h.opts.Verbose || h.opts.VeryVerbose, h.opts.VeryVerbose)

	fmt.Printf("\n🚀 Running %d scenarios\n", len(scenarios))
	fmt.Println(strings.Repeat("=", 60))

	for i, scenario := range scenarios {
		select {
		case <-ctx.Done():
			return results, fmt.Errorf("batch execution cancelled: %w", ctx.Err())
		default:
		}

		fmt.Printf("\n[%d/%d] Running %s...\n", i+1, len(scenarios), scenario.Name)

		result, err := h.Run(ctx, scenario)
		results = append(results, result)

		if err != nil && !h.opts.ContinueOnError {
			ui.Error(fmt.Sprintf("Scenario %s failed", scenario.Name), err)
			return results, fmt.Errorf("scenario %s failed: %w", scenario.Name, err)
		}
	}

	// Summary
	passed := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			passed++
		} else {
			failed++
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	if failed == 0 {
		fmt.Printf("✅ All %d scenarios passed!\n", len(results))
	} else {
		fmt.Printf("❌ %d/%d scenarios failed\n", failed, len(results))
		fmt.Printf("✓ Passed: %d\n", passed)
		fmt.Printf("✗ Failed: %d\n", failed)
	}

	if failed > 0 {
		return results, fmt.Errorf("%d scenarios failed", failed)
	}

	return results, nil
}

// createTempManager creates a temporary directory manager for a scenario
func (h *Harness) createTempManager(scenarioName string) (*fs.TempDirManager, error) {
	prefix := fmt.Sprintf("grove-tend-%s-", scenarioName)
	return fs.NewTempDirManager(prefix)
}

// executeStep runs a single step with proper error handling
func (h *Harness) executeStep(ctx context.Context, testCtx *Context, step Step, ui *UI) StepResult {
	start := time.Now()
	result := StepResult{
		Name:      step.Name,
		Success:   true,
		StartTime: start,
	}

	// Create a panic handler
	defer func() {
		if r := recover(); r != nil {
			result.Error = fmt.Errorf("step panicked: %v", r)
			result.Success = false
			ui.Error("Step panic", result.Error)
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// Run the step function
	stepErr := make(chan error, 1)
	go func() {
		stepErr <- step.Func(testCtx)
	}()

	// Wait for completion or context cancellation
	select {
	case err := <-stepErr:
		if err != nil {
			result.Error = err
			result.Success = false
		}
		return result
	case <-ctx.Done():
		result.Error = fmt.Errorf("step cancelled: %w", ctx.Err())
		result.Success = false
		return result
	}
}

// resolveBinary finds the grove binary to test
func (h *Harness) resolveBinary() string {
	// Check options first (from --grove flag)
	if h.opts.GroveBinary != "" {
		return h.opts.GroveBinary
	}
	
	// Check environment variable
	if bin := os.Getenv("GROVE_BINARY"); bin != "" {
		return bin
	}

	// Try to find binary via grove.yml
	rootDir := h.opts.RootDir
	if rootDir == "" {
		rootDir, _ = os.Getwd()
	}
	
	if binaryPath, err := project.GetBinaryPath(rootDir); err == nil {
		return binaryPath
	}

	// Default to looking for 'grove' in PATH
	// This assumes gvm or proper PATH setup will make it available
	return "grove"
}

// setupTmuxPane manages the creation or reuse of a tmux pane for debugging.
func setupTmuxPane(client *tmux.Client, ui *UI, paneType string, splitHorizontal bool, commandStr string) (string, error) {
	paneIDFile := filepath.Join(os.TempDir(), fmt.Sprintf("tend-debug-%s-pane-id", paneType))
	var paneID string

	// Check for a cached pane ID
	if data, err := os.ReadFile(paneIDFile); err == nil {
		existingPaneID := strings.TrimSpace(string(data))
		if client.PaneExists(context.Background(), existingPaneID) {
			ui.Info("tmux", fmt.Sprintf("Reusing existing %s pane...", paneType))
			paneID = existingPaneID

			// Kill any running editor to ensure a clean state
			if currentCmd, err := client.GetPaneCommand(context.Background(), paneID); err == nil {
				currentCmd = strings.TrimSpace(currentCmd)
				if currentCmd == "nvim" || currentCmd == "vim" || currentCmd == "vi" {
					client.SendKeys(context.Background(), paneID, "Escape", ":qa!", "C-m")
					time.Sleep(100 * time.Millisecond)
				}
			}
		} else {
			ui.Info("tmux", fmt.Sprintf("Previous %s pane no longer exists, creating new one...", paneType))
		}
	}

	// Create a new pane if needed
	if paneID == "" {
		ui.Info("tmux", fmt.Sprintf("Creating %s pane...", paneType))
		newPaneID, err := client.SplitWindow(context.Background(), "", splitHorizontal, 0, "")
		if err != nil {
			return "", fmt.Errorf("failed to split tmux window for %s pane: %w", paneType, err)
		}
		paneID = newPaneID
		_ = os.WriteFile(paneIDFile, []byte(paneID), 0644)
	}

	// Send the command to the pane
	if err := client.SendKeys(context.Background(), paneID, commandStr, "C-m"); err != nil {
		// Clean up and return error if sending keys fails
		command.New("tmux", "kill-pane", "-t", paneID).Run()
		os.Remove(paneIDFile)
		return "", fmt.Errorf("failed to send keys to %s pane: %w", paneType, err)
	}

	return paneID, nil
}

// setupDebugPanes creates and configures the shell and editor panes for debug mode.
func (h *Harness) setupDebugPanes(ctx *Context, ui *UI, scenario *Scenario) error {
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("not in a tmux session")
	}

	tmuxClient, err := tmux.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create tmux client: %w", err)
	}

	// Setup Shell Pane if requested
	if h.opts.TmuxSplit {
		shellCommand := fmt.Sprintf("cd '%s'", ctx.RootDir)
		// If nvim isn't requested separately, open it in the shell pane for old behavior
		if !h.opts.Nvim {
			shellCommand += " && nvim"
		}

		shellPaneID, err := setupTmuxPane(tmuxClient, ui, "shell", false, shellCommand)
		if err != nil {
			return err
		}
		ctx.shellPaneID = shellPaneID
	}

	// Setup Editor Pane if requested
	if h.opts.Nvim {
		// Determine which file to open - prefer scenario file, fall back to first step
		var initialFile string
		var initialLine int

		if scenario.File != "" {
			// Scenario has source location (using NewScenario constructor)
			initialFile = scenario.File
			initialLine = scenario.Line
		} else if len(scenario.Steps) > 0 && scenario.Steps[0].File != "" {
			// Fall back to first step's location
			initialFile = scenario.Steps[0].File
			initialLine = scenario.Steps[0].Line
		}

		// Only open editor if we have source location info
		if initialFile != "" {
			editorCommand := fmt.Sprintf("cd '%s' && nvim '%s'", ctx.ProjectRoot, initialFile)
			editorPaneID, err := setupTmuxPane(tmuxClient, ui, "editor", true, editorCommand)
			if err != nil {
				return err
			}
			ctx.editorPaneID = editorPaneID
			ctx.currentEditorFile = initialFile

			// Jump to initial line if available
			if editorPaneID != "" && initialLine > 0 {
				time.Sleep(100 * time.Millisecond) // Give nvim time to start
				jumpCmd := fmt.Sprintf(":%d", initialLine)
				tmuxClient.SendKeys(context.Background(), editorPaneID, jumpCmd, "C-m")
			}
		}
	}

	return nil
}

