package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/pkg/tmux"
	"github.com/grovetools/core/tui/theme"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/harness/reporters"
	"github.com/grovetools/tend/pkg/ui"
	"github.com/grovetools/tend/internal/tui/prunner"
	tea "github.com/charmbracelet/bubbletea"
)

var ulogRun = grovelogging.NewUnifiedLogger("grove-tend.cmd.run")

var (
	parallel            bool
	timeout             time.Duration
	noCleanup           bool
	outputFormat        string
	junitOutput         string
	jsonOutput          string
	tmuxSplit           bool
	nvim                bool
	debug               bool
	debugSession        bool
	debugServer         string
	testRootDirOverride string
	tmuxSocketOverride  string
	tmuxEditorTarget    string
	useRealDeps         []string
	includeLocal        bool
	explicitOnly        bool
	runSetup            bool
	runSteps            string
	recordTUIDir        string
	jobs                int
)

// newRunCmd creates the run command with the provided scenarios
func newRunCmd(allScenarios []*harness.Scenario) *cobra.Command {
	runCmd := &cobra.Command{
	Use:   "run [scenario...]",
	Short: "Run test scenarios",
	Long: `Run one or more test scenarios.

If no scenarios are specified, all scenarios in the scenarios directory will be run.
Scenarios can be filtered by tags using the --tags flag.

Examples:
  tend run                           # Run all scenarios
  tend run agent-isolation           # Run specific scenario
  tend run --tags=smoke              # Run scenarios tagged with 'smoke'
  tend run --interactive agent-*     # Run agent scenarios interactively
  tend run --parallel --timeout=5m   # Run with 5 minute timeout in parallel`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScenarios(cmd, args, allScenarios)
	},
	}

	runCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Run scenarios in parallel")
	runCmd.Flags().IntVarP(&jobs, "jobs", "j", 0, "Number of parallel jobs (default: half of CPU cores)")
	runCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Timeout for scenario execution")
	runCmd.Flags().BoolVar(&noCleanup, "no-cleanup", false, "Skip cleanup after scenario execution")
	runCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format (text, json, junit)")
	runCmd.Flags().StringVar(&junitOutput, "junit", "", "Write JUnit XML to file")
	runCmd.Flags().StringVar(&jsonOutput, "json", "", "Write JSON report to file")
	runCmd.Flags().BoolVar(&tmuxSplit, "tmux-split", false, "Split tmux window and cd to test directory")
	runCmd.Flags().BoolVar(&nvim, "nvim", false, "Start nvim in the new tmux split (requires --tmux-split)")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode (shorthand for -i --no-cleanup --tmux-split --nvim --very-verbose)")
	runCmd.Flags().BoolVar(&debugSession, "debug-session", false, "Enable debug mode in a new tmux session with windows (implies -i, --no-cleanup)")
	runCmd.Flags().StringVar(&debugServer, "server", "", "Debug session server type ('main' or 'dedicated'). Overrides config. (requires --debug-session)")
	runCmd.Flags().StringVar(&testRootDirOverride, "_test-root-dir", "", "Internal use: override the test root directory")
	runCmd.Flags().StringVar(&tmuxSocketOverride, "_tmux-socket", "", "Internal use: tmux socket name for debug session")
	runCmd.Flags().StringVar(&tmuxEditorTarget, "_tmux-editor", "", "Internal use: tmux editor window target")
	_ = runCmd.Flags().MarkHidden("_test-root-dir")
	_ = runCmd.Flags().MarkHidden("_tmux-socket")
	_ = runCmd.Flags().MarkHidden("_tmux-editor")
	runCmd.Flags().StringSliceVar(&useRealDeps, "use-real-deps", []string{}, "A comma-separated list of dependencies to use real binaries for instead of mocks (e.g., flow,cx). Use 'all' to swap all.")
	runCmd.Flags().BoolVar(&includeLocal, "include-local", false, "Include local-only scenarios even when in a CI environment")
	runCmd.Flags().BoolVar(&explicitOnly, "explicit", false, "Run only explicit-only scenarios (automatically enables --no-cleanup)")
	runCmd.Flags().StringVar(&recordTUIDir, "record-tui", "", "Directory to save TUI session recordings for failed tests")
	runCmd.Flags().BoolVar(&runSetup, "run-setup", false, "Run setup phase then pause (or switch to interactive if no setup)")
	runCmd.Flags().StringVar(&runSteps, "run-steps", "", "Run specific test steps at startup then pause (e.g., '1,2,3')")
	_ = runCmd.Flags().MarkHidden("run-setup")

	return runCmd
}

