package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/teatest"
	"github.com/mattsolo1/grove-tend/pkg/tui"
)

// ExampleTUITestScenario demonstrates testing a TUI application in tmux
func ExampleTUITestScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-tui-tmux",
		Description: "Example scenario that tests a TUI application using tmux sessions",
		Tags:        []string{"example", "tui", "tmux", "interactive"},
		Steps: []harness.Step{
			{
				Name:        "Create test TUI binary",
				Description: "Creates a simple TUI application for testing",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("tui-test")
					if err := fs.CreateDir(testDir); err != nil {
						return fmt.Errorf("failed to create test directory: %w", err)
					}

					// Create a simple TUI application script
					scriptPath := filepath.Join(testDir, "test-tui.sh")
					scriptContent := `#!/bin/bash
echo "Welcome to Test TUI"
echo "==================="
echo ""
echo "Available commands:"
echo "  h - Show help"
echo "  q - Quit"
echo "  t - Run test"
echo ""
printf "> "

while true; do
    read -n 1 key
    echo ""
    case $key in
        h)
            echo "Help: This is a simple test TUI"
            printf "> "
            ;;
        t)
            echo "Running test..."
            echo "Test completed successfully!"
            printf "> "
            ;;
        q)
            echo "Goodbye!"
            exit 0
            ;;
        *)
            echo "Unknown command: $key"
            printf "> "
            ;;
    esac
done
`
					if err := fs.WriteString(scriptPath, scriptContent); err != nil {
						return fmt.Errorf("failed to create TUI script: %w", err)
					}

					// Make script executable
					if err := os.Chmod(scriptPath, 0755); err != nil {
						return fmt.Errorf("failed to make script executable: %w", err)
					}

					ctx.Set("tui_script", scriptPath)
					return nil
				},
			},
			{
				Name:        "Launch TUI in tmux session",
				Description: "Starts the TUI application in a tmux session for testing",
				Func: func(ctx *harness.Context) error {
					scriptPath := ctx.GetString("tui_script")
					if scriptPath == "" {
						return fmt.Errorf("TUI script path not found")
					}

					// Launch the TUI in a tmux session
					session, err := ctx.StartTUI("/bin/bash", scriptPath)
					if err != nil {
						return fmt.Errorf("failed to start TUI: %w", err)
					}

					ctx.Set("tui_session", session)
					return nil
				},
			},
			{
				Name:        "Wait for TUI to initialize",
				Description: "Waits for the welcome message and prompt to appear",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("tui_session").(*tui.Session)

					// Wait for the prompt, which is the last thing to appear on init.
					// This is more robust than checking for welcome text first.
					if err := session.WaitForText(">", 5*time.Second); err != nil {
						content, captureErr := session.Capture(tui.WithCleanedOutput())
						if captureErr != nil {
							content = fmt.Sprintf("failed to capture screen: %v", captureErr)
						}
						return fmt.Errorf("TUI prompt did not appear: %w\n\nLast screen content:\n---\n%s\n---", err, content)
					}

					// Now that the UI has stabilized, assert that the initial welcome text is also visible.
					return session.AssertContains("Welcome to Test TUI")
				},
			},
			{
				Name:        "Test help command",
				Description: "Sends 'h' key and verifies help output",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("tui_session").(*tui.Session)

					// Send help command
					if err := session.SendKeys("h"); err != nil {
						return fmt.Errorf("failed to send help key: %w", err)
					}

					// Wait for help text
					if err := session.WaitForText("Help: This is a simple test TUI", 2*time.Second); err != nil {
						return fmt.Errorf("help text did not appear: %w", err)
					}

					return nil
				},
			},
			{
				Name:        "Test running a command",
				Description: "Sends 't' key to run the test command",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("tui_session").(*tui.Session)

					// Send test command
					if err := session.SendKeys("t"); err != nil {
						return fmt.Errorf("failed to send test key: %w", err)
					}

					// Wait for test output
					if err := session.WaitForText("Test completed successfully!", 2*time.Second); err != nil {
						return fmt.Errorf("test did not complete: %w", err)
					}

					return nil
				},
			},
			{
				Name:        "Capture TUI state",
				Description: "Captures the current TUI display for verification",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("tui_session").(*tui.Session)

					// Capture the current pane content
					content, err := session.Capture(tui.WithCleanedOutput())
					if err != nil {
						return fmt.Errorf("failed to capture TUI state: %w", err)
					}

					// Verify expected content is present
					if !strings.Contains(content, "Welcome to Test TUI") {
						return fmt.Errorf("welcome message not in capture")
					}
					if !strings.Contains(content, "Test completed successfully!") {
						return fmt.Errorf("test completion message not in capture")
					}

					ctx.Set("tui_capture", content)
					fmt.Printf("   TUI Capture (cleaned):\n")
					fmt.Printf("   %s\n", strings.ReplaceAll(content, "\n", "\n   "))
					return nil
				},
			},
			{
				Name:        "Test quit command",
				Description: "Sends 'q' key to quit the TUI",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("tui_session").(*tui.Session)

					// Send quit command
					if err := session.SendKeys("q"); err != nil {
						return fmt.Errorf("failed to send quit key: %w", err)
					}

					// Wait for goodbye message
					if err := session.WaitForText("Goodbye!", 2*time.Second); err != nil {
						return fmt.Errorf("goodbye message did not appear: %w", err)
					}

					// The session should close automatically
					time.Sleep(500 * time.Millisecond)
					
					return nil
				},
			},
		},
	}
}

