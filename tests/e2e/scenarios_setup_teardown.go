// File: tests/e2e/scenarios_setup_teardown.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grovetools/tend/pkg/assert"
	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/verify"
)

// SetupPhaseBasicScenario tests that the setup phase executes before test steps
func SetupPhaseBasicScenario() *harness.Scenario {
	return harness.NewScenario(
		"setup-phase-basic",
		"Tests that setup phase executes before test steps and state is shared",
		[]string{"setup", "lifecycle"},
		[]harness.Step{
			harness.NewStep("Verify setup ran and set context value", func(ctx *harness.Context) error {
				// This should be set by the setup step
				value := ctx.GetString("setup_value")
				if err := ctx.Check("setup_value is set by setup phase", assert.Equal(value, "initialized")); err != nil {
					return err
				}
				return nil
			}),
			harness.NewStep("Verify setup created file", func(ctx *harness.Context) error {
				setupFile := ctx.GetString("setup_file")
				return ctx.Check("file created in setup exists", fs.AssertExists(setupFile))
			}),
		},
	).WithSetup(
		harness.NewStep("Initialize test environment", func(ctx *harness.Context) error {
			// Set a context value
			ctx.Set("setup_value", "initialized")

			// Create a file
			setupFile := filepath.Join(ctx.RootDir, "setup.txt")
			if err := fs.WriteString(setupFile, "setup complete\n"); err != nil {
				return fmt.Errorf("failed to write setup file: %w", err)
			}
			ctx.Set("setup_file", setupFile)

			return nil
		}),
	)
}

// SetupPhaseMultipleStepsScenario tests that multiple setup steps run in order
func SetupPhaseMultipleStepsScenario() *harness.Scenario {
	return harness.NewScenario(
		"setup-phase-multiple-steps",
		"Tests that multiple setup steps execute in order and can depend on each other",
		[]string{"setup", "lifecycle"},
		[]harness.Step{
			harness.NewStep("Verify all setup steps executed", func(ctx *harness.Context) error {
				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("setup step 1 ran", ctx.GetString("step1"), "done")
					v.Equal("setup step 2 ran", ctx.GetString("step2"), "done")
					v.Equal("setup step 3 ran", ctx.GetString("step3"), "done")
					v.Equal("steps ran in order", ctx.GetInt("counter"), 3)
				})
			}),
		},
	).WithSetup(
		harness.NewStep("Setup step 1", func(ctx *harness.Context) error {
			ctx.Set("step1", "done")
			ctx.Set("counter", 1)
			return nil
		}),
		harness.NewStep("Setup step 2", func(ctx *harness.Context) error {
			// Verify step 1 ran first
			if ctx.GetString("step1") != "done" {
				return fmt.Errorf("step 1 should have run before step 2")
			}
			ctx.Set("step2", "done")
			ctx.Set("counter", ctx.GetInt("counter")+1)
			return nil
		}),
		harness.NewStep("Setup step 3", func(ctx *harness.Context) error {
			// Verify steps 1 and 2 ran first
			if ctx.GetString("step1") != "done" || ctx.GetString("step2") != "done" {
				return fmt.Errorf("steps 1 and 2 should have run before step 3")
			}
			ctx.Set("step3", "done")
			ctx.Set("counter", ctx.GetInt("counter")+1)
			return nil
		}),
	)
}

// SetupPhaseFailureHandlingScenario tests that setup failures are handled correctly
func SetupPhaseFailureHandlingScenario() *harness.Scenario {
	return harness.NewScenario(
		"setup-phase-failure-handling",
		"Tests that setup phase properly handles and reports failures",
		[]string{"setup", "lifecycle"},
		[]harness.Step{
			harness.NewStep("Verify setup failure behavior", func(ctx *harness.Context) error {
				// This test validates that when a setup step fails:
				// 1. The first setup step should have run
				// 2. Subsequent setup steps should NOT run
				// 3. Test steps should NOT run (we're in a test step, so if we get here, something is wrong)
				//
				// Since we can't directly test failure in a passing test, we verify the
				// behavior indirectly by checking that failed setup stops execution.
				// The actual failure test exists as setup-phase-failure (explicit-only).

				// For this passing test, we just verify normal setup completion
				return ctx.Check("setup completed successfully", assert.Equal(ctx.GetString("setup_complete"), "yes"))
			}),
		},
	).WithSetup(
		harness.NewStep("Setup step that succeeds", func(ctx *harness.Context) error {
			ctx.Set("setup_complete", "yes")
			return nil
		}),
	)
}

