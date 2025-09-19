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

// WaitForUIStable polls the TUI screen until its content remains unchanged for a specified duration,
// or until a timeout is reached. This is useful for waiting for animations or asynchronous updates to complete.
func (s *Session) WaitForUIStable(timeout time.Duration, pollInterval time.Duration, stableDuration time.Duration) error {
	var lastContent string
	var stableSince time.Time
	var initialized bool

	opts := wait.Options{
		Timeout:      timeout,
		PollInterval: pollInterval,
		Immediate:    true,
	}

	return wait.ForWithMessage(func() (bool, string, error) {
		currentContent, err := s.Capture(WithCleanedOutput())
		if err != nil {
			return false, "failed to capture screen", err
		}

		if !initialized {
			lastContent = currentContent
			stableSince = time.Now()
			initialized = true
			return false, "initializing stability check", nil
		}

		if currentContent != lastContent {
			lastContent = currentContent
			stableSince = time.Now()
			return false, "screen content changed", nil
		}

		if time.Since(stableSince) >= stableDuration {
			return true, "screen has been stable", nil
		}

		return false, fmt.Sprintf("screen stable for %v, waiting for %v", time.Since(stableSince).Round(time.Millisecond), stableDuration), nil
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

// GetCursorPosition returns the 1-based (row, col) of the cursor.
func (s *Session) GetCursorPosition() (row, col int, err error) {
	return s.client.GetCursorPosition(context.Background(), s.sessionName)
}

// FindTextLocation searches the screen for the given text and returns its 1-based (row, col) position if found.
func (s *Session) FindTextLocation(text string) (row, col int, found bool, err error) {
	content, err := s.Capture(WithCleanedOutput())
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to capture screen for text search: %w", err)
	}

	lines := strings.Split(content, "\n")
	for r, line := range lines {
		if c := strings.Index(line, text); c != -1 {
			// Return 1-based coordinates
			return r + 1, c + 1, true, nil
		}
	}

	return 0, 0, false, nil
}

// NavigateToText moves the cursor from its current position to the location of the specified text.
// It calculates the required up/down/left/right key presses.
func (s *Session) NavigateToText(text string) error {
	targetRow, targetCol, found, err := s.FindTextLocation(text)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("text '%s' not found on screen", text)
	}

	currentRow, currentCol, err := s.GetCursorPosition()
	if err != nil {
		return fmt.Errorf("failed to get current cursor position: %w", err)
	}

	// Calculate vertical movement
	rowDiff := targetRow - currentRow
	if rowDiff > 0 {
		for i := 0; i < rowDiff; i++ {
			if err := s.SendKeys("Down"); err != nil {
				return err
			}
		}
	} else if rowDiff < 0 {
		for i := 0; i < -rowDiff; i++ {
			if err := s.SendKeys("Up"); err != nil {
				return err
			}
		}
	}

	// Calculate horizontal movement
	colDiff := targetCol - currentCol
	if colDiff > 0 {
		for i := 0; i < colDiff; i++ {
			if err := s.SendKeys("Right"); err != nil {
				return err
			}
		}
	} else if colDiff < 0 {
		for i := 0; i < -colDiff; i++ {
			if err := s.SendKeys("Left"); err != nil {
				return err
			}
		}
	}

	return nil
}

// Close terminates the tmux session associated with the TUI.
// This is typically called automatically by the test harness on cleanup.
func (s *Session) Close() error {
	return s.client.KillSession(context.Background(), s.sessionName)
}