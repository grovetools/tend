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
				tendBinary, err := FindProjectBinary()
				if err != nil {
					return err
				}
				cmd := command.New(tendBinary, "list", "--keyword=git")
				result := cmd.Run()

				if result.Error != nil {
					return fmt.Errorf("tend list --keyword=git failed: %w", result.Error)
				}

				// Check that output contains git-workflow scenario
				if !strings.Contains(result.Stdout, "git-workflow") {
					return fmt.Errorf("expected 'git-workflow' in output, got: %s", result.Stdout)
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