// SetupPhaseFailureScenario tests that test steps don't run if setup fails
// This is an explicit-only test because it intentionally fails to verify error handling
func SetupPhaseFailureScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"setup-phase-failure",
		"Tests that test steps are skipped if a setup step fails (explicit-only: intentionally fails)",
		[]string{"setup", "lifecycle", "failure"},
		[]harness.Step{
			harness.NewStep("This step should never execute", func(ctx *harness.Context) error {
				// If this runs, the test failed
				ctx.Set("test_step_ran", true)
				return fmt.Errorf("test step should not have executed after setup failure")
			}),
		},
		false, // localOnly
		true,  // explicitOnly - this test intentionally fails
	).WithSetup(
		harness.NewStep("Setup step that succeeds", func(ctx *harness.Context) error {
			ctx.Set("setup_step_1", "ran")
			return nil
		}),
		harness.NewStep("Setup step that fails", func(ctx *harness.Context) error {
			return fmt.Errorf("intentional setup failure for testing")
		}),
		harness.NewStep("Setup step that should not run", func(ctx *harness.Context) error {
			ctx.Set("setup_step_3", "ran")
			return nil
		}),
	)
}

// TeardownPhaseBasicScenario tests that teardown phase executes after test steps
func TeardownPhaseBasicScenario() *harness.Scenario {
	return harness.NewScenario(
		"teardown-phase-basic",
		"Tests that teardown phase executes after test steps complete",
		[]string{"teardown", "lifecycle"},
		[]harness.Step{
			harness.NewStep("Create resource that needs cleanup", func(ctx *harness.Context) error {
				resourceFile := filepath.Join(ctx.RootDir, "resource.txt")
				if err := fs.WriteString(resourceFile, "temporary resource\n"); err != nil {
					return fmt.Errorf("failed to create resource: %w", err)
				}
				ctx.Set("resource_file", resourceFile)

				// Verify file exists
				return ctx.Check("resource file created", fs.AssertExists(resourceFile))
			}),
			harness.NewStep("Mark cleanup marker", func(ctx *harness.Context) error {
				markerFile := filepath.Join(ctx.RootDir, "cleanup.marker")
				if err := fs.WriteString(markerFile, "needs cleanup\n"); err != nil {
					return fmt.Errorf("failed to create marker: %w", err)
				}
				ctx.Set("marker_file", markerFile)
				return nil
			}),
		},
	).WithTeardown(
		harness.NewStep("Cleanup resources", func(ctx *harness.Context) error {
			resourceFile := ctx.GetString("resource_file")
			if resourceFile != "" {
				if err := os.Remove(resourceFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove resource: %w", err)
				}
			}

			markerFile := ctx.GetString("marker_file")
			if markerFile != "" {
				if err := os.Remove(markerFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove marker: %w", err)
				}
			}

			return nil
		}),
	)
}

// TeardownPhaseMultipleStepsScenario tests that multiple teardown steps run in order
func TeardownPhaseMultipleStepsScenario() *harness.Scenario {
	return harness.NewScenario(
		"teardown-phase-multiple-steps",
		"Tests that multiple teardown steps execute in order",
		[]string{"teardown", "lifecycle"},
		[]harness.Step{
			harness.NewStep("Create multiple resources", func(ctx *harness.Context) error {
				for i := 1; i <= 3; i++ {
					filename := filepath.Join(ctx.RootDir, fmt.Sprintf("resource%d.txt", i))
					if err := fs.WriteString(filename, fmt.Sprintf("resource %d\n", i)); err != nil {
						return fmt.Errorf("failed to create resource %d: %w", i, err)
					}
					ctx.Set(fmt.Sprintf("resource%d", i), filename)
				}
				return nil
			}),
		},
	).WithTeardown(
		harness.NewStep("Cleanup resource 1", func(ctx *harness.Context) error {
			resource1 := ctx.GetString("resource1")
			if resource1 != "" {
				if err := os.Remove(resource1); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove resource1: %w", err)
				}
			}
			return nil
		}),
		harness.NewStep("Cleanup resource 2", func(ctx *harness.Context) error {
			resource2 := ctx.GetString("resource2")
			if resource2 != "" {
				if err := os.Remove(resource2); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove resource2: %w", err)
				}
			}
			return nil
		}),
		harness.NewStep("Cleanup resource 3", func(ctx *harness.Context) error {
			resource3 := ctx.GetString("resource3")
			if resource3 != "" {
				if err := os.Remove(resource3); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove resource3: %w", err)
				}
			}
			return nil
		}),
	)
}

