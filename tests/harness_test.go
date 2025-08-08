package tend_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/fs"
)

func TestHarnessExecution(t *testing.T) {
	// Create a simple test scenario
	scenario := &harness.Scenario{
		Name:        "test-scenario",
		Description: "A test scenario to validate the harness",
		Tags:        []string{"test", "validation"},
		Steps: []harness.Step{
			{
				Name: "Create test directory",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("test")
					if err := fs.CreateDir(testDir); err != nil {
						return err
					}
					ctx.Set("created_dir", testDir)
					return nil
				},
			},
			{
				Name: "Write test file",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.Dir("test")
					if testDir == "" {
						return fmt.Errorf("test directory not found")
					}
					
					testFile := filepath.Join(testDir, "test.txt")
					return fs.WriteString(testFile, "Hello from harness test!")
				},
			},
			{
				Name: "Verify file exists",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.Dir("test")
					testFile := filepath.Join(testDir, "test.txt")
					
					if !fs.Exists(testFile) {
						return fmt.Errorf("test file does not exist")
					}
					return nil
				},
			},
		},
	}

	// Run the scenario
	h := harness.New(harness.Options{
		Verbose:   true,
		NoCleanup: false, // Allow cleanup for testing
	})

	ctx := context.Background()
	result, err := h.Run(ctx, scenario)

	if err != nil {
		t.Fatalf("Scenario failed: %v", err)
	}

	if !result.Success {
		t.Fatal("Scenario should have succeeded")
	}

	if result.ScenarioName != "test-scenario" {
		t.Errorf("Expected scenario name 'test-scenario', got %s", result.ScenarioName)
	}

	if len(result.StepResults) != 3 {
		t.Errorf("Expected 3 step results, got %d", len(result.StepResults))
	}

	// Verify all steps succeeded
	for i, stepResult := range result.StepResults {
		if !stepResult.Success {
			t.Errorf("Step %d (%s) should have succeeded", i, stepResult.Name)
		}
		if stepResult.Error != nil {
			t.Errorf("Step %d (%s) should not have error: %v", i, stepResult.Name, stepResult.Error)
		}
	}

	t.Logf("Scenario completed in %v", result.Duration)
}

func TestHarnessFailure(t *testing.T) {
	// Create a scenario that should fail
	scenario := &harness.Scenario{
		Name:        "failing-scenario",
		Description: "A scenario that should fail",
		Steps: []harness.Step{
			{
				Name: "Successful step",
				Func: func(ctx *harness.Context) error {
					return nil
				},
			},
			{
				Name: "Failing step",
				Func: func(ctx *harness.Context) error {
					return fmt.Errorf("intentional failure")
				},
			},
			{
				Name: "Should not run",
				Func: func(ctx *harness.Context) error {
					t.Error("This step should not have run")
					return nil
				},
			},
		},
	}

	h := harness.New(harness.Options{
		Verbose: false, // Reduce noise in test output
	})

	ctx := context.Background()
	result, err := h.Run(ctx, scenario)

	if err == nil {
		t.Fatal("Scenario should have failed")
	}

	if result.Success {
		t.Fatal("Result should indicate failure")
	}

	if result.FailedStep != "Failing step" {
		t.Errorf("Expected failed step 'Failing step', got %s", result.FailedStep)
	}

	// Should have run 2 steps (successful + failing)
	if len(result.StepResults) != 2 {
		t.Errorf("Expected 2 step results, got %d", len(result.StepResults))
	}

	// First step should succeed, second should fail
	if !result.StepResults[0].Success {
		t.Error("First step should have succeeded")
	}
	if result.StepResults[1].Success {
		t.Error("Second step should have failed")
	}
}

func TestStepBuilders(t *testing.T) {
	// Test the step builder functions
	ctx := &harness.Context{
		RootDir: "/tmp/test",
	}

	t.Run("NewStep", func(t *testing.T) {
		step := harness.NewStep("test step", func(ctx *harness.Context) error {
			return nil
		})

		if step.Name != "test step" {
			t.Errorf("Expected name 'test step', got %s", step.Name)
		}

		if err := step.Func(ctx); err != nil {
			t.Errorf("Step function should not error: %v", err)
		}
	})

	t.Run("DelayStep", func(t *testing.T) {
		step := harness.DelayStep("delay", 10*time.Millisecond)
		
		start := time.Now()
		err := step.Func(ctx)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Delay step should not error: %v", err)
		}

		if duration < 10*time.Millisecond {
			t.Errorf("Delay should be at least 10ms, was %v", duration)
		}
	})

	t.Run("ConditionalStep", func(t *testing.T) {
		// Test condition that returns true
		step := harness.ConditionalStep("conditional", 
			func(*harness.Context) bool { return true },
			func(*harness.Context) error { return nil })

		if err := step.Func(ctx); err != nil {
			t.Errorf("Conditional step (true) should not error: %v", err)
		}

		// Test condition that returns false
		step = harness.ConditionalStep("conditional",
			func(*harness.Context) bool { return false },
			func(*harness.Context) error { return fmt.Errorf("should not run") })

		if err := step.Func(ctx); err != nil {
			t.Errorf("Conditional step (false) should not error: %v", err)
		}
	})
}

func TestCommandIntegration(t *testing.T) {
	// Test that command package works with harness
	scenario := &harness.Scenario{
		Name: "command-test",
		Steps: []harness.Step{
			{
				Name: "Run echo command",
				Func: func(ctx *harness.Context) error {
					cmd := command.New("echo", "hello", "world")
					result := cmd.Run()
					
					if result.Error != nil {
						return result.Error
					}
					
					if result.ExitCode != 0 {
						return fmt.Errorf("expected exit code 0, got %d", result.ExitCode)
					}
					
					ctx.Set("echo_output", result.Stdout)
					return nil
				},
			},
			{
				Name: "Verify echo output",
				Func: func(ctx *harness.Context) error {
					output := ctx.GetString("echo_output")
					if output == "" {
						return fmt.Errorf("echo output not found")
					}
					
					expected := "hello world\n"
					if output != expected {
						return fmt.Errorf("expected %q, got %q", expected, output)
					}
					
					return nil
				},
			},
		},
	}

	h := harness.New(harness.Options{Verbose: false})
	result, err := h.Run(context.Background(), scenario)

	if err != nil {
		t.Fatalf("Command scenario failed: %v", err)
	}

	if !result.Success {
		t.Fatal("Command scenario should have succeeded")
	}
}