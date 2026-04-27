package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/tui/theme"
	"github.com/grovetools/tend/pkg/command"
	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/git"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/verify"
)

var ulog = grovelogging.NewUnifiedLogger("grove-tend.cmd.demo")

func main() {
	ulog.Info("Grove Tend Framework Demo").
		Pretty(theme.IconDebugStart + " Grove Tend Framework Demo\n" + "=" + string(make([]rune, 50))).
		Emit()

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

				lineCount := len(strings.Split(result.Stdout, "\n"))
				ulog.Info("Directory listing completed").
					Field("line_count", lineCount).
					Pretty(fmt.Sprintf("Directory listing shows %d lines of output", lineCount)).
					Emit()
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

				ulog.Success("Repository validation passed").
					Field("branch", branch).
					Pretty(theme.IconSuccess + fmt.Sprintf(" Repository is on branch '%s' with clean state", branch)).
					Emit()
				return nil
			}),

			harness.NewStep("Demonstrate assertion styles", func(ctx *harness.Context) error {
				ulog.Info("Demonstrating assertion styles").
					Pretty(theme.IconSuccess + " Demonstrating hard (fail-fast) and soft (collecting) assertions.").
					Emit()

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
					ulog.Info("Collected assertion failures").
						Err(err).
						Pretty(fmt.Sprintf("\nCollected assertion failures (as expected for demo):\n%v", err)).
						Emit()
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
						ulog.Success("Conditional step executed").
							Pretty(theme.IconSuccess + " Conditional step executed successfully").
							Emit()
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
					ulog.Success("Retry step succeeded").
						Field("attempt", retryCounter).
						Pretty(theme.IconSuccess + fmt.Sprintf(" Retry step succeeded on attempt %d", retryCounter)).
						Emit()
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

	ulog.Success("Demo completed successfully").
		Field("duration", result.Duration).
		Field("steps_executed", len(result.StepResults)).
		Pretty(fmt.Sprintf("\n%s Demo completed successfully!\n   Duration: %v\n   Steps executed: %d",
			theme.IconStatusCompleted, result.Duration, len(result.StepResults))).
		Emit()

	// Show capabilities summary
	capabilitiesSummary := fmt.Sprintf(`
%s Framework Capabilities Demonstrated:
  %s Scenario definition and execution
  %s Step-by-step progress tracking
  %s Temporary directory management
  %s Filesystem operations
  %s Git repository operations
  %s Command execution and output capture
  %s Context state management between steps
  %s Hard and soft assertion styles
  %s Error handling and reporting
  %s Step builder utilities
  %s Automatic cleanup

%s Ready for:
  %s Converting existing bash test scripts
  %s Building complex multi-step scenarios
  %s Interactive debugging mode
  %s CI integration and reporting`,
		theme.IconChecklist,
		theme.IconSuccess, theme.IconSuccess, theme.IconSuccess, theme.IconSuccess,
		theme.IconSuccess, theme.IconSuccess, theme.IconSuccess, theme.IconSuccess,
		theme.IconSuccess, theme.IconSuccess, theme.IconSuccess,
		theme.IconBuild,
		theme.IconBullet, theme.IconBullet, theme.IconBullet, theme.IconBullet)

	ulog.Info("Framework capabilities summary").
		Pretty(capabilitiesSummary).
		PrettyOnly().
		Emit()
}
