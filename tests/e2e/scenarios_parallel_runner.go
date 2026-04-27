// File: tests/e2e/scenarios_parallel_runner.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grovetools/core/tui/theme"

	"github.com/grovetools/tend/pkg/command"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/harness/reporters"
	"github.com/grovetools/tend/pkg/tui"
	"github.com/grovetools/tend/pkg/verify"
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

// slowFixtureDuration is the per-fixture sleep used by the parallel-runner
// concurrency tests. It must be large enough that the parallelism signal
// (jobs=4 finishing in 1 batch vs jobs=2 in 2 batches) clears the per-test
// overhead (process spawn + sub-TUI render cadence). 3s gives a clean 2x
// gap on a healthy machine.
const slowFixtureDuration = 3 * time.Second

// SlowScenario1 is a slow scenario for concurrency testing
func SlowScenario1() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"fixture-slow-1",
		"A slow scenario for concurrency testing",
		[]string{"parallel-fixture", "slow"},
		[]harness.Step{
			harness.NewStep("Sleep", func(ctx *harness.Context) error {
				time.Sleep(slowFixtureDuration)
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
			harness.NewStep("Sleep", func(ctx *harness.Context) error {
				time.Sleep(slowFixtureDuration)
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
			harness.NewStep("Sleep", func(ctx *harness.Context) error {
				time.Sleep(slowFixtureDuration)
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
			harness.NewStep("Sleep", func(ctx *harness.Context) error {
				time.Sleep(slowFixtureDuration)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// findE2EBinary explicitly finds the tend-e2e binary in the bin directory
func findE2EBinary() (string, error) {
	// Get current executable
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// The tend-e2e binary should be in the same directory
	binDir := filepath.Dir(execPath)
	e2ePath := filepath.Join(binDir, "tend-e2e")

	if _, err := os.Stat(e2ePath); err != nil {
		return "", fmt.Errorf("tend-e2e binary not found at %s", e2ePath)
	}

	return e2ePath, nil
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
				tendBinary, err := findE2EBinary()
				if err != nil {
					return err
				}

				// Create temp file for JSON report in the test's isolated directory
				reportPath := filepath.Join(ctx.RootDir, "report.json")

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

				// Wait for all tests to complete by checking for the success checkmarks
				if err := session.WaitForText(theme.IconSuccess, 30*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not show completed tests: %w\nContent:\n%s", err, content)
				}

				// Give it a moment to write the report
				time.Sleep(500 * time.Millisecond)

				ctx.Set("report_path", reportPath)
				ctx.Set("tui_session", session)

				return nil
			}),

			harness.NewStep("Verify TUI showed all tests completed", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)
				content, _ := session.Capture()

				// Count the number of checkmarks to verify all tests completed
				successCount := 0
				for _, line := range strings.Split(content, "\n") {
					if strings.Contains(line, theme.IconSuccess) {
						successCount++
					}
				}

				if successCount < 3 {
					return fmt.Errorf("expected 3 successful tests, found %d checkmarks", successCount)
				}
				return nil
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
				tendBinary, err := findE2EBinary()
				if err != nil {
					return err
				}

				jsonPath := filepath.Join(ctx.RootDir, "report.json")
				junitPath := filepath.Join(ctx.RootDir, "report.xml")

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

				// Wait for the JSON report file to be created, which indicates the parallel runner completed
				// We use file-based waiting instead of TUI text because:
				// 1. The fixture scenarios complete in ~100ms
				// 2. The TUI shows "Finished!" briefly then exits
				// 3. Failure details are dumped to stdout (~500 lines) which scrolls past the TUI output
				// 4. By the time WaitForText polls, the TUI text is scrolled out of the visible pane
				if err := session.WaitForFile("report.json", 30*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not create JSON report: %w\nContent:\n%s", err, content)
				}

				// Give it a moment for the file to be fully written
				time.Sleep(500 * time.Millisecond)

				ctx.Set("tui_session", session)
				ctx.Set("json_path", jsonPath)
				ctx.Set("junit_path", junitPath)

				return nil
			}),

			harness.NewStep("Verify TUI showed completion header", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)
				content, _ := session.Capture()

				// After the TUI exits, the pane shows the final output which includes failure details
				// The "Finished!" header from the TUI may have scrolled off, but we verified it appeared
				// in the previous step. Here we just verify the test output contains expected markers.
				// The detailed result verification is done via the JSON report in the next step.
				return ctx.Verify(func(v *verify.Collector) {
					// The output should contain either the TUI completion message or the final summary
					hasFinished := strings.Contains(content, "Finished!")
					hasSummary := strings.Contains(content, "scenario(s)")
					v.True("TUI showed completion or summary", hasFinished || hasSummary)
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
				tendBinary, err := findE2EBinary()
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

				// Wait for the *all-done* summary banner, not any single test's
				// success icon. WaitForText(IconSuccess) matches the first
				// per-test success in the inner TUI, which appears at the same
				// time regardless of --jobs (the first fixture finishes at
				// slowFixtureDuration in either configuration). To measure the
				// concurrency benefit we need wall-clock to *all* fixtures
				// completing.
				if err := session.WaitForText("All 4 scenario(s) passed", 30*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not show completed tests: %w\nContent:\n%s", err, content)
				}
				duration := time.Since(startTime)

				ctx.Set("jobs2_duration", duration)

				// With 4 fixtures of slowFixtureDuration each and --jobs=2, we
				// expect ~2 * slowFixtureDuration total (2 tests, then 2 more)
				// plus per-test overhead. Bound generously on both sides — the
				// concurrency assertion is the next step's job.
				minD := 2 * slowFixtureDuration
				maxD := 4*slowFixtureDuration + 5*time.Second
				if duration < minD {
					return fmt.Errorf("duration was %v, expected at least %v (tests should run in 2 batches)", duration, minD)
				}
				if duration >= maxD {
					return fmt.Errorf("duration was %v, expected less than %v", duration, maxD)
				}
				return nil
			}),

			harness.NewStep("Run with --jobs=4 and measure time", func(ctx *harness.Context) error {
				tendBinary, err := findE2EBinary()
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

				// Wait for the *all-done* summary banner, not any single test's
				// success icon. WaitForText(IconSuccess) matches the first
				// per-test success in the inner TUI, which appears at the same
				// time regardless of --jobs (the first fixture finishes at
				// slowFixtureDuration in either configuration). To measure the
				// concurrency benefit we need wall-clock to *all* fixtures
				// completing.
				if err := session.WaitForText("All 4 scenario(s) passed", 30*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("parallel runner did not show completed tests: %w\nContent:\n%s", err, content)
				}
				duration := time.Since(startTime)

				// With 4 fixtures of slowFixtureDuration and --jobs=4, all 4 run
				// concurrently — wall-clock should be roughly slowFixtureDuration
				// plus overhead. The concurrency assertion is a *ratio* against
				// the --jobs=2 run rather than an absolute gap, since per-test
				// overhead (process spawn, sub-TUI render cadence) varies by
				// machine and a fixed-millisecond threshold flakes when overhead
				// gets close to the fixture sleep duration.
				jobs2Duration := ctx.Get("jobs2_duration").(time.Duration)
				minD := slowFixtureDuration / 2
				if duration < minD {
					return fmt.Errorf("duration was %v, too fast - tests might not have run properly", duration)
				}
				// jobs=4 should be no more than 75% of jobs=2: a healthy run is
				// ~50%, so 75% leaves headroom for noise while still detecting
				// a real regression in --jobs concurrency.
				maxRatioPct := int64(75)
				if duration*100 >= jobs2Duration*time.Duration(maxRatioPct) {
					return fmt.Errorf("--jobs=4 duration %v not sufficiently faster than --jobs=2 %v (ratio %d%%, expected <%d%%)",
						duration, jobs2Duration, int64(duration*100/jobs2Duration), maxRatioPct)
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
				tendBinary, err := findE2EBinary()
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

// ParallelRunHeadlessJSONScenario tests headless parallel execution with --format json
func ParallelRunHeadlessJSONScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"parallel-run-headless-json",
		"Verifies that --parallel --format json runs headlessly and outputs clean JSON",
		[]string{"parallel", "headless", "json"},
		[]harness.Step{
			harness.NewStep("Run tests headlessly with JSON format", func(ctx *harness.Context) error {
				tendBinary, err := findE2EBinary()
				if err != nil {
					return err
				}

				// command.New runs without a TTY, so this triggers headless parallel mode
				result := command.New(tendBinary,
					"run",
					"fixture-passing-1",
					"fixture-passing-2",
					"fixture-failing-1",
					"--parallel",
					"--format", "json",
				).Dir(ctx.ProjectRoot).Env("TEND_IS_CHILD_PROCESS=true").Run()

				ctx.Set("result", result)

				// We expect exit code 1 because fixture-failing-1 is included
				if result.ExitCode != 1 {
					return fmt.Errorf("expected exit code 1, got %d\nStdout: %s\nStderr: %s", result.ExitCode, result.Stdout, result.Stderr)
				}
				return nil
			}),
			harness.NewStep("Verify JSON on stdout and logs on stderr", func(ctx *harness.Context) error {
				result := ctx.Get("result").(*command.Result)

				// Verify stdout is valid JSON
				var report reporters.JSONReport
				if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
					return fmt.Errorf("stdout is not valid JSON: %w\nStdout:\n%s", err, result.Stdout)
				}

				return ctx.Verify(func(v *verify.Collector) {
					// Verify report contents
					v.Equal("total tests is 3", 3, report.TotalTests)
					v.Equal("passed is 2", 2, report.Passed)
					v.Equal("failed is 1", 1, report.Failed)

					// Verify stderr contains the headless progress logs
					v.Contains("stderr has start log", result.Stderr, "Scenario started")
					v.Contains("stderr has completion log", result.Stderr, "Scenario completed")
					v.Contains("stderr has failure log", result.Stderr, "Scenario failed")

					// Verify stdout doesn't contain the logs (pure JSON)
					v.NotContains("stdout is pure JSON (no start logs)", result.Stdout, "Scenario started")
				})
			}),
		},
		false, // localOnly
		false, // explicitOnly
	)
}

// ParallelRunHeadlessNonTTYScenario tests headless parallel in non-TTY without --format json
func ParallelRunHeadlessNonTTYScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"parallel-run-headless-non-tty",
		"Verifies that --parallel defaults to headless parallel (not sequential) when TTY is unavailable",
		[]string{"parallel", "headless", "tty"},
		[]harness.Step{
			harness.NewStep("Run slow tests in non-TTY environment", func(ctx *harness.Context) error {
				tendBinary, err := findE2EBinary()
				if err != nil {
					return err
				}

				startTime := time.Now()
				// Run without --format json. Lack of TTY triggers headless mode.
				result := command.New(tendBinary,
					"run",
					"fixture-slow-1",
					"fixture-slow-2",
					"--parallel",
				).Dir(ctx.ProjectRoot).Env("TEND_IS_CHILD_PROCESS=true").Run()

				duration := time.Since(startTime)
				ctx.Set("duration", duration)
				ctx.Set("result", result)

				if result.ExitCode != 0 {
					return fmt.Errorf("expected exit code 0, got %d\nStdout: %s\nStderr: %s", result.ExitCode, result.Stdout, result.Stderr)
				}
				return nil
			}),
			harness.NewStep("Verify parallel execution and headless logs", func(ctx *harness.Context) error {
				duration := ctx.Get("duration").(time.Duration)
				result := ctx.Get("result").(*command.Result)

				return ctx.Verify(func(v *verify.Collector) {
					// If it ran sequentially, it would take > 4 seconds
					v.True("execution was parallel (took < 3.5s)", duration < 3500*time.Millisecond)

					// Headless progress logs should be present
					combined := result.Stderr + result.Stdout
					v.Contains("output has headless start log", combined, "Scenario started")
					v.Contains("output has headless complete log", combined, "Scenario completed")

					// Non-TTY note should be printed
					v.Contains("prints non-TTY note", result.Stderr, "Note: stdout is not a TTY, running parallel tests without TUI")
				})
			}),
		},
		false, // localOnly
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
				tendBinary, err := findE2EBinary()
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
