package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/grovetools/core/pkg/tmux"

	"github.com/grovetools/tend/pkg/assert"
	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/wait"
)

// Session represents an active TUI application running in a tmux session.
// It provides a high-level API for interaction and assertion.
type Session struct {
	sessionName     string
	client          *tmux.Client
	recording       *SessionRecording // For recording and debugging
	rootDir         string
	debugClient     *http.Client // HTTP client connected via Unix socket to the terminal debug server
	debugSocketPath string       // Path to the terminal's debug Unix socket
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
//
//	session.Type("j")           // Navigate down and wait
//	session.Type("g", "g")      // Go to top (sends separately for chord recognition)
//	session.Type("/", "search") // Open search and type
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
			if err := s.SendKeys(keys[1]); err != nil { //nolint:gosec // len(keys)==2 checked above
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

// WaitForUIStable polls the TUI screen until its content remains unchanged for the specified
// stable duration, or until the timeout is reached. This is useful for waiting for animations
// or asynchronous updates to complete.
//
// Parameters:
// - timeout: Maximum time to wait for stability (e.g., 10*time.Second)
// - pollInterval: How often to check the screen (e.g., 100*time.Millisecond)
// - stableDuration: How long the screen must be unchanged to be considered stable (e.g., 200*time.Millisecond)
//
// For most use cases, use WaitStable() instead, which provides sensible defaults.
func (s *Session) WaitForUIStable(timeout, pollInterval, stableDuration time.Duration) error {
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
// This is equivalent to WaitForUIStable(10*time.Second, 100*time.Millisecond, 200*time.Millisecond).
//
// The 10 second timeout accommodates slow CI environments but returns immediately once
// the UI stabilizes (typically 300-500ms for test fixtures).
//
// Use this method for most testing scenarios. Use WaitForUIStable() directly only if you
// need custom timing parameters.
func (s *Session) WaitStable() error {
	return s.WaitForUIStable(10*time.Second, 100*time.Millisecond, 200*time.Millisecond)
}

// AssertContains immediately checks if the TUI's current content contains the specified text.
// NOTE: For better test reporting, it is recommended to use this with the harness context:
//
//	err := ctx.Check("should show welcome message", session.AssertContains("Welcome"))
func (s *Session) AssertContains(text string) error {
	content, err := s.Capture()
	if err != nil {
		return fmt.Errorf("failed to capture pane for assertion: %w", err)
	}
	return assert.Contains(content, text)
}

// AssertNotContains immediately checks if the TUI's current content does not contain the specified text.
// NOTE: For better test reporting, it is recommended to use this with the harness context:
//
//	err := ctx.Check("should not show error", session.AssertNotContains("Error:"))
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

// NavigateToText moves the selection to the line containing the specified text.
// For selection-based TUIs (like lists), it finds lines with selection indicators (">", "*", "•")
// and calculates navigation from the current selection. For cursor-based interfaces, it uses
// tmux cursor position. Waits for UI to stabilize after each keypress.
func (s *Session) NavigateToText(text string) error {
	targetRow, _, found, err := s.FindTextLocation(text)
	if err != nil {
		return err
	}
	if !found {
		err := fmt.Errorf("text '%s' not found on screen", text)
		s.recordEvent("navigate", map[string]interface{}{"text": text}, "", err)
		return err
	}

	// Try to find current selection by looking for common selection indicators
	content, err := s.Capture(WithCleanedOutput())
	if err != nil {
		return fmt.Errorf("failed to capture screen: %w", err)
	}

	lines := strings.Split(content, "\n")
	currentRow := -1

	// Look for selection indicators at the start of lines
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) > 0 {
			// Check for common selection indicators
			firstChar := rune(trimmed[0])
			if firstChar == '>' || firstChar == '*' || strings.HasPrefix(trimmed, "•") || strings.HasPrefix(trimmed, "▶") {
				currentRow = i + 1 // Convert to 1-based
				break
			}
		}
	}

	// Fall back to tmux cursor position if no selection indicator found
	if currentRow == -1 {
		currentRow, _, err = s.GetCursorPosition()
		if err != nil {
			return fmt.Errorf("failed to get cursor position: %w", err)
		}
	}

	// Calculate vertical movement needed
	rowDiff := targetRow - currentRow
	if rowDiff > 0 {
		for i := 0; i < rowDiff; i++ {
			if err := s.Type("Down"); err != nil {
				return err
			}
		}
	} else if rowDiff < 0 {
		for i := 0; i < -rowDiff; i++ {
			if err := s.Type("Up"); err != nil {
				return err
			}
		}
	}

	// Record successful navigation
	s.recordEvent("navigate", map[string]interface{}{
		"text": text,
		"from": fmt.Sprintf("row %d", currentRow),
		"to":   fmt.Sprintf("row %d", targetRow),
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
					if err := s.Type("Down"); err != nil {
						return err
					}
				}
			} else if rowDiff < 0 {
				for j := 0; j < -rowDiff; j++ {
					if err := s.Type("Up"); err != nil {
						return err
					}
				}
			}

			// Press the selection key
			return s.Type(key)
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
func (s *Session) AssertFileContains(relPath, content string) error {
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

// ---------------------------------------------------------------------------
// Debug server integration (Locator API)
// ---------------------------------------------------------------------------

// SetDebugSocket configures the session to talk to the terminal's debug server
// over the given Unix domain socket path. It creates the HTTP client and waits
// for the server to become ready.
func (s *Session) SetDebugSocket(socketPath string, readyTimeout time.Duration) error {
	s.debugSocketPath = socketPath
	s.debugClient = newDebugHTTPClient(socketPath)
	return waitForDebugServer(s.debugClient, readyTimeout)
}

// GetDebugState fetches the current debug snapshot from the terminal via the
// debug Unix socket. Returns an error if the debug client is not configured.
func (s *Session) GetDebugState() (*DebugSnapshot, error) {
	if s.debugClient == nil {
		return nil, fmt.Errorf("debug client not configured (no GROVETERM_DEBUG_SOCKET)")
	}

	resp, err := s.debugClient.Get("http://unix/debug/state")
	if err != nil {
		return nil, fmt.Errorf("GET /debug/state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /debug/state returned %d", resp.StatusCode)
	}

	var snap DebugSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return nil, fmt.Errorf("decode debug state: %w", err)
	}
	return &snap, nil
}

// postDebug sends a POST request with a JSON body to the given debug API endpoint.
func (s *Session) postDebug(path string, body interface{}) error {
	if s.debugClient == nil {
		return fmt.Errorf("debug client not configured (no GROVETERM_DEBUG_SOCKET)")
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal debug request: %w", err)
	}

	resp, err := s.debugClient.Post("http://unix"+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %s returned %d", path, resp.StatusCode)
	}
	return nil
}

// DiagnosticOutput is implemented by types that can display labelled output
// (e.g. harness.Context). Defined here to avoid a circular import.
type DiagnosticOutput interface {
	ShowCommandOutput(command, stdout, stderr string)
}

// DiagnosticSnapshot returns a formatted string combining the visual tmux
// capture with the structural debug state (if the debug server is available).
// When the debug server is not configured the structural section is omitted.
func (s *Session) DiagnosticSnapshot() string {
	var b strings.Builder

	// Visual capture
	visual, err := s.Capture(WithCleanedOutput())
	if err != nil {
		visual = fmt.Sprintf("(capture error: %v)", err)
	}
	b.WriteString("[Visual Capture]\n")
	b.WriteString(visual)
	b.WriteString("\n")

	// Structural state — gracefully degrade when unavailable
	snap, err := s.GetDebugState()
	if err == nil && snap != nil {
		b.WriteString("\n[Structural State]\n")

		// HUD
		b.WriteString(fmt.Sprintf("HUD:          %q\n", snap.HUD))

		// Active panel
		b.WriteString(fmt.Sprintf("Active Panel: %s\n", snap.ActivePanelID))

		// Rail
		if len(snap.Rail) > 0 {
			var items []string
			for _, ri := range snap.Rail {
				if ri.IsActive {
					items = append(items, fmt.Sprintf("*%s*", ri.Label))
				} else {
					items = append(items, ri.Label)
				}
			}
			b.WriteString(fmt.Sprintf("Rail:         %s\n", strings.Join(items, " | ")))
		}

		// Panels
		b.WriteString("Panels:\n")
		for _, pi := range snap.Panels {
			focus := "-"
			if pi.IsFocused {
				focus = "*"
			}
			b.WriteString(fmt.Sprintf("  %s [%s] (type: %s)\n", focus, pi.ID, pi.Type))
			b.WriteString(fmt.Sprintf("    Bounds: x=%d, y=%d, w=%d, h=%d\n",
				pi.Bounds.X, pi.Bounds.Y, pi.Bounds.W, pi.Bounds.H))
			if len(pi.State) > 0 {
				stateJSON, _ := json.Marshal(pi.State)
				b.WriteString(fmt.Sprintf("    State:  %s\n", string(stateJSON)))
			}
		}
	}

	return b.String()
}

// LogDiagnostic captures a diagnostic snapshot and displays it via the given
// output sink (typically a *harness.Context).
func (s *Session) LogDiagnostic(out DiagnosticOutput, label string) {
	out.ShowCommandOutput(label, s.DiagnosticSnapshot(), "")
}

// GetEvents returns the navigation event log from the debug server.
// Calls GET /debug/events.
func (s *Session) GetEvents() ([]string, error) {
	if s.debugClient == nil {
		return nil, fmt.Errorf("debug client not configured")
	}

	resp, err := s.debugClient.Get("http://unix/debug/events")
	if err != nil {
		return nil, fmt.Errorf("GET /debug/events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /debug/events returned %d", resp.StatusCode)
	}

	var result struct {
		Events []string `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode events response: %w", err)
	}

	return result.Events, nil
}

// ClearEvents resets the navigation event log on the debug server.
// Calls DELETE /debug/events.
func (s *Session) ClearEvents() error {
	if s.debugClient == nil {
		return fmt.Errorf("debug client not configured")
	}

	req, err := http.NewRequest(http.MethodDelete, "http://unix/debug/events", nil)
	if err != nil {
		return fmt.Errorf("create DELETE request: %w", err)
	}

	resp, err := s.debugClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE /debug/events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DELETE /debug/events returned %d", resp.StatusCode)
	}
	return nil
}

// SendKittyKey injects a synthesized KittyKeyMsg into the bubbletea event
// loop for the given panel. This simulates a CSI-u key event from the host
// terminal. keycode is a Unicode codepoint, mods is the kitty modifier
// bitmask (shift=1, alt=2, ctrl=4, super=8).
func (s *Session) SendKittyKey(panelID string, keycode, mods int) error {
	return s.postDebug("/debug/kitty-keys", map[string]interface{}{
		"panel_id": panelID,
		"keycode":  keycode,
		"mods":     mods,
	})
}

// Panel returns a PanelLocator for the given panel ID.
func (s *Session) Panel(id string) *PanelLocator {
	return &PanelLocator{session: s, panelID: id}
}

// RailItem returns a RailLocator for the given rail item label.
func (s *Session) RailItem(label string) *RailLocator {
	return &RailLocator{session: s, label: label}
}

// HUD returns a HUDLocator for the head-up display.
func (s *Session) HUD() *HUDLocator {
	return &HUDLocator{session: s}
}
