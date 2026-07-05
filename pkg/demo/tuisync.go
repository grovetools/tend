package demo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grovetools/core/config"
	"gopkg.in/yaml.v3"
)

// tuiWhitelist is the narrow projection of the user's [tui] config that is
// copied into a demo. It deliberately mirrors ONLY the fields we want a demo
// to inherit — theme, leader/action keys, vim pane nav, icons, and the whole
// focus block — so that marshaling it can never leak the rest of TUIConfig
// (keybindings, panels, plugins, hide_splash_on_startup, …) into the demo.
type tuiWhitelist struct {
	Theme                 string              `yaml:"theme,omitempty"`
	LeaderKey             string              `yaml:"leader_key,omitempty"`
	ActionKey             string              `yaml:"action_key,omitempty"`
	VimControlHjklPaneNav bool                `yaml:"vim_control_hjkl_pane_nav,omitempty"`
	Icons                 string              `yaml:"icons,omitempty"`
	Focus                 *config.FocusConfig `yaml:"focus,omitempty"`
}

// isEmpty reports whether the whitelist projected nothing worth writing.
func (w tuiWhitelist) isEmpty() bool {
	return w.Theme == "" &&
		w.LeaderKey == "" &&
		w.ActionKey == "" &&
		!w.VimControlHjklPaneNav &&
		w.Icons == "" &&
		w.Focus == nil
}

// SyncUserTUIConfig projects the user's real [tui] choices (theme, leader/
// action keys, vim pane nav, icons, focus) into the demo's global override
// config at <demoDir>/config/grove/grove.override.yml, so a freshly created or
// re-entered demo feels native to the user's TUI. It is one-way (real → demo):
// the override is full-replaced on every call, so in-demo edits never flow back
// and stale keys are dropped.
//
// It must be called PRE-re-exec, in the real environment (before GROVE_HOME is
// pointed at the demo): the generate step, the CLI attach path, and the
// treemux enter seam all satisfy this. As a guard it is a no-op when
// GROVE_DEMO=1 so a demo can never sync its own config into itself.
//
// The user's effective config is read via config.LoadFrom from a NEUTRAL temp
// directory rather than the cwd: LoadFrom walks up from its start dir merging
// any ecosystem/project [tui] it finds, and the cwd may sit inside a grove
// workspace whose project [tui] would contaminate the copy. A fresh /tmp dir
// has no plausible grove config ancestor, so the result is exactly the user's
// global base + fragments + global override + env overlay. GROVE_HOME/XDG is
// honored by core, so this reads the real user config when called pre-re-exec.
//
// If the user has no whitelisted [tui] at all, any existing demo override file
// is removed rather than replaced with an empty one.
func SyncUserTUIConfig(demoDir string) error {
	// Never sync a demo's config into itself.
	if os.Getenv("GROVE_DEMO") == "1" {
		return nil
	}

	neutral, err := os.MkdirTemp("", "grove-tuisync")
	if err != nil {
		return fmt.Errorf("creating neutral load dir: %w", err)
	}
	defer os.RemoveAll(neutral)

	cfg, err := config.LoadFrom(neutral)
	if err != nil {
		return fmt.Errorf("loading user config: %w", err)
	}

	destPath := filepath.Join(demoDir, "config", "grove", "grove.override.yml")

	whitelist := projectTUIWhitelist(cfg)
	if whitelist.isEmpty() {
		// Nothing to inherit: drop any stale override so old values (including
		// a legacy credential-bearing override) don't linger.
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing stale demo override: %w", err)
		}
		return nil
	}

	// Wrap under the tui key so the override merges as [tui] into the demo's
	// global config layer.
	payload := map[string]any{"tui": whitelist}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling tui override: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0o644); err != nil { //nolint:gosec // no secrets: TUI settings only
		return fmt.Errorf("writing demo override: %w", err)
	}
	return nil
}

// projectTUIWhitelist copies the whitelisted [tui] fields off the resolved
// user config into a tuiWhitelist. A nil cfg.TUI yields an empty projection.
func projectTUIWhitelist(cfg *config.Config) tuiWhitelist {
	var w tuiWhitelist
	if cfg == nil || cfg.TUI == nil {
		return w
	}
	tui := cfg.TUI
	w.Theme = tui.Theme
	w.LeaderKey = tui.LeaderKey
	w.ActionKey = tui.ActionKey
	w.VimControlHjklPaneNav = tui.VimControlHjklPaneNav
	w.Icons = tui.Icons
	if tui.Focus != nil {
		// Copy the whole FocusConfig — it is a bounded, TUI-only struct
		// (style/active_color/inactive_color/thickness/dim_inactive) with no
		// forbidden fields, so wholesale copy is safe.
		focus := *tui.Focus
		w.Focus = &focus
	}
	return w
}
