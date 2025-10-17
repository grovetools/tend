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
	return &harness.Scenario{
		Name:        "test-keyword-filtering",
		Description: "Tests the tend list --keyword functionality",
		Tags:        []string{"test", "filtering", "cli"},
		Steps: []harness.Step{
			{
				Name:        "Test keyword filtering for 'git'",
				Description: "Verify that --keyword=git returns only git-related scenarios",
				Func: func(ctx *harness.Context) error {
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
				},
			},
		},
	}
}

// LocalOnlyScenario demonstrates a scenario that should only run on dev machines
func LocalOnlyScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "local-only-example",
		Description: "Tests that should be skipped in CI environments",
		Tags:        []string{"local", "dev"},
		LocalOnly:   true,
		Steps: []harness.Step{
			{
				Name:        "Check local environment",
				Description: "Performs checks that are only valid in a local dev environment",
				Func: func(ctx *harness.Context) error {
					fmt.Println("Running local-only test logic...")
					ctx.Set("local_test_ran", true)
					return nil
				},
			},
		},
	}
}

// ExplicitOnlyScenario demonstrates a scenario that must be run explicitly
func ExplicitOnlyScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:         "explicit-only-example",
		Description:  "Tests that use real resources and must be run explicitly by name",
		Tags:         []string{"integration", "expensive"},
		ExplicitOnly: true,
		Steps: []harness.Step{
			{
				Name:        "Simulate expensive operation",
				Description: "Performs an operation that should not run automatically",
				Func: func(ctx *harness.Context) error {
					fmt.Println("⚠️  Running explicit-only test...")
					time.Sleep(100 * time.Millisecond)
					ctx.Set("expensive_operation_completed", true)
					return nil
				},
			},
		},
	}
}
