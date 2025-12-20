package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mattsolo1/grove-core/pkg/tmux"
	"github.com/mattsolo1/grove-tend/pkg/assert"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/wait"
)

// Session represents an active TUI application running in a tmux session.
// It provides a high-level API for interaction and assertion.
type Session struct {
	sessionName string
	client      *tmux.Client
	recording   *SessionRecording // For recording and debugging
	rootDir     string
}

// NewSession creates a new TUI session handle. It is intended for internal use by the harness.
func NewSession(sessionName string, client *tmux.Client, rootDir string) *Session {
	return &Session{
		sessionName: sessionName,
		client:      client,
		rootDir:     rootDir,
	}
}

// Type sends keys to the TUI and waits for the screen to stabilize.
// This is the recommended method for most interactions, as it combines
// SendKeys + WaitStable which is the pattern used in 90% of tests.
//
// Special handling for vim-style chord commands:
// When multiple identical single-character keys are passed (like "g", "g"),
// they are sent individually with stabilization between each key.
// This ensures vim chord commands like "gg" are properly recognized.
//
// Example:
//   session.Type("j")           // Navigate down and wait
//   session.Type("g", "g")      // Go to top (sends separately for chord recognition)
//   session.Type("/", "search") // Open search and type
func (s *Session) Type(keys ...string) error {
	// Detect vim chord commands: exactly 2 single-character keys
	// Examples: "g"+"g" (go to top), "z"+"M" (close all), "z"+"R" (open all)
	if len(keys) == 2 {
		allSingleChar := true
		for _, key := range keys {
			if len(key) != 1 {
				allSingleChar = false
				break
			}
		}

		// If both keys are single characters, treat as vim chord
		// Send them individually with stabilization between them
		if allSingleChar {
			if err := s.SendKeys(keys[0]); err != nil {
				return err
			}
			if err := s.WaitStable(); err != nil {
				return err
			}
			if err := s.SendKeys(keys[1]); err != nil {
				return err
			}
			return s.WaitStable()
		}
	}

	// Default behavior: send all keys together and wait
	if err := s.SendKeys(keys...); err != nil {
		return err
	}
	return s.WaitStable()
}

