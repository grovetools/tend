package sessions

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/grovetools/core/tui/keymap"
)

// KeyMap defines the keybindings for the sessions TUI.
type KeyMap struct {
	keymap.Base
	Attach  key.Binding
	Kill    key.Binding
	Refresh key.Binding
}

// newKeyMap creates a new KeyMap with default bindings.
func newKeyMap() KeyMap {
	return KeyMap{
		Base: keymap.NewBase(),
		Attach: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "attach to session"),
		),
		Kill: key.NewBinding(
			key.WithKeys("x", "X"),
			key.WithHelp("x", "kill session"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh list"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Attach, k.Kill, k.Refresh, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Attach, k.Kill, k.Refresh},
		{k.Help, k.Quit},
	}
}

// Sections returns all keybinding sections for the sessions TUI.
func (k KeyMap) Sections() []keymap.Section {
	return []keymap.Section{
		{
			Name:     "Navigation",
			Bindings: []key.Binding{k.Up, k.Down, k.PageUp, k.PageDown},
		},
		{
			Name:     "Actions",
			Bindings: []key.Binding{k.Attach, k.Kill, k.Refresh},
		},
		k.Base.SystemSection(),
	}
}

// KeymapInfo returns the keymap metadata for the tend sessions TUI.
// Used by the grove keys registry generator to aggregate all TUI keybindings.
func KeymapInfo() keymap.TUIInfo {
	km := newKeyMap()
	return keymap.MakeTUIInfo(
		"tend-sessions",
		"tend",
		"Debug session manager",
		km,
	)
}