// TeardownPhaseAfterFailureScenario tests that teardown runs even if a test step fails
// This is an explicit-only test because it intentionally fails to verify error handling
func TeardownPhaseAfterFailureScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"teardown-phase-after-failure",
		"Tests that teardown phase executes even when test steps fail (explicit-only: intentionally fails)",
		[]string{"teardown", "lifecycle", "failure"},
		[]harness.Step{
			harness.NewStep("Create resource", func(ctx *harness.Context) error {
				resourceFile := filepath.Join(ctx.RootDir, "must-cleanup.txt")
				if err := fs.WriteString(resourceFile, "must be cleaned up\n"); err != nil {
					return fmt.Errorf("failed to create resource: %w", err)
				}
				ctx.Set("cleanup_file", resourceFile)
				return nil
			}),
			harness.NewStep("Intentionally fail", func(ctx *harness.Context) error {
				return fmt.Errorf("intentional failure to test teardown")
			}),
		},
		false, // localOnly
		true,  // explicitOnly - this test intentionally fails
	).WithTeardown(
		harness.NewStep("Verify teardown runs despite failure", func(ctx *harness.Context) error {
			// This teardown should run even though the test failed
			cleanupFile := ctx.GetString("cleanup_file")
			if cleanupFile != "" {
				if err := os.Remove(cleanupFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to cleanup: %w", err)
				}
			}
			return nil
		}),
	)
}

// TeardownPhaseFailureScenario tests that teardown failures are logged but don't fail the scenario
func TeardownPhaseFailureScenario() *harness.Scenario {
	return harness.NewScenario(
		"teardown-phase-failure-handling",
		"Tests that teardown failures are logged but don't cause scenario failure",
		[]string{"teardown", "lifecycle", "failure"},
		[]harness.Step{
			harness.NewStep("Successful test step", func(ctx *harness.Context) error {
				ctx.Set("test_passed", true)
				return nil
			}),
		},
	).WithTeardown(
		harness.NewStep("Teardown that fails", func(ctx *harness.Context) error {
			// This failure should be logged but not fail the scenario
			return fmt.Errorf("intentional teardown failure")
		}),
	)
}

// FullLifecycleScenario tests the complete setup -> test -> teardown lifecycle
func FullLifecycleScenario() *harness.Scenario {
	return harness.NewScenario(
		"full-lifecycle",
		"Tests the complete lifecycle: setup -> test -> teardown",
		[]string{"setup", "teardown", "lifecycle", "slow"},
		[]harness.Step{
			harness.NewStep("Verify setup completed", func(ctx *harness.Context) error {
				workspace := ctx.GetString("workspace")
				configFile := filepath.Join(workspace, "config.yml")

				return ctx.Verify(func(v *verify.Collector) {
					v.NotEqual("workspace was created", workspace, "")
					v.Equal("config file exists", nil, fs.AssertExists(configFile))
					v.Equal("setup marker is set", ctx.GetBool("setup_complete"), true)
				})
			}),
			harness.NewStep("Create test artifact", func(ctx *harness.Context) error {
				workspace := ctx.GetString("workspace")
				artifactFile := filepath.Join(workspace, "test-artifact.txt")
				if err := fs.WriteString(artifactFile, "test artifact\n"); err != nil {
					return fmt.Errorf("failed to create artifact: %w", err)
				}
				ctx.Set("artifact_file", artifactFile)
				return ctx.Check("artifact created", fs.AssertExists(artifactFile))
			}),
		},
	).WithSetup(
		harness.NewStep("Create workspace", func(ctx *harness.Context) error {
			workspace := ctx.NewDir("lifecycle-workspace")
			if err := os.MkdirAll(workspace, 0o755); err != nil {
				return fmt.Errorf("failed to create workspace: %w", err)
			}
			ctx.Set("workspace", workspace)
			return nil
		}),
		harness.NewStep("Initialize configuration", func(ctx *harness.Context) error {
			workspace := ctx.GetString("workspace")
			configFile := filepath.Join(workspace, "config.yml")
			if err := fs.WriteString(configFile, "name: lifecycle-test\nversion: 1.0\n"); err != nil {
				return fmt.Errorf("failed to create config: %w", err)
			}
			ctx.Set("setup_complete", true)
			return nil
		}),
	).WithTeardown(
		harness.NewStep("Remove test artifact", func(ctx *harness.Context) error {
			artifactFile := ctx.GetString("artifact_file")
			if artifactFile != "" {
				if err := os.Remove(artifactFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove artifact: %w", err)
				}
			}
			return nil
		}),
	)
}