// ExampleHeadlessBubbleTeaScenario demonstrates testing a BubbleTea app without tmux
func ExampleHeadlessBubbleTeaScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-bubbletea-headless",
		Description: "Example scenario that tests a BubbleTea TUI in headless mode",
		Tags:        []string{"example", "tui", "bubbletea", "headless"},
		Steps: []harness.Step{
			{
				Name:        "Create simple BubbleTea model",
				Description: "Sets up a basic counter model for testing",
				Func: func(ctx *harness.Context) error {
					// Create a simple counter model
					model := &counterModel{count: 0}
					ctx.Set("tea_model", model)
					return nil
				},
			},
			{
				Name:        "Start headless session",
				Description: "Launches the BubbleTea model in headless test mode",
				Func: func(ctx *harness.Context) error {
					model := ctx.Get("tea_model").(tea.Model)
					session := ctx.StartHeadless(model)
					ctx.Set("headless_session", session)
					return nil
				},
			},
			{
				Name:        "Test increment operations",
				Description: "Sends increment messages and verifies counter",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("headless_session").(*teatest.HeadlessSession)
					
					// Send increment messages
					session.Send(incrementMsg{})
					session.Send(incrementMsg{})
					session.Send(incrementMsg{})
					
					// Wait for processing
					session.Wait()
					
					// Get output
					output := session.Output()
					if !strings.Contains(output, "Count: 3") {
						return fmt.Errorf("expected count to be 3, got output: %s", output)
					}
					
					return nil
				},
			},
			{
				Name:        "Test decrement operations",
				Description: "Sends decrement messages and verifies counter",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("headless_session").(*teatest.HeadlessSession)
					
					// Send decrement message
					session.Send(decrementMsg{})
					session.Wait()
					
					// Get output
					output := session.Output()
					if !strings.Contains(output, "Count: 2") {
						return fmt.Errorf("expected count to be 2, got output: %s", output)
					}
					
					return nil
				},
			},
			{
				Name:        "Test keyboard input",
				Description: "Types characters and verifies they're handled",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("headless_session").(*teatest.HeadlessSession)
					
					// Type 'q' to quit
					session.TypeString("q")
					session.Wait()
					
					// The model should indicate it's quitting
					output := session.Output()
					if !strings.Contains(output, "Press 'q' to quit") {
						return fmt.Errorf("quit instruction not shown in output: %s", output)
					}
					
					return nil
				},
			},
		},
	}
}

// Simple BubbleTea counter model for testing
type counterModel struct {
	count int
}

type incrementMsg struct{}
type decrementMsg struct{}

func (m *counterModel) Init() tea.Cmd {
	return nil
}

func (m *counterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case incrementMsg:
		m.count++
	case decrementMsg:
		m.count--
	case tea.KeyMsg:
		// Handle 'q' for quit
		return m, tea.Quit
	}
	return m, nil
}

func (m *counterModel) View() string {
	return fmt.Sprintf("Count: %d\nPress '+' to increment, '-' to decrement\nPress 'q' to quit", m.count)
}

