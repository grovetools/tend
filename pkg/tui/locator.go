package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/grovetools/tend/pkg/wait"
)

// DebugSnapshot mirrors the JSON returned by GET /debug/state on the terminal's
// debug Unix socket. Keep in sync with terminal/internal/app/debug.go.
type DebugSnapshot struct {
	ActivePanelID string                    `json:"active_panel_id"`
	HUD           string                    `json:"hud"`
	Rail          []DebugRailItem           `json:"rail"`
	Panels        map[string]DebugPanelInfo `json:"panels"`
}

// DebugRailItem mirrors the terminal's DebugRailItem.
type DebugRailItem struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Icon     string `json:"icon"`
	Group    string `json:"group"`
	IsActive bool   `json:"is_active"`
}

// DebugPanelBounds holds the absolute layout position of a panel leaf.
type DebugPanelBounds struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// DebugPanelInfo describes a single panel in the BSP tree.
type DebugPanelInfo struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Bounds    DebugPanelBounds       `json:"bounds"`
	IsFocused bool                   `json:"is_focused"`
	Text      string                 `json:"text"`
	State     map[string]interface{} `json:"state,omitempty"`
}

// defaultLocatorTimeout is the default timeout for polling locator methods.
const defaultLocatorTimeout = 10 * time.Second

// defaultLocatorPoll is the default poll interval for locator methods.
const defaultLocatorPoll = 50 * time.Millisecond

// ---------------------------------------------------------------------------
// PanelLocator
// ---------------------------------------------------------------------------

// PanelLocator provides Playwright-style assertions and queries for a specific
// panel identified by its panel ID. Obtain one via Session.Panel(id).
type PanelLocator struct {
	session *Session
	panelID string
}

// AssertExists polls the debug state until the panel appears in the snapshot.
func (l *PanelLocator) AssertExists() error {
	err := wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		if _, ok := snap.Panels[l.panelID]; ok {
			return true, fmt.Sprintf("panel %q exists", l.panelID), nil
		}
		return false, fmt.Sprintf("panel %q not found", l.panelID), nil
	}, wait.Options{
		Timeout:      defaultLocatorTimeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
	if err != nil {
		return fmt.Errorf("%w\n\n=== Diagnostic Context ===\n%s", err, l.session.DiagnosticSnapshot())
	}
	return nil
}

// AssertFocused polls the debug state until the panel is the active panel.
func (l *PanelLocator) AssertFocused() error {
	err := wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		p, ok := snap.Panels[l.panelID]
		if !ok {
			return false, fmt.Sprintf("panel %q not found", l.panelID), nil
		}
		if p.IsFocused {
			return true, fmt.Sprintf("panel %q is focused", l.panelID), nil
		}
		return false, fmt.Sprintf("panel %q not focused (active: %s)", l.panelID, snap.ActivePanelID), nil
	}, wait.Options{
		Timeout:      defaultLocatorTimeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
	if err != nil {
		return fmt.Errorf("%w\n\n=== Diagnostic Context ===\n%s", err, l.session.DiagnosticSnapshot())
	}
	return nil
}

// AssertBounds polls the debug state until the panel has the exact given bounds.
func (l *PanelLocator) AssertBounds(x, y, w, h int) error {
	err := wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		p, ok := snap.Panels[l.panelID]
		if !ok {
			return false, fmt.Sprintf("panel %q not found", l.panelID), nil
		}
		b := p.Bounds
		if b.X == x && b.Y == y && b.W == w && b.H == h {
			return true, "bounds match", nil
		}
		return false, fmt.Sprintf("bounds mismatch: got (%d,%d,%d,%d) want (%d,%d,%d,%d)",
			b.X, b.Y, b.W, b.H, x, y, w, h), nil
	}, wait.Options{
		Timeout:      defaultLocatorTimeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
	if err != nil {
		return fmt.Errorf("%w\n\n=== Diagnostic Context ===\n%s", err, l.session.DiagnosticSnapshot())
	}
	return nil
}

// WaitForText polls until the panel's rendered text contains the given substring.
func (l *PanelLocator) WaitForText(text string, timeout time.Duration) error {
	return wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		p, ok := snap.Panels[l.panelID]
		if !ok {
			return false, fmt.Sprintf("panel %q not found", l.panelID), nil
		}
		if strings.Contains(p.Text, text) {
			return true, fmt.Sprintf("found %q in panel %q", text, l.panelID), nil
		}
		return false, fmt.Sprintf("text %q not found in panel %q", text, l.panelID), nil
	}, wait.Options{
		Timeout:      timeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
}

// WaitForState polls until the panel's state map contains key with a matching value.
// The expected value is compared using fmt.Sprintf("%v") for both sides.
func (l *PanelLocator) WaitForState(key string, expected interface{}, timeout time.Duration) error {
	expectedStr := fmt.Sprintf("%v", expected)
	return wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		p, ok := snap.Panels[l.panelID]
		if !ok {
			return false, fmt.Sprintf("panel %q not found", l.panelID), nil
		}
		val, exists := p.State[key]
		if !exists {
			return false, fmt.Sprintf("state key %q not found in panel %q", key, l.panelID), nil
		}
		actualStr := fmt.Sprintf("%v", val)
		if actualStr == expectedStr {
			return true, fmt.Sprintf("state[%q] = %v", key, expected), nil
		}
		return false, fmt.Sprintf("state[%q] = %v, want %v", key, val, expected), nil
	}, wait.Options{
		Timeout:      timeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
}

