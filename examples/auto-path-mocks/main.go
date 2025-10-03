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

// AutoPathMocksScenario demonstrates automatic PATH handling for TUI sessions with mocks.
// When test_bin_dir is set in the context, StartTUI automatically prepends it to PATH,
// allowing mock binaries to be found without manual PATH manipulation or wrapper scripts.
func AutoPathMocksScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "auto-path-mocks",
		Description: "Tests automatic PATH prepending for mock binaries in TUI sessions",
		Tags:        []string{"tui", "mocks", "path", "example"},
		Steps: []harness.Step{
			{
				Name:        "Create mock binaries",
				Description: "Creates a directory with mock executables",
				Func: func(ctx *harness.Context) error {
					// Create a directory for mock binaries
					mockDir := ctx.NewDir("mocks")
					if err := fs.CreateDir(mockDir); err != nil {
						return fmt.Errorf("failed to create mock directory: %w", err)
					}

					// Create a mock 'git' command that prints a custom message
					mockGitPath := filepath.Join(mockDir, "git")
					mockGitContent := `#!/bin/bash
echo "MOCK GIT: This is a mock git binary!"
echo "MOCK GIT: Called with args: $@"
`
					if err := fs.WriteString(mockGitPath, mockGitContent); err != nil {
						return fmt.Errorf("failed to create mock git: %w", err)
					}
					if err := os.Chmod(mockGitPath, 0755); err != nil {
						return fmt.Errorf("failed to chmod mock git: %w", err)
					}

					// Create a mock 'curl' command
					mockCurlPath := filepath.Join(mockDir, "curl")
					mockCurlContent := `#!/bin/bash
echo "MOCK CURL: Pretending to fetch $@"
echo "MOCK CURL: Mock response data"
`
					if err := fs.WriteString(mockCurlPath, mockCurlContent); err != nil {
						return fmt.Errorf("failed to create mock curl: %w", err)
					}
					if err := os.Chmod(mockCurlPath, 0755); err != nil {
						return fmt.Errorf("failed to chmod mock curl: %w", err)
					}

					// IMPORTANT: Store mock directory using the 'test_bin_dir' key.
					// This is the convention that StartTUI looks for to automatically
					// prepend to PATH. No manual PATH manipulation needed!
					ctx.Set("test_bin_dir", mockDir)

					fmt.Printf("   Created mocks in: %s\n", mockDir)
					return nil
				},
			},
			{
				Name:        "Create test script that prints PATH and calls mocks",
				Description: "Creates a script that will show PATH and call mocked commands",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("test-scripts")
					if err := fs.CreateDir(testDir); err != nil {
						return fmt.Errorf("failed to create test directory: %w", err)
					}

					scriptPath := filepath.Join(testDir, "test-mocks.sh")
					scriptContent := `#!/bin/bash
echo "=== Testing Mock Binaries ==="
echo "PATH begins with:"
echo "$PATH" | cut -d: -f1
echo ""
echo "Calling 'git status':"
git status
echo ""
echo "Calling 'curl https://example.com':"
curl https://example.com
echo "=== Test Complete ==="
`
					if err := fs.WriteString(scriptPath, scriptContent); err != nil {
						return fmt.Errorf("failed to create test script: %w", err)
					}
					if err := os.Chmod(scriptPath, 0755); err != nil {
						return fmt.Errorf("failed to make script executable: %w", err)
					}

					ctx.Set("test_script", scriptPath)
					return nil
				},
			},
			{
				Name:        "Launch TUI with automatic PATH handling",
				Description: "StartTUI automatically prepends test_bin_dir to PATH",
				Func: func(ctx *harness.Context) error {
					scriptPath := ctx.GetString("test_script")

					// Launch TUI WITHOUT any manual PATH manipulation.
					// Because we set 'test_bin_dir' in the context, StartTUI
					// will automatically prepend it to PATH for the subprocess.
					// This is the new, cleaner approach!
					session, err := ctx.StartTUI(scriptPath, []string{})
					if err != nil {
						return fmt.Errorf("failed to start TUI: %w", err)
					}

					fmt.Println("   ✓ TUI launched with automatic PATH handling")
					ctx.Set("tui_session", session)
					return nil
				},
			},
			{
				Name:        "Verify mocks were called",
				Description: "Checks that our mock binaries were executed instead of real ones",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("tui_session").(*tui.Session)

					// Wait for the script to complete execution
					if err := session.WaitForText("Test Complete", 10*time.Second); err != nil {
						// If we can't find the completion text, still capture and show what we got
						content, _ := session.Capture()
						fmt.Printf("\n   Debug - Captured output:\n%s\n", content)
						return fmt.Errorf("script did not complete: %w", err)
					}

					// Capture the output
					content, err := session.Capture()
					if err != nil {
						return fmt.Errorf("failed to capture output: %w", err)
					}

					fmt.Printf("\n   Captured TUI output:\n---\n%s\n---\n", content)

					// Verify our mock directory is first in PATH
					mockDir := ctx.GetString("test_bin_dir")
					if !strings.Contains(content, mockDir) {
						return fmt.Errorf("mock directory not found in PATH")
					}

					// Verify our mock git was called (not the real git)
					if !strings.Contains(content, "MOCK GIT") {
						return fmt.Errorf("mock git was not called - PATH not set correctly")
					}

					// Verify our mock curl was called
					if !strings.Contains(content, "MOCK CURL") {
						return fmt.Errorf("mock curl was not called - PATH not set correctly")
					}

					fmt.Println("   ✓ Mock directory is first in PATH!")
					fmt.Println("   ✓ Mock binaries successfully executed via automatic PATH!")
					fmt.Println("   ✓ No wrapper scripts or manual PATH manipulation needed!")
					return nil
				},
			},
		},
	}
}

func main() {
	scenarios := []*harness.Scenario{
		AutoPathMocksScenario(),
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
