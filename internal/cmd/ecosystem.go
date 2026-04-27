package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/tui/components/table"
	"github.com/grovetools/core/tui/theme"
	"github.com/spf13/cobra"

	"github.com/grovetools/tend/internal/tui/e_runner"
	"github.com/grovetools/tend/pkg/command"
	"github.com/grovetools/tend/pkg/harness/reporters"
	"github.com/grovetools/tend/pkg/ui"
)

var ulog = grovelogging.NewUnifiedLogger("grove-tend.cmd.ecosystem")

func newEcosystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ecosystem",
		Short: "Run tests across the entire Grove ecosystem",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Discover and run all E2E test suites",
		RunE:  runEcosystemTests,
	}

	runCmd.Flags().BoolP("parallel", "p", false, "Run ecosystem tests in parallel using a TUI")
	runCmd.Flags().IntP("jobs", "j", 0, "Number of parallel jobs (default: half of CPU cores)")

	cmd.AddCommand(runCmd)
	return cmd
}

type testResult struct {
	ProjectName string
	Success     bool
	ReportPath  string
	Error       error
	Duration    time.Duration
}

func runEcosystemTests(cmd *cobra.Command, args []string) error {
	renderer := ui.NewRenderer(os.Stdout, verbose, 120)
	renderer.RenderInfo("Starting ecosystem test run...")

	root, err := findEcosystemRoot()
	if err != nil {
		return fmt.Errorf("could not find ecosystem root: %w", err)
	}
	renderer.RenderInfo(fmt.Sprintf("Found ecosystem root at: %s", root))

	packages, err := parseRootMakefile(root)
	if err != nil {
		return fmt.Errorf("could not parse root Makefile: %w", err)
	}

	testSuites := discoverTestSuites(root, packages)
	if len(testSuites) == 0 {
		renderer.RenderInfo("No tend test suites found in the ecosystem.")
		return nil
	}

	renderer.RenderInfo(fmt.Sprintf("Found %d test suites to run.", len(testSuites)))

	parallel, _ := cmd.Flags().GetBool("parallel")
	jobs, _ := cmd.Flags().GetInt("jobs")

	var allResults []*testResult
	var allPassed bool

	if parallel {
		allResults, err = runEcosystemTestsParallel(testSuites, jobs)
		if err != nil {
			return err
		}
	} else {
		allResults, err = runEcosystemTestsSequential(testSuites)
		if err != nil {
			return err
		}
	}

	allPassed = true
	for _, res := range allResults {
		if !res.Success {
			allPassed = false
			break
		}
	}

	if err := aggregateAndDisplayResults(allResults); err != nil {
		renderer.RenderError(fmt.Errorf("error processing results: %w", err))
		return err
	}

	if !allPassed {
		// Error already displayed in the summary, just exit with non-zero status
		os.Exit(1)
	}

	renderer.RenderSuccess("All ecosystem tests passed!")
	return nil
}

func runEcosystemTestsSequential(testSuites map[string]string) ([]*testResult, error) {
	resultsDir, err := os.MkdirTemp("", "tend-ecosystem-results-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary results directory: %w", err)
	}
	defer os.RemoveAll(resultsDir)

	var wg sync.WaitGroup
	resultsChan := make(chan testResult, len(testSuites))
	semaphore := make(chan struct{}, 8) // Limit concurrency

	for projectPath, makeTarget := range testSuites {
		wg.Add(1)
		go func(path, target string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			projectName := filepath.Base(path)
			reportPath := filepath.Join(resultsDir, projectName+".json")
			makeArgs := fmt.Sprintf("ARGS=--json %s", reportPath)

			cmd := command.New("make", target).Dir(path).Env(makeArgs).Timeout(5 * time.Minute)
			startTime := time.Now()
			result := cmd.Run()
			duration := time.Since(startTime)

			success := result.ExitCode == 0
			if !success {
				ulog.Error("Test failed").
					Field("project", projectName).
					Field("duration", duration).
					Pretty(theme.IconError + " " + theme.DefaultTheme.Error.Render(projectName)).
					Emit()
			} else {
				ulog.Success("Test passed").
					Field("project", projectName).
					Field("duration", duration).
					Pretty(theme.IconSuccess + " " + theme.DefaultTheme.Success.Render(projectName)).
					Emit()
			}
			resultsChan <- testResult{
				ProjectName: projectName,
				Success:     success,
				ReportPath:  reportPath,
				Error:       result.Error,
				Duration:    duration,
			}
		}(projectPath, makeTarget)
	}

	wg.Wait()
	close(resultsChan)

	var allResults []*testResult
	for res := range resultsChan {
		allResults = append(allResults, &res)
	}
	return allResults, nil
}