// Text returns the panel's current rendered text (ANSI-stripped).
func (l *PanelLocator) Text() (string, error) {
	snap, err := l.session.GetDebugState()
	if err != nil {
		return "", err
	}
	p, ok := snap.Panels[l.panelID]
	if !ok {
		return "", fmt.Errorf("panel %q not found in debug state", l.panelID)
	}
	return p.Text, nil
}

// State returns the panel's current test state map.
func (l *PanelLocator) State() (map[string]interface{}, error) {
	snap, err := l.session.GetDebugState()
	if err != nil {
		return nil, err
	}
	p, ok := snap.Panels[l.panelID]
	if !ok {
		return nil, fmt.Errorf("panel %q not found in debug state", l.panelID)
	}
	return p.State, nil
}

// Bounds returns the panel's current layout bounds.
func (l *PanelLocator) Bounds() (x, y, w, h int, err error) {
	snap, err := l.session.GetDebugState()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	p, ok := snap.Panels[l.panelID]
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("panel %q not found in debug state", l.panelID)
	}
	b := p.Bounds
	return b.X, b.Y, b.W, b.H, nil
}

// Focus sends a POST /debug/focus request to switch focus to this panel.
func (l *PanelLocator) Focus() error {
	return l.session.postDebug("/debug/focus", map[string]string{"panel_id": l.panelID})
}

// SendKeys sends a POST /debug/keys request to inject keystrokes into this panel.
// The keys string uses standard notation: individual characters are sent as-is,
// special keys use C-<char> for ctrl, and names like Enter, Esc, Tab, Space,
// Backspace, Up, Down, Left, Right.
func (l *PanelLocator) SendKeys(keys string) error {
	return l.session.postDebug("/debug/keys", map[string]string{"panel_id": l.panelID, "keys": keys})
}

// ---------------------------------------------------------------------------
// RailLocator
// ---------------------------------------------------------------------------

// RailLocator provides assertions for icon rail items, identified by label.
// Obtain one via Session.RailItem(label).
type RailLocator struct {
	session *Session
	label   string
}

// AssertExists polls the debug state until a rail item with the given label exists.
func (l *RailLocator) AssertExists() error {
	err := wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		for _, item := range snap.Rail {
			if item.Label == l.label {
				return true, fmt.Sprintf("rail item %q exists", l.label), nil
			}
		}
		return false, fmt.Sprintf("rail item %q not found", l.label), nil
	}, wait.Options{
		Timeout:      defaultLocatorTimeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
	if err != nil {
		return fmt.Errorf("%w\n\n=== Diagnostic Context ===\n%s", err, l.session.DiagnosticSnapshot())
	}
	return nil
}

// Click sends a POST /debug/focus request with the rail item's panel ID,
// effectively focusing the panel associated with this rail item.
func (l *RailLocator) Click() error {
	snap, err := l.session.GetDebugState()
	if err != nil {
		return fmt.Errorf("failed to fetch debug state for rail click: %w", err)
	}
	for _, item := range snap.Rail {
		if item.Label == l.label {
			return l.session.postDebug("/debug/focus", map[string]string{"panel_id": item.ID})
		}
	}
	return fmt.Errorf("rail item %q not found", l.label)
}

// AssertActive polls the debug state until the rail item with the given label is active.
func (l *RailLocator) AssertActive() error {
	err := wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		for _, item := range snap.Rail {
			if item.Label == l.label {
				if item.IsActive {
					return true, fmt.Sprintf("rail item %q is active", l.label), nil
				}
				return false, fmt.Sprintf("rail item %q exists but not active", l.label), nil
			}
		}
		return false, fmt.Sprintf("rail item %q not found", l.label), nil
	}, wait.Options{
		Timeout:      defaultLocatorTimeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
	if err != nil {
		return fmt.Errorf("%w\n\n=== Diagnostic Context ===\n%s", err, l.session.DiagnosticSnapshot())
	}
	return nil
}

// ---------------------------------------------------------------------------
// HUDLocator
// ---------------------------------------------------------------------------

// HUDLocator provides assertions for the HUD (head-up display) text.
// Obtain one via Session.HUD().
type HUDLocator struct {
	session *Session
}

// WaitForText polls the debug state until the HUD text contains the given substring.
func (l *HUDLocator) WaitForText(text string, timeout time.Duration) error {
	err := wait.ForWithMessage(func() (bool, string, error) {
		snap, err := l.session.GetDebugState()
		if err != nil {
			return false, "failed to fetch debug state", err
		}
		if strings.Contains(snap.HUD, text) {
			return true, fmt.Sprintf("found %q in HUD", text), nil
		}
		return false, fmt.Sprintf("text %q not found in HUD", text), nil
	}, wait.Options{
		Timeout:      timeout,
		PollInterval: defaultLocatorPoll,
		Immediate:    true,
	})
	if err != nil {
		return fmt.Errorf("%w\n\n=== Diagnostic Context ===\n%s", err, l.session.DiagnosticSnapshot())
	}
	return nil
}

// Text returns the current HUD text (ANSI-stripped).
func (l *HUDLocator) Text() (string, error) {
	snap, err := l.session.GetDebugState()
	if err != nil {
		return "", err
	}
	return snap.HUD, nil
}
