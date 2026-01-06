package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/git"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/verify"
)

func main() {
	fmt.Printf("%s Grove Tend Framework Demo\n", theme.IconDebugStart)
	fmt.Println("=" + string(make([]rune, 50)))

	// Create a sample scenario that demonstrates the framework capabilities
	scenario := harness.NewScenario(
		"grove-tend-demo",
		"Demonstrates the Grove Tend testing framework",
		[]string{"demo", "showcase"},
		[]harness.Step{
			harness.NewStep("Setup workspace", func(ctx *harness.Context) error {
				// Create project structure
				workspaceDir := ctx.NewDir("workspace")
				if err := fs.CreateProjectStructure(workspaceDir); err != nil {
					return err
				}

				// Write grove.yml
				if err := fs.WriteBasicGroveConfig(workspaceDir); err != nil {
					return err
				}

				ctx.Set("workspace", workspaceDir)
				return nil
			}),

			harness.NewStep("Initialize git repository", func(ctx *harness.Context) error {
				workspaceDir := ctx.GetString("workspace")
				
				repo, err := git.SetupTestRepo(workspaceDir)
				if err != nil {
					return err
				}

				// Add files and create initial commit
				if err := repo.AddCommit("Initial commit with Grove config"); err != nil {
					return err
				}

				ctx.Set("repo", repo)
				return nil
			}),

			harness.NewStep("Create feature branch", func(ctx *harness.Context) error {
				repo := ctx.Get("repo").(*git.Git)

				// Create feature branch with changes
				changes := map[string]string{
					"feature.txt": "This is a new feature!",
					"src/feature.go": `package main

import "fmt"

func newFeature() {
    fmt.Println("New feature implemented!")
}`,
				}

				return repo.CreateBranchWithChanges("feature/demo", changes)
			}),

			harness.NewStep("Simulate command execution", func(ctx *harness.Context) error {
				workspaceDir := ctx.GetString("workspace")

				// Test command execution
				cmd := command.New("ls", "-la")
				cmd.Dir(workspaceDir)
				result := cmd.Run()

				if result.Error != nil {
					return result.Error
				}

				fmt.Printf("Directory listing shows %d lines of output\n", len(strings.Split(result.Stdout, "\n")))
				return nil
			}),

			harness.NewStep("Validate repository state", func(ctx *harness.Context) error {
				repo := ctx.Get("repo").(*git.Git)

				// Check current branch
				branch, err := repo.CurrentBranch()
				if err != nil {
					return err
				}

				if branch != "feature/demo" {
					return fmt.Errorf("expected branch 'feature/demo', got '%s'", branch)
				}

				// Check for uncommitted changes
				hasChanges, err := repo.HasUncommittedChanges()
				if err != nil {
					return err
				}

				if hasChanges {
					return fmt.Errorf("repository has uncommitted changes")
				}

				fmt.Printf("%s Repository is on branch '%s' with clean state\n", theme.IconSuccess, branch)
				return nil
			}),

			harness.NewStep("Demonstrate assertion styles", func(ctx *harness.Context) error {
				fmt.Printf("%s Demonstrating hard (fail-fast) and soft (collecting) assertions.\n", theme.IconSuccess)

				// Example of a successful hard assertion
				if err := ctx.Check("repository has a clean state", func() error {
					repo := ctx.Get("repo").(*git.Git)
					hasChanges, err := repo.HasUncommittedChanges()
					if err != nil {
						return err
					}
					if hasChanges {
						return fmt.Errorf("repository should be clean but has uncommitted changes")
					}
					return nil
				}()); err != nil {
					return err // This would fail the step
				}

				// Example of soft assertions (collecting multiple failures)
				// This block will intentionally fail to demonstrate the aggregated error report.
				err := ctx.Verify(func(v *verify.Collector) {
					v.Contains("a passing soft assertion", "hello world", "hello")
					v.Equal("a failing equality check", 1, 2)
					v.True("a failing boolean check", false)
					v.Contains("another failing contains check", "foo bar", "baz")
				})

				if err != nil {
					fmt.Printf("\nCollected assertion failures (as expected for demo):\n%v\n", err)
					// In a real test, you would 'return err' here.
					// We return nil so the demo scenario can continue and pass.
					return nil
				}

				// This part should not be reached in the demo.
				return fmt.Errorf("expected soft assertion block to fail but it passed")
			}),

			harness.NewStep("Demonstrate step builders", func(ctx *harness.Context) error {
				// Test conditional step
				conditional := harness.ConditionalStep("conditional test",
					func(ctx *harness.Context) bool {
						return ctx.HasKey("workspace")
					},
					func(ctx *harness.Context) error {
						fmt.Printf("%s Conditional step executed successfully\n", theme.IconSuccess)
						return nil
					})

				if err := conditional.Func(ctx); err != nil {
					return err
				}

				// Test retry step
				retryCounter := 0
				retry := harness.RetryStep("retry test", 3, 0, func(ctx *harness.Context) error {
					retryCounter++
					if retryCounter < 2 {
						return fmt.Errorf("attempt %d failed", retryCounter)
					}
					fmt.Printf("%s Retry step succeeded on attempt %d\n", theme.IconSuccess, retryCounter)
					return nil
				})

				return retry.Func(ctx)
			}),
		},
	)

	// Run the scenario
	h := harness.New(harness.Options{
		Interactive: false, // Set to true for interactive mode
		Verbose:     true,
		NoCleanup:   false, // Set to true to preserve files for inspection
	})

	result, err := h.Run(context.Background(), scenario)
	if err != nil {
		log.Fatalf("Demo scenario failed: %v", err)
	}

	fmt.Printf("\n%s Demo completed successfully!\n", theme.IconStatusCompleted)
	fmt.Printf("   Duration: %v\n", result.Duration)
	fmt.Printf("   Steps executed: %d\n", len(result.StepResults))

	// Show capabilities summary
	fmt.Printf("\n%s Framework Capabilities Demonstrated:\n", theme.IconChecklist)
	fmt.Printf("  %s Scenario definition and execution\n", theme.IconSuccess)
	fmt.Printf("  %s Step-by-step progress tracking\n", theme.IconSuccess)
	fmt.Printf("  %s Temporary directory management\n", theme.IconSuccess)
	fmt.Printf("  %s Filesystem operations\n", theme.IconSuccess)
	fmt.Printf("  %s Git repository operations\n", theme.IconSuccess)
	fmt.Printf("  %s Command execution and output capture\n", theme.IconSuccess)
	fmt.Printf("  %s Context state management between steps\n", theme.IconSuccess)
	fmt.Printf("  %s Hard and soft assertion styles\n", theme.IconSuccess)
	fmt.Printf("  %s Error handling and reporting\n", theme.IconSuccess)
	fmt.Printf("  %s Step builder utilities\n", theme.IconSuccess)
	fmt.Printf("  %s Automatic cleanup\n", theme.IconSuccess)

	fmt.Printf("\n%s Ready for:\n", theme.IconBuild)
	fmt.Printf("  %s Converting existing bash test scripts\n", theme.IconBullet)
	fmt.Printf("  %s Building complex multi-step scenarios\n", theme.IconBullet)
	fmt.Printf("  %s Interactive debugging mode\n", theme.IconBullet)
	fmt.Printf("  %s CI integration and reporting\n", theme.IconBullet)
}