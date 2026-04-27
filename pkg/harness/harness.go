package harness

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/pkg/tmux"
	"github.com/grovetools/core/tui/theme"
	"github.com/grovetools/tend/pkg/command"
	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/project"
	"github.com/grovetools/tend/pkg/tui"
)

var ulogHarness = grovelogging.NewUnifiedLogger("grove-tend.harness")

// Context carries state through a scenario execution
type Context struct {
	// Core paths
	RootDir     string // Root temporary directory for the scenario
	ProjectRoot string // Root of the project being tested
	GroveBinary string // Path to the grove binary under test
	TestID      string // Unique ID for this test run

	// Sandboxed home environment
	homeDir    string
	configDir  string
	dataDir    string
	stateDir   string
	cacheDir   string
	runtimeDir string // Short path for XDG_RUNTIME_DIR to avoid Unix socket length limits

	// State management
	dirs              map[string]string      // Named directories created during test
	values            map[string]interface{} // Generic key-value store for step communication
	assertions        []*AssertionResult     // Log of assertions for the current step
	UseRealDeps       map[string]bool        // Map of dependencies to use real binaries for
	mockOverrides     map[string]string      // Map of command name to its mock binary path
	shellPaneID       string                 // Tmux pane ID for the shell
	editorPaneID      string                 // Tmux pane ID for the editor
	currentEditorFile string                 // The file currently open in the editor pane
	recordTUIDir      string                 // Directory to save TUI session recordings
	tmuxSocket        string                 // Socket name for isolated tmux server for test TUI sessions

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
	Setup        []Step
	Steps        []Step
	Teardown     []Step
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
	// TestRootDir specifies a pre-existing directory to use for the test run.
	// If empty, a new temporary directory will be created.
	TestRootDir string
	// TmuxSocket specifies a custom tmux socket for debug-session mode.
	TmuxSocket string
	// TmuxEditorTarget specifies the tmux target for the editor window in debug-session mode.
	TmuxEditorTarget string
	// RunSetup runs the setup phase then pauses (or switches to interactive if no setup).
	RunSetup bool
	// RunSteps specifies which test steps to auto-run at startup before pausing.
	// Format: "1,2,3" runs test steps 1-3 then pauses at step 4
	RunSteps string
	// RecordTUIDir specifies the directory to save TUI session recordings for failed tests.
	RecordTUIDir string
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
	Name       string
	Success    bool
	Error      error
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Assertions []*AssertionResult
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
	sandboxedState := filepath.Join(sandboxedHome, ".local", "state")
	sandboxedCache := filepath.Join(sandboxedHome, ".cache")

	for _, dir := range []string{sandboxedHome, sandboxedConfig, sandboxedData, sandboxedState, sandboxedCache} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("creating sandboxed home directory structure: %w", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, result.Error
		}
	}

	// Generate a unique test ID based on the temp directory early (needed for runtime dir)
	testID := filepath.Base(tempMgr.BaseDir())

	// Create short runtime dir for Unix socket paths (groved, etc.)
	// Unix sockets have ~104 char path limit on macOS, so we must use /tmp directly
	sandboxedRuntime, err := fs.CreateShortTempDir(fmt.Sprintf("tend-%s-", testID))
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("creating short runtime dir: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	// Declare testCtx variable here so defer can reference it
	var testCtx *Context

	// Create cleanup function that can be called from multiple places
	cleanupFunc := func() {
		ui.Cleanup()
		// Clean up all TUI sessions if testCtx was initialized
		if testCtx != nil {
			// If using an isolated tmux server, kill the entire server
			// This cleans up all sessions and the server in one operation
			if testCtx.tmuxSocket != "" {
				ulogHarness.Debug("Attempting to kill tmux server").
					Field("socket", testCtx.tmuxSocket).
					Emit()
				client, err := tmux.NewClientWithSocket(testCtx.tmuxSocket)
				if err != nil {
					ulogHarness.Error("Failed to create tmux client for cleanup").
						Field("socket", testCtx.tmuxSocket).
						Field("error", err.Error()).
						Emit()
				} else {
					if err := client.KillServer(context.Background()); err != nil {
						ulogHarness.Error("Failed to kill tmux server").
							Field("socket", testCtx.tmuxSocket).
							Field("error", err.Error()).
							Emit()
					} else {
						ulogHarness.Debug("Successfully killed tmux server").
							Field("socket", testCtx.tmuxSocket).
							Emit()
					}
				}
			} else {
				// Fallback: kill individual sessions on the default tmux server
				sessions := testCtx.GetStringSlice("tui_sessions")
				if len(sessions) > 0 {
					tmuxClient, err := tmux.NewClient()
					if err == nil {
						for _, sessionName := range sessions {
							_ = tmuxClient.KillSession(context.Background(), sessionName)
						}
					}
				}
			}
		}
		// Clean up the short runtime dir (created outside tempMgr for socket path length)
		if testCtx != nil && testCtx.runtimeDir != "" {
			if cleanErr := os.RemoveAll(testCtx.runtimeDir); cleanErr != nil {
				ui.Error("Runtime dir cleanup failed", cleanErr)
			}
		}
		if cleanErr := tempMgr.Cleanup(); cleanErr != nil {
			ui.Error("Cleanup failed", cleanErr)
		}
	}

	// reapDaemons kills any groved processes spawned inside the sandbox.
	// Must run BEFORE directory removal so PID files are still readable.
	reapDaemons := func() {
		if testCtx != nil && testCtx.runtimeDir != "" {
			killDaemonsInTree(testCtx.runtimeDir)
		}
		if tempMgr != nil {
			if base := tempMgr.BaseDir(); base != "" {
				killDaemonsInTree(base)
			}
		}
	}

	// Always reap groved daemons, even with --no-cleanup.
	// When cleanup IS enabled, reapDaemons also runs inside cleanupFunc
	// (before dir removal); this defer catches the --no-cleanup path.
	defer reapDaemons()

	// Setup cleanup
	if !h.opts.NoCleanup {
		defer func() {
			reapDaemons()
			cleanupFunc()
		}()

		// Setup signal handler to ensure cleanup happens even on interrupt
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Clean up signal handler when done
		defer signal.Stop(sigChan)

		go func() {
			<-sigChan
			ulogHarness.Info("Received interrupt signal, cleaning up...").Emit()
			reapDaemons()
			cleanupFunc()
			os.Exit(130) // Standard exit code for SIGINT
		}()
	} else {
		ui.Info("Cleanup disabled", fmt.Sprintf("Test files preserved in: %s", tempMgr.BaseDir()))
	}

	// Initialize context
	groveBinary := h.resolveBinary()
	if h.opts.Verbose || h.opts.VeryVerbose {
		ui.Info("Grove binary", groveBinary)
	}
	// Unix socket paths are limited to 104 chars on macOS (sun_path). tmux places
	// sockets in $TMPDIR/tmux-<uid>/ which can already be ~50 chars on macOS, so
	// the socket name itself must stay short. Use a short prefix of testID for log
	// traceability plus a timestamp-derived hash for uniqueness.
	shortPrefix := testID
	if len(shortPrefix) > 8 {
		shortPrefix = shortPrefix[:8]
	}
	socketName := fmt.Sprintf("tt-%s-%x", shortPrefix, time.Now().UnixNano()%0x100000000)

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
		stateDir:      sandboxedState,
		cacheDir:      sandboxedCache,
		runtimeDir:    sandboxedRuntime,
		dirs:          make(map[string]string),
		values:        make(map[string]interface{}),
		UseRealDeps:   realDepsMap,
		mockOverrides: make(map[string]string),
		recordTUIDir:  h.opts.RecordTUIDir,
		tmuxSocket:    socketName,
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

	// Defer teardown to ensure it runs after setup and test steps, even on failure.
	defer func() {
		// Don't run teardown in run-setup mode or if cleanup is disabled.
		if h.opts.RunSetup || h.opts.NoCleanup {
			return
		}
		if len(scenario.Teardown) > 0 {
			ui.PhaseStart("Teardown")
			for i, step := range scenario.Teardown {
				// We don't fail the scenario if a teardown step fails, but we log it.
				ui.StepStart(i+1, len(scenario.Teardown), step.Name)
				stepResult := h.executeStep(ctx, testCtx, step, ui)
				if stepResult.Error != nil {
					ui.Error(fmt.Sprintf("Teardown step '%s' failed", step.Name), stepResult.Error)
				}
			}
		}
	}()

	// --- SETUP PHASE ---
	if len(scenario.Setup) > 0 {
		ui.PhaseStart("Setup")
		for i, step := range scenario.Setup {
			ui.StepStart(i+1, len(scenario.Setup), step.Name)
			testCtx.clearAssertions()
			stepResult := h.executeStep(ctx, testCtx, step, ui)
			stepResult.Assertions = testCtx.getAssertions()
			stepResults = append(stepResults, stepResult)

			if stepResult.Error != nil {
				ui.StepFailed(stepResult)
				result.Success = false
				result.FailedStep = step.Name
				result.Error = &StepError{StepName: step.Name, Err: stepResult.Error}
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				result.StepResults = stepResults
				return result, result.Error
			}
			ui.StepSuccess(stepResult)
		}
	}

	// --- RUN SETUP MODE ---
	// When --run-setup is used, we prepare the workspace for interactive debugging.
	// If setup exists, we've run it and halt. If not, we switch to interactive mode
	// to let the user step through the test interactively.
	if h.opts.RunSetup {
		if len(scenario.Setup) > 0 {
			// Setup was run, halt here so user can explore the prepared environment
			ui.Info("Run Setup", "Halting execution after setup phase.")
			result.Success = true
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.StepResults = stepResults
			return result, nil
		} else {
			// No setup steps exist, switch to interactive mode so user can step through tests
			ui.Info("Run Setup", "No setup steps found. Switching to interactive mode.")
			h.opts.Interactive = true
		}
	}

	// --- RUN STEPS MODE ---
	// Parse --run-steps to determine which test steps to auto-run before pausing
	var runTestSteps []int
	if h.opts.RunSteps != "" {
		// Enable interactive mode whenever --run-steps is used
		// We'll auto-run the specified test steps then pause
		h.opts.Interactive = true

		parts := strings.Split(h.opts.RunSteps, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			// Parse as test step number
			var stepNum int
			if _, err := fmt.Sscanf(part, "%d", &stepNum); err == nil {
				runTestSteps = append(runTestSteps, stepNum)
			}
		}
	}

	// --- TEST PHASE ---
	if len(scenario.Steps) > 0 {
		ui.PhaseStart("Test")
	}
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
		if step.Line > 0 {
			var tmuxClient *tmux.Client
			var editorTarget string
			var err error

			// Check for debug-session mode (dedicated tmux server)
			if h.opts.TmuxSocket != "" && h.opts.TmuxEditorTarget != "" {
				tmuxClient, err = tmux.NewClientWithSocket(h.opts.TmuxSocket)
				editorTarget = h.opts.TmuxEditorTarget
			} else if h.opts.Nvim && testCtx.editorPaneID != "" {
				// Pane-based debug mode
				tmuxClient, err = tmux.NewClient()
				editorTarget = testCtx.editorPaneID
			}

			if err == nil && tmuxClient != nil && editorTarget != "" {
				// Check if the step is in a different file and switch if needed
				if step.File != testCtx.currentEditorFile {
					_ = tmuxClient.SendKeys(context.Background(), editorTarget, fmt.Sprintf(":e %s", step.File), "C-m")
					testCtx.currentEditorFile = step.File
					time.Sleep(100 * time.Millisecond) // Give nvim time to open the file
				}
				// Now, jump to the line
				jumpCmd := fmt.Sprintf(":%d", step.Line)
				_ = tmuxClient.SendKeys(context.Background(), editorTarget, jumpCmd, "C-m")
			}
		}

		ui.StepStart(i+1, len(scenario.Steps), step.Name)

		// Display TUI state if verbose and a session is active
		if h.opts.Verbose || h.opts.VeryVerbose {
			if sessionName := testCtx.GetString("active_tui_session_name"); sessionName != "" {
				tmuxClient, err := tmux.NewClient()
				if err == nil {
					if content, err := tmuxClient.CapturePane(context.Background(), sessionName); err == nil {
						ui.RenderTUICapture(content)
					}
				}
			}
		}

		// Interactive pause (unless this step should be auto-run)
		shouldAutoRun := false
		for _, autoRunStep := range runTestSteps {
			if autoRunStep == i+1 { // Step numbers are 1-indexed
				shouldAutoRun = true
				break
			}
		}

		if h.opts.Interactive && !shouldAutoRun {
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
					cmd := tmux.Command("attach-session", "-t", sessionName)
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

		// Before running the step, clear any previous assertion results.
		testCtx.clearAssertions()

		// Execute step
		stepResult := h.executeStep(ctx, testCtx, step, ui)
		stepResult.Assertions = testCtx.getAssertions() // Collect assertions after step execution
		stepResults = append(stepResults, stepResult)

		if stepResult.Error != nil {
			ui.StepFailed(stepResult)
			result.Success = false
			result.FailedStep = step.Name
			result.Error = &StepError{
				StepName: step.Name,
				Err:      stepResult.Error,
			}
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.StepResults = stepResults

			// Stop TUI recording on failure if enabled
			if h.opts.RecordTUIDir != "" {
				if tuiSession, ok := testCtx.Get("tui_session").(*tui.Session); ok && tuiSession != nil {
					if err := tuiSession.StopRecording(); err != nil {
						ui.Error("Failed to stop TUI recording", err)
					} else {
						ui.Info("TUI Recording saved", fmt.Sprintf("Recording saved to %s", h.opts.RecordTUIDir))
					}
				}
			}

			return result, result.Error
		}

		ui.StepSuccess(stepResult)
	}

	// Success
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.StepResults = stepResults
	ui.ScenarioSuccess(scenario.Name, result.Duration)

	// Stop TUI recording on success if enabled
	// Note: For successful tests, we could optionally delete the recording
	// or keep it based on configuration. For now, we save it.
	if h.opts.RecordTUIDir != "" {
		if tuiSession, ok := testCtx.Get("tui_session").(*tui.Session); ok && tuiSession != nil {
			if err := tuiSession.StopRecording(); err != nil {
				ui.Error("Failed to stop TUI recording", err)
			}
		}
	}

	return result, nil
}

