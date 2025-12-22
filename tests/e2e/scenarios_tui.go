// File: tests/e2e/scenarios_tui.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/tui"
	"github.com/mattsolo1/grove-tend/pkg/verify"
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

// ExampleAdvancedTuiNavigation demonstrates the advanced navigation and timing controls.
func ExampleAdvancedTuiNavigation() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"example-advanced-tui-navigation",
		"Demonstrates TUI navigation with arrow keys, FindTextLocation, and WaitForUIStable",
		[]string{"example", "tui", "navigation"},
		[]harness.Step{
			harness.NewStep("Launch TUI and wait for it to stabilize", func(ctx *harness.Context) error {
				// Use the pre-built list-tui fixture from tests/e2e/fixtures/bin
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
				binPath := filepath.Join(cwd, "tests", "e2e", "fixtures", "bin", "list-tui")

				if !fs.Exists(binPath) {
					return fmt.Errorf("list-tui fixture not found at %s - run 'make build-e2e-fixtures' first", binPath)
				}

				session, err := ctx.StartTUI(binPath, []string{})
				if err != nil {
					return err
				}
				ctx.Set("advanced_session", session)

				// First, wait for the TUI to actually start by checking for expected content
				if err := session.WaitForText("File Browser", 5*time.Second); err != nil {
					return fmt.Errorf("TUI did not start: %w", err)
				}

				// OLD WAY: time.Sleep(1 * time.Second)
				// NEW WAY: Wait for the UI to stop changing content.
				// This is more reliable than a fixed sleep.
				fmt.Println("   Waiting for UI to stabilize...")
				if err := session.WaitStable(); err != nil {
					return err
				}

				// After stabilizing, verify all items have loaded
				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("README.md is visible", nil, session.AssertContains("README.md"))
					v.Equal("main.go is visible", nil, session.AssertContains("main.go"))
					v.Equal("docs/guide.md is visible", nil, session.AssertContains("docs/guide.md"))
				})
			}),
			harness.NewStep("Test FindTextLocation functionality", func(ctx *harness.Context) error {
				session := ctx.Get("advanced_session").(*tui.Session)

				// Test finding specific text
				fmt.Println("   Searching for 'docs/guide.md'...")
				row, col, found, err := session.FindTextLocation("docs/guide.md")
				if err != nil {
					return fmt.Errorf("failed to find text location: %w", err)
				}
				if !found {
					return fmt.Errorf("text 'docs/guide.md' not found on screen")
				}

				fmt.Printf("   Found 'docs/guide.md' at row %d, col %d\n", row, col)
				return nil
			}),
			harness.NewStep("Navigate to docs/guide.md using NavigateToText", func(ctx *harness.Context) error {
				session := ctx.Get("advanced_session").(*tui.Session)

				// OLD WAY (brittle - breaks if order changes):
				// session.Type("Down")
				// session.Type("Down")

				// NEW WAY: Navigate directly using NavigateToText
				fmt.Println("   Using NavigateToText to select 'docs/guide.md'...")
				if err := session.NavigateToText("docs/guide.md"); err != nil {
					return fmt.Errorf("failed to navigate: %w", err)
				}

				// Verify the selection indicator moved
				if err := session.AssertLine(func(line string) bool {
					return strings.Contains(line, "> docs/guide.md")
				}, "expected '> docs/guide.md' to be selected"); err != nil {
					return err
				}

				fmt.Println("   ✓ Successfully selected 'docs/guide.md'")
				return nil
			}),
			harness.NewStep("Navigate back to main.go using NavigateToText", func(ctx *harness.Context) error {
				session := ctx.Get("advanced_session").(*tui.Session)

				// Navigate back using NavigateToText
				fmt.Println("   Using NavigateToText to select 'main.go'...")
				if err := session.NavigateToText("main.go"); err != nil {
					return fmt.Errorf("failed to navigate: %w", err)
				}

				// Verify the selection indicator moved
				if err := session.AssertLine(func(line string) bool {
					return strings.Contains(line, "> main.go")
				}, "expected '> main.go' to be selected"); err != nil {
					return err
				}

				fmt.Println("   ✓ Successfully selected 'main.go'")
				fmt.Println("   ✓ NavigateToText works for selection-based TUIs!")
				return nil
			}),
			harness.NewStep("Test SelectItem functionality", func(ctx *harness.Context) error {
				session := ctx.Get("advanced_session").(*tui.Session)

				fmt.Println("   Using SelectItem to choose and select 'README.md'...")
				if err := session.SelectItem(func(line string) bool {
					return strings.Contains(line, "README.md")
				}); err != nil {
					return fmt.Errorf("failed to select item: %w", err)
				}

				// Verify the selection was made by checking the output
				return ctx.Check("README.md selection confirmed",
					session.AssertContains("Selected: README.md"))
			}),
			harness.NewStep("Cleanup", func(ctx *harness.Context) error {
				session := ctx.Get("advanced_session").(*tui.Session)
				return session.Type("q")
			}),
		},
		true,  // localOnly - TUI tests require tmux
		false, // explicitOnly
	)
}

