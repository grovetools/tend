package harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/fs"
)

// Context carries state through a scenario execution
type Context struct {
	// Core paths
	RootDir     string // Root temporary directory for the scenario
	GroveBinary string // Path to the grove binary under test
	TestID      string // Unique ID for this test run

	// State management
	dirs   map[string]string      // Named directories created during test
	values map[string]interface{} // Generic key-value store for step communication
	
	// UI for displaying command output
	ui *UI
}

// Step represents a single action in a scenario
type Step struct {
	Name        string
	Description string
	Func        func(ctx *Context) error
}

// Scenario is a collection of steps defining a test
type Scenario struct {
	Name        string
	Description string
	Tags        []string
	Steps       []Step
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

	// Setup cleanup
	if !h.opts.NoCleanup {
		defer func() {
			ui.Cleanup()
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
	testCtx := &Context{
		RootDir:     tempMgr.BaseDir(),
		GroveBinary: groveBinary,
		TestID:      testID,
		dirs:        make(map[string]string),
		values:      make(map[string]interface{}),
		ui:          ui,
	}
	
	// Set the test ID in the UI for container filtering
	ui.SetTestID(testID)

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

		ui.StepStart(i+1, len(scenario.Steps), step.Name)

		// Interactive pause
		if h.opts.Interactive {
			if !ui.WaitForUser() {
				result.Success = false
				result.Error = fmt.Errorf("user cancelled at step: %s", step.Name)
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				result.StepResults = stepResults
				return result, result.Error
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
	// Check options first
	if h.opts.GroveBinary != "" {
		return h.opts.GroveBinary
	}
	
	// Check environment variable
	if bin := os.Getenv("GROVE_BINARY"); bin != "" {
		return bin
	}

	// Default to looking for 'grove' in PATH
	// This assumes gvm or proper PATH setup will make it available
	return "grove"
}