// RunAll executes multiple scenarios in sequence
func (h *Harness) RunAll(ctx context.Context, scenarios []*Scenario) ([]*Result, error) {
	results := make([]*Result, 0, len(scenarios))
	ui := NewUI(h.opts.Interactive, h.opts.Verbose || h.opts.VeryVerbose, h.opts.VeryVerbose)

	ulogHarness.Info("Starting scenario batch").
		Field("count", len(scenarios)).
		Pretty(fmt.Sprintf("\n%s Running %d scenarios\n%s", theme.IconDebugStart, len(scenarios), strings.Repeat("=", 60))).
		Emit()

	for i, scenario := range scenarios {
		select {
		case <-ctx.Done():
			return results, fmt.Errorf("batch execution cancelled: %w", ctx.Err())
		default:
		}

		ulogHarness.Progress("Running scenario").
			Field("current", i+1).
			Field("total", len(scenarios)).
			Field("name", scenario.Name).
			Pretty(fmt.Sprintf("\n[%d/%d] Running %s...", i+1, len(scenarios), scenario.Name)).
			Emit()

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

	prettyMsg := strings.Repeat("=", 60) + "\n"
	if failed == 0 {
		prettyMsg += fmt.Sprintf("%s All %d scenarios passed!", theme.IconSuccess, len(results))
		ulogHarness.Success("All scenarios passed").
			Field("total", len(results)).
			Pretty(prettyMsg).
			Emit()
	} else {
		prettyMsg += fmt.Sprintf("%s %d/%d scenarios failed\n", theme.IconError, failed, len(results))
		prettyMsg += fmt.Sprintf("%s Passed: %d\n", theme.IconSuccess, passed)
		prettyMsg += fmt.Sprintf("%s Failed: %d", theme.IconError, failed)
		ulogHarness.Error("Some scenarios failed").
			Field("passed", passed).
			Field("failed", failed).
			Field("total", len(results)).
			Pretty(prettyMsg).
			Emit()
	}

	if failed > 0 {
		return results, fmt.Errorf("%d scenarios failed", failed)
	}

	return results, nil
}

// createTempManager creates a temporary directory manager for a scenario
func (h *Harness) createTempManager(scenarioName string) (*fs.TempDirManager, error) {
	// If TestRootDir is set, use the existing directory
	if h.opts.TestRootDir != "" {
		return fs.NewTempDirManagerForExisting(h.opts.TestRootDir)
	}
	// Otherwise, create a new temporary directory
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
	var isReusedPane bool

	// Check for a cached pane ID
	if data, err := os.ReadFile(paneIDFile); err == nil {
		existingPaneID := strings.TrimSpace(string(data))
		if client.PaneExists(context.Background(), existingPaneID) {
			ui.Info("tmux", fmt.Sprintf("Reusing existing %s pane...", paneType))
			paneID = existingPaneID
			isReusedPane = true

			// Kill any running editor to ensure a clean state
			if currentCmd, err := client.GetPaneCommand(context.Background(), paneID); err == nil {
				currentCmd = strings.TrimSpace(currentCmd)
				if currentCmd == "nvim" || currentCmd == "vim" || currentCmd == "vi" {
					_ = client.SendKeys(context.Background(), paneID, "Escape", ":qa!", "C-m")
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
		// If this was a reused pane, the cache might be stale - try creating a new one
		if isReusedPane {
			ui.Info("tmux", fmt.Sprintf("Cached %s pane is stale, creating new one...", paneType))
			os.Remove(paneIDFile)
			newPaneID, createErr := client.SplitWindow(context.Background(), "", splitHorizontal, 0, "")
			if createErr != nil {
				return "", fmt.Errorf("failed to split tmux window for %s pane: %w", paneType, createErr)
			}
			paneID = newPaneID
			_ = os.WriteFile(paneIDFile, []byte(paneID), 0644)

			// Retry sending keys to the new pane
			if retryErr := client.SendKeys(context.Background(), paneID, commandStr, "C-m"); retryErr != nil {
				command.New("tmux", "kill-pane", "-t", paneID).Run()
				os.Remove(paneIDFile)
				return "", fmt.Errorf("failed to send keys to %s pane: %w", paneType, retryErr)
			}
			return paneID, nil
		}

		// Clean up and return error if sending keys fails on a new pane
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
		// Always open nvim in the shell pane for exploring test artifacts
		shellCommand := fmt.Sprintf("cd '%s' && nvim", ctx.RootDir)

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
				_ = tmuxClient.SendKeys(context.Background(), editorPaneID, jumpCmd, "C-m")
			}
		}
	}

	return nil
}


// killDaemonsInTree finds groved PID files anywhere under root and kills
// the recorded PIDs. Matches both the unscoped "groved.pid" and scoped
// "groved-<name>-<hash>.pid" variants. Best-effort: missing files,
// malformed contents, and already-dead PIDs are silently ignored.
func killDaemonsInTree(root string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "groved") || !strings.HasSuffix(name, ".pid") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if parseErr != nil || pid <= 1 {
			return nil
		}
		proc, findErr := os.FindProcess(pid)
		if findErr != nil {
			return nil
		}
		// SIGTERM first; the daemons handle it for graceful shutdown.
		// We do not wait — the process will exit asynchronously and the
		// pidfile/dir is about to be removed anyway.
		_ = proc.Signal(syscall.SIGTERM)
		return nil
	})
}