// ExampleConditionalFlowsAndRecording demonstrates the new conditional flow and recording features.
func ExampleConditionalFlowsAndRecording() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"example-conditional-flows-recording",
		"Demonstrates WaitForAnyText, pattern matching, SelectItem, and session recording",
		[]string{"example", "tui", "conditional", "recording"},
		[]harness.Step{
			harness.NewStep("Launch task manager TUI and start recording", func(ctx *harness.Context) error {
				// Use pre-built task-manager fixture
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
				binPath := filepath.Join(cwd, "tests", "e2e", "fixtures", "bin", "task-manager")

				if !fs.Exists(binPath) {
					return fmt.Errorf("task-manager fixture not found at %s - run 'make build-e2e-fixtures' first", binPath)
				}

				session, err := ctx.StartTUI(binPath, []string{})
				if err != nil {
					return err
				}
				ctx.Set("cond_session", session)

				// Start recording the session
				recordingPath := filepath.Join(ctx.RootDir, "session-recording")
				if err := session.StartRecording(recordingPath); err != nil {
					return fmt.Errorf("failed to start recording: %w", err)
				}
				fmt.Printf("   📹 Recording session to: %s\n", recordingPath)

				// Wait for menu to appear
				if err := session.WaitForText("Select action:", 5*time.Second); err != nil {
					return err
				}

				return nil
			}),
			harness.NewStep("Select option and wait for processing", func(ctx *harness.Context) error {
				session := ctx.Get("cond_session").(*tui.Session)

				// Choose option 1 (Process files)
				fmt.Println("   Selecting 'Process files' option...")
				if err := session.Type("1"); err != nil {
					return err
				}

				return nil
			}),
			harness.NewStep("Test WaitForAnyText for conditional outcomes", func(ctx *harness.Context) error {
				session := ctx.Get("cond_session").(*tui.Session)

				// Wait for one of multiple possible outcomes
				fmt.Println("   Waiting for operation result...")
				result, err := session.WaitForAnyText(
					[]string{"✓ Success", "✗ Failed", "⚠ Warning"},
					3*time.Second,
				)
				if err != nil {
					return fmt.Errorf("failed waiting for outcome: %w", err)
				}

				fmt.Printf("   Got result: %s\n", result)

				// Handle based on result
				switch result {
				case "✓ Success":
					fmt.Println("   ✅ Operation completed successfully!")
				case "✗ Failed":
					fmt.Println("   ❌ Operation failed!")
				case "⚠ Warning":
					fmt.Println("   ⚠️  Operation completed with warnings")
				}

				return nil
			}),
			harness.NewStep("Test pattern matching for file counts", func(ctx *harness.Context) error {
				session := ctx.Get("cond_session").(*tui.Session)

				// Use regex pattern to find file counts
				fmt.Println("   Looking for file count patterns...")
				pattern := regexp.MustCompile(`\d+ files? (modified|added|deleted)`)

				match, err := session.WaitForTextPattern(pattern, 2*time.Second)
				if err != nil {
					// Pattern might not be present depending on choice
					fmt.Println("   No file counts found (might have chosen different option)")
					return nil
				}

				fmt.Printf("   Found pattern match: %s\n", match)

				// Assert pattern exists
				if err := session.AssertContainsPattern(pattern); err != nil {
					return fmt.Errorf("pattern assertion failed: %w", err)
				}

				return nil
			}),
			harness.NewStep("Take screenshot and stop recording", func(ctx *harness.Context) error {
				session := ctx.Get("cond_session").(*tui.Session)

				// Take a screenshot
				screenshotPath := filepath.Join(ctx.RootDir, "final-state.ansi")
				if err := session.TakeScreenshot(screenshotPath); err != nil {
					return fmt.Errorf("failed to take screenshot: %w", err)
				}
				fmt.Printf("   📸 Screenshot saved to: %s\n", screenshotPath)

				// Stop recording
				if err := session.StopRecording(); err != nil {
					return fmt.Errorf("failed to stop recording: %w", err)
				}

				// Show key history
				history := session.GetKeyHistory()
				fmt.Printf("   Key history: %v\n", history)

				recordingPath := filepath.Join(ctx.RootDir, "session-recording")
				fmt.Printf("   📊 Recording saved to:\n")
				fmt.Printf("      - HTML: %s.html\n", recordingPath)
				fmt.Printf("      - JSON: %s.json\n", recordingPath)

				return nil
			}),
			harness.NewStep("Cleanup", func(ctx *harness.Context) error {
				session := ctx.Get("cond_session").(*tui.Session)
				return session.Type("q")
			}),
		},
		true,  // localOnly - TUI tests require tmux
		false, // explicitOnly
	)
}

