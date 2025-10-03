package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/tui"
)

// EnvPassingTestScenario demonstrates passing environment variables to TUI sessions
func EnvPassingTestScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "env-passing-test",
		Description: "Tests that environment variables are correctly passed to TUI subprocess",
		Tags:        []string{"tui", "env", "test"},
		Steps: []harness.Step{
			{
				Name:        "Create test script that prints env vars",
				Description: "Creates a simple script that echoes an environment variable",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("env-test")
					if err := fs.CreateDir(testDir); err != nil {
						return fmt.Errorf("failed to create test directory: %w", err)
					}

					scriptPath := filepath.Join(testDir, "env-test.sh")
					scriptContent := `#!/bin/bash
echo "CLICOLOR_FORCE is: $CLICOLOR_FORCE"
echo "CUSTOM_VAR is: $CUSTOM_VAR"
sleep 1
`
					if err := fs.WriteString(scriptPath, scriptContent); err != nil {
						return fmt.Errorf("failed to create script: %w", err)
					}

					if err := os.Chmod(scriptPath, 0755); err != nil {
						return fmt.Errorf("failed to make script executable: %w", err)
					}

					ctx.Set("env_script", scriptPath)
					return nil
				},
			},
			{
				Name:        "Launch TUI with environment variables",
				Description: "Starts TUI with CLICOLOR_FORCE and custom env vars",
				Func: func(ctx *harness.Context) error {
					scriptPath := ctx.GetString("env_script")
					if scriptPath == "" {
						return fmt.Errorf("script path not found")
					}

					// Launch with environment variables using the new WithEnv option
					session, err := ctx.StartTUI(
						"/bin/bash",
						[]string{scriptPath},
						tui.WithEnv("CLICOLOR_FORCE=1", "CUSTOM_VAR=test_value"),
					)
					if err != nil {
						return fmt.Errorf("failed to start TUI: %w", err)
					}

					ctx.Set("env_session", session)
					return nil
				},
			},
			{
				Name:        "Verify environment variables were set",
				Description: "Checks that the environment variables are visible in the TUI output",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("env_session").(*tui.Session)

					// Wait for the script to output the environment variables
					time.Sleep(500 * time.Millisecond)

					// Capture the output
					content, err := session.Capture(tui.WithCleanedOutput())
					if err != nil {
						return fmt.Errorf("failed to capture output: %w", err)
					}

					fmt.Printf("\n   Captured output:\n---\n%s\n---\n", content)

					// Verify CLICOLOR_FORCE was set
					if !strings.Contains(content, "CLICOLOR_FORCE is: 1") {
						return fmt.Errorf("CLICOLOR_FORCE not set correctly in subprocess")
					}

					// Verify CUSTOM_VAR was set
					if !strings.Contains(content, "CUSTOM_VAR is: test_value") {
						return fmt.Errorf("CUSTOM_VAR not set correctly in subprocess")
					}

					fmt.Println("   ✓ Environment variables successfully passed to TUI subprocess!")
					return nil
				},
			},
		},
	}
}

func main() {
	scenarios := []*harness.Scenario{
		EnvPassingTestScenario(),
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Execute the test
	if err := app.Execute(ctx, scenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
