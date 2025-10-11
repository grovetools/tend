package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mattsolo1/grove-core/tui/components/table"
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/harness/reporters"
	"github.com/mattsolo1/grove-tend/pkg/ui"
	"github.com/spf13/cobra"
)

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

	cmd.AddCommand(runCmd)
	return cmd
}

type testResult struct {
	ProjectName string
	Success     bool
	ReportPath  string
	Error       error
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

	resultsDir, err := os.MkdirTemp("", "tend-ecosystem-results-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary results directory: %w", err)
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
			result := cmd.Run()

			success := result.ExitCode == 0
			if !success {
				fmt.Printf("❌ %s\n", ui.ErrorStyle.Render(projectName))
			} else {
				fmt.Printf("✅ %s\n", ui.SuccessStyle.Render(projectName))
			}
			resultsChan <- testResult{
				ProjectName: projectName,
				Success:     success,
				ReportPath:  reportPath,
				Error:       result.Error,
			}
		}(projectPath, makeTarget)
	}

	wg.Wait()
	close(resultsChan)

	allPassed, err := aggregateAndDisplayResults(resultsDir, resultsChan)
	if err != nil {
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

func findEcosystemRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "Makefile")); err == nil {
			if _, err := os.Stat(filepath.Join(current, "grove-core")); err == nil {
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

func aggregateAndDisplayResults(resultsDir string, resultsChan <-chan testResult) (bool, error) {
	var allReports []reporters.JSONReport
	var failedProjects []testResult
	projectResults := make(map[string]testResult)

	for res := range resultsChan {
		projectResults[res.ProjectName] = res
		if !res.Success {
			failedProjects = append(failedProjects, res)
		}

		if _, err := os.Stat(res.ReportPath); os.IsNotExist(err) {
			// If make failed, a report might not be generated.
			continue
		}

		data, err := os.ReadFile(res.ReportPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read report for %s: %v\n", res.ProjectName, err)
			continue
		}

		var report reporters.JSONReport
		if err := json.Unmarshal(data, &report); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not parse report for %s: %v\n", res.ProjectName, err)
			continue
		}
		// Add project name to the report for context
		if len(report.Results) > 0 {
			report.Results[0].Name = res.ProjectName
		}
		allReports = append(allReports, report)
	}

	fmt.Println("\n--- Ecosystem Test Summary ---")
	tbl := table.NewStyledTable().Headers("Project", "Status", "Duration", "Passed", "Failed")
	allPassed := len(failedProjects) == 0

	// Add all projects with reports
	for _, report := range allReports {
		if len(report.Results) == 0 {
			continue
		}
		res := report.Results[0]
		status := ui.SuccessStyle.Render("✅ PASS")
		if !res.Success || report.Failed > 0 {
			status = ui.ErrorStyle.Render("❌ FAIL")
			allPassed = false
		}
		tbl.Row(res.Name, status, res.Duration, fmt.Sprintf("%d", report.Passed), fmt.Sprintf("%d", report.Failed))
	}

	// Add failed projects without reports
	for _, failed := range failedProjects {
		hasReport := false
		for _, report := range allReports {
			if len(report.Results) > 0 && report.Results[0].Name == failed.ProjectName {
				hasReport = true
				break
			}
		}
		if !hasReport {
			status := ui.ErrorStyle.Render("❌ FAIL")
			errorMsg := "Build/Make failed"
			if failed.Error != nil {
				errorMsg = failed.Error.Error()
				if len(errorMsg) > 30 {
					errorMsg = errorMsg[:30] + "..."
				}
			}
			tbl.Row(failed.ProjectName, status, errorMsg, "0", "0")
		}
	}

	fmt.Println(tbl)

	if len(failedProjects) > 0 {
		fmt.Println("\n" + ui.TitleStyle.Render("Failure Details"))
		fmt.Println()

		for i, proj := range failedProjects {
			if i > 0 {
				fmt.Println()
			}

			fmt.Printf("%s %s\n", ui.ErrorStyle.Render("❌"), ui.TitleStyle.Render(proj.ProjectName))
			reportPath := filepath.Join(resultsDir, proj.ProjectName+".json")

			if data, err := os.ReadFile(reportPath); err == nil {
				var report reporters.JSONReport
				if json.Unmarshal(data, &report) == nil {
					failedScenarios := 0
					for _, res := range report.Results {
						if !res.Success {
							failedScenarios++
							if failedScenarios > 1 {
								fmt.Println()
							}
							fmt.Printf("   %s %s\n", ui.MutedStyle.Render("Scenario:"), res.Name)
							fmt.Printf("   %s %s\n", ui.MutedStyle.Render("Step:    "), res.FailedStep)
							// Only show first line of error
							errorLines := strings.Split(res.Error, "\n")
							if len(errorLines) > 0 {
								fmt.Printf("   %s %s\n", ui.MutedStyle.Render("Error:   "), errorLines[0])
							}
						}
					}
					if failedScenarios == 0 {
						fmt.Printf("   %s\n", ui.MutedStyle.Render("Build/test runner failed (no detailed failure info available)"))
					}
				}
			} else if proj.Error != nil {
				// Clean up error message - only show first line
				errMsg := proj.Error.Error()
				errLines := strings.Split(errMsg, "\n")
				if len(errLines) > 0 {
					fmt.Printf("   %s %s\n", ui.MutedStyle.Render("Error:"), errLines[0])
				}
			}
		}
		fmt.Println()
	}

	return allPassed, nil
}
