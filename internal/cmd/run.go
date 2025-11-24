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
	"github.com/mattsolo1/grove-core/pkg/tmux"
	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/spf13/cobra"

	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/harness/reporters"
	"github.com/mattsolo1/grove-tend/pkg/ui"
)

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
	testRootDirOverride string
	tmuxSocketOverride  string
	tmuxEditorTarget    string
	useRealDeps         []string
	includeLocal        bool
	explicitOnly        bool
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
	runCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Timeout for scenario execution")
	runCmd.Flags().BoolVar(&noCleanup, "no-cleanup", false, "Skip cleanup after scenario execution")
	runCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format (text, json, junit)")
	runCmd.Flags().StringVar(&junitOutput, "junit", "", "Write JUnit XML to file")
	runCmd.Flags().StringVar(&jsonOutput, "json", "", "Write JSON report to file")
	runCmd.Flags().BoolVar(&tmuxSplit, "tmux-split", false, "Split tmux window and cd to test directory")
	runCmd.Flags().BoolVar(&nvim, "nvim", false, "Start nvim in the new tmux split (requires --tmux-split)")
	runCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode (shorthand for -i --no-cleanup --tmux-split --nvim --very-verbose)")
	runCmd.Flags().BoolVar(&debugSession, "debug-session", false, "Enable debug mode in a new tmux session with windows (implies -i, --no-cleanup)")
	runCmd.Flags().StringVar(&testRootDirOverride, "_test-root-dir", "", "Internal use: override the test root directory")
	runCmd.Flags().StringVar(&tmuxSocketOverride, "_tmux-socket", "", "Internal use: tmux socket name for debug session")
	runCmd.Flags().StringVar(&tmuxEditorTarget, "_tmux-editor", "", "Internal use: tmux editor window target")
	_ = runCmd.Flags().MarkHidden("_test-root-dir")
	_ = runCmd.Flags().MarkHidden("_tmux-socket")
	_ = runCmd.Flags().MarkHidden("_tmux-editor")
	runCmd.Flags().StringSliceVar(&useRealDeps, "use-real-deps", []string{}, "A comma-separated list of dependencies to use real binaries for instead of mocks (e.g., flow,cx). Use 'all' to swap all.")
	runCmd.Flags().BoolVar(&includeLocal, "include-local", false, "Include local-only scenarios even when in a CI environment")
	runCmd.Flags().BoolVar(&explicitOnly, "explicit", false, "Run only explicit-only scenarios (automatically enables --no-cleanup)")

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

	// Debug session orchestration - creates a dedicated tmux server and re-executes tend
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

		// 1. Prepare Environments
		// Sandboxed environment for 'runner' and 'logs' windows
		sandboxEnvSlice := []string{
			fmt.Sprintf("HOME=%s", homeDir),
			fmt.Sprintf("XDG_CONFIG_HOME=%s", configDir),
			fmt.Sprintf("XDG_DATA_HOME=%s", dataDir),
			fmt.Sprintf("XDG_CACHE_HOME=%s", cacheDir),
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

		// Set up a dedicated tmux server for all debug sessions
		socketName := "tend-debug"
		sanitizedName := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(scenario.Name, "_")
		sessionName := sanitizedName

		client, err := tmux.NewClientWithSocket(socketName)
		if err != nil {
			return fmt.Errorf("failed to create tmux client: %w", err)
		}

		// Kill any existing session with the same name
		_ = client.KillSession(ctx, sessionName)

		// The project directory is the CWD of the tend process
		projectRoot := rootDir
		if projectRoot == "" {
			projectRoot, _ = os.Getwd()
		}

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
		for _, arg := range os.Args[1:] {
			if arg != "--debug-session" {
				newArgs = append(newArgs, arg)
			}
		}
		// The editor target will point to editor_test_steps window
		editorTarget := sessionName + ":editor_test_steps"
		newArgs = append(newArgs,
			"-i", "--no-cleanup", "--very-verbose",
			"--_test-root-dir="+testRootDir,
			"--_tmux-socket="+socketName,
			"--_tmux-editor="+editorTarget,
		)
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
		// Build nvim command with optional file and line jump
		var editorCmd string
		if scenario.File != "" {
			if scenario.Line > 0 {
				editorCmd = fmt.Sprintf("nvim +%d '%s'", scenario.Line, scenario.File)
			} else {
				editorCmd = fmt.Sprintf("nvim '%s'", scenario.File)
			}
		} else {
			editorCmd = "nvim"
		}
		if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
			Target:     sessionName,
			WindowName: "editor_test_steps",
			WorkingDir: projectRoot,
			Env:        realEnvSlice,
			Command:    editorCmd,
		}); err != nil {
			return fmt.Errorf("failed to create editor_test_steps window: %w", err)
		}

		// Window: term - Interactive shell with sandboxed environment
		if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
			Target:     sessionName,
			WindowName: "term",
			WorkingDir: testRootDir,
			Env:        sandboxEnvSlice,
			Command:    "", // Empty command for interactive shell
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

		// Print instructions for attaching
		renderer.RenderInfo(fmt.Sprintf("Debug session '%s' created in tmux server '%s'", sessionName, socketName))
		renderer.RenderInfo(fmt.Sprintf("To attach: tmux -L %s attach -t %s", socketName, sessionName))
		renderer.RenderInfo(fmt.Sprintf("List sessions: tmux -L %s ls", socketName))

		return nil
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
	}
	
	// Configure for CI if needed
	harness.ConfigureForCI(&opts)
	
	// Setup CI environment
	harness.SetupCIEnvironment()
	
	// Create harness
	h := harness.New(opts)
	
	// Run scenarios
	var results []*harness.Result
	var totalSuccess int
	var err error
	
	if parallel {
		results, err = runScenariosParallel(ctx, h, selectedScenarios, renderer)
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
	
	// Write reports
	if err := writeReports(results); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write reports: %v\n", err)
	}
	
	// Display summary
	renderFinalSummary(renderer, results, totalSuccess, len(selectedScenarios))
	
	// Exit with error code if any scenarios failed
	if totalSuccess < len(selectedScenarios) {
		os.Exit(1)
	}
	
	return nil
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

func runScenariosParallel(ctx context.Context, h *harness.Harness, scenarios []*harness.Scenario, renderer *ui.Renderer) ([]*harness.Result, error) {
	// For now, implement as sequential since parallel execution requires more complex coordination
	// TODO: Implement true parallel execution with goroutines and channels
	renderer.RenderInfo("Parallel execution not yet implemented, running sequentially")
	return runScenariosSequential(ctx, h, scenarios, renderer)
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
	fmt.Println()
	
	if success == total {
		renderer.RenderSuccess(fmt.Sprintf("All %d scenario(s) passed!", total))
	} else {
		renderer.RenderError(fmt.Errorf("%d of %d scenario(s) failed", total-success, total))
	}
	
	// Create results table
	fmt.Println()
	
	// Build table data
	headers := []string{"STATUS", "SCENARIO", "DURATION", "DETAILS"}
	var rows [][]string
	
	for _, result := range results {
		status := "✅ PASS"
		statusStyle := theme.DefaultTheme.Success
		if !result.Success {
			status = "❌ FAIL"
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
	
	fmt.Println(t)
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