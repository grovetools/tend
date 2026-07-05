package demo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grovetools/core/config"
	"gopkg.in/yaml.v3"
)

// writeUserFixture creates a GROVE_HOME fixture whose global grove.toml carries
// the given [tui] TOML body, points GROVE_HOME at it, and returns the home dir.
func writeUserFixture(t *testing.T, tuiTOML string) string {
	t.Helper()
	home := t.TempDir()
	cfgDir := filepath.Join(home, "config", "grove")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "name = \"user\"\n"
	if tuiTOML != "" {
		body += tuiTOML
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "grove.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROVE_HOME", home)
	return home
}

// newDemoDir creates a demo dir with the config/grove subdir the sync writes to.
func newDemoDir(t *testing.T) string {
	t.Helper()
	demoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(demoDir, "config", "grove"), 0o755); err != nil {
		t.Fatal(err)
	}
	return demoDir
}

func overridePath(demoDir string) string {
	return filepath.Join(demoDir, "config", "grove", "grove.override.yml")
}

// readOverrideTUI reads the demo override and returns its `tui` sub-map.
func readOverrideTUI(t *testing.T, demoDir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(overridePath(demoDir))
	if err != nil {
		t.Fatalf("reading demo override: %v", err)
	}
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshaling demo override: %v", err)
	}
	tui, ok := out["tui"].(map[string]any)
	if !ok {
		t.Fatalf("demo override has no [tui] section: %s", data)
	}
	return tui
}

// TestSyncUserTUIConfig_WhitelistOnly verifies the written override carries
// only the whitelisted [tui] fields, never forbidden ones (hide_splash_on_
// startup, keybindings).
func TestSyncUserTUIConfig_WhitelistOnly(t *testing.T) {
	writeUserFixture(t, `[tui]
theme = "dracula"
leader_key = "ctrl+a"
action_key = "ctrl+x"
icons = "nerd"
vim_control_hjkl_pane_nav = true
hide_splash_on_startup = true
sidebar_expanded = true

[tui.focus]
style = "gutter"
thickness = 3
dim_inactive = true

[tui.keybindings.navigation]
up = ["k", "up"]
`)
	demoDir := newDemoDir(t)

	if err := SyncUserTUIConfig(demoDir); err != nil {
		t.Fatalf("SyncUserTUIConfig: %v", err)
	}

	tui := readOverrideTUI(t, demoDir)

	// Whitelisted fields present.
	for k, want := range map[string]any{
		"theme":                     "dracula",
		"leader_key":                "ctrl+a",
		"action_key":                "ctrl+x",
		"icons":                     "nerd",
		"vim_control_hjkl_pane_nav": true,
	} {
		if tui[k] != want {
			t.Errorf("override tui.%s = %v, want %v", k, tui[k], want)
		}
	}
	focus, ok := tui["focus"].(map[string]any)
	if !ok {
		t.Fatalf("override missing tui.focus: %v", tui)
	}
	if focus["thickness"] != 3 || focus["dim_inactive"] != true || focus["style"] != "gutter" {
		t.Errorf("focus not fully copied: %v", focus)
	}

	// Forbidden fields absent.
	for _, k := range []string{"hide_splash_on_startup", "sidebar_expanded", "keybindings"} {
		if _, present := tui[k]; present {
			t.Errorf("forbidden field tui.%s leaked into demo override", k)
		}
	}
}