func runEcosystemTestsParallel(testSuites map[string]string, numJobs int) ([]*testResult, error) {
	model := e_runner.New(testSuites, numJobs)
	p := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running parallel test runner: %w", err)
	}

	runnerModel := finalModel.(e_runner.Model)
	eRunnerResults := runnerModel.Results()

	var allResults []*testResult
	for _, res := range eRunnerResults {
		allResults = append(allResults, &testResult{
			ProjectName: res.ProjectName,
			Success:     res.Success,
			ReportPath:  res.ReportPath,
			Error:       res.Error,
			Duration:    res.Duration,
		})
	}

	// Print detailed failures after the TUI closes
	if len(eRunnerResults) > 0 {
		var failedProjects []*e_runner.ProjectState
		for _, s := range runnerModel.ProjectStates() {
			if s.Status() == e_runner.StatusFailure {
				failedProjects = append(failedProjects, s)
			}
		}
		if len(failedProjects) > 0 {
			printEcosystemFailureDetails(failedProjects)
		}
	}

	return allResults, nil
}

func findEcosystemRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "Makefile")); err == nil {
			if _, err := os.Stat(filepath.Join(current, "core")); err == nil {
				return current, nil
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find root Makefile")
		}
		current = parent
	}
}

func parseRootMakefile(root string) ([]string, error) {
	content, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`(?m)^PACKAGES = (.*)$`)
	matches := re.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find PACKAGES variable in root Makefile")
	}
	return strings.Fields(matches[1]), nil
}

func discoverTestSuites(root string, packages []string) map[string]string {
	suites := make(map[string]string)
	for _, pkg := range packages {
		pkgPath := filepath.Join(root, pkg)
		makefile := filepath.Join(pkgPath, "Makefile")
		if _, err := os.Stat(makefile); err != nil {
			continue
		}
		content, err := os.ReadFile(makefile)
		if err != nil {
			continue
		}
		contentStr := string(content)

		if strings.Contains(contentStr, "\ntest-e2e-tend:") {
			suites[pkgPath] = "test-e2e-tend"
		} else if strings.Contains(contentStr, "\ntest-e2e:") && (strings.Contains(contentStr, "tend") || strings.Contains(contentStr, "E2E_BINARY_NAME")) {
			suites[pkgPath] = "test-e2e"
		} else if strings.Contains(contentStr, "\nrun-tend-tests:") {
			suites[pkgPath] = "run-tend-tests"
		}
	}
	return suites
}

