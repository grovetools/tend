package drive

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grovetools/tend/pkg/tui"
)

// Default runner timings. These mirror Session.WaitStable's defaults so that a
// `wait` step behaves like the engine-plane WaitStable it stands in for, but
// implemented over the debug socket.
const (
	defaultStepTimeout  = 10 * time.Second
	defaultStableWindow = 200 * time.Millisecond
	defaultPollInterval = 100 * time.Millisecond
)

// Outcome is the recorded result of a single step.
type Outcome string

const (
	// OutcomeOK means the step ran and passed.
	OutcomeOK Outcome = "ok"
	// OutcomeAssertFailed means an assertion step ran and did not match.
	OutcomeAssertFailed Outcome = "assert-failed"
	// OutcomeError means the step could not be executed (infrastructure error).
	OutcomeError Outcome = "error"
	// OutcomeSkipped means the run stopped before reaching this step.
	OutcomeSkipped Outcome = "skipped"
)

// Driver is the narrow debug-socket surface the runner needs. It is satisfied by
// a real *tui.Session (via SessionDriver) and by fakes in unit tests.
type Driver interface {
	// State fetches the current structural debug snapshot.
	State() (*tui.DebugSnapshot, error)
	// SendKeys injects keystrokes into the given panel.
	SendKeys(panelID, keys string) error
	// SendKittyKey injects a synthesized CSI-u key event into the given panel.
	SendKittyKey(panelID string, keycode, mods int) error
}

// SessionDriver adapts a *tui.Session to the Driver interface. It uses only the
// debug-socket plane, which is the only plane available in attach mode.
type SessionDriver struct {
	Session *tui.Session
}

// State implements Driver.
func (d SessionDriver) State() (*tui.DebugSnapshot, error) { return d.Session.GetDebugState() }

// SendKeys implements Driver.
func (d SessionDriver) SendKeys(panelID, keys string) error {
	return d.Session.Panel(panelID).SendKeys(keys)
}

// SendKittyKey implements Driver.
func (d SessionDriver) SendKittyKey(panelID string, keycode, mods int) error {
	return d.Session.SendKittyKey(panelID, keycode, mods)
}

// StepResult is the recorded execution of one step.
type StepResult struct {
	Index     int
	Kind      StepKind
	Arg       string // human-readable argument (keys, text, pattern, label, kitty summary)
	StartedAt time.Time
	EndedAt   time.Time
	Outcome   Outcome
	Failure   string   // failure text; "" when Outcome is ok/skipped
	Files     []string // snapshot artifact basenames written for this step
}

// RunResult is the whole run's outcome, consumed by the manifest writer and the
// command's exit-code / diagnostics handling.
type RunResult struct {
	StartedAt time.Time
	EndedAt   time.Time
	Steps     []StepResult

	// FailedIndex is the 1-based index of the step that stopped the run, or 0
	// when every step passed.
	FailedIndex int
	// Diagnostic is a rendered snapshot of the debug state captured at the
	// moment of failure (empty on full success).
	Diagnostic string
}

// ExitCode maps outcomes to the process exit code contract:
//
//	0 — all steps passed
//	1 — an infrastructure error stopped the run
//	2 — an assertion failed but the run completed to the failure point
func (r *RunResult) ExitCode() int {
	assertFailed := false
	for _, s := range r.Steps {
		switch s.Outcome {
		case OutcomeError:
			return 1
		case OutcomeAssertFailed:
			assertFailed = true
		}
	}
	if assertFailed {
		return 2
	}
	return 0
}

// Runner replays a script against a Driver and writes snapshot artifacts into
// OutDir as it goes.
type Runner struct {
	Driver Driver
	Steps  []Step
	OutDir string

	// DefaultTimeout governs wait steps that do not set their own timeout.
	DefaultTimeout time.Duration
	// StableWindow is how long the rendered state must be unchanged to count as
	// stable. PollInterval is how often it is sampled.
	StableWindow time.Duration
	PollInterval time.Duration

	// now is injectable for deterministic tests; defaults to time.Now.
	now func() time.Time
}

// Run replays every step in order, stopping at the first failure. Steps after a
// failure are recorded as skipped so the manifest always lists the full script.
func (r *Runner) Run() *RunResult {
	if r.now == nil {
		r.now = time.Now
	}
	if r.DefaultTimeout <= 0 {
		r.DefaultTimeout = defaultStepTimeout
	}
	if r.StableWindow <= 0 {
		r.StableWindow = defaultStableWindow
	}
	if r.PollInterval <= 0 {
		r.PollInterval = defaultPollInterval
	}

	result := &RunResult{StartedAt: r.now()}
	stopped := false

	for i, step := range r.Steps {
		sr := StepResult{Index: i + 1, Kind: step.Kind, Arg: stepArg(step)}

		if stopped {
			sr.Outcome = OutcomeSkipped
			result.Steps = append(result.Steps, sr)
			continue
		}

		sr.StartedAt = r.now()
		outcome, failure, files := r.runStep(step)
		sr.EndedAt = r.now()
		sr.Outcome = outcome
		sr.Failure = failure
		sr.Files = files
		result.Steps = append(result.Steps, sr)

		if outcome == OutcomeAssertFailed || outcome == OutcomeError {
			stopped = true
			result.FailedIndex = i + 1
			result.Diagnostic = r.captureDiagnostic()
		}
	}

	result.EndedAt = r.now()
	return result
}

// runStep executes a single step and returns its outcome, failure text, and any
// artifact files written.
func (r *Runner) runStep(step Step) (Outcome, string, []string) {
	switch step.Kind {
	case StepType:
		snap, err := r.Driver.State()
		if err != nil {
			return OutcomeError, fmt.Sprintf("fetch state before type: %v", err), nil
		}
		if err := r.Driver.SendKeys(snap.ActivePanelID, step.Text); err != nil {
			return OutcomeError, fmt.Sprintf("send keys to %q: %v", snap.ActivePanelID, err), nil
		}
		return OutcomeOK, "", nil

	case StepKittyKey:
		panel := step.Kitty.Panel
		if panel == "" {
			snap, err := r.Driver.State()
			if err != nil {
				return OutcomeError, fmt.Sprintf("fetch state before kittykey: %v", err), nil
			}
			panel = snap.ActivePanelID
		}
		if err := r.Driver.SendKittyKey(panel, step.Kitty.Keycode, step.Kitty.Mods); err != nil {
			return OutcomeError, fmt.Sprintf("send kitty key to %q: %v", panel, err), nil
		}
		return OutcomeOK, "", nil

	case StepWait:
		timeout := step.Timeout
		if timeout <= 0 {
			timeout = r.DefaultTimeout
		}
		if err := r.waitStable(timeout); err != nil {
			return OutcomeError, err.Error(), nil
		}
		return OutcomeOK, "", nil

	case StepAssertContains:
		text, err := r.renderState()
		if err != nil {
			return OutcomeError, fmt.Sprintf("fetch state for assert_contains: %v", err), nil
		}
		if !strings.Contains(text, step.Text) {
			return OutcomeAssertFailed, fmt.Sprintf("expected content to contain %q", step.Text), nil
		}
		return OutcomeOK, "", nil

	case StepAssertPattern:
		text, err := r.renderState()
		if err != nil {
			return OutcomeError, fmt.Sprintf("fetch state for assert_pattern: %v", err), nil
		}
		if !step.pattern.MatchString(text) {
			return OutcomeAssertFailed, fmt.Sprintf("expected content to match pattern %q", step.Text), nil
		}
		return OutcomeOK, "", nil

	case StepSnapshot:
		files, err := r.writeSnapshot(step.Text)
		if err != nil {
			return OutcomeError, fmt.Sprintf("write snapshot %q: %v", step.Text, err), nil
		}
		return OutcomeOK, "", files

	default:
		return OutcomeError, fmt.Sprintf("unhandled step kind %q", step.Kind), nil
	}
}

// waitStable polls the rendered debug state until it is unchanged for
// StableWindow, or returns an error when timeout elapses first. There are no
// sleeps outside this explicit wait step.
func (r *Runner) waitStable(timeout time.Duration) error {
	deadline := r.now().Add(timeout)
	var last string
	var stableSince time.Time
	initialized := false

	for {
		text, err := r.renderState()
		if err != nil {
			return fmt.Errorf("wait: fetch state: %w", err)
		}
		nowT := r.now()
		switch {
		case !initialized:
			last = text
			stableSince = nowT
			initialized = true
		case text != last:
			last = text
			stableSince = nowT
		case nowT.Sub(stableSince) >= r.StableWindow:
			return nil
		}

		if nowT.After(deadline) {
			return fmt.Errorf("wait: content did not stabilize within %s", timeout)
		}
		time.Sleep(r.PollInterval)
	}
}

// renderState fetches and renders the current debug state as flat text for
// substring / pattern assertions.
func (r *Runner) renderState() (string, error) {
	snap, err := r.Driver.State()
	if err != nil {
		return "", err
	}
	return RenderText(snap), nil
}

// captureDiagnostic renders the current state for failure reporting, degrading
// to an error note when the state cannot be fetched.
func (r *Runner) captureDiagnostic() string {
	snap, err := r.Driver.State()
	if err != nil {
		return fmt.Sprintf("(failed to capture diagnostic state: %v)", err)
	}
	return RenderSnapshot(snap)
}

// writeSnapshot fetches the debug state and writes <label>.txt (visual render)
// and <label>.json (structural state) into OutDir. It returns the basenames.
func (r *Runner) writeSnapshot(label string) ([]string, error) {
	snap, err := r.Driver.State()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(r.OutDir, 0o755); err != nil {
		return nil, err
	}

	txtName := label + ".txt"
	jsonName := label + ".json"

	if err := os.WriteFile(filepath.Join(r.OutDir, txtName), []byte(RenderSnapshot(snap)), 0o644); err != nil {
		return nil, err
	}
	jsonBytes, err := marshalSnapshot(snap)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(r.OutDir, jsonName), jsonBytes, 0o644); err != nil {
		return nil, err
	}
	return []string{txtName, jsonName}, nil
}

// stepArg renders a step's argument for the manifest and result log.
func stepArg(step Step) string {
	switch step.Kind {
	case StepType, StepAssertContains, StepAssertPattern, StepSnapshot:
		return step.Text
	case StepWait:
		if step.Timeout > 0 {
			return step.Timeout.String()
		}
		return ""
	case StepKittyKey:
		return fmt.Sprintf("panel=%q keycode=%d mods=%d", step.Kitty.Panel, step.Kitty.Keycode, step.Kitty.Mods)
	default:
		return ""
	}
}

// RenderText flattens a debug snapshot into the text that assertions match
// against: the HUD followed by each panel's rendered text, ordered by panel ID
// for determinism.
func RenderText(snap *tui.DebugSnapshot) string {
	if snap == nil {
		return ""
	}
	var b strings.Builder
	if snap.HUD != "" {
		b.WriteString(snap.HUD)
		b.WriteString("\n")
	}
	for _, id := range sortedPanelIDs(snap) {
		b.WriteString(snap.Panels[id].Text)
		b.WriteString("\n")
	}
	return b.String()
}

// RenderSnapshot produces the human-readable <label>.txt content: the flattened
// visual text plus a compact structural summary. In attach mode the "visual"
// evidence is the debug snapshot's rendered panel text (there is no tmux
// capture-pane because attach mode has no mux engine).
func RenderSnapshot(snap *tui.DebugSnapshot) string {
	var b strings.Builder
	b.WriteString("[Visual Capture]\n")
	b.WriteString(RenderText(snap))
	if snap == nil {
		return b.String()
	}

	b.WriteString("\n[Structural State]\n")
	b.WriteString(fmt.Sprintf("HUD:          %q\n", snap.HUD))
	b.WriteString(fmt.Sprintf("Active Panel: %s\n", snap.ActivePanelID))
	if len(snap.Rail) > 0 {
		items := make([]string, 0, len(snap.Rail))
		for _, ri := range snap.Rail {
			if ri.IsActive {
				items = append(items, fmt.Sprintf("*%s*", ri.Label))
			} else {
				items = append(items, ri.Label)
			}
		}
		b.WriteString(fmt.Sprintf("Rail:         %s\n", strings.Join(items, " | ")))
	}
	b.WriteString("Panels:\n")
	for _, id := range sortedPanelIDs(snap) {
		pi := snap.Panels[id]
		focus := "-"
		if pi.IsFocused {
			focus = "*"
		}
		b.WriteString(fmt.Sprintf("  %s [%s] (type: %s)\n", focus, pi.ID, pi.Type))
		b.WriteString(fmt.Sprintf("    Bounds: x=%d, y=%d, w=%d, h=%d\n",
			pi.Bounds.X, pi.Bounds.Y, pi.Bounds.W, pi.Bounds.H))
	}
	return b.String()
}

// sortedPanelIDs returns the snapshot's panel IDs in stable order.
func sortedPanelIDs(snap *tui.DebugSnapshot) []string {
	ids := make([]string, 0, len(snap.Panels))
	for id := range snap.Panels {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