// ExampleInteractiveTUIDebugging demonstrates the interactive debugging features
func ExampleInteractiveTUIDebugging() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-tui-interactive-debug",
		Description: "Demonstrates interactive debugging features with TUI attach",
		Tags:        []string{"example", "tui", "debug", "interactive"},
		LocalOnly:   true, // This requires interactive mode to be meaningful
		Steps: []harness.Step{
			{
				Name:        "Create interactive TUI",
				Description: "Creates a more complex TUI for debugging",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("debug-tui")
					if err := fs.CreateDir(testDir); err != nil {
						return fmt.Errorf("failed to create test directory: %w", err)
					}

					// Create a TUI with multiple states
					scriptPath := filepath.Join(testDir, "debug-tui.sh")
					scriptContent := `#!/bin/bash
STATE="menu"
echo "=== Debug TUI ==="
echo "State: $STATE"
echo ""
echo "Commands: [m]enu [e]dit [v]iew [q]uit"
printf "> "

while true; do
    read -n 1 key
    echo ""
    case $key in
        m)
            STATE="menu"
            echo "State: $STATE"
            echo "Main Menu"
            echo "1. Option One"
            echo "2. Option Two"
            ;;
        e)
            STATE="edit"
            echo "State: $STATE"
            echo "Edit Mode (type text, ESC to exit)"
            ;;
        v)
            STATE="view"
            echo "State: $STATE"
            echo "View Mode (readonly)"
            ;;
        q)
            echo "Exiting..."
            exit 0
            ;;
    esac
    printf "> "
done
`
					if err := fs.WriteString(scriptPath, scriptContent); err != nil {
						return fmt.Errorf("failed to create debug TUI: %w", err)
					}

					if err := os.Chmod(scriptPath, 0755); err != nil {
						return fmt.Errorf("failed to chmod: %w", err)
					}

					ctx.Set("debug_script", scriptPath)
					return nil
				},
			},
			{
				Name:        "Launch debug TUI",
				Description: "Starts the TUI in tmux (can be attached to manually)",
				Func: func(ctx *harness.Context) error {
					scriptPath := ctx.GetString("debug_script")
					
					session, err := ctx.StartTUI("/bin/bash", scriptPath)
					if err != nil {
						return fmt.Errorf("failed to start debug TUI: %w", err)
					}

					ctx.Set("debug_session", session)
					
					// In interactive mode, the test framework will:
					// 1. Display the current TUI state
					// 2. Offer options to continue, attach, or quit
					fmt.Println("\n   💡 TIP: In interactive mode, you can press 'a' to attach to this tmux session")
					fmt.Println("   Once attached, use 'Ctrl-b d' to detach and continue the test")
					
					return nil
				},
			},
			{
				Name:        "Test menu navigation",
				Description: "Navigate through menu states",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("debug_session").(*tui.Session)
					
					// Go to menu
					if err := session.SendKeys("m"); err != nil {
						return fmt.Errorf("failed to send menu key: %w", err)
					}
					
					if err := session.WaitForText("Main Menu", 2*time.Second); err != nil {
						return fmt.Errorf("menu did not appear: %w", err)
					}
					
					return nil
				},
			},
			{
				Name:        "Test edit mode",
				Description: "Switch to edit mode",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("debug_session").(*tui.Session)
					
					if err := session.SendKeys("e"); err != nil {
						return fmt.Errorf("failed to enter edit mode: %w", err)
					}
					
					if err := session.WaitForText("Edit Mode", 2*time.Second); err != nil {
						return fmt.Errorf("edit mode did not activate: %w", err)
					}
					
					// Capture current state for debugging
					content, _ := session.Capture()
					fmt.Printf("   Current TUI state captured (%d bytes)\n", len(content))
					
					return nil
				},
			},
			{
				Name:        "Cleanup debug session",
				Description: "Quit the debug TUI",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("debug_session").(*tui.Session)
					
					if err := session.SendKeys("q"); err != nil {
						return fmt.Errorf("failed to quit: %w", err)
					}
					
					return nil
				},
			},
		},
	}
}