func runScenarios(cmd *cobra.Command, args []string, allScenarios []*harness.Scenario) error {
	ctx := cmd.Context()

	// Validate mutually exclusive flags
	if debug && debugSession {
		return fmt.Errorf("--debug and --debug-session cannot be used at the same time")
	}

	// Handle the debug flag shorthand
	if debug {
		interactive = true
		noCleanup = true
		tmuxSplit = true
		nvim = true
		veryVerbose = true
		verbose = true // --very-verbose implies --verbose
	}

	// Handle the debug-session flag shorthand
	if debugSession {
		interactive = true
		noCleanup = true
		veryVerbose = true
		verbose = true // --very-verbose implies --verbose
	}

	// If running explicit-only scenarios, automatically enable no-cleanup
	if explicitOnly {
		noCleanup = true
	}

	if nvim && !tmuxSplit {
		return fmt.Errorf("--nvim can only be used with --tmux-split")
	}
	
	// Create UI renderer
	renderer := ui.NewRenderer(os.Stdout, verbose, 80)
	
	
	// Filter scenarios based on --explicit flag
	var selectedScenarios []*harness.Scenario
	if explicitOnly {
		// When --explicit is used, only select explicit-only scenarios
		for _, scenario := range allScenarios {
			if scenario.ExplicitOnly {
				selectedScenarios = append(selectedScenarios, scenario)
			}
		}
		if len(selectedScenarios) == 0 {
			renderer.RenderInfo("No explicit-only scenarios found")
			return nil
		}
		renderer.RenderInfo(fmt.Sprintf("Running %d explicit-only scenario(s) (--no-cleanup enabled)", len(selectedScenarios)))
	} else {
		// Normal filtering
		selectedScenarios = filterScenarios(allScenarios, args, tags)
		
		// Filter ExplicitOnly scenarios when running all (and not using --explicit)
		if len(args) == 0 && len(selectedScenarios) > 0 {
			var filtered []*harness.Scenario
			var explicitCount int
			for _, scenario := range selectedScenarios {
				if !scenario.ExplicitOnly {
					filtered = append(filtered, scenario)
				} else {
					explicitCount++
				}
			}
			if explicitCount > 0 {
				renderer.RenderInfo(fmt.Sprintf("Skipped %d explicit-only scenario(s) (must be run by name or use --explicit)", explicitCount))
			}
			selectedScenarios = filtered
		}
	}
	
	// Filter LocalOnly scenarios in CI
	if harness.IsCI() && !includeLocal && len(selectedScenarios) > 0 {
		var filtered []*harness.Scenario
		var localCount int
		for _, scenario := range selectedScenarios {
			if !scenario.LocalOnly {
				filtered = append(filtered, scenario)
			} else {
				localCount++
			}
		}
		if localCount > 0 {
			renderer.RenderInfo(fmt.Sprintf("Skipped %d local-only scenario(s) in CI environment (use --include-local to override)", localCount))
		}
		selectedScenarios = filtered
	}
	
	if len(selectedScenarios) == 0 {
		renderer.RenderInfo("No scenarios match the specified criteria")
		return nil
	}

	// Debug session orchestration - creates a tmux session and re-executes tend
	// Creates 5 windows:
	//   1. runner        - Runs the actual test with sandboxed environment
	//   2. editor_test_dir   - nvim viewing test directory with real user environment
	//   3. editor_test_steps - nvim for editing test steps with real user environment
	//   4. term          - Interactive shell with sandboxed environment
	//   5. logs          - Log viewer with sandboxed environment
	if debugSession {
		if len(selectedScenarios) != 1 {
			return fmt.Errorf("--debug-session requires exactly one scenario to be specified, but found %d", len(selectedScenarios))
		}
		scenario := selectedScenarios[0]

		// Create the temporary directory for the test
		testRootDir, err := os.MkdirTemp("", "tend-debug-*")
		if err != nil {
			return fmt.Errorf("failed to create temp dir for debug session: %w", err)
		}
		renderer.RenderInfo(fmt.Sprintf("Debug session root directory: %s", testRootDir))

		// Create the sandboxed home directory structure
		homeDir := filepath.Join(testRootDir, "home")
		configDir := filepath.Join(homeDir, ".config")
		dataDir := filepath.Join(homeDir, ".local", "share")
		cacheDir := filepath.Join(homeDir, ".cache")
		for _, dir := range []string{homeDir, configDir, dataDir, cacheDir} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create sandbox dir %s: %w", dir, err)
			}
		}

		// Create short runtime dir for Unix socket paths (groved, etc.)
		// Unix sockets have ~104 char path limit on macOS, so we must use /tmp directly
		runtimeDir, err := fs.CreateShortTempDir("tend-debug-")
		if err != nil {
			return fmt.Errorf("failed to create short runtime dir for debug session: %w", err)
		}
		renderer.RenderInfo(fmt.Sprintf("Debug session runtime directory: %s", runtimeDir))

		// 1. Prepare Environments
		// Sandboxed environment for 'runner', 'term', and 'logs' windows
		sandboxEnvSlice := []string{
			fmt.Sprintf("HOME=%s", homeDir),
			fmt.Sprintf("XDG_CONFIG_HOME=%s", configDir),
			fmt.Sprintf("XDG_DATA_HOME=%s", dataDir),
			fmt.Sprintf("XDG_CACHE_HOME=%s", cacheDir),
			fmt.Sprintf("XDG_RUNTIME_DIR=%s", runtimeDir),
		}

		// Real environment for editor windows to ensure user's nvim config loads
		realHome, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not get user home directory: %w", err)
		}
		realEnvSlice := []string{
			fmt.Sprintf("HOME=%s", realHome),
		}
		// Propagate user's XDG dirs if they exist, so Neovim finds its config
		if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
			realEnvSlice = append(realEnvSlice, fmt.Sprintf("XDG_CONFIG_HOME=%s", xdgConfigHome))
		}
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			realEnvSlice = append(realEnvSlice, fmt.Sprintf("XDG_DATA_HOME=%s", xdgDataHome))
		}
		if xdgCacheHome := os.Getenv("XDG_CACHE_HOME"); xdgCacheHome != "" {
			realEnvSlice = append(realEnvSlice, fmt.Sprintf("XDG_CACHE_HOME=%s", xdgCacheHome))
		}

		// Determine server mode from config and flag
		// TODO: Load config to get default server mode
		// Default to "main" for better integration with tmux workflows
		serverMode := "main"
		if debugServer != "" {
			serverMode = debugServer
		}

		// Set up tmux client based on server mode
		var client *tmux.Client
		var sessionName string
		sanitizedName := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(scenario.Name, "_")

		if serverMode == "main" {
			// New behavior: use main server with namespaced session name
			// TODO: Get workspace identifier when available
			// For now, use simple naming with "tend_" prefix
			sessionName = fmt.Sprintf("tend_%s", sanitizedName)
			client, err = tmux.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create tmux client: %w", err)
			}
		} else {
			// Old behavior: use dedicated server with simple session name
			socketName := "tend-debug"
			sessionName = sanitizedName
			client, err = tmux.NewClientWithSocket(socketName)
			if err != nil {
				return fmt.Errorf("failed to create tmux client: %w", err)
			}
		}

		// Kill any existing session with the same name
		_ = client.KillSession(ctx, sessionName)

		// The project directory is the CWD of the tend process
		projectRoot := rootDir
		if projectRoot == "" {
			projectRoot, _ = os.Getwd()
		}

		// Add project bin directory to PATH for sandboxed environment
		projectBinDir := filepath.Join(projectRoot, "bin")
		currentPath := os.Getenv("PATH")
		sandboxPath := fmt.Sprintf("%s:%s", projectBinDir, currentPath)
		sandboxEnvSlice = append(sandboxEnvSlice, fmt.Sprintf("PATH=%s", sandboxPath))

		// 2. Create Initial Session with 'runner' Window
		launchOpts := tmux.LaunchOptions{
			SessionName:      sessionName,
			WorkingDirectory: projectRoot,
			WindowName:       "runner",
			WindowIndex:      -1,
			Panes: []tmux.PaneOptions{
				{}, // Start with an empty shell
			},
		}
		if err := client.Launch(ctx, launchOpts); err != nil {
			return fmt.Errorf("failed to launch debug session: %w", err)
		}

		// 3. Reconstruct the command, replacing --debug-session with interactive flags
		newArgs := []string{}
		hasRunSteps := false
		for _, arg := range os.Args[1:] {
			if arg != "--debug-session" {
				newArgs = append(newArgs, arg)
				// Check if user provided --run-steps
				if strings.HasPrefix(arg, "--run-steps") {
					hasRunSteps = true
				}
			}
		}
		// The editor target will point to editor_test_steps window
		editorTarget := sessionName + ":editor_test_steps"

		// If user didn't provide --run-steps, default to --run-setup behavior
		if !hasRunSteps {
			newArgs = append(newArgs, "--run-setup") // Run setup then halt, or switch to interactive if no setup
		}

		newArgs = append(newArgs,
			"--no-cleanup", "--very-verbose",
			"--_test-root-dir="+testRootDir,
			"--_tmux-editor="+editorTarget,
		)
		// Only add socket override for dedicated server mode
		if serverMode == "dedicated" {
			newArgs = append(newArgs, "--_tmux-socket=tend-debug")
		}
		// Prepending with a space is a convention for many shells (bash, zsh, fish)
		// to not save the command in history.
		tendCmd := " " + os.Args[0] + " " + strings.Join(newArgs, " ")

		runnerTarget := sessionName + ":runner"
		if err := client.SendKeys(ctx, runnerTarget, tendCmd, "C-m"); err != nil {
			return fmt.Errorf("failed to send command to runner pane: %w", err)
		}

		// 4. Create additional windows

		// Window: editor_test_dir - nvim viewing test directory with real user env
		if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
			Target:     sessionName,
			WindowName: "editor_test_dir",
			WorkingDir: testRootDir,
			Env:        realEnvSlice,
			Command:    "nvim .",
		}); err != nil {
			return fmt.Errorf("failed to create editor_test_dir window: %w", err)
		}

		// Window: editor_test_steps - nvim for editing test steps with real user env
		// Create window with empty shell first
		if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
			Target:     sessionName,
			WindowName: "editor_test_steps",
			WorkingDir: projectRoot,
			Env:        realEnvSlice,
			Command:    "", // Empty shell, we'll send keys next
		}); err != nil {
			return fmt.Errorf("failed to create editor_test_steps window: %w", err)
		}

		// Send nvim command to open the file
		// Since WorkingDir is projectRoot, relative paths in scenario.File will work
		editorStepsTarget := sessionName + ":editor_test_steps"
		if scenario.File != "" {
			var nvimCmd string
			if scenario.Line > 0 {
				nvimCmd = fmt.Sprintf("nvim +%d %s", scenario.Line, scenario.File)
			} else {
				nvimCmd = fmt.Sprintf("nvim %s", scenario.File)
			}
			if err := client.SendKeys(ctx, editorStepsTarget, nvimCmd, "C-m"); err != nil {
				return fmt.Errorf("failed to send nvim command: %w", err)
			}
		} else {
			// No file, open nvim at tests/e2e directory for easy navigation to scenarios
			if err := client.SendKeys(ctx, editorStepsTarget, "nvim tests/e2e", "C-m"); err != nil {
				return fmt.Errorf("failed to send nvim command: %w", err)
			}
		}

		// Window: term - Interactive shell with sandboxed environment
		// Build export commands from sandboxEnvSlice and exec the shell
		var exports []string
		for _, env := range sandboxEnvSlice {
			exports = append(exports, fmt.Sprintf("export %s", env))
		}
		termCmd := fmt.Sprintf("%s && exec $SHELL", strings.Join(exports, " && "))
		if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
			Target:     sessionName,
			WindowName: "term",
			WorkingDir: testRootDir,
			Command:    termCmd,
		}); err != nil {
			return fmt.Errorf("failed to create term window: %w", err)
		}

		// Window: logs - Log viewer with sandboxed environment
		if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
			Target:     sessionName,
			WindowName: "logs",
			WorkingDir: testRootDir,
			Env:        sandboxEnvSlice,
			Command:    "sh -c 'core logs --tui || read'", // Keep window open if command fails
		}); err != nil {
			return fmt.Errorf("failed to create logs window: %w", err)
		}

		// 5. Finalize Session State
		// Select the runner window to be the active one on attach
		if err := client.SelectWindow(ctx, runnerTarget); err != nil {
			return fmt.Errorf("failed to select runner window: %w", err)
		}

		// Print instructions for attaching based on server mode
		if serverMode == "main" {
			renderer.RenderInfo(fmt.Sprintf("Debug session '%s' created in main tmux server", sessionName))
			renderer.RenderInfo(fmt.Sprintf("To attach: tmux attach -t %s", sessionName))
			renderer.RenderInfo("List all tend sessions: tend sessions")
		} else {
			renderer.RenderInfo(fmt.Sprintf("Debug session '%s' created in tmux server 'tend-debug'", sessionName))
			renderer.RenderInfo(fmt.Sprintf("To attach: tmux -L tend-debug attach -t %s", sessionName))
			renderer.RenderInfo("List sessions: tmux -L tend-debug ls")
		}

		return nil
	}

	// When --format json, redirect all pretty/styled output to stderr so that
	// only the JSON report is written to stdout.
	if outputFormat == "json" {
		grovelogging.SetGlobalOutput(os.Stderr)
		renderer = ui.NewRenderer(os.Stderr, verbose, 80)
	}

	// Display selected scenarios
	scenarioNames := make([]string, len(selectedScenarios))
	for i, scenario := range selectedScenarios {
		scenarioNames[i] = scenario.Name
	}
	renderer.RenderList(fmt.Sprintf("Running %d scenario(s):", len(selectedScenarios)), scenarioNames)

	// Create harness options
	opts := harness.Options{
		Verbose:          verbose,
		VeryVerbose:      veryVerbose,
		Interactive:      interactive,
		NoCleanup:        noCleanup,
		Timeout:          timeout,
		GroveBinary:      groveBinary,
		RootDir:          rootDir,
		MonitorDocker:    monitorDocker,
		DockerFilter:     dockerFilter,
		TmuxSplit:        tmuxSplit,
		Nvim:             nvim,
		UseRealDeps:      useRealDeps,
		TestRootDir:      testRootDirOverride,
		TmuxSocket:       tmuxSocketOverride,
		TmuxEditorTarget: tmuxEditorTarget,
		RunSetup:         runSetup,
		RunSteps:         runSteps,
		RecordTUIDir:     recordTUIDir,
	}
	
	// Configure for CI if needed
	harness.ConfigureForCI(&opts)
	
	// Setup CI environment
	harness.SetupCIEnvironment()
	
	// Create harness
	h := harness.New(opts)
	
	// Determine if we can use the interactive TUI for parallel mode.
	// Fall back to sequential runner when stdout is not a terminal (e.g., piped
	// to a file, captured by a subprocess, or running inside a non-interactive agent).
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	useTUI := parallel && isTTY && outputFormat != "json"

	if parallel && !isTTY && outputFormat != "json" {
		fmt.Fprintf(os.Stderr, "Note: stdout is not a TTY, running parallel tests without TUI\n")
	}

	// Run scenarios
	var results []*harness.Result
	var scenarioStates []*prunner.ScenarioState
	var totalSuccess int
	var err error

	if useTUI {
		results, scenarioStates, err = runScenariosParallel(ctx, h, selectedScenarios, renderer, rootDir)

		// Clean up orphaned tmux servers from parallel test runs
		// Parallel tests use --no-cleanup to avoid interference, so we clean up here
		if err := cleanupOrphanedTmuxServers(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cleanup orphaned tmux servers: %v\n", err)
		}
	} else if parallel {
		results, err = runScenariosParallelHeadless(ctx, selectedScenarios, rootDir)

		// Clean up orphaned tmux servers from parallel test runs
		if cleanupErr := cleanupOrphanedTmuxServers(ctx); cleanupErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cleanup orphaned tmux servers: %v\n", cleanupErr)
		}
	} else {
		results, err = runScenariosSequential(ctx, h, selectedScenarios, renderer)
	}

	if err != nil {
		renderer.RenderError(err)
		return err
	}

	// Count successes
	for _, result := range results {
		if result.Success {
			totalSuccess++
		}
	}

	// Write file-based reports (--json <file>, --junit <file>)
	if err := writeReports(results); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write reports: %v\n", err)
	}

	// When --format json is set, write JSON to stdout and skip styled output
	if outputFormat == "json" {
		reporter := reporters.NewJSONReporter(true, true)
		if err := reporter.WriteReport(os.Stdout, results); err != nil {
			return fmt.Errorf("writing json to stdout: %w", err)
		}
	} else {
		// Display styled summary
		if !parallel || !useTUI {
			renderFinalSummary(renderer, results, totalSuccess, len(selectedScenarios))
		} else {
			renderFinalSummary(renderer, results, totalSuccess, len(selectedScenarios))
			if totalSuccess < len(selectedScenarios) {
				printParallelFailureDetails(scenarioStates)
			}
		}
	}

	// Exit with error code if any scenarios failed
	if totalSuccess < len(selectedScenarios) {
		os.Exit(1)
	}

	return nil
}

