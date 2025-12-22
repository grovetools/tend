// File: tests/e2e/scenarios_cli.go
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// TestKeywordFilteringScenario creates a scenario that tests the keyword filtering feature
func TestKeywordFilteringScenario() *harness.Scenario {
	return harness.NewScenario(
		"test-keyword-filtering",
		"Tests the tend list --keyword functionality",
		[]string{"test", "filtering", "cli"},
		[]harness.Step{
			harness.NewStep("Test keyword filtering for 'git'", func(ctx *harness.Context) error {
				tendBinary, err := FindTendBinary()
				if err != nil {
					return err
				}
				// Run from project root so it can discover scenarios
				// Set TEND_IS_CHILD_PROCESS to prevent delegation to tend-e2e
				result := command.New(tendBinary, "list", "--keyword=git").
					Dir(ctx.ProjectRoot).
					Env("TEND_IS_CHILD_PROCESS=true").
					Run()

				if result.Error != nil {
					return fmt.Errorf("tend list --keyword=git failed (dir=%s): %w\nStdout: %s\nStderr: %s",
						ctx.ProjectRoot, result.Error, result.Stdout, result.Stderr)
				}

				// Check that output contains git-workflow scenario
				if !strings.Contains(result.Stdout, "git-workflow") {
					return fmt.Errorf("expected 'git-workflow' in output (dir=%s), got: %s\nStderr: %s",
						ctx.ProjectRoot, result.Stdout, result.Stderr)
				}

				// Check that output doesn't contain unrelated scenarios
				if strings.Contains(result.Stdout, "docker-operations") {
					return fmt.Errorf("unexpected 'docker-operations' in filtered output")
				}

				ctx.ShowCommandOutput("tend list --keyword=git", result.Stdout, result.Stderr)
				return nil
			}),
		},
	)
}

// LocalOnlyScenario demonstrates a scenario that should only run on dev machines
func LocalOnlyScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"local-only-example",
		"Tests that should be skipped in CI environments",
		[]string{"local", "dev"},
		[]harness.Step{
			harness.NewStep("Check local environment", func(ctx *harness.Context) error {
				fmt.Println("Running local-only test logic...")
				ctx.Set("local_test_ran", true)
				return nil
			}),
		},
		true,  // localOnly
		false, // explicitOnly
	)
}

// ExplicitOnlyScenario demonstrates a scenario that must be run explicitly
func ExplicitOnlyScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"explicit-only-example",
		"Tests that use real resources and must be run explicitly by name",
		[]string{"integration", "expensive"},
		[]harness.Step{
			harness.NewStep("Simulate expensive operation", func(ctx *harness.Context) error {
				fmt.Println("⚠️  Running explicit-only test...")
				time.Sleep(100 * time.Millisecond)
				ctx.Set("expensive_operation_completed", true)
				return nil
			}),
		},
		false, // localOnly
		true,  // explicitOnly
	)
}

// SetupOnlyFlagE2EScenario tests the --setup-only flag behavior end-to-end
func SetupOnlyFlagE2EScenario() *harness.Scenario {
	return harness.NewScenario(
		"setup-only-flag-e2e",
		"Tests the --setup-only flag with both scenarios with and without setup steps",
		[]string{"cli", "setup", "e2e"},
		[]harness.Step{
			harness.NewStep("Test --setup-only with a scenario that has setup steps", func(ctx *harness.Context) error {
				// Explicitly use the tend-e2e binary which has the helper scenarios
				tendE2EBinary := ctx.ProjectRoot + "/bin/tend-e2e"

				// Run with --setup-only on a scenario that has setup steps
				// This should halt after setup and return success
				result := command.New(tendE2EBinary, "run", "setup-only-with-setup-steps", "--setup-only").
					Dir(ctx.ProjectRoot).
					Run()

				// Should succeed (setup runs successfully and halts)
				if result.Error != nil {
					return fmt.Errorf("--setup-only with setup steps failed: %w\nStdout: %s\nStderr: %s",
						result.Error, result.Stdout, result.Stderr)
				}

				// Verify it shows "Halting execution after setup phase"
				if !strings.Contains(result.Stdout, "Halting execution after setup phase") {
					return fmt.Errorf("expected 'Halting execution after setup phase' in output, got:\n%s", result.Stdout)
				}

				// Verify test step did NOT run
				if strings.Contains(result.Stdout, "This test step should not run with --setup-only") {
					return fmt.Errorf("test step should not have executed with --setup-only")
				}

				ctx.ShowCommandOutput("tend-e2e run setup-only-with-setup-steps --setup-only", result.Stdout, result.Stderr)
				return nil
			}),

			harness.NewStep("Test --setup-only with a scenario that has NO setup steps", func(ctx *harness.Context) error {
				// Explicitly use the tend-e2e binary which has the helper scenarios
				tendE2EBinary := ctx.ProjectRoot + "/bin/tend-e2e"

				// Run with --setup-only on a scenario WITHOUT setup steps
				// This should switch to interactive mode and display the info message, then run the test normally
				result := command.New(tendE2EBinary, "run", "setup-only-without-setup-steps", "--setup-only").
					Dir(ctx.ProjectRoot).
					Run()

				// Should succeed when run normally (without actual interactive prompts in CI)
				if result.Error != nil {
					return fmt.Errorf("--setup-only without setup steps failed: %w\nStdout: %s\nStderr: %s",
						result.Error, result.Stdout, result.Stderr)
				}

				// Verify it shows "No setup steps found. Switching to interactive mode."
				if !strings.Contains(result.Stdout, "No setup steps found. Switching to interactive mode") {
					return fmt.Errorf("expected 'No setup steps found. Switching to interactive mode' in output, got:\n%s", result.Stdout)
				}

				// Verify the test step executed (because it switched to continuing with the test phase)
				if !strings.Contains(result.Stdout, "Mark test step executed") {
					return fmt.Errorf("expected test step to execute after switching to interactive mode")
				}

				ctx.ShowCommandOutput("tend-e2e run setup-only-without-setup-steps --setup-only", result.Stdout, result.Stderr)
				return nil
			}),
		},
	)
}