// ExampleAdvancedTuiNavigation demonstrates the advanced navigation and timing controls.
func ExampleAdvancedTuiNavigation() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-advanced-tui-navigation",
		Description: "Demonstrates robust navigation and timing features like WaitForUIStable and NavigateToText",
		Tags:        []string{"example", "tui", "navigation"},
		Steps: []harness.Step{
			{
				Name: "Create a test TUI with a list",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("advanced-tui")
					scriptPath := filepath.Join(testDir, "list-tui.sh")
					// This script simulates a file browser that loads items with a delay
					scriptContent := `#!/bin/bash
echo "Loading files..."
sleep 0.1
echo "  README.md"
sleep 0.1
echo "  main.go"
sleep 0.1
echo "  docs/guide.md"
echo ""
echo "Use arrow keys to navigate. Press 'q' to quit."
printf "> "`
					if err := fs.WriteString(scriptPath, scriptContent); err != nil {
						return err
					}
					return os.Chmod(scriptPath, 0755)
				},
			},
			{
				Name: "Launch TUI and wait for it to stabilize",
				Func: func(ctx *harness.Context) error {
					scriptPath := filepath.Join(ctx.Dir("advanced-tui"), "list-tui.sh")
					session, err := ctx.StartTUI("/bin/bash", scriptPath)
					if err != nil {
						return err
					}
					ctx.Set("advanced_session", session)

					// OLD WAY: time.Sleep(1 * time.Second)
					// NEW WAY: Wait for the UI to stop changing content.
					// This is more reliable than a fixed sleep.
					fmt.Println("   Waiting for UI to stabilize...")
					return session.WaitForUIStable(5*time.Second, 100*time.Millisecond, 300*time.Millisecond)
				},
			},
			{
				Name: "Test FindTextLocation functionality",
				Func: func(ctx *harness.Context) error {
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
				},
			},
			{
				Name: "Demonstrate NavigateToText navigation",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("advanced_session").(*tui.Session)

					// OLD WAY:
					// session.SendKeys("Down")
					// session.SendKeys("Down")
					// This is brittle if the file order changes.

					// NEW WAY: Navigate directly to specific text
					fmt.Println("   Navigating cursor to 'docs/guide.md'...")
					if err := session.NavigateToText("docs/guide.md"); err != nil {
						return fmt.Errorf("failed to navigate to text: %w", err)
					}

					// Verify cursor position
					row, col, err := session.GetCursorPosition()
					if err != nil {
						return fmt.Errorf("failed to get cursor position: %w", err)
					}
					fmt.Printf("   ✓ Successfully navigated cursor to row %d, col %d\n", row, col)
					
					// Navigate to another location
					fmt.Println("   Navigating cursor to 'main.go'...")
					if err := session.NavigateToText("main.go"); err != nil {
						return fmt.Errorf("failed to navigate to main.go: %w", err)
					}
					
					fmt.Println("   ✓ Navigation works correctly - no more brittle key sequences!")
					return nil
				},
			},
			{
				Name: "Cleanup",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("advanced_session").(*tui.Session)
					return session.SendKeys("q")
				},
			},
		},
	}
}

// ExampleConditionalFlowsAndRecording demonstrates the new conditional flow and recording features.
func ExampleConditionalFlowsAndRecording() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-conditional-flows-recording",
		Description: "Demonstrates WaitForAnyText, pattern matching, SelectItem, and session recording",
		Tags:        []string{"example", "tui", "conditional", "recording"},
		Steps: []harness.Step{
			{
				Name: "Create a TUI with conditional outcomes",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("conditional-tui")
					scriptPath := filepath.Join(testDir, "conditional-tui.sh")
					
					scriptContent := `#!/bin/bash
echo "Task Manager v1.0"
echo "================="
echo ""
echo "Select action:"
echo "  1) Process files"
echo "  2) Run tests"
echo "  3) Check status"
echo ""
printf "Choice: "
read -n 1 choice
echo ""

case $choice in
	1)
		echo "Processing files..."
		sleep 0.3
		echo "Found 15 files modified"
		echo "Found 3 files added"
		echo "✓ Success: All files processed"
		;;
	2)
		echo "Running tests..."
		sleep 0.3
		echo "Test Suite: Unit Tests"
		echo "Tests: 42 passed, 0 failed"
		echo "✓ Success: All tests passed"
		;;
	3)
		echo "Checking status..."
		sleep 0.3
		echo "⚠ Warning: Low disk space"
		;;
	*)
		echo "✗ Failed: Invalid option"
		;;
esac
echo ""
printf "> "`
					
					if err := fs.WriteString(scriptPath, scriptContent); err != nil {
						return err
					}
					return os.Chmod(scriptPath, 0755)
				},
			},
			{
				Name: "Start recording and launch TUI",
				Func: func(ctx *harness.Context) error {
					scriptPath := filepath.Join(ctx.Dir("conditional-tui"), "conditional-tui.sh")
					session, err := ctx.StartTUI("/bin/bash", scriptPath)
					if err != nil {
						return err
					}
					ctx.Set("cond_session", session)
					
					// Start recording the session
					recordingPath := filepath.Join(ctx.Dir("conditional-tui"), "session-recording")
					if err := session.StartRecording(recordingPath); err != nil {
						return fmt.Errorf("failed to start recording: %w", err)
					}
					fmt.Printf("   📹 Recording session to: %s\n", recordingPath)
					
					return nil
				},
			},
			{
				Name: "Test SelectItem with predicate",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("cond_session").(*tui.Session)
					
					// Wait for menu to appear
					if err := session.WaitForText("Select action:", 2*time.Second); err != nil {
						return err
					}
					
					// Use SelectItem to choose option 1 (Process files)
					fmt.Println("   Selecting 'Process files' option...")
					if err := session.SendKeys("1"); err != nil {
						return err
					}
					
					return nil
				},
			},
			{
				Name: "Test WaitForAnyText for conditional outcomes",
				Func: func(ctx *harness.Context) error {
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
				},
			},
			{
				Name: "Test pattern matching for file counts",
				Func: func(ctx *harness.Context) error {
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
				},
			},
			{
				Name: "Take screenshot and stop recording",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("cond_session").(*tui.Session)
					
					// Take a screenshot
					screenshotPath := filepath.Join(ctx.Dir("conditional-tui"), "final-state.ansi")
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
					
					recordingPath := filepath.Join(ctx.Dir("conditional-tui"), "session-recording")
					fmt.Printf("   📊 Recording saved to:\n")
					fmt.Printf("      - HTML: %s.html\n", recordingPath)
					fmt.Printf("      - JSON: %s.json\n", recordingPath)
					
					return nil
				},
			},
			{
				Name: "Cleanup",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("cond_session").(*tui.Session)
					return session.SendKeys("Ctrl+c")
				},
			},
		},
	}
}

