package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/grovepm/grove-tend/internal/harness"
	"github.com/grovepm/grove-tend/pkg/command"
	"github.com/grovepm/grove-tend/pkg/fs"
	"github.com/grovepm/grove-tend/pkg/git"
)

func main() {
	fmt.Println("🚀 Grove Tend Framework Demo")
	fmt.Println("=" + string(make([]rune, 50)))

	// Create a sample scenario that demonstrates the framework capabilities
	scenario := &harness.Scenario{
		Name:        "grove-tend-demo",
		Description: "Demonstrates the Grove Tend testing framework",
		Tags:        []string{"demo", "showcase"},
		Steps: []harness.Step{
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

				fmt.Printf("✓ Repository is on branch '%s' with clean state\n", branch)
				return nil
			}),

			harness.NewStep("Demonstrate step builders", func(ctx *harness.Context) error {
				// Test conditional step
				conditional := harness.ConditionalStep("conditional test",
					func(ctx *harness.Context) bool {
						return ctx.HasKey("workspace")
					},
					func(ctx *harness.Context) error {
						fmt.Println("✓ Conditional step executed successfully")
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
					fmt.Printf("✓ Retry step succeeded on attempt %d\n", retryCounter)
					return nil
				})

				return retry.Func(ctx)
			}),
		},
	}

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

	fmt.Printf("\n🎉 Demo completed successfully!\n")
	fmt.Printf("   Duration: %v\n", result.Duration)
	fmt.Printf("   Steps executed: %d\n", len(result.StepResults))

	// Show capabilities summary
	fmt.Println("\n📋 Framework Capabilities Demonstrated:")
	fmt.Println("  ✓ Scenario definition and execution")
	fmt.Println("  ✓ Step-by-step progress tracking")
	fmt.Println("  ✓ Temporary directory management")
	fmt.Println("  ✓ Filesystem operations")
	fmt.Println("  ✓ Git repository operations")
	fmt.Println("  ✓ Command execution and output capture")
	fmt.Println("  ✓ Context state management between steps")
	fmt.Println("  ✓ Error handling and reporting")
	fmt.Println("  ✓ Step builder utilities")
	fmt.Println("  ✓ Automatic cleanup")
	
	fmt.Println("\n🔧 Ready for:")
	fmt.Println("  • Converting existing bash test scripts")
	fmt.Println("  • Building complex multi-step scenarios")
	fmt.Println("  • Interactive debugging mode")
	fmt.Println("  • CI integration and reporting")
}