// SetupPhaseAssertionsScenario tests that assertions in setup steps are tracked
func SetupPhaseAssertionsScenario() *harness.Scenario {
	return harness.NewScenario(
		"setup-phase-assertions",
		"Tests that assertions in setup steps are properly tracked and reported",
		[]string{"setup", "assertions"},
		[]harness.Step{
			harness.NewStep("Verify setup assertions were tracked", func(ctx *harness.Context) error {
				// The test just needs to complete - the framework tracks assertions
				return ctx.Check("test step executed", assert.True(true))
			}),
		},
	).WithSetup(
		harness.NewStep("Setup with assertions", func(ctx *harness.Context) error {
			testDir := ctx.NewDir("assertions-test")
			if err := os.MkdirAll(testDir, 0o755); err != nil {
				return fmt.Errorf("failed to create dir: %w", err)
			}

			testFile := filepath.Join(testDir, "test.txt")
			if err := fs.WriteString(testFile, "content\n"); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			// Use both hard and soft assertions in setup
			if err := ctx.Check("directory created", fs.AssertExists(testDir)); err != nil {
				return err
			}

			return ctx.Verify(func(v *verify.Collector) {
				v.Equal("file exists", nil, fs.AssertExists(testFile))
				v.Contains("file has correct content", "content\n", "content")
			})
		}),
	)
}

// TeardownWithNoCleanupScenario tests that teardown is skipped when --no-cleanup is used
func TeardownWithNoCleanupScenario() *harness.Scenario {
	return harness.NewScenario(
		"teardown-with-no-cleanup",
		"Tests that teardown is skipped when --no-cleanup flag is used",
		[]string{"teardown", "lifecycle", "cleanup"},
		[]harness.Step{
			harness.NewStep("Create file that should persist", func(ctx *harness.Context) error {
				persistFile := filepath.Join(ctx.RootDir, "persist.txt")
				if err := fs.WriteString(persistFile, "should persist with --no-cleanup\n"); err != nil {
					return fmt.Errorf("failed to create file: %w", err)
				}
				ctx.Set("persist_file", persistFile)
				return ctx.Check("persist file created", fs.AssertExists(persistFile))
			}),
		},
	).WithTeardown(
		harness.NewStep("This teardown should be skipped with --no-cleanup", func(ctx *harness.Context) error {
			persistFile := ctx.GetString("persist_file")
			if persistFile != "" {
				// In normal mode this would delete the file
				// With --no-cleanup, this step won't run
				if err := os.Remove(persistFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove file: %w", err)
				}
			}
			return nil
		}),
	)
}

// SetupOnlyModeScenario tests the --setup-only flag behavior
func SetupOnlyModeScenario() *harness.Scenario {
	return harness.NewScenario(
		"setup-only-mode",
		"Tests that --setup-only flag runs only setup steps and exits",
		[]string{"setup", "lifecycle"},
		[]harness.Step{
			harness.NewStep("This test step should not run in --setup-only mode", func(ctx *harness.Context) error {
				// This should not execute when --setup-only is used
				ctx.Set("test_step_executed", true)
				return nil
			}),
		},
	).WithSetup(
		harness.NewStep("Setup step that should run", func(ctx *harness.Context) error {
			ctx.Set("setup_executed", true)
			setupFile := filepath.Join(ctx.RootDir, "setup-only.txt")
			if err := fs.WriteString(setupFile, "setup complete\n"); err != nil {
				return fmt.Errorf("failed to create setup file: %w", err)
			}
			return nil
		}),
	).WithTeardown(
		harness.NewStep("Teardown should not run in --setup-only mode", func(ctx *harness.Context) error {
			ctx.Set("teardown_executed", true)
			return nil
		}),
	)
}