// TestSyncUserTUIConfig_IdempotentFullReplace verifies a re-sync after the user
// changes their config overwrites the override and drops now-absent keys.
func TestSyncUserTUIConfig_IdempotentFullReplace(t *testing.T) {
	home := writeUserFixture(t, `[tui]
theme = "dracula"
leader_key = "ctrl+a"
`)
	demoDir := newDemoDir(t)

	if err := SyncUserTUIConfig(demoDir); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if tui := readOverrideTUI(t, demoDir); tui["theme"] != "dracula" || tui["leader_key"] != "ctrl+a" {
		t.Fatalf("first sync payload wrong: %v", tui)
	}

	// User switches theme and drops the leader_key override.
	if err := os.WriteFile(filepath.Join(home, "config", "grove", "grove.toml"),
		[]byte("name = \"user\"\n[tui]\ntheme = \"nord\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SyncUserTUIConfig(demoDir); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	tui := readOverrideTUI(t, demoDir)
	if tui["theme"] != "nord" {
		t.Errorf("re-sync did not update theme: %v", tui)
	}
	if _, present := tui["leader_key"]; present {
		t.Errorf("stale leader_key survived re-sync (full replace expected): %v", tui)
	}
}

// TestSyncUserTUIConfig_NoTUIRemovesOverride verifies a user with no [tui]
// yields no override file, and that a pre-existing override is removed.
func TestSyncUserTUIConfig_NoTUIRemovesOverride(t *testing.T) {
	writeUserFixture(t, "")
	demoDir := newDemoDir(t)

	// Seed a stale override to prove it gets removed.
	if err := os.WriteFile(overridePath(demoDir), []byte("tui:\n  theme: old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SyncUserTUIConfig(demoDir); err != nil {
		t.Fatalf("SyncUserTUIConfig: %v", err)
	}
	if _, err := os.Stat(overridePath(demoDir)); !os.IsNotExist(err) {
		t.Errorf("override should be removed when user has no [tui]; stat err = %v", err)
	}
}

// TestSyncUserTUIConfig_DemoNoOp verifies the sync is a no-op inside a demo
// (GROVE_DEMO=1), so a demo can never sync its own config into itself.
func TestSyncUserTUIConfig_DemoNoOp(t *testing.T) {
	writeUserFixture(t, `[tui]
theme = "dracula"
`)
	t.Setenv("GROVE_DEMO", "1")
	demoDir := newDemoDir(t)

	if err := SyncUserTUIConfig(demoDir); err != nil {
		t.Fatalf("SyncUserTUIConfig: %v", err)
	}
	if _, err := os.Stat(overridePath(demoDir)); !os.IsNotExist(err) {
		t.Errorf("GROVE_DEMO=1 must be a no-op; override file was written")
	}
}

// TestSyncUserTUIConfig_RoundTripEffectiveConfig is the regression test for the
// core merge fix: a user config with vim_control_hjkl_pane_nav and
// focus.thickness + dim_inactive must survive the whole path — sync into the
// demo override, then LoadFrom over the demo as GROVE_HOME merges the override
// as the global override layer — and land in the demo's EFFECTIVE config.
// Forbidden fields (hide_splash_on_startup) must NOT be inherited.
func TestSyncUserTUIConfig_RoundTripEffectiveConfig(t *testing.T) {
	writeUserFixture(t, `[tui]
theme = "nord"
vim_control_hjkl_pane_nav = true
hide_splash_on_startup = true

[tui.focus]
thickness = 2
dim_inactive = true
`)
	demoDir := newDemoDir(t)

	// A realistic demo base global config that carries no [tui].
	if err := os.WriteFile(filepath.Join(demoDir, "config", "grove", "grove.toml"),
		[]byte("name = \"grove-demo-test\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SyncUserTUIConfig(demoDir); err != nil {
		t.Fatalf("SyncUserTUIConfig: %v", err)
	}

	// Load the demo's effective config: GROVE_HOME=demoDir makes the demo's
	// grove.toml the global base and its grove.override.yml the global override.
	// Use a neutral start dir so no project config contaminates the merge.
	t.Setenv("GROVE_HOME", demoDir)
	neutral := t.TempDir()
	cfg, err := config.LoadFrom(neutral)
	if err != nil {
		t.Fatalf("loading demo effective config: %v", err)
	}
	if cfg.TUI == nil {
		t.Fatal("demo effective config has no [tui]")
	}
	if cfg.TUI.Theme != "nord" {
		t.Errorf("theme = %q, want nord", cfg.TUI.Theme)
	}
	if !cfg.TUI.VimControlHjklPaneNav {
		t.Error("vim_control_hjkl_pane_nav did not reach the demo effective config (core merge regression)")
	}
	if cfg.TUI.Focus == nil || cfg.TUI.Focus.Thickness != 2 {
		t.Errorf("focus.thickness did not survive to effective config: %+v", cfg.TUI.Focus)
	}
	if cfg.TUI.Focus == nil || !cfg.TUI.Focus.DimInactive {
		t.Errorf("focus.dim_inactive did not survive to effective config: %+v", cfg.TUI.Focus)
	}
	// Forbidden field must not be inherited by the demo.
	if cfg.TUI.HideSplashOnStartup {
		t.Error("hide_splash_on_startup leaked into the demo effective config")
	}
}