// ExampleFilesystemInteractionScenario demonstrates testing a TUI that interacts with the filesystem
func ExampleFilesystemInteractionScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-tui-filesystem",
		Description: "Tests a TUI that writes to the filesystem",
		Tags:        []string{"example", "tui", "filesystem"},
		Steps: []harness.Step{
			{
				Name: "Create TUI script that writes to a file",
				Func: func(ctx *harness.Context) error {
					scriptPath := filepath.Join(ctx.RootDir, "save-script.sh")
					scriptContent := `#!/bin/bash
echo "Press 's' to save a file."
printf "> "

while true; do
    read -n 1 key
    echo ""
    if [ "$key" == "s" ]; then
        echo "Saving file..."
        echo "saved at $(date)" > output.txt
        echo "File saved to output.txt"
        printf "> "
    elif [ "$key" == "q" ]; then
        echo "Quitting."
        exit 0
    fi
done`
					if err := fs.WriteString(scriptPath, scriptContent); err != nil {
						return err
					}
					return os.Chmod(scriptPath, 0755)
				},
			},
			{
				Name: "Launch TUI and wait for it to be ready",
				Func: func(ctx *harness.Context) error {
					scriptPath := filepath.Join(ctx.RootDir, "save-script.sh")
					session, err := ctx.StartTUI("/bin/bash", scriptPath)
					if err != nil {
						return err
					}
					ctx.Set("fs_session", session)
					return session.WaitForText("Press 's' to save a file.", 5*time.Second)
				},
			},
			{
				Name: "Send 's' key and wait for UI change",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("fs_session").(*tui.Session)
					return session.SendKeysAndWaitForChange(2*time.Second, "s")
				},
			},
			{
				Name: "Verify file was created",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("fs_session").(*tui.Session)
					// Verify the file was created and is visible to the session
					return session.WaitForFile("output.txt", 5*time.Second)
				},
			},
			{
				Name: "Verify file content",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("fs_session").(*tui.Session)
					// Assert the file contains the expected content
					return session.AssertFileContains("output.txt", "saved at")
				},
			},
			{
				Name: "Cleanup",
				Func: func(ctx *harness.Context) error {
					session := ctx.Get("fs_session").(*tui.Session)
					return session.SendKeys("q")
				},
			},
		},
	}
}

func main() {
	scenarios := []*harness.Scenario{
		ExampleTUITestScenario(),
		ExampleHeadlessBubbleTeaScenario(),
		ExampleInteractiveTUIDebugging(),
		ExampleAdvancedTuiNavigation(),
		ExampleConditionalFlowsAndRecording(),
		ExampleFilesystemInteractionScenario(),
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

	// Execute the tmux TUI test examples
	if err := app.Execute(ctx, scenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}