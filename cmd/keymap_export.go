package cmd

import (
	"github.com/grovetools/core/tui/keymap"
	"github.com/grovetools/tend/internal/tui/runner"
)

// RunnerKeymapInfo returns the keymap metadata for the tend runner TUI.
// Used by the grove keys registry generator to aggregate all TUI keybindings.
func RunnerKeymapInfo() keymap.TUIInfo {
	return runner.KeymapInfo()
}
