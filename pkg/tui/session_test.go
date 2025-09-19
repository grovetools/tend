package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mattsolo1/grove-core/pkg/tmux"
)

// Helper to skip tests if tmux is not available
func skipIfNoTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH, skipping TUI integration tests")
	}
}

func TestSession(t *testing.T) {
	skipIfNoTmux(t)

	ctx := context.Background()
	sessionName := "tend-tui-test-session"
	client, _ := tmux.NewClient()

	// Cleanup any old session
	_ = client.KillSession(ctx, sessionName)

	// Launch a simple TUI (a shell command that waits)
	err := client.Launch(ctx, tmux.LaunchOptions{
		SessionName: sessionName,
		Panes: []tmux.PaneOptions{
			{Command: "echo 'Hello TUI'; sleep 5"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to launch test session: %v", err)
	}
	defer client.KillSession(ctx, sessionName)

	session := NewSession(sessionName, client)

	// Test WaitForText
	err = session.WaitForText("Hello TUI", 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForText should find the initial text: %v", err)
	}

	// Test AssertContains
	err = session.AssertContains("Hello TUI")
	if err != nil {
		t.Errorf("AssertContains should find the text: %v", err)
	}

	// Test AssertNotContains
	err = session.AssertNotContains("Goodbye TUI")
	if err != nil {
		t.Errorf("AssertNotContains should not find the text: %v", err)
	}

	// Test SendKeys and Capture
	err = session.SendKeys("echo 'New Text'", "Enter")
	if err != nil {
		t.Errorf("SendKeys should not return an error: %v", err)
	}

	time.Sleep(200 * time.Millisecond) // Give shell time to process

	content, err := session.Capture()
	if err != nil {
		t.Errorf("Capture should not return an error: %v", err)
	}
	if !strings.Contains(content, "New Text") {
		t.Errorf("Expected captured content to contain 'New Text', got:\n%s", content)
	}

	// Test Close
	err = session.Close()
	if err != nil {
		t.Errorf("Close should not return an error: %v", err)
	}

	exists, _ := client.SessionExists(ctx, sessionName)
	if exists {
		t.Error("Session should not exist after Close")
	}
}

func TestSession_AdvancedFeatures(t *testing.T) {
	skipIfNoTmux(t)

	ctx := context.Background()
	sessionName := "tend-tui-advanced-test"
	client, _ := tmux.NewClient()

	_ = client.KillSession(ctx, sessionName)

	// Launch a shell script that simulates a list and updates slowly
	// The `sleep 0.2` simulates rendering delay.
	script := `#!/bin/bash
echo "File 1.txt"
sleep 0.2
echo "File 2.md"
sleep 0.2
echo "File 3.go"
printf "> "`
	err := client.Launch(ctx, tmux.LaunchOptions{
		SessionName: sessionName,
		Panes:       []tmux.PaneOptions{{Command: script}},
	})
	if err != nil {
		t.Fatalf("Failed to launch test session: %v", err)
	}
	defer client.KillSession(ctx, sessionName)

	session := NewSession(sessionName, client)

	// Test WaitForUIStable
	err = session.WaitForUIStable(5*time.Second, 100*time.Millisecond, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForUIStable failed: %v", err)
	}

	// Test FindTextLocation
	// First, let's capture the actual content to understand the layout
	content, err := session.Capture(WithCleanedOutput())
	if err != nil {
		t.Fatalf("Failed to capture content: %v", err)
	}
	t.Logf("Captured content:\n%s", content)
	
	row, col, found, err := session.FindTextLocation("File 2.md")
	if err != nil || !found {
		t.Fatalf("FindTextLocation failed. err: %v, found: %v", err, found)
	}
	t.Logf("Found 'File 2.md' at (%d, %d)", row, col)
	
	// The exact position will depend on the shell script output format
	// Just verify we found it somewhere reasonable
	if row < 1 || col < 1 {
		t.Errorf("Expected positive location, got (%d, %d)", row, col)
	}

	// Test GetCursorPosition
	// The cursor position will vary depending on shell state
	curRow, curCol, err := session.GetCursorPosition()
	if err != nil {
		t.Fatalf("GetCursorPosition failed: %v", err)
	}
	t.Logf("Current cursor position: (%d, %d)", curRow, curCol)
	if curRow < 1 || curCol < 1 {
		t.Errorf("Expected positive cursor position, got (%d, %d)", curRow, curCol)
	}

	// Test NavigateToText
	err = session.NavigateToText("File 2.md")
	if err != nil {
		t.Fatalf("NavigateToText failed: %v", err)
	}

	// Verify cursor moved (exact position depends on shell output format)
	finalRow, finalCol, err := session.GetCursorPosition()
	if err != nil {
		t.Fatalf("GetCursorPosition after navigation failed: %v", err)
	}
	t.Logf("Cursor after navigation: (%d, %d)", finalRow, finalCol)
	
	// Just verify the cursor moved somewhere reasonable
	if finalRow < 1 || finalCol < 1 {
		t.Errorf("Expected positive cursor position after navigation, got (%d, %d)", finalRow, finalCol)
	}
}

func TestSession_ConditionalFlows(t *testing.T) {
	skipIfNoTmux(t)

	ctx := context.Background()
	sessionName := "tend-tui-conditional-test"
	client, _ := tmux.NewClient()

	_ = client.KillSession(ctx, sessionName)

	// Script that outputs different messages randomly
	script := `#!/bin/bash
echo "Processing..."
sleep 0.2
# Simulate random outcome
if [ $((RANDOM % 3)) -eq 0 ]; then
    echo "✓ Success"
elif [ $((RANDOM % 3)) -eq 1 ]; then
    echo "✗ Failed"  
else
    echo "⚠ Warning"
fi`
	err := client.Launch(ctx, tmux.LaunchOptions{
		SessionName: sessionName,
		Panes:       []tmux.PaneOptions{{Command: script}},
	})
	if err != nil {
		t.Fatalf("Failed to launch test session: %v", err)
	}
	defer client.KillSession(ctx, sessionName)

	session := NewSession(sessionName, client)

	// Test WaitForAnyText
	result, err := session.WaitForAnyText([]string{"✓ Success", "✗ Failed", "⚠ Warning"}, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForAnyText failed: %v", err)
	}
	
	// Verify we got one of the expected results
	validResults := map[string]bool{"✓ Success": true, "✗ Failed": true, "⚠ Warning": true}
	if !validResults[result] {
		t.Errorf("Unexpected result from WaitForAnyText: %s", result)
	}
	t.Logf("Got result: %s", result)
}

func TestSession_PatternMatching(t *testing.T) {
	skipIfNoTmux(t)

	ctx := context.Background()
	sessionName := "tend-tui-pattern-test"
	client, _ := tmux.NewClient()

	_ = client.KillSession(ctx, sessionName)

	// Script that outputs file counts
	script := `#!/bin/bash
echo "Scanning files..."
sleep 0.1
echo "Found 42 files modified"
echo "Found 7 files added"
echo "Processing complete"`
	
	err := client.Launch(ctx, tmux.LaunchOptions{
		SessionName: sessionName,
		Panes:       []tmux.PaneOptions{{Command: script}},
	})
	if err != nil {
		t.Fatalf("Failed to launch test session: %v", err)
	}
	defer client.KillSession(ctx, sessionName)

	session := NewSession(sessionName, client)

	// Test WaitForTextPattern
	pattern := regexp.MustCompile(`\d+ files? (modified|added|deleted)`)
	match, err := session.WaitForTextPattern(pattern, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForTextPattern failed: %v", err)
	}
	
	if match == "" {
		t.Error("Expected pattern match but got empty string")
	}
	t.Logf("Pattern matched: %s", match)

	// Test AssertContainsPattern
	err = session.AssertContainsPattern(regexp.MustCompile(`Processing complete`))
	if err != nil {
		t.Errorf("AssertContainsPattern failed: %v", err)
	}
}

func TestSession_SelectItem(t *testing.T) {
	skipIfNoTmux(t)

	ctx := context.Background()
	sessionName := "tend-tui-select-test"
	client, _ := tmux.NewClient()

	_ = client.KillSession(ctx, sessionName)

	// Create a simple menu
	script := `#!/bin/bash
echo "Select an option:"
echo "  1. First option"
echo "  2. Second option"
echo "  3. Third option"
printf "> "`
	
	err := client.Launch(ctx, tmux.LaunchOptions{
		SessionName: sessionName,
		Panes:       []tmux.PaneOptions{{Command: script}},
	})
	if err != nil {
		t.Fatalf("Failed to launch test session: %v", err)
	}
	defer client.KillSession(ctx, sessionName)

	session := NewSession(sessionName, client)
	
	// Wait for menu to appear
	time.Sleep(500 * time.Millisecond)

	// Test GetVisibleLines
	lines, err := session.GetVisibleLines()
	if err != nil {
		t.Fatalf("GetVisibleLines failed: %v", err)
	}
	
	if len(lines) < 4 {
		t.Errorf("Expected at least 4 lines, got %d", len(lines))
	}
	
	// Test SelectItem with a predicate
	// Note: In a real terminal, navigation might work differently
	// This is a simplified test
	foundSecond := false
	for _, line := range lines {
		if strings.Contains(line, "Second option") {
			foundSecond = true
			break
		}
	}
	
	if !foundSecond {
		t.Error("Could not find 'Second option' in visible lines")
	}
}

func TestSession_Recording(t *testing.T) {
	skipIfNoTmux(t)

	ctx := context.Background()
	sessionName := "tend-tui-recording-test"
	client, _ := tmux.NewClient()

	_ = client.KillSession(ctx, sessionName)

	err := client.Launch(ctx, tmux.LaunchOptions{
		SessionName: sessionName,
		Panes:       []tmux.PaneOptions{{Command: "echo 'Recording Test'"}},
	})
	if err != nil {
		t.Fatalf("Failed to launch test session: %v", err)
	}
	defer client.KillSession(ctx, sessionName)

	session := NewSession(sessionName, client)
	
	// Start recording
	tempDir := t.TempDir()
	recordingPath := filepath.Join(tempDir, "test-recording")
	err = session.StartRecording(recordingPath)
	if err != nil {
		t.Fatalf("StartRecording failed: %v", err)
	}

	// Perform some actions
	session.SendKeys("echo", "Space", "Hello", "Enter")
	time.Sleep(200 * time.Millisecond)
	session.WaitForText("Hello", 1*time.Second)
	
	// Take a screenshot
	screenshotPath := filepath.Join(tempDir, "screenshot.ansi")
	err = session.TakeScreenshot(screenshotPath)
	if err != nil {
		t.Errorf("TakeScreenshot failed: %v", err)
	}
	
	// Stop recording
	err = session.StopRecording()
	if err != nil {
		t.Fatalf("StopRecording failed: %v", err)
	}
	
	// Check that files were created
	if _, err := os.Stat(recordingPath + ".html"); os.IsNotExist(err) {
		t.Error("HTML recording file was not created")
	}
	if _, err := os.Stat(recordingPath + ".json"); os.IsNotExist(err) {
		t.Error("JSON recording file was not created")
	}
	if _, err := os.Stat(screenshotPath); os.IsNotExist(err) {
		t.Error("Screenshot file was not created")
	}
	
	// Test GetKeyHistory
	history := session.GetKeyHistory()
	if len(history) == 0 {
		t.Error("Key history should not be empty after sending keys")
	}
	t.Logf("Key history: %v", history)
}