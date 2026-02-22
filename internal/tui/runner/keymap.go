package runner

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/grovetools/core/config"
	"github.com/grovetools/core/tui/keymap"
)

// KeyMap defines the keybindings for the test runner TUI.
// It embeds keymap.Base for standard navigation, actions, search, selection, and fold bindings.
// Only TUI-specific bindings that don't exist in Base are defined here.
type KeyMap struct {
	keymap.Base
	// Run operations (TUI-specific)
	Run          key.Binding
	DebugRun     key.Binding
	DebugSession key.Binding
	// Focus operations (TUI-specific)
	FocusSelected  key.Binding
	FocusEcosystem key.Binding
	ClearFocus     key.Binding
}

// newKeyMap creates a new KeyMap with user configuration applied.
// Base bindings (navigation, actions, search, selection, fold) come from keymap.Load().
// Only TUI-specific bindings are defined here.
func newKeyMap(cfg *config.Config) KeyMap {
	km := KeyMap{
		Base: keymap.Load(cfg, "tend.runner"),
		// Run operations
		Run: key.NewBinding(
			key.WithKeys("r", "enter"),
			key.WithHelp("r/⏎", "run"),
		),
		DebugRun: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "run in debug mode"),
		),
		DebugSession: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "run in debug session"),
		),
		// Focus operations
		FocusSelected: key.NewBinding(
			key.WithKeys("."),
			key.WithHelp(".", "focus selected"),
		),
		FocusEcosystem: key.NewBinding(
			key.WithKeys("@"),
			key.WithHelp("@", "focus ecosystem/project"),
		),
		ClearFocus: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "clear focus"),
		),
	}

	// Apply TUI-specific overrides from config
	keymap.ApplyTUIOverrides(cfg, "tend", "runner", &km)

	return km
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// Sections returns all keybinding sections for the tend runner TUI.
// It includes the base sections plus runner-specific sections.
func (k KeyMap) Sections() []keymap.Section {
	return []keymap.Section{
		keymap.NavigationSection(k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom),
		keymap.FoldSection(k.FoldClose, k.FoldOpen, k.FoldToggle, k.FoldOpenAll, k.FoldCloseAll),
		keymap.NewSection(keymap.SectionFocus, k.FocusSelected, k.FocusEcosystem, k.ClearFocus),
		keymap.ActionsSection(k.Run, k.DebugRun, k.DebugSession, k.Search, k.Help, k.Quit),
	}
}

// KeymapInfo returns the keymap metadata for the tend runner TUI.
// Used by the grove keys registry generator to aggregate all TUI keybindings.
func KeymapInfo() keymap.TUIInfo {
	km := newKeyMap(nil)
	return keymap.MakeTUIInfo(
		"tend-runner",
		"tend",
		"Test runner and e2e test browser",
		km,
	)
}