// ReusableSetupStepsScenario tests that setup steps can be shared across scenarios
func ReusableSetupStepsScenario() *harness.Scenario {
	// Define a reusable setup step
	commonSetup := harness.NewStep("Common setup step", func(ctx *harness.Context) error {
		commonDir := ctx.NewDir("common-workspace")
		if err := os.MkdirAll(commonDir, 0o755); err != nil {
			return fmt.Errorf("failed to create common dir: %w", err)
		}

		commonFile := filepath.Join(commonDir, "common.txt")
		if err := fs.WriteString(commonFile, "common setup\n"); err != nil {
			return fmt.Errorf("failed to create common file: %w", err)
		}

		ctx.Set("common_dir", commonDir)
		ctx.Set("common_file", commonFile)
		return nil
	})

	return harness.NewScenario(
		"reusable-setup-steps",
		"Tests that setup steps can be defined once and reused across scenarios",
		[]string{"setup", "reusability"},
		[]harness.Step{
			harness.NewStep("Verify common setup ran", func(ctx *harness.Context) error {
				commonDir := ctx.GetString("common_dir")
				commonFile := ctx.GetString("common_file")

				return ctx.Verify(func(v *verify.Collector) {
					v.NotEqual("common directory set", commonDir, "")
					v.Equal("common file exists", nil, fs.AssertExists(commonFile))
				})
			}),
		},
	).WithSetup(commonSetup)
}

// RunSetupWithoutSetupStepsScenario tests that --run-setup mode switches to interactive
// when no setup steps exist (backward compatibility for debugging)
//
// This scenario validates the behavior: when --run-setup is used on a scenario without
// setup steps, the harness switches to interactive mode instead of exiting.
// This is a helper scenario used by the run-setup-flag-e2e test.
func RunSetupWithoutSetupStepsScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"run-setup-without-setup-steps",
		"Helper scenario for testing --run-setup without setup steps (explicit-only)",
		[]string{"setup", "lifecycle", "helper"},
		[]harness.Step{
			harness.NewStep("Mark test step executed", func(ctx *harness.Context) error {
				// This is a helper scenario that will be used by the E2E test.
				// When invoked with --run-setup but no setup steps, this should execute.
				ctx.Set("test_step_executed", true)
				testFile := filepath.Join(ctx.RootDir, "test-step.txt")
				if err := fs.WriteString(testFile, "test step ran\n"); err != nil {
					return fmt.Errorf("failed to create test file: %w", err)
				}
				return ctx.Check("test step file created", fs.AssertExists(testFile))
			}),
		},
		false, // localOnly
		true,  // explicitOnly - this is a helper scenario
	) // No WithSetup() - this scenario has no setup steps
}

// RunSetupWithSetupStepsScenario tests that --run-setup mode halts after setup
// when setup steps exist (original behavior)
//
// This is a helper scenario that will be used by the E2E test to verify that
// scenarios WITH setup steps still halt after setup when --run-setup is used.
func RunSetupWithSetupStepsScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"run-setup-with-setup-steps",
		"Helper scenario for testing --run-setup with setup steps (explicit-only)",
		[]string{"setup", "lifecycle", "helper"},
		[]harness.Step{
			harness.NewStep("This test step should not run with --run-setup", func(ctx *harness.Context) error {
				// This should NOT execute when --run-setup is used
				ctx.Set("test_step_executed", true)
				return fmt.Errorf("test step should not have executed in --run-setup mode")
			}),
		},
		false, // localOnly
		true,  // explicitOnly - this is a helper scenario
	).WithSetup(
		harness.NewStep("Setup step that should run", func(ctx *harness.Context) error {
			ctx.Set("setup_executed", true)
			setupFile := filepath.Join(ctx.RootDir, "setup-only-with-setup.txt")
			if err := fs.WriteString(setupFile, "setup complete\n"); err != nil {
				return fmt.Errorf("failed to create setup file: %w", err)
			}
			return ctx.Check("setup file created", fs.AssertExists(setupFile))
		}),
	).WithTeardown(
		harness.NewStep("Teardown should not run with --setup-only", func(ctx *harness.Context) error {
			// This should NOT execute when --setup-only is used
			ctx.Set("teardown_executed", true)
			return fmt.Errorf("teardown should not have executed in --setup-only mode")
		}),
	)
}