// SendKeys sends a sequence of keystrokes to the TUI without waiting.
// Special keys can be sent (e.g., "Enter", "Esc", "Ctrl+c").
//
// NOTE: In most cases, you should use Type() instead, which automatically
// waits for the screen to stabilize after sending keys.
func (s *Session) SendKeys(keys ...string) error {
	// Tmux's send-keys sends all arguments as a single sequence.
	err := s.client.SendKeys(context.Background(), s.sessionName, keys...)

	// Record the event if recording is enabled
	s.recordEvent("key", map[string]interface{}{"keys": keys}, "", err)

	// Auto-capture screen after sending keys (async)
	if s.recording != nil && s.recording.enabled {
		go func() {
			time.Sleep(100 * time.Millisecond) // Small delay for UI update
			s.captureForRecording()
		}()
	}

	return err
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

	err := wait.ForWithMessage(func() (bool, string, error) {
		content, err := s.Capture()
		if err != nil {
			return false, "failed to capture pane", err
		}
		if strings.Contains(content, text) {
			return true, fmt.Sprintf("found text '%s'", text), nil
		}
		return false, fmt.Sprintf("text '%s' not found", text), nil
	}, opts)
	
	// Record the wait event
	data := map[string]interface{}{"text": text, "timeout": timeout.String()}
	result := ""
	if err == nil {
		result = fmt.Sprintf("found text '%s'", text)
	}
	s.recordEvent("wait", data, result, err)
	
	return err
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

// WaitStable waits for the UI to stabilize using sensible defaults.
// Equivalent to WaitForUIStable(2*time.Second, 100*time.Millisecond, 200*time.Millisecond).
func (s *Session) WaitStable() error {
	return s.WaitForUIStable(2*time.Second, 100*time.Millisecond, 200*time.Millisecond)
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

// AssertLine iterates through each visible line of the TUI and passes if the predicate
// function returns true for any line. This is a flexible way to assert complex states
// like focus or selection without relying on specific ANSI codes.
func (s *Session) AssertLine(predicate func(line string) bool, message string) error {
	content, err := s.Capture(WithCleanedOutput())
	if err != nil {
		return fmt.Errorf("failed to capture screen for AssertLine: %w", err)
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if predicate(line) {
			return nil // Predicate matched, assertion passes.
		}
	}

	// If no line matched, the assertion fails.
	return fmt.Errorf("AssertLine failed: %s. Screen content:\n%s", message, content)
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
		err := fmt.Errorf("text '%s' not found on screen", text)
		s.recordEvent("navigate", map[string]interface{}{"text": text}, "", err)
		return err
	}

	currentRow, currentCol, err := s.GetCursorPosition()
	if err != nil {
		err := fmt.Errorf("failed to get current cursor position: %w", err)
		s.recordEvent("navigate", map[string]interface{}{"text": text}, "", err)
		return err
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

	// Record successful navigation
	s.recordEvent("navigate", map[string]interface{}{
		"text": text,
		"from": fmt.Sprintf("(%d,%d)", currentRow, currentCol),
		"to":   fmt.Sprintf("(%d,%d)", targetRow, targetCol),
	}, fmt.Sprintf("navigated to '%s'", text), nil)

	return nil
}

// WaitForAnyText waits for any of the specified texts to appear on screen.
// Returns the first matching text found, or an error if timeout is reached.
func (s *Session) WaitForAnyText(texts []string, timeout time.Duration) (string, error) {
	opts := wait.Options{
		Timeout:      timeout,
		PollInterval: 100 * time.Millisecond,
		Immediate:    true,
	}

	var foundText string
	err := wait.ForWithMessage(func() (bool, string, error) {
		content, err := s.Capture(WithCleanedOutput())
		if err != nil {
			return false, "failed to capture", err
		}

		for _, text := range texts {
			if strings.Contains(content, text) {
				foundText = text
				return true, fmt.Sprintf("found '%s'", text), nil
			}
		}

		return false, fmt.Sprintf("waiting for any of: %v", texts), nil
	}, opts)

	return foundText, err
}

// WaitForTextPattern waits for text matching the regex pattern to appear.
// Returns the matched text, or an error if timeout is reached.
func (s *Session) WaitForTextPattern(pattern *regexp.Regexp, timeout time.Duration) (string, error) {
	opts := wait.Options{
		Timeout:      timeout,
		PollInterval: 100 * time.Millisecond,
		Immediate:    true,
	}

	var matched string
	err := wait.ForWithMessage(func() (bool, string, error) {
		content, err := s.Capture(WithCleanedOutput())
		if err != nil {
			return false, "failed to capture", err
		}

		if match := pattern.FindString(content); match != "" {
			matched = match
			return true, fmt.Sprintf("found pattern match: %s", match), nil
		}

		return false, fmt.Sprintf("waiting for pattern: %s", pattern.String()), nil
	}, opts)

	return matched, err
}

// AssertContainsPattern checks if the current screen content matches the regex pattern.
func (s *Session) AssertContainsPattern(pattern *regexp.Regexp) error {
	content, err := s.Capture(WithCleanedOutput())
	if err != nil {
		return fmt.Errorf("failed to capture for assertion: %w", err)
	}

	if !pattern.MatchString(content) {
		return fmt.Errorf("pattern '%s' not found in content", pattern.String())
	}

	return nil
}

// GetVisibleLines returns the current screen content as a slice of lines.
func (s *Session) GetVisibleLines() ([]string, error) {
	content, err := s.Capture(WithCleanedOutput())
	if err != nil {
		return nil, err
	}
	return strings.Split(content, "\n"), nil
}

// SelectItem navigates to and selects an item matching the predicate.
// It searches visible lines and moves the cursor to the first match, then presses Enter.
func (s *Session) SelectItem(predicate func(line string) bool) error {
	return s.SelectItemWithKey(predicate, "Enter")
}

// SelectItemWithKey navigates to an item matching the predicate and presses the specified key.
func (s *Session) SelectItemWithKey(predicate func(line string) bool, key string) error {
	lines, err := s.GetVisibleLines()
	if err != nil {
		return fmt.Errorf("failed to get visible lines: %w", err)
	}

	// Find the first matching line
	for i, line := range lines {
		if predicate(line) {
			// Get current cursor position
			currentRow, _, err := s.GetCursorPosition()
			if err != nil {
				return fmt.Errorf("failed to get cursor position: %w", err)
			}

			// Navigate to the target line (1-based indexing)
			targetRow := i + 1
			rowDiff := targetRow - currentRow

			// Move cursor vertically
			if rowDiff > 0 {
				for j := 0; j < rowDiff; j++ {
					if err := s.SendKeys("Down"); err != nil {
						return err
					}
				}
			} else if rowDiff < 0 {
				for j := 0; j < -rowDiff; j++ {
					if err := s.SendKeys("Up"); err != nil {
						return err
					}
				}
			}

			// Press the selection key
			return s.SendKeys(key)
		}
	}

	return fmt.Errorf("no item matching predicate found")
}

// WaitForFile waits for a file to exist within the scenario's temporary directory.
func (s *Session) WaitForFile(relPath string, timeout time.Duration) error {
	if s.rootDir == "" {
		return fmt.Errorf("session is not aware of test root directory")
	}
	fullPath := filepath.Join(s.rootDir, relPath)

	opts := wait.DefaultOptions()
	opts.Timeout = timeout

	return wait.For(func() (bool, error) {
		_, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	}, opts)
}

// AssertFileExists asserts that a file exists within the scenario's temporary directory.
func (s *Session) AssertFileExists(relPath string) error {
	if s.rootDir == "" {
		return fmt.Errorf("session is not aware of test root directory")
	}
	fullPath := filepath.Join(s.rootDir, relPath)

	if !fs.Exists(fullPath) {
		return fmt.Errorf("file does not exist: %s", fullPath)
	}
	return nil
}

// AssertFileContains asserts that a file within the scenario's temporary directory contains specific content.
func (s *Session) AssertFileContains(relPath string, content string) error {
	if s.rootDir == "" {
		return fmt.Errorf("session is not aware of test root directory")
	}
	fullPath := filepath.Join(s.rootDir, relPath)

	fileContent, err := fs.ReadString(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	return assert.Contains(fileContent, content)
}

// SendKeysAndWaitForChange sends keys and waits for the screen to change.
func (s *Session) SendKeysAndWaitForChange(timeout time.Duration, keys ...string) error {
	initialContent, err := s.Capture(WithCleanedOutput())
	if err != nil {
		return fmt.Errorf("failed to capture initial screen state: %w", err)
	}

	if err := s.SendKeys(keys...); err != nil {
		return fmt.Errorf("failed to send keys: %w", err)
	}

	opts := wait.DefaultOptions()
	opts.Timeout = timeout

	return wait.For(func() (bool, error) {
		currentContent, err := s.Capture(WithCleanedOutput())
		if err != nil {
			// Don't fail the wait, just try again
			return false, nil
		}
		return currentContent != initialContent, nil
	}, opts)
}

// Close terminates the tmux session associated with the TUI.
// This is typically called automatically by the test harness on cleanup.
func (s *Session) Close() error {
	return s.client.KillSession(context.Background(), s.sessionName)
}