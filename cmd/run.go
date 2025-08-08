package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattsolo1/grove-tend/internal/harness"
	"github.com/mattsolo1/grove-tend/internal/harness/reporters"
	"github.com/mattsolo1/grove-tend/pkg/ui"
	"github.com/mattsolo1/grove-tend/scenarios"
)

var (
	parallel    bool
	timeout     time.Duration
	noCleanup   bool
	scenarioDir string
	outputFormat string
	junitOutput  string
	jsonOutput   string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
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
	RunE: runScenarios,
}

func init() {
	runCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Run scenarios in parallel")
	runCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Timeout for scenario execution")
	runCmd.Flags().BoolVar(&noCleanup, "no-cleanup", false, "Skip cleanup after scenario execution")
	runCmd.Flags().StringVar(&scenarioDir, "scenario-dir", "scenarios", "Directory containing scenarios")
	runCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format (text, json, junit)")
	runCmd.Flags().StringVar(&junitOutput, "junit", "", "Write JUnit XML to file")
	runCmd.Flags().StringVar(&jsonOutput, "json", "", "Write JSON report to file")
}

func runScenarios(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	// Create UI renderer
	renderer := ui.NewRenderer(os.Stdout, verbose, 80)
	
	// Load scenarios
	scenarioLoader := scenarios.NewLoader(filepath.Join(rootDir, scenarioDir))
	allScenarios, err := scenarioLoader.LoadAll()
	if err != nil {
		renderer.RenderError(fmt.Errorf("failed to load scenarios: %w", err))
		return err
	}
	
	// Filter scenarios
	selectedScenarios := filterScenarios(allScenarios, args, tags)
	
	if len(selectedScenarios) == 0 {
		renderer.RenderInfo("No scenarios match the specified criteria")
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
		Verbose:       verbose,
		VeryVerbose:   veryVerbose,
		Interactive:   interactive,
		NoCleanup:     noCleanup,
		Timeout:       timeout,
		GroveBinary:   groveBinary,
		RootDir:       rootDir,
		MonitorDocker: monitorDocker,
		DockerFilter:  dockerFilter,
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
		
		result, err := runSingleScenario(ctx, h, scenario, renderer)
		if err != nil {
			return results, err
		}
		
		results = append(results, result)
		
		// Stop on first failure in sequential mode if not in interactive mode
		if !result.Success && !interactive {
			break
		}
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
	
	// Show individual results
	for _, result := range results {
		status := "PASS"
		if !result.Success {
			status = "FAIL"
		}
		
		fmt.Printf("  %s %s (%v)\n", 
			status, 
			result.ScenarioName, 
			result.Duration.Round(time.Millisecond))
	}
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