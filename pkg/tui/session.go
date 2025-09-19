package tui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mattsolo1/grove-core/pkg/tmux"
	"github.com/mattsolo1/grove-tend/pkg/assert"
	"github.com/mattsolo1/grove-tend/pkg/wait"
)

// Session represents an active TUI application running in a tmux session.
// It provides a high-level API for interaction and assertion.
type Session struct {
	sessionName string
	client      *tmux.Client
}

// NewSession creates a new TUI session handle. It is intended for internal use by the harness.
func NewSession(sessionName string, client *tmux.Client) *Session {
	return &Session{
		sessionName: sessionName,
		client:      client,
	}
}

// SendKeys sends a sequence of keystrokes to the TUI.
// Special keys can be sent (e.g., "Enter", "Esc", "Ctrl+c").
func (s *Session) SendKeys(keys ...string) error {
	// Tmux's send-keys sends all arguments as a single sequence.
	return s.client.SendKeys(context.Background(), s.sessionName, keys...)
}

// captureConfig holds configuration for capture options
type captureConfig struct {
	raw bool
}

// CaptureOption configures capture behavior
type CaptureOption func(*captureConfig)

// WithRawOutput is a capture option to get the raw pane content with ANSI codes.
func WithRawOutput() CaptureOption {
	return func(c *captureConfig) {
		c.raw = true
	}
}

// WithCleanedOutput is a capture option to get pane content with ANSI codes stripped.
func WithCleanedOutput() CaptureOption {
	return func(c *captureConfig) {
		c.raw = false
	}
}

// Capture returns the visible text content of the TUI pane.
// By default, it returns raw content with ANSI escape codes.
func (s *Session) Capture(opts ...CaptureOption) (string, error) {
	cfg := &captureConfig{raw: true} // Default to raw
	for _, opt := range opts {
		opt(cfg)
	}

	content, err := s.client.CapturePane(context.Background(), s.sessionName)
	if err != nil {
		return "", err
	}

	if cfg.raw {
		return content, nil
	}

	// Simple ANSI stripping logic
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(content, ""), nil
}

// WaitForText polls the TUI's content until the specified text appears or the timeout is reached.
func (s *Session) WaitForText(text string, timeout time.Duration) error {
	opts := wait.DefaultOptions()
	opts.Timeout = timeout

	return wait.ForWithMessage(func() (bool, string, error) {
		content, err := s.Capture()
		if err != nil {
			return false, "failed to capture pane", err
		}
		if strings.Contains(content, text) {
			return true, fmt.Sprintf("found text '%s'", text), nil
		}
		return false, fmt.Sprintf("text '%s' not found", text), nil
	}, opts)
}

// AssertContains immediately checks if the TUI's current content contains the specified text.
func (s *Session) AssertContains(text string) error {
	content, err := s.Capture()
	if err != nil {
		return fmt.Errorf("failed to capture pane for assertion: %w", err)
	}
	return assert.Contains(content, text)
}

// AssertNotContains immediately checks if the TUI's current content does not contain the specified text.
func (s *Session) AssertNotContains(text string) error {
	content, err := s.Capture()
	if err != nil {
		return fmt.Errorf("failed to capture pane for assertion: %w", err)
	}
	return assert.NotContains(content, text)
}

// Close terminates the tmux session associated with the TUI.
// This is typically called automatically by the test harness on cleanup.
func (s *Session) Close() error {
	return s.client.KillSession(context.Background(), s.sessionName)
}