func printParallelFailureDetails(states []*prunner.ScenarioState) {
	var failedScenarios []*prunner.ScenarioState
	for _, s := range states {
		if s.Status() == prunner.StatusFailure {
			failedScenarios = append(failedScenarios, s)
		}
	}

	if len(failedScenarios) == 0 {
		return
	}

	prettyMsg := "\n" + strings.Repeat("=", 80) + "\n"
	prettyMsg += fmt.Sprintf("%s Test run failed: %d/%d scenarios failed\n", theme.IconError, len(failedScenarios), len(states))
	prettyMsg += strings.Repeat("=", 80)

	ulogRun.Error("Test run failed").
		Field("failed_count", len(failedScenarios)).
		Field("total_count", len(states)).
		Pretty(prettyMsg).
		Emit()

	for _, s := range failedScenarios {
		prettyMsg := fmt.Sprintf("\n%s %s (failed in %v)\n%s",
			theme.IconError, s.Scenario().Name, s.Duration().Round(time.Millisecond),
			strings.Repeat("-", 80))

		if s.Output() != "" {
			// Trim leading/trailing whitespace from output for cleaner presentation
			prettyMsg += "\n" + strings.TrimSpace(s.Output())
		}

		ulogRun.Error("Scenario failed").
			Field("scenario", s.Scenario().Name).
			Field("duration", s.Duration()).
			Pretty(prettyMsg).
			Emit()
	}
}

