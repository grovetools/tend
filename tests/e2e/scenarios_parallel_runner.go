// File: tests/e2e/scenarios_parallel_runner.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/harness/reporters"
	"github.com/mattsolo1/grove-tend/pkg/tui"
	"github.com/mattsolo1/grove-tend/pkg/verify"
)

// Test fixture scenarios - these will be used by the parallel runner tests
// They need to be registered in main.go but marked as explicit-only so they don't run by default

// PassingScenario1 is a simple passing scenario for testing
func PassingScenario1() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-passing-1",
		"A simple passing scenario for parallel runner tests",
		[]string{"parallel-fixture"},
		[]harness.Step{
			harness.NewStep("Do some work", func(ctx *harness.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly - don't run this in normal test runs
	)
}

// PassingScenario2 is another simple passing scenario
func PassingScenario2() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-passing-2",
		"Another passing scenario for parallel runner tests",
		[]string{"parallel-fixture"},
		[]harness.Step{
			harness.NewStep("Do some work", func(ctx *harness.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// PassingScenario3 is yet another simple passing scenario
func PassingScenario3() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-passing-3",
		"Yet another passing scenario for parallel runner tests",
		[]string{"parallel-fixture"},
		[]harness.Step{
			harness.NewStep("Do some work", func(ctx *harness.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// FailingScenario1 is a scenario that fails
func FailingScenario1() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-failing-1",
		"A failing scenario for parallel runner tests",
		[]string{"parallel-fixture"},
		[]harness.Step{
			harness.NewStep("This step will fail", func(ctx *harness.Context) error {
				time.Sleep(50 * time.Millisecond)
				return fmt.Errorf("intentional failure in fixture-failing-1")
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// FailingScenario2 is another scenario that fails
func FailingScenario2() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-failing-2",
		"Another failing scenario for parallel runner tests",
		[]string{"parallel-fixture"},
		[]harness.Step{
			harness.NewStep("First step passes", func(ctx *harness.Context) error {
				time.Sleep(50 * time.Millisecond)
				return nil
			}),
			harness.NewStep("Second step fails", func(ctx *harness.Context) error {
				return fmt.Errorf("intentional failure in step 2")
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// SlowScenario1 is a slow scenario for concurrency testing
func SlowScenario1() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-slow-1",
		"A slow scenario for concurrency testing",
		[]string{"parallel-fixture", "slow"},
		[]harness.Step{
			harness.NewStep("Sleep for 2 seconds", func(ctx *harness.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// SlowScenario2 is another slow scenario
func SlowScenario2() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-slow-2",
		"Another slow scenario for concurrency testing",
		[]string{"parallel-fixture", "slow"},
		[]harness.Step{
			harness.NewStep("Sleep for 2 seconds", func(ctx *harness.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// SlowScenario3 is yet another slow scenario
func SlowScenario3() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-slow-3",
		"Yet another slow scenario for concurrency testing",
		[]string{"parallel-fixture", "slow"},
		[]harness.Step{
			harness.NewStep("Sleep for 2 seconds", func(ctx *harness.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// SlowScenario4 is the fourth slow scenario
func SlowScenario4() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-slow-4",
		"Fourth slow scenario for concurrency testing",
		[]string{"parallel-fixture", "slow"},
		[]harness.Step{
			harness.NewStep("Sleep for 2 seconds", func(ctx *harness.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// Actual test scenarios for the parallel runner feature

// ParallelRunAllPassingScenario tests parallel execution with all passing tests
func ParallelRunAllPassingScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"parallel-run-all-passing",
		"Verifies that the parallel runner executes passing scenarios correctly",
		[]string{"parallel", "smoke", "success"},
		[]harness.Step{
			harness.NewStep("Run three passing scenarios in parallel", func(ctx *harness.Context) error {
				tendBinary, err := FindTendBinary()
				if err != nil {
					return err
				}

				// Create temp file for JSON report in a predictable location
				reportPath := filepath.Join(ctx.ProjectRoot, "report.json")

				// The parallel runner requires a TTY for the TUI
				// We need to use StartTUI to provide that
				session, err := ctx.StartTUI(tendBinary, []string{
					"run",
					"fixture-passing-1",
					"fixture-passing-2",
					"fixture-passing-3",
					"--parallel",
					"--json", reportPath,
				})
				if err != nil {
					return fmt.Errorf("failed to start parallel runner: %w", err)
				}

				// Wait for the TUI to finish (look for "Finished!")
				if err := session.WaitForText("Finished!", 30*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not finish: %w\nContent:\n%s", err, content)
				}

				// Give it a moment to write the report
				time.Sleep(500 * time.Millisecond)

				ctx.Set("report_path", reportPath)
				ctx.Set("tui_session", session)

				return nil
			}),

			harness.NewStep("Verify TUI showed success", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)

				// Verify the TUI showed successful completion
				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("TUI shows 3 successful tests", nil, session.AssertContains("Success: 3"))
					v.Equal("TUI shows 0 failed tests", nil, session.AssertContains("Failed: 0"))
				})
			}),

			harness.NewStep("Verify JSON report is valid", func(ctx *harness.Context) error {
				reportPath := ctx.Get("report_path").(string)
				data, err := os.ReadFile(reportPath)
				if err != nil {
					return fmt.Errorf("failed to read report: %w", err)
				}

				var report reporters.JSONReport
				if err := json.Unmarshal(data, &report); err != nil {
					return fmt.Errorf("failed to parse JSON report: %w", err)
				}

				ctx.Set("report", &report)

				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("total tests is 3", 3, report.TotalTests)
					v.Equal("passed is 3", 3, report.Passed)
					v.Equal("failed is 0", 0, report.Failed)
					v.Equal("results count is 3", 3, len(report.Results))

					// Check each result
					for i, r := range report.Results {
						v.Equal(fmt.Sprintf("result %d is successful", i), true, r.Success)
					}
				})
			}),
		},
		true,  // localOnly - requires tmux for TUI
		false, // explicitOnly
	)
}

// ParallelRunWithFailuresScenario tests parallel execution with mixed results
func ParallelRunWithFailuresScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"parallel-run-with-failures",
		"Ensures the parallel runner handles mixed pass/fail results correctly",
		[]string{"parallel", "failure"},
		[]harness.Step{
			harness.NewStep("Run scenarios with mixed results", func(ctx *harness.Context) error {
				tendBinary, err := FindTendBinary()
				if err != nil {
					return err
				}

				jsonPath := filepath.Join(ctx.ProjectRoot, "report.json")
				junitPath := filepath.Join(ctx.ProjectRoot, "report.xml")

				session, err := ctx.StartTUI(tendBinary, []string{
					"run",
					"fixture-passing-1",
					"fixture-passing-2",
					"fixture-failing-1",
					"fixture-failing-2",
					"--parallel",
					"--json", jsonPath,
					"--junit", junitPath,
				})
				if err != nil {
					return fmt.Errorf("failed to start parallel runner: %w", err)
				}

				// Wait for completion
				if err := session.WaitForText("Finished!", 30*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not finish: %w\nContent:\n%s", err, content)
				}

				time.Sleep(500 * time.Millisecond)

				ctx.Set("tui_session", session)
				ctx.Set("json_path", jsonPath)
				ctx.Set("junit_path", junitPath)

				return nil
			}),

			harness.NewStep("Verify TUI showed mixed results", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)

				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("TUI shows 2 successful tests", nil, session.AssertContains("Success: 2"))
					v.Equal("TUI shows 2 failed tests", nil, session.AssertContains("Failed: 2"))
				})
			}),

			harness.NewStep("Verify JSON report shows correct results", func(ctx *harness.Context) error {
				jsonPath := ctx.Get("json_path").(string)
				data, err := os.ReadFile(jsonPath)
				if err != nil {
					return fmt.Errorf("failed to read JSON report: %w", err)
				}

				var report reporters.JSONReport
				if err := json.Unmarshal(data, &report); err != nil {
					return fmt.Errorf("failed to parse JSON report: %w", err)
				}

				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("total tests is 4", 4, report.TotalTests)
					v.Equal("passed is 2", 2, report.Passed)
					v.Equal("failed is 2", 2, report.Failed)

					// Count successes and failures
					successes := 0
					failures := 0
					for _, r := range report.Results {
						if r.Success {
							successes++
						} else {
							failures++
							v.NotEqual(fmt.Sprintf("failed test %s has error message", r.Name), "", r.Error)
							v.NotEqual(fmt.Sprintf("failed test %s has failed_step", r.Name), "", r.FailedStep)
						}
					}
					v.Equal("2 successful results", 2, successes)
					v.Equal("2 failed results", 2, failures)
				})
			}),

			harness.NewStep("Verify JUnit report is valid", func(ctx *harness.Context) error {
				junitPath := ctx.Get("junit_path").(string)
				data, err := os.ReadFile(junitPath)
				if err != nil {
					return fmt.Errorf("failed to read JUnit report: %w", err)
				}

				xmlContent := string(data)

				return ctx.Verify(func(v *verify.Collector) {
					v.Contains("JUnit has testsuites element", xmlContent, "<testsuites")
					v.Contains("JUnit shows 4 tests", xmlContent, `tests="4"`)
					v.Contains("JUnit shows 2 failures", xmlContent, `failures="2"`)
					v.Contains("JUnit has failure elements", xmlContent, "<failure")
				})
			}),
		},
		true,  // localOnly - requires tmux for TUI
		false, // explicitOnly
	)
}

// ParallelRunJobsFlagScenario tests the --jobs flag for concurrency control
func ParallelRunJobsFlagScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"parallel-run-jobs-flag",
		"Verifies that the --jobs flag correctly limits concurrency",
		[]string{"parallel", "concurrency"},
		[]harness.Step{
			harness.NewStep("Run with --jobs=2 and measure time", func(ctx *harness.Context) error {
				tendBinary, err := FindTendBinary()
				if err != nil {
					return err
				}

				startTime := time.Now()
				session, err := ctx.StartTUI(tendBinary, []string{
					"run",
					"fixture-slow-1",
					"fixture-slow-2",
					"fixture-slow-3",
					"fixture-slow-4",
					"--parallel",
					"--jobs=2",
				})
				if err != nil {
					return fmt.Errorf("failed to start parallel runner: %w", err)
				}

				// Wait for completion
				if err := session.WaitForText("Finished!", 15*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not finish: %w\nContent:\n%s", err, content)
				}
				duration := time.Since(startTime)

				ctx.Set("jobs2_duration", duration)

				// With 4 tests of 2s each, and --jobs=2, we expect ~4s total
				// (2 tests run, then 2 more tests run)
				if duration < 4*time.Second {
					return fmt.Errorf("duration was %v, expected at least 4s", duration)
				}
				if duration >= 5*time.Second {
					return fmt.Errorf("duration was %v, expected less than 5s", duration)
				}
				return nil
			}),

			harness.NewStep("Run with --jobs=4 and measure time", func(ctx *harness.Context) error {
				tendBinary, err := FindTendBinary()
				if err != nil {
					return err
				}

				startTime := time.Now()
				session, err := ctx.StartTUI(tendBinary, []string{
					"run",
					"fixture-slow-1",
					"fixture-slow-2",
					"fixture-slow-3",
					"fixture-slow-4",
					"--parallel",
					"--jobs=4",
				})
				if err != nil {
					return fmt.Errorf("failed to start parallel runner: %w", err)
				}

				// Wait for completion
				if err := session.WaitForText("Finished!", 10*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not finish: %w\nContent:\n%s", err, content)
				}
				duration := time.Since(startTime)

				// With 4 tests of 2s each, and --jobs=4, we expect ~2s total
				// (all 4 tests run concurrently)
				if duration < 2*time.Second {
					return fmt.Errorf("duration was %v, expected at least 2s", duration)
				}
				if duration >= 3*time.Second {
					return fmt.Errorf("duration was %v, expected less than 3s", duration)
				}
				return nil
			}),
		},
		true,  // localOnly - requires tmux for TUI
		false, // explicitOnly
	)
}

// ParallelRunInteractiveQuitScenario tests graceful interruption of the TUI
func ParallelRunInteractiveQuitScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"parallel-run-interactive-quit",
		"Tests that the parallel runner TUI can be gracefully interrupted",
		[]string{"parallel", "tui", "interactive"},
		[]harness.Step{
			harness.NewStep("Start parallel runner with slow tests", func(ctx *harness.Context) error {
				tendBinary, err := FindTendBinary()
				if err != nil {
					return err
				}

				// Create a very slow scenario for this test
				session, err := ctx.StartTUI(tendBinary, []string{
					"run",
					"fixture-slow-1",
					"fixture-slow-2",
					"fixture-slow-3",
					"fixture-slow-4",
					"--parallel",
				})
				if err != nil {
					return fmt.Errorf("failed to start parallel runner: %w", err)
				}

				ctx.Set("tui_session", session)
				return nil
			}),

			harness.NewStep("Wait for TUI to show running tests", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)

				// Wait for the TUI to show at least one running test
				if err := session.WaitForText("Running", 5*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("TUI did not show running state: %w\nContent:\n%s", err, content)
				}

				return nil
			}),

			harness.NewStep("Send quit command", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)

				// Send 'q' to quit
				// The harness will handle cleanup and verification automatically
				if err := session.Type("q"); err != nil {
					return fmt.Errorf("failed to send quit command: %w", err)
				}

				// Give it a moment to process the quit
				time.Sleep(500 * time.Millisecond)

				return nil
			}),
		},
		true,  // localOnly - requires tmux
		false, // explicitOnly
	)
}

// ParallelRunNoScenariosScenario tests behavior when no scenarios match
// This scenario doesn't need tmux since it should exit before launching the TUI
func ParallelRunNoScenariosScenario() *harness.Scenario {
	return harness.NewScenario(
		"parallel-run-no-scenarios",
		"Ensures correct behavior when no scenarios match the filter",
		[]string{"parallel", "edge-case"},
		[]harness.Step{
			harness.NewStep("Run with non-matching tag filter", func(ctx *harness.Context) error {
				tendBinary, err := FindTendBinary()
				if err != nil {
					return err
				}

				startTime := time.Now()
				result := command.New(tendBinary,
					"run",
					"--parallel",
					"--tags=non-existent-tag",
				).Dir(ctx.ProjectRoot).Run()
				duration := time.Since(startTime)

				ctx.Set("result", result)
				ctx.Set("duration", duration)

				return nil
			}),

			harness.NewStep("Verify command exits successfully", func(ctx *harness.Context) error {
				result := ctx.Get("result").(*command.Result)

				if result.ExitCode != 0 {
					return fmt.Errorf("expected exit code 0, got %d", result.ExitCode)
				}

				hasNoScenariosMsg := strings.Contains(result.Stdout, "No scenarios") ||
					strings.Contains(result.Stderr, "No scenarios")
				if !hasNoScenariosMsg {
					return fmt.Errorf("expected 'No scenarios' message in output, got:\nStdout: %s\nStderr: %s",
						result.Stdout, result.Stderr)
				}

				return nil
			}),

			harness.NewStep("Verify TUI was not launched", func(ctx *harness.Context) error {
				duration := ctx.Get("duration").(time.Duration)

				// If TUI was launched, it would take longer
				// Should exit immediately (within 1 second)
				if duration >= 1*time.Second {
					return fmt.Errorf("command took too long (%v), TUI may have been launched", duration)
				}

				return nil
			}),
		},
	)
}
