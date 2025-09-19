package tui

import (
	"context"
	"os/exec"
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