func runScenariosSequential(ctx context.Context, h *harness.Harness, scenarios []*harness.Scenario, renderer *ui.Renderer) ([]*harness.Result, error) {
	var results []*harness.Result
	
	for i, scenario := range scenarios {
		renderer.RenderProgress(i, len(scenarios))
		
		result, _ := runSingleScenario(ctx, h, scenario, renderer)
		// Ignore the error - we want to continue running all scenarios
		// The result object contains the success/failure information
		
		results = append(results, result)
	}
	
	return results, nil
}

func runScenariosParallel(ctx context.Context, h *harness.Harness, scenarios []*harness.Scenario, renderer *ui.Renderer, projectRoot string) ([]*harness.Result, []*prunner.ScenarioState, error) {
	model := prunner.New(scenarios, projectRoot, jobs)
	// Use stdin/stdout instead of trying to open /dev/tty
	// This allows the TUI to work in various contexts (tmux, pipes, etc.)
	p := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	finalModel, err := p.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("error running parallel test runner: %w", err)
	}

	// The TUI has finished, get the results from the final model
	runnerModel := finalModel.(prunner.Model)
	results := runnerModel.Results()
	states := runnerModel.ScenarioStates()
	return results, states, nil
}

func runScenariosParallelHeadless(ctx context.Context, scenarios []*harness.Scenario, projectRoot string) ([]*harness.Result, error) {
	eventsChan := prunner.Run(ctx, scenarios, projectRoot, jobs)
	var results []*harness.Result

	for event := range eventsChan {
		switch event.Type {
		case "start":
			ulogRun.Info("Scenario started").
				Field("scenario", scenarios[event.Index].Name).
				Emit()
		case "finish":
			results = append(results, event.Result)

			if event.Result.Success {
				ulogRun.Success("Scenario completed").
					Field("scenario", event.Result.ScenarioName).
					Field("duration", event.Result.Duration).
					Emit()
			} else {
				errMsg := ""
				if event.Result.Error != nil {
					errMsg = event.Result.Error.Error()
				}
				ulogRun.Error("Scenario failed").
					Field("scenario", event.Result.ScenarioName).
					Field("duration", event.Result.Duration).
					Field("error", errMsg).
					Emit()

				// Print raw output to aid debugging in headless/CI environments
				if output := strings.TrimSpace(event.Output); output != "" {
					fmt.Fprintf(os.Stderr, "\n--- Output: %s ---\n%s\n---\n\n", event.Result.ScenarioName, output)
				}
			}
		}
	}
	return results, nil
}

func runSingleScenario(ctx context.Context, h *harness.Harness, scenario *harness.Scenario, renderer *ui.Renderer) (*harness.Result, error) {
	renderer.RenderScenarioStart(scenario)
	
	result, err := h.Run(ctx, scenario)
	
	renderer.RenderScenarioEnd(result)
	
	return result, err
}

func filterScenarios(scenarios []*harness.Scenario, names []string, tags []string) []*harness.Scenario {
	var filtered []*harness.Scenario
	
	for _, scenario := range scenarios {
		// Filter by name patterns if specified
		if len(names) > 0 {
			nameMatch := false
			for _, pattern := range names {
				if matched, _ := filepath.Match(pattern, scenario.Name); matched {
					nameMatch = true
					break
				}
			}
			if !nameMatch {
				continue
			}
		}
		
		// Filter by tags if specified
		if len(tags) > 0 {
			tagMatch := false
			for _, requiredTag := range tags {
				for _, scenarioTag := range scenario.Tags {
					if scenarioTag == requiredTag {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue
			}
		}
		
		filtered = append(filtered, scenario)
	}
	
	return filtered
}

func renderFinalSummary(renderer *ui.Renderer, results []*harness.Result, success, total int) {
	ulogRun.Info("Final summary separator").
		Pretty("").
		PrettyOnly().
		Emit()

	if success == total {
		renderer.RenderSuccess(fmt.Sprintf("All %d scenario(s) passed!", total))
	} else {
		renderer.RenderError(fmt.Errorf("%d of %d scenario(s) failed", total-success, total))
	}

	// Create results table
	ulogRun.Info("Results table separator").
		Pretty("").
		PrettyOnly().
		Emit()
	
	// Build table data
	headers := []string{"STATUS", "SCENARIO", "DURATION", "DETAILS"}
	var rows [][]string
	
	for _, result := range results {
		status := theme.IconSuccess + " PASS"
		statusStyle := theme.DefaultTheme.Success
		if !result.Success {
			status = theme.IconError + " FAIL"
			statusStyle = theme.DefaultTheme.Error
		}

		details := "-"
		if !result.Success && result.FailedStep != "" {
			details = result.FailedStep
		}
		
		row := []string{
			statusStyle.Render(status),
			result.ScenarioName,
			result.Duration.Round(time.Millisecond).String(),
			details,
		}
		rows = append(rows, row)
	}
	
	// Create table renderer
	re := lipgloss.NewRenderer(os.Stdout)
	
	// Define styles
	baseStyle := re.NewStyle().Padding(0, 1)
	headerStyle := baseStyle.Copy().Bold(true).Foreground(lipgloss.Color("#5FAFFF"))
	
	// Create the table
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))).
		Headers(headers...).
		Rows(rows...)
	
	// Apply styling
	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == 0 {
			return headerStyle
		}
		// Apply base style to all cells
		return baseStyle
	})

	ulogRun.Info("Test results").
		Field("success_count", success).
		Field("total_count", total).
		Pretty(t.String()).
		PrettyOnly().
		Emit()
}

// writeReports writes test results in various formats
func writeReports(results []*harness.Result) error {
	// JUnit output
	if junitOutput != "" {
		file, err := os.Create(junitOutput)
		if err != nil {
			return fmt.Errorf("creating junit file: %w", err)
		}
		defer file.Close()
		
		reporter := reporters.NewJUnitReporter("Grove Tend Tests")
		if err := reporter.WriteReport(file, results); err != nil {
			return fmt.Errorf("writing junit report: %w", err)
		}
	}
	
	// JSON output
	if jsonOutput != "" {
		file, err := os.Create(jsonOutput)
		if err != nil {
			return fmt.Errorf("creating json file: %w", err)
		}
		defer file.Close()
		
		reporter := reporters.NewJSONReporter(true, true)
		if err := reporter.WriteReport(file, results); err != nil {
			return fmt.Errorf("writing json report: %w", err)
		}
	}
	
	// GitHub Actions annotations
	if harness.DetectCIProvider() == harness.CIProviderGitHubActions {
		reporter := reporters.NewGitHubReporter()
		if err := reporter.WriteReport(os.Stdout, results); err != nil {
			return fmt.Errorf("writing github annotations: %w", err)
		}
	}

	return nil
}

// cleanupOrphanedTmuxServers kills all tend-test tmux servers and removes their socket files.
// This is called after parallel test runs to clean up servers left by --no-cleanup.
func cleanupOrphanedTmuxServers(ctx context.Context) error {
	// Get the tmux socket directory
	socketDir := fmt.Sprintf("/tmp/tmux-%d", os.Getuid())

	// Check if directory exists
	entries, err := os.ReadDir(socketDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No socket directory, nothing to clean
		}
		return fmt.Errorf("failed to read socket directory: %w", err)
	}

	// Find all tend-test sockets
	var tendSockets []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "tend-test-") {
			tendSockets = append(tendSockets, entry.Name())
		}
	}

	if len(tendSockets) == 0 {
		return nil // Nothing to clean up
	}

	ulogRun.Debug("Cleaning up orphaned tmux servers").
		Field("count", len(tendSockets)).
		Emit()

	// Kill servers and remove socket files
	var cleanupErrors []string
	for _, socketName := range tendSockets {
		socketPath := filepath.Join(socketDir, socketName)

		// Try to kill the server (if running)
		client, err := tmux.NewClientWithSocket(socketName)
		if err == nil {
			if err := client.KillServer(ctx); err != nil {
				// Log but continue - server might already be dead
				ulogRun.Debug("Failed to kill tmux server during cleanup").
					Field("socket", socketName).
					Field("error", err.Error()).
					Emit()
			}
		}

		// Remove the socket file
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("%s: %v", socketName, err))
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("some socket files could not be removed: %s", strings.Join(cleanupErrors, "; "))
	}

	ulogRun.Debug("Successfully cleaned up orphaned tmux servers").
		Field("count", len(tendSockets)).
		Emit()

	return nil
}