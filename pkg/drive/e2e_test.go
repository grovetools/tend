package drive

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grovetools/tend/pkg/tui"
)

// debugFixture is a minimal, hermetic stand-in for a real app's debug socket. It
// serves the same /debug/* protocol the driver attaches to (GET /debug/state,
// POST /debug/keys, POST /debug/kitty-keys, POST /debug/root-keys) over a Unix
// socket, so the e2e test exercises the entire drive path — attach, key
// injection round-trip, stability polling, assertions, and bundle writing —
// without spawning treemux/tuimux.
type debugFixture struct {
	mu       sync.Mutex
	snap     tui.DebugSnapshot
	kittyLog []string
	chordLog []string
}

func (f *debugFixture) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/state", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.snap)
	})

	mux.HandleFunc("/debug/keys", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PanelID string `json:"panel_id"`
			Keys    string `json:"keys"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		p := f.snap.Panels[body.PanelID]
		p.Text += body.Keys
		f.snap.Panels[body.PanelID] = p
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/debug/kitty-keys", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PanelID string `json:"panel_id"`
			Keycode int    `json:"keycode"`
			Mods    int    `json:"mods"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.kittyLog = append(f.kittyLog, body.PanelID)
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/debug/root-keys", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Keys string `json:"keys"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.chordLog = append(f.chordLog, body.Keys)
		// Simulate a root-handled panel switch so a following assert_structural
		// can prove the chord round-trip end-to-end.
		f.snap.ActivePanelID = "nav"
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	return mux
}

// startFixture serves the fixture on a short-pathed Unix socket and returns the
// socket path plus a cleanup func.
func startFixture(t *testing.T) (*debugFixture, string) {
	t.Helper()

	// Keep the socket path short: macOS caps AF_UNIX paths at ~104 bytes, and
	// t.TempDir() under /var/folders can overflow that.
	dir, err := os.MkdirTemp("/tmp", "tdrive")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	sockPath := filepath.Join(dir, "d.sock")

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}

	f := &debugFixture{
		snap: tui.DebugSnapshot{
			ActivePanelID: "main",
			HUD:           "READY",
			Rail: []tui.DebugRailItem{
				{ID: "sessions", Label: "Sessions", IsActive: true},
			},
			Panels: map[string]tui.DebugPanelInfo{
				"main": {
					ID:        "main",
					Type:      "shell",
					IsFocused: true,
					Bounds:    tui.DebugPanelBounds{X: 0, Y: 0, W: 80, H: 24},
					Text:      "",
				},
			},
		},
	}

	srv := &http.Server{Handler: f.handler()}
	go func() { _ = srv.Serve(ln) }()

	t.Cleanup(func() {
		_ = srv.Close()
		_ = ln.Close()
		_ = os.RemoveAll(dir)
	})
	return f, sockPath
}

func TestDrive_EndToEnd_DebugSocketMode(t *testing.T) {
	fixture, sockPath := startFixture(t)

	script := `
- type: "PLAYWRIGHT_OK"
- wait: {}
- kittykey: {panel: "main", keycode: 97, mods: 4}
- assert_contains: "PLAYWRIGHT_OK"
- assert_pattern: "PLAYWRIGHT_[A-Z]+"
- assert_structural:
    active_panel: main
    rail_active: sessions
    focused: main
    focused_count: 1
    panel_type:
      main: shell
- chord: "C-g w"
- wait: {}
- assert_structural: {active_panel: nav}
- snapshot: "after-type"
`
	steps, err := ParseScript([]byte(script))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}

	// Attach exactly as the CLI does in debug-socket mode (no --session).
	driver, err := Attach(AttachOptions{Socket: sockPath, ReadyTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	out := t.TempDir()
	runner := &Runner{
		Driver:         driver,
		Steps:          steps,
		OutDir:         out,
		DefaultTimeout: 3 * time.Second,
		StableWindow:   50 * time.Millisecond,
		PollInterval:   20 * time.Millisecond,
	}
	result := runner.Run()

	if code := result.ExitCode(); code != 0 {
		t.Fatalf("expected exit 0, got %d (failed step %d: %s)",
			code, result.FailedIndex, failureText(result))
	}

	// The kittykey step must have reached the fixture.
	fixture.mu.Lock()
	kittyHits := len(fixture.kittyLog)
	chordLog := append([]string(nil), fixture.chordLog...)
	fixture.mu.Unlock()
	if kittyHits != 1 {
		t.Errorf("expected 1 kitty key injected, got %d", kittyHits)
	}

	// The chord step must have reached the fixture as one whole sequence, and
	// the assert_structural above already proved its effect (active panel moved
	// to nav).
	if len(chordLog) != 1 || chordLog[0] != "C-g w" {
		t.Errorf("chordLog = %v, want [\"C-g w\"]", chordLog)
	}

	// Snapshot artifacts must contain the typed text and structural state.
	txt, err := os.ReadFile(filepath.Join(out, "after-type.txt"))
	if err != nil {
		t.Fatalf("read snapshot txt: %v", err)
	}
	if !strings.Contains(string(txt), "PLAYWRIGHT_OK") {
		t.Errorf("snapshot txt missing typed text:\n%s", txt)
	}
	if !strings.Contains(string(txt), "[Structural State]") {
		t.Errorf("snapshot txt missing structural section:\n%s", txt)
	}

	jsonBytes, err := os.ReadFile(filepath.Join(out, "after-type.json"))
	if err != nil {
		t.Fatalf("read snapshot json: %v", err)
	}
	var snap tui.DebugSnapshot
	if err := json.Unmarshal(jsonBytes, &snap); err != nil {
		t.Fatalf("snapshot json is not a valid DebugSnapshot: %v", err)
	}
	if snap.ActivePanelID != "nav" {
		t.Errorf("snapshot json active panel = %q, want nav (post-chord)", snap.ActivePanelID)
	}

	// Write the bundle manifest exactly as the CLI does after a run.
	if err := WriteManifest(out, BuildManifest(
		ManifestMeta{Socket: sockPath, Mode: ModeDebugSocket, Script: "script.yaml"}, result)); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	// Manifest must exist, be schema-versioned, and record every step ok.
	m := readManifest(t, out)
	if m.SchemaVersion != ManifestSchemaVersion {
		t.Errorf("manifest schema_version = %q", m.SchemaVersion)
	}
	if m.ExitCode != 0 {
		t.Errorf("manifest exit_code = %d, want 0", m.ExitCode)
	}
	if len(m.Steps) != 10 {
		t.Fatalf("manifest steps = %d, want 10", len(m.Steps))
	}
	for _, s := range m.Steps {
		if s.Outcome != string(OutcomeOK) {
			t.Errorf("manifest step %d (%s): outcome %q", s.Index, s.Kind, s.Outcome)
		}
	}
	// The snapshot step must list its two artifacts.
	last := m.Steps[len(m.Steps)-1]
	if len(last.Files) != 2 {
		t.Errorf("snapshot step files = %v, want 2 entries", last.Files)
	}
}

func TestDrive_EndToEnd_AssertFailureWritesBundle(t *testing.T) {
	_, sockPath := startFixture(t)

	script := `
- type: "abc"
- assert_contains: "NEVER_PRESENT"
- snapshot: "unreached"
`
	steps, err := ParseScript([]byte(script))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	driver, err := Attach(AttachOptions{Socket: sockPath, ReadyTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	out := t.TempDir()
	result := (&Runner{
		Driver:         driver,
		Steps:          steps,
		OutDir:         out,
		DefaultTimeout: 2 * time.Second,
		StableWindow:   50 * time.Millisecond,
		PollInterval:   20 * time.Millisecond,
	}).Run()

	if code := result.ExitCode(); code != 2 {
		t.Fatalf("expected exit 2 on assert failure, got %d", code)
	}

	// The bundle (most valuable exactly on failure) must still be written.
	m := BuildManifest(ManifestMeta{Socket: sockPath, Mode: ModeDebugSocket}, result)
	if err := WriteManifest(out, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	got := readManifest(t, out)
	if got.FailedStep != 2 {
		t.Errorf("manifest failed_step = %d, want 2", got.FailedStep)
	}
	if got.Steps[2].Outcome != string(OutcomeSkipped) {
		t.Errorf("step 3 outcome = %q, want skipped", got.Steps[2].Outcome)
	}
}

func TestDrive_EndToEnd_StructuralAssertFailureWritesBundle(t *testing.T) {
	_, sockPath := startFixture(t)

	script := `
- type: "abc"
- assert_structural: {active_panel: "not-a-panel", focused_count: 9}
- snapshot: "unreached"
`
	steps, err := ParseScript([]byte(script))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	driver, err := Attach(AttachOptions{Socket: sockPath, ReadyTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}

	out := t.TempDir()
	result := (&Runner{
		Driver:         driver,
		Steps:          steps,
		OutDir:         out,
		DefaultTimeout: 2 * time.Second,
		StableWindow:   50 * time.Millisecond,
		PollInterval:   20 * time.Millisecond,
	}).Run()

	if code := result.ExitCode(); code != 2 {
		t.Fatalf("expected exit 2 on structural assert failure, got %d", code)
	}

	// The bundle (most valuable exactly on failure) must still be written.
	m := BuildManifest(ManifestMeta{Socket: sockPath, Mode: ModeDebugSocket}, result)
	if err := WriteManifest(out, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	got := readManifest(t, out)
	if got.FailedStep != 2 {
		t.Errorf("manifest failed_step = %d, want 2", got.FailedStep)
	}
	if got.Steps[1].Outcome != string(OutcomeAssertFailed) {
		t.Errorf("step 2 outcome = %q, want assert-failed", got.Steps[1].Outcome)
	}
	// The failure message must list every mismatched field.
	for _, part := range []string{
		`active_panel: want "not-a-panel", got "main"`,
		"focused_count: want 9, got 1",
	} {
		if !strings.Contains(got.Steps[1].Failure, part) {
			t.Errorf("failure missing %q; got: %s", part, got.Steps[1].Failure)
		}
	}
	if got.Steps[2].Outcome != string(OutcomeSkipped) {
		t.Errorf("step 3 outcome = %q, want skipped", got.Steps[2].Outcome)
	}
}

func readManifest(t *testing.T, dir string) Manifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ManifestFileName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return m
}

func failureText(r *RunResult) string {
	if r.FailedIndex == 0 || r.FailedIndex > len(r.Steps) {
		return ""
	}
	return r.Steps[r.FailedIndex-1].Failure
}
