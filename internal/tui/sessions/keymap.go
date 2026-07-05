package sessions

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/grovetools/core/config"
	"github.com/grovetools/core/tui/keymap"
)

// KeyMap defines the keybindings for the sessions TUI.
type KeyMap struct {
	keymap.Base
	Attach  key.Binding
	Kill    key.Binding
	Refresh key.Binding
}

// newKeyMap creates a new KeyMap with user configuration applied.
// Base bindings (navigation, actions, search, selection, fold) come from keymap.Load().
// Only TUI-specific bindings are defined here.
func newKeyMap(cfg *config.Config) KeyMap {
	km := KeyMap{
		Base: keymap.Load(cfg, "tend.sessions"),
		Attach: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "attach to session"),
		),
		Kill: key.NewBinding(
			key.WithKeys("x", "X"),
			key.WithHelp("x/X", "kill session"),
		),
		// ctrl+r is the canonical refresh key (Decision 3); r is kept as an
		// alias to preserve muscle memory. Adding ctrl+r to Keys() also lets
		// the disabled Base.Refresh (ctrl+r) drop out without losing the key.
		Refresh: key.NewBinding(
			key.WithKeys("r", "ctrl+r"),
			key.WithHelp("r", "refresh list"),
		),
	}

	// Apply TUI-specific overrides from config
	keymap.ApplyTUIOverrides(cfg, "tend", "sessions", &km)

	// This TUI matches exactly four bindings in its own Update (Quit, Attach,
	// Kill, Refresh) and delegates the rest to a stock bubbles/list, which
	// owns j/k navigation and '/' filtering. Disable every Base binding this
	// TUI does not actually honor so help/registry never advertise dead keys.
	// Kept enabled: Up, Down (coincide with the list's j/k), Search ('/' filter),
	// Help ('?' overlay added below), Quit. Refresh gains ctrl+r above, so the
	// duplicate Base.Refresh is disabled.
	disable(
		&km.Left, &km.Right, &km.PageUp, &km.PageDown, &km.Home, &km.End,
		&km.Top, &km.Bottom,
		&km.Confirm, &km.Cancel, &km.Back, &km.Edit, &km.Delete, &km.Yank,
		&km.Rename, &km.Base.Refresh, &km.CopyPath,
		&km.SearchNext, &km.SearchPrev, &km.ClearSearch, &km.Grep,
		&km.SwitchView, &km.NextTab, &km.PrevTab, &km.FocusNext, &km.FocusPrev,
		&km.TogglePreview,
		&km.Tab1, &km.Tab2, &km.Tab3, &km.Tab4, &km.Tab5, &km.Tab6, &km.Tab7,
		&km.Tab8, &km.Tab9,
		&km.Select, &km.SelectAll, &km.SelectNone,
		&km.FoldOpen, &km.FoldClose, &km.FoldToggle, &km.FoldOpenAll, &km.FoldCloseAll,
	)

	return km
}

// disable turns off a set of bindings in place. A disabled binding is skipped
// by the keymap audit and never rendered in help.
func disable(bindings ...*key.Binding) {
	for _, b := range bindings {
		b.SetEnabled(false)
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Attach, k.Kill, k.Refresh, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Search},
		{k.Attach, k.Kill, k.Refresh},
		{k.Help, k.Quit},
	}
}

// Compile-time guard: KeyMap satisfies the sectioned help/audit contract.
// Value receiver — Sections() has a value receiver and help.New receives the
// value form (see NewModel).
var _ keymap.SectionedKeyMap = KeyMap{}

// Sections returns the keybinding sections for the sessions TUI, scoped to
// only the keys this TUI actually honors: its own Attach/Kill/Refresh plus the
// stock list's j/k navigation and '/' filter, and the added '?' help overlay.
func (k KeyMap) Sections() []keymap.Section {
	return []keymap.Section{
		keymap.NavigationSection(k.Up, k.Down, k.Search),
		keymap.ActionsSection(k.Attach, k.Kill, k.Refresh),
		k.Base.SystemSection(),
	}
}

// KeymapInfo returns the keymap metadata for the tend sessions TUI.
// Used by the grove keys registry generator to aggregate all TUI keybindings.
func KeymapInfo() keymap.TUIInfo {
	km := newKeyMap(nil)
	return keymap.MakeTUIInfo(
		"tend-sessions",
		"tend",
		"Debug session manager",
		km,
	)
}
