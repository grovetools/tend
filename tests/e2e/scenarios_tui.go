// File: tests/e2e/scenarios_tui.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/tui"
)

// AutoPathMocksScenario demonstrates automatic PATH handling for TUI sessions with mocks.
func AutoPathMocksScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"auto-path-mocks",
		"Tests automatic PATH prepending for mock binaries in TUI sessions",
		[]string{"tui", "mocks", "path"},
		[]harness.Step{
			harness.NewStep("Create mock binaries", func(ctx *harness.Context) error {
				mockDir := ctx.NewDir("mocks")
				if err := os.MkdirAll(mockDir, 0755); err != nil {
					return fmt.Errorf("failed to create mocks directory: %w", err)
				}
				ctx.Set("test_bin_dir", mockDir)

				mockGitPath := filepath.Join(mockDir, "git")
				if err := fs.WriteString(mockGitPath, `#!/bin/bash
echo "MOCK GIT: This is a mock git binary!"`); err != nil {
					return err
				}
				return os.Chmod(mockGitPath, 0755)
			}),
			harness.NewStep("Create test script", func(ctx *harness.Context) error {
				scriptDir := ctx.NewDir("test-scripts")
				if err := os.MkdirAll(scriptDir, 0755); err != nil {
					return fmt.Errorf("failed to create test-scripts directory: %w", err)
				}
				scriptPath := filepath.Join(scriptDir, "test-mocks.sh")
				if err := fs.WriteString(scriptPath, `#!/bin/bash
set -e
echo "Testing mock binaries"
git status
echo "Mock test complete"`); err != nil {
					return err
				}
				ctx.Set("test_script", scriptPath)
				return os.Chmod(scriptPath, 0755)
			}),
			harness.NewStep("Launch TUI with automatic PATH handling", func(ctx *harness.Context) error {
				scriptPath := ctx.GetString("test_script")
				session, err := ctx.StartTUI("/bin/bash", []string{scriptPath})
				if err != nil {
					return err
				}
				ctx.Set("tui_session", session)
				return nil
			}),
			harness.NewStep("Verify mocks were called", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)
				// Wait for script completion - if this succeeds, the mock was called
				if err := session.WaitForText("Mock test complete", 5*time.Second); err != nil {
					content, _ := session.Capture()
					return fmt.Errorf("script did not complete: %w\nOutput:\n%s", err, content)
				}
				return nil
			}),
		},
		true,  // localOnly - TUI tests require tmux which may not be available in CI
		false, // explicitOnly
	)
}

// EnvPassingTestScenario demonstrates passing environment variables to TUI sessions
func EnvPassingTestScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"env-passing-test",
		"Tests that environment variables are correctly passed to TUI subprocess",
		[]string{"tui", "env"},
		[]harness.Step{
			harness.NewStep("Create test script that prints env vars", func(ctx *harness.Context) error {
				scriptDir := ctx.NewDir("env-test")
				if err := os.MkdirAll(scriptDir, 0755); err != nil {
					return fmt.Errorf("failed to create env-test directory: %w", err)
				}
				scriptPath := filepath.Join(scriptDir, "env-test.sh")
				if err := fs.WriteString(scriptPath, `#!/bin/bash
echo "CUSTOM_VAR is: $CUSTOM_VAR"`); err != nil {
					return err
				}
				ctx.Set("env_script", scriptPath)
				return os.Chmod(scriptPath, 0755)
			}),
			harness.NewStep("Launch TUI with environment variables", func(ctx *harness.Context) error {
				scriptPath := ctx.GetString("env_script")
				session, err := ctx.StartTUI("/bin/bash", []string{scriptPath}, tui.WithEnv("CUSTOM_VAR=test_value"))
				if err != nil {
					return err
				}
				ctx.Set("env_session", session)
				return nil
			}),
			harness.NewStep("Verify environment variables were set", func(ctx *harness.Context) error {
				session := ctx.Get("env_session").(*tui.Session)
				return session.WaitForText("CUSTOM_VAR is: test_value", 5*time.Second)
			}),
		},
		true,  // localOnly - TUI tests require tmux which may not be available in CI
		false, // explicitOnly
	)
}
