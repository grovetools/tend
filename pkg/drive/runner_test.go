package drive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grovetools/tend/pkg/tui"
)

// fakeDriver is an in-memory Driver: typed keys append to the active panel's
// text, so assertions and snapshots have something deterministic to read.
type fakeDriver struct {
	snap      tui.DebugSnapshot
	stateErr  error
	sendErr   error
	kittyErr  error
	kittyLog  []string
	stateHits int
}

func newFakeDriver() *fakeDriver {
	return &fakeDriver{
		snap: tui.DebugSnapshot{
			ActivePanelID: "main",
			HUD:           "READY",
			Panels: map[string]tui.DebugPanelInfo{
				"main": {ID: "main", Type: "shell", IsFocused: true, Text: ""},
			},
		},
	}
}

func (d *fakeDriver) State() (*tui.DebugSnapshot, error) {
	d.stateHits++
	if d.stateErr != nil {
		return nil, d.stateErr
	}
	// Return a copy so callers cannot mutate internal state.
	cp := d.snap
	cp.Panels = map[string]tui.DebugPanelInfo{}
	for k, v := range d.snap.Panels {
		cp.Panels[k] = v
	}
	return &cp, nil
}

func (d *fakeDriver) SendKeys(panelID, keys string) error {
	if d.sendErr != nil {
		return d.sendErr
	}
	p := d.snap.Panels[panelID]
	p.Text += keys
	d.snap.Panels[panelID] = p
	return nil
}

func (d *fakeDriver) SendKittyKey(panelID string, keycode, mods int) error {
	if d.kittyErr != nil {
		return d.kittyErr
	}
	d.kittyLog = append(d.kittyLog, panelID)
	return nil
}

// fastRunner builds a Runner with tiny timings so wait steps resolve instantly.
func fastRunner(d Driver, steps []Step, outDir string) *Runner {
	return &Runner{
		Driver:         d,
		Steps:          steps,
		OutDir:         outDir,
		DefaultTimeout: time.Second,
		StableWindow:   time.Millisecond,
		PollInterval:   time.Millisecond,
	}
}

func TestRunner_HappyPath(t *testing.T) {
	steps := mustParse(t, `
- type: "hello"
- wait: {}
- assert_contains: "hello"
- assert_pattern: "he[l]+o"
- snapshot: "final"
`)
	out := t.TempDir()
	d := newFakeDriver()
	result := fastRunner(d, steps, out).Run()

	if code := result.ExitCode(); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, s := range result.Steps {
		if s.Outcome != OutcomeOK {
			t.Errorf("step %d (%s): expected ok, got %s (%s)", s.Index, s.Kind, s.Outcome, s.Failure)
		}
	}

	// Snapshot artifacts must exist.
	for _, name := range []string{"final.txt", "final.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("expected snapshot artifact %s: %v", name, err)
		}
	}
	if d.kittyLog != nil {
		t.Errorf("no kittykey steps ran, but kittyLog is %v", d.kittyLog)
	}
}

func TestRunner_AssertFailedExit2AndStops(t *testing.T) {
	steps := mustParse(t, `
- type: "abc"
- assert_contains: "zzz"
- snapshot: "unreached"
`)
	out := t.TempDir()
	result := fastRunner(newFakeDriver(), steps, out).Run()

	if code := result.ExitCode(); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if result.FailedIndex != 2 {
		t.Errorf("expected failure at step 2, got %d", result.FailedIndex)
	}
	if result.Steps[1].Outcome != OutcomeAssertFailed {
		t.Errorf("step 2: expected assert-failed, got %s", result.Steps[1].Outcome)
	}
	if result.Steps[2].Outcome != OutcomeSkipped {
		t.Errorf("step 3: expected skipped after failure, got %s", result.Steps[2].Outcome)
	}
	if result.Diagnostic == "" {
		t.Error("expected a diagnostic snapshot on failure")
	}
	// The skipped snapshot must not have written artifacts.
	if _, err := os.Stat(filepath.Join(out, "unreached.txt")); err == nil {
		t.Error("skipped snapshot should not have written artifacts")
	}
}

func TestRunner_AssertStructural_Pass(t *testing.T) {
	steps := mustParse(t, `
- assert_structural:
    active_panel: main
    rail_active: sessions
    focused: main
    focused_count: 1
    panel_type:
      main: shell
- assert_structural: {rail_active: "Sessions"}
`)
	d := newFakeDriver()
	d.snap.Rail = []tui.DebugRailItem{{ID: "sessions", Label: "Sessions", IsActive: true}}
	result := fastRunner(d, steps, t.TempDir()).Run()

	if code := result.ExitCode(); code != 0 {
		t.Fatalf("expected exit 0, got %d (%s)", code, result.Steps[result.FailedIndex-1].Failure)
	}
	for _, s := range result.Steps {
		if s.Outcome != OutcomeOK {
			t.Errorf("step %d: expected ok, got %s (%s)", s.Index, s.Outcome, s.Failure)
		}
	}
}

func TestRunner_AssertStructural_FailListsEveryMismatch(t *testing.T) {
	steps := mustParse(t, `
- assert_structural:
    active_panel: nav
    rail_active: sessions
    focused: side
    focused_count: 2
    panel_type:
      main: nav
      ghost: shell
- snapshot: "unreached"
`)
	d := newFakeDriver()
	d.snap.Panels["side"] = tui.DebugPanelInfo{ID: "side", Type: "nav", IsFocused: false}
	result := fastRunner(d, steps, t.TempDir()).Run()

	if code := result.ExitCode(); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if result.Steps[0].Outcome != OutcomeAssertFailed {
		t.Fatalf("step 1: expected assert-failed, got %s", result.Steps[0].Outcome)
	}
	if result.Steps[1].Outcome != OutcomeSkipped {
		t.Errorf("step 2: expected skipped after failure, got %s", result.Steps[1].Outcome)
	}

	failure := result.Steps[0].Failure
	for _, part := range []string{
		`active_panel: want "nav", got "main"`,
		`rail_active: want "sessions", got "none"`,
		`focused: want "side", got "main"`,
		"focused_count: want 2, got 1",
		`panel_type[ghost]: want "shell", got absent`,
		`panel_type[main]: want "nav", got "shell"`,
	} {
		if !contains(failure, part) {
			t.Errorf("failure message missing %q; got: %s", part, failure)
		}
	}
}

func TestRunner_AssertStructural_InfraErrorExit1(t *testing.T) {
	steps := mustParse(t, `
- assert_structural: {active_panel: main}
`)
	d := newFakeDriver()
	d.stateErr = errFake("socket closed")
	result := fastRunner(d, steps, t.TempDir()).Run()

	if code := result.ExitCode(); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if result.Steps[0].Outcome != OutcomeError {
		t.Errorf("expected error outcome, got %s", result.Steps[0].Outcome)
	}
}

func TestRunner_InfraErrorExit1(t *testing.T) {
	steps := mustParse(t, `
- assert_contains: "x"
`)
	d := newFakeDriver()
	d.stateErr = errFake("socket closed")
	result := fastRunner(d, steps, t.TempDir()).Run()

	if code := result.ExitCode(); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if result.Steps[0].Outcome != OutcomeError {
		t.Errorf("expected error outcome, got %s", result.Steps[0].Outcome)
	}
}

func TestRunResult_ExitCodePrecedence(t *testing.T) {
	cases := []struct {
		name     string
		outcomes []Outcome
		want     int
	}{
		{"all ok", []Outcome{OutcomeOK, OutcomeOK}, 0},
		{"assert fail", []Outcome{OutcomeOK, OutcomeAssertFailed, OutcomeSkipped}, 2},
		{"error wins over assert", []Outcome{OutcomeAssertFailed, OutcomeError}, 1},
		{"empty", nil, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &RunResult{}
			for _, o := range tc.outcomes {
				r.Steps = append(r.Steps, StepResult{Outcome: o})
			}
			if got := r.ExitCode(); got != tc.want {
				t.Errorf("ExitCode = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBuildManifest_AlwaysPresentKeys(t *testing.T) {
	steps := mustParse(t, `
- type: "hi"
- snapshot: "s1"
`)
	out := t.TempDir()
	result := fastRunner(newFakeDriver(), steps, out).Run()

	meta := ManifestMeta{Socket: "/tmp/sock", Session: "", Mode: ModeDebugSocket, Script: "script.yaml"}
	m := BuildManifest(meta, result)

	if m.SchemaVersion != ManifestSchemaVersion {
		t.Errorf("schema_version = %q, want %q", m.SchemaVersion, ManifestSchemaVersion)
	}
	if m.Mode != ModeDebugSocket {
		t.Errorf("mode = %q, want %q", m.Mode, ModeDebugSocket)
	}
	if len(m.Steps) != 2 {
		t.Fatalf("expected 2 manifest steps, got %d", len(m.Steps))
	}

	// Round-trip through JSON and confirm the documented keys are all present,
	// including zero-value ones (session, failed_step, empty failure/files).
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	for _, key := range []string{
		"schema_version", "socket", "session", "mode", "script",
		"started_at", "ended_at", "exit_code", "failed_step", "steps",
	} {
		if _, ok := raw[key]; !ok {
			t.Errorf("manifest missing key %q", key)
		}
	}

	// Each step must carry all documented keys, files as an array not null.
	var rawSteps []map[string]json.RawMessage
	if err := json.Unmarshal(raw["steps"], &rawSteps); err != nil {
		t.Fatalf("unmarshal steps: %v", err)
	}
	for i, rs := range rawSteps {
		for _, key := range []string{"index", "kind", "arg", "started_at", "ended_at", "outcome", "failure", "files"} {
			if _, ok := rs[key]; !ok {
				t.Errorf("step %d missing key %q", i, key)
			}
		}
		if string(rs["files"]) == "null" {
			t.Errorf("step %d files should be [] not null", i)
		}
	}
}

func TestWriteManifest_RoundTrip(t *testing.T) {
	out := t.TempDir()
	m := BuildManifest(ManifestMeta{Mode: ModeDebugSocket}, &RunResult{
		StartedAt: time.Now(),
		EndedAt:   time.Now(),
		Steps:     []StepResult{{Index: 1, Kind: StepType, Arg: "x", Outcome: OutcomeOK}},
	})
	if err := WriteManifest(out, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(out, ManifestFileName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal written manifest: %v", err)
	}
	if got.SchemaVersion != ManifestSchemaVersion {
		t.Errorf("round-tripped schema_version = %q", got.SchemaVersion)
	}
}

// --- helpers ---

func mustParse(t *testing.T, script string) []Step {
	t.Helper()
	steps, err := ParseScript([]byte(script))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	return steps
}

type errFake string

func (e errFake) Error() string { return string(e) }