func aggregateAndDisplayResults(results []*testResult) error {
	var allReports []reporters.JSONReport
	var failedProjects []*testResult

	for _, res := range results {
		if !res.Success {
			failedProjects = append(failedProjects, res)
		}

		if _, err := os.Stat(res.ReportPath); os.IsNotExist(err) {
			// If make failed, a report might not be generated.
			continue
		}

		data, err := os.ReadFile(res.ReportPath)
		if err != nil {
			ulog.Warn("Could not read report").
				Field("project", res.ProjectName).
				Err(err).
				Pretty(fmt.Sprintf("Warning: could not read report for %s: %v", res.ProjectName, err)).
				Emit()
			continue
		}

		var report reporters.JSONReport
		if err := json.Unmarshal(data, &report); err != nil {
			ulog.Warn("Could not parse report").
				Field("project", res.ProjectName).
				Err(err).
				Pretty(fmt.Sprintf("Warning: could not parse report for %s: %v", res.ProjectName, err)).
				Emit()
			continue
		}
		// Add project name to the report for context
		if len(report.Results) > 0 {
			report.Results[0].Name = res.ProjectName
		}
		allReports = append(allReports, report)
	}

	ulog.Info("Ecosystem test summary").
		Pretty("\n--- Ecosystem Test Summary ---").
		PrettyOnly().
		Emit()

	tbl := table.NewStyledTable().Headers("Project", "Status", "Duration", "Passed", "Failed")

	var allProjects []string
	projectResultMap := make(map[string]*testResult)
	for _, r := range results {
		allProjects = append(allProjects, r.ProjectName)
		projectResultMap[r.ProjectName] = r
	}
	sort.Strings(allProjects)

	for _, projectName := range allProjects {
		res := projectResultMap[projectName]
		status := theme.DefaultTheme.Success.Render(theme.IconSuccess + " PASS")
		var passed, failed string
		var durationStr string

		var reportForProject *reporters.JSONReport
		for _, r := range allReports {
			if len(r.Results) > 0 && r.Results[0].Name == projectName {
				reportForProject = &r
				break
			}
		}

		if !res.Success {
			status = theme.DefaultTheme.Error.Render(theme.IconError + " FAIL")
		}

		if reportForProject != nil {
			passed = fmt.Sprintf("%d", reportForProject.Passed)
			failed = fmt.Sprintf("%d", reportForProject.Failed)
			if d, err := time.ParseDuration(reportForProject.Duration); err == nil {
				durationStr = d.Round(time.Millisecond).String()
			}
		} else {
			passed = "0"
			failed = "0"
			durationStr = res.Duration.Round(time.Millisecond).String()
		}

		if !res.Success && failed == "0" {
			failed = "1" // Mark as at least one failure if the run failed but no report was parsed
		}

		tbl.Row(projectName, status, durationStr, passed, failed)
	}

	ulog.Info("Test results table").
		Pretty(tbl.String()).
		PrettyOnly().
		Emit()

	if len(failedProjects) > 0 {
		ulog.Info("Failure details").
			Pretty("\n" + theme.DefaultTheme.Title.Render("Failure Details")).
			PrettyOnly().
			Emit()

		for i, proj := range failedProjects {
			if i > 0 {
				ulog.Info("Failure details separator").
					Pretty("").
					PrettyOnly().
					Emit()
			}

			ulog.Error("Failed project").
				Field("project", proj.ProjectName).
				Pretty(theme.DefaultTheme.Error.Render(theme.IconError) + " " + theme.DefaultTheme.Title.Render(proj.ProjectName)).
				Emit()
			var reportForProject *reporters.JSONReport
			for _, r := range allReports {
				if len(r.Results) > 0 && r.Results[0].Name == proj.ProjectName {
					reportForProject = &r
					break
				}
			}

			if reportForProject != nil {
				failedScenarios := 0
				for _, res := range reportForProject.Results {
					if !res.Success {
						failedScenarios++
						if failedScenarios > 1 {
							ulog.Info("Failed scenario separator").
								Pretty("").
								PrettyOnly().
								Emit()
						}
						prettyMsg := fmt.Sprintf("   %s %s\n   %s %s",
							theme.DefaultTheme.Muted.Render("Scenario:"), res.Name,
							theme.DefaultTheme.Muted.Render("Step:    "), res.FailedStep)

						// Only show first line of error
						errorLines := strings.Split(res.Error, "\n")
						if len(errorLines) > 0 {
							prettyMsg += fmt.Sprintf("\n   %s %s",
								theme.DefaultTheme.Muted.Render("Error:   "), errorLines[0])
						}

						ulog.Error("Failed scenario details").
							Field("scenario", res.Name).
							Field("failed_step", res.FailedStep).
							Pretty(prettyMsg).
							Emit()
					}
				}
				if failedScenarios == 0 {
					ulog.Info("No scenario details").
						Pretty("   " + theme.DefaultTheme.Muted.Render("Build/test runner failed (no detailed failure info available)")).
						PrettyOnly().
						Emit()
				}
			} else if proj.Error != nil {
				// Clean up error message - only show first line
				errMsg := proj.Error.Error()
				errLines := strings.Split(errMsg, "\n")
				if len(errLines) > 0 {
					ulog.Error("Project error").
						Err(proj.Error).
						Pretty(fmt.Sprintf("   %s %s",
							theme.DefaultTheme.Muted.Render("Error:"), errLines[0])).
						Emit()
				}
			}
		}
		ulog.Info("Failure details end").
			Pretty("").
			PrettyOnly().
			Emit()
	}

	return nil
}

func printEcosystemFailureDetails(states []*e_runner.ProjectState) {
	if len(states) == 0 {
		return
	}

	prettyMsg := "\n" + strings.Repeat("=", 80) + "\n"
	prettyMsg += fmt.Sprintf("%s Test run failed: %d projects failed\n", theme.IconError, len(states))
	prettyMsg += strings.Repeat("=", 80)

	ulog.Error("Ecosystem test run failed").
		Field("failed_count", len(states)).
		Pretty(prettyMsg).
		Emit()

	for _, s := range states {
		prettyMsg := fmt.Sprintf("\n%s %s (failed in %v)\n%s",
			theme.IconError, s.Title(), s.Duration().Round(time.Millisecond),
			strings.Repeat("-", 80))

		if s.Output() != "" {
			// Trim leading/trailing whitespace from output for cleaner presentation
			prettyMsg += "\n" + strings.TrimSpace(s.Output())
		}

		ulog.Error("Project failed").
			Field("project", s.Title()).
			Field("duration", s.Duration()).
			Pretty(prettyMsg).
			Emit()
	}
}
