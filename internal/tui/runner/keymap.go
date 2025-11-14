package runner

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/mattsolo1/grove-core/tui/keymap"
)

// KeyMap defines the keybindings for the test runner TUI.
type KeyMap struct {
	keymap.Base
	DebugRun       key.Binding
	FocusSelected  key.Binding
	FocusEcosystem key.Binding
	ClearFocus     key.Binding
	GoToTop        key.Binding
	GoToBottom     key.Binding
	Fold           key.Binding
	Unfold         key.Binding
	FoldPrefix     key.Binding // z
}

func newKeyMap() KeyMap {
	base := keymap.NewBase()
	return KeyMap{
		Base: base,
		DebugRun: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "run in debug mode"),
		),
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
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
		Fold: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "close fold"),
		),
		Unfold: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "open fold"),
		),
		FoldPrefix: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z*", "fold commands (za, zc, zo, zR, zM)"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.GoToTop, k.GoToBottom},
		{k.Fold, k.Unfold, k.FoldPrefix},
		{k.FocusSelected, k.FocusEcosystem, k.ClearFocus},
		{k.DebugRun, k.Help, k.Quit},
	}
}