// ExampleFilesystemInteractionScenario demonstrates testing a TUI that interacts with the filesystem
func ExampleFilesystemInteractionScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"example-tui-filesystem",
		"Tests a TUI that writes to the filesystem",
		[]string{"example", "tui", "filesystem"},
		[]harness.Step{
			harness.NewStep("Launch file-saver TUI", func(ctx *harness.Context) error {
				// Use pre-built file-saver fixture
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
				binPath := filepath.Join(cwd, "tests", "e2e", "fixtures", "bin", "file-saver")

				if !fs.Exists(binPath) {
					return fmt.Errorf("file-saver fixture not found at %s - run 'make build-e2e-fixtures' first", binPath)
				}

				session, err := ctx.StartTUI(binPath, []string{}, tui.WithCwd(ctx.RootDir))
				if err != nil {
					return err
				}
				ctx.Set("fs_session", session)

				// Wait for the TUI to be ready
				return ctx.Check("TUI is ready with save prompt",
					session.WaitForText("Press 's' to save", 5*time.Second))
			}),
			harness.NewStep("Save file using 's' key", func(ctx *harness.Context) error {
				session := ctx.Get("fs_session").(*tui.Session)

				// Press 's' to save
				if err := session.Type("s"); err != nil {
					return err
				}

				// Wait for confirmation message
				return ctx.Check("file save confirmation appears",
					session.WaitForText("File saved to output.txt", 2*time.Second))
			}),
			harness.NewStep("Verify file creation and content", func(ctx *harness.Context) error {
				session := ctx.Get("fs_session").(*tui.Session)

				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("output.txt file was created", nil,
						session.WaitForFile("output.txt", 5*time.Second))
					v.Equal("file contains expected content", nil,
						session.AssertFileContains("output.txt", "saved at"))
				})
			}),
			harness.NewStep("Cleanup", func(ctx *harness.Context) error {
				session := ctx.Get("fs_session").(*tui.Session)
				return session.Type("q")
			}),
		},
		true,  // localOnly - TUI tests require tmux
		false, // explicitOnly
	)
}
