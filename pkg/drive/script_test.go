package drive

import (
	"testing"
	"time"
)

func TestParseScript_ValidAllStepTypes(t *testing.T) {
	script := `
- type: "hello"
- kittykey: {panel: "shell-0", keycode: 97, mods: 4}
- chord: "C-g w"
- wait: {}
- wait: {timeout: 5s}
- assert_contains: "hello"
- assert_pattern: "he[l]+o"
- snapshot: "after-type"
- assert_structural:
    active_panel: nav
    rail_active: sessions
    focused: nav
    focused_count: 1
    panel_type:
      nav: nav
`
	steps, err := ParseScript([]byte(script))
	if err != nil {
		t.Fatalf("ParseScript returned error: %v", err)
	}
	if len(steps) != 9 {
		t.Fatalf("expected 9 steps, got %d", len(steps))
	}

	want := []StepKind{
		StepType, StepKittyKey, StepChord, StepWait, StepWait,
		StepAssertContains, StepAssertPattern, StepSnapshot, StepAssertStructural,
	}
	for i, k := range want {
		if steps[i].Kind != k {
			t.Errorf("step %d: expected kind %q, got %q", i, k, steps[i].Kind)
		}
	}

	if steps[0].Text != "hello" {
		t.Errorf("type: expected text %q, got %q", "hello", steps[0].Text)
	}
	if steps[1].Kitty != (KittyKey{Panel: "shell-0", Keycode: 97, Mods: 4}) {
		t.Errorf("kittykey: unexpected value %+v", steps[1].Kitty)
	}
	if steps[2].Text != "C-g w" {
		t.Errorf("chord: expected keys %q, got %q", "C-g w", steps[2].Text)
	}
	if steps[3].Timeout != 0 {
		t.Errorf("empty wait: expected zero timeout, got %v", steps[3].Timeout)
	}
	if steps[4].Timeout != 5*time.Second {
		t.Errorf("wait timeout: expected 5s, got %v", steps[4].Timeout)
	}
	if steps[6].pattern == nil || !steps[6].pattern.MatchString("hello") {
		t.Errorf("assert_pattern: pattern not compiled/usable")
	}
	if steps[7].Text != "after-type" {
		t.Errorf("snapshot: expected label %q, got %q", "after-type", steps[7].Text)
	}
	sa := steps[8].Structural
	if sa.ActivePanel != "nav" || sa.RailActive != "sessions" || sa.Focused != "nav" {
		t.Errorf("assert_structural: unexpected string fields %+v", sa)
	}
	if sa.FocusedCount == nil || *sa.FocusedCount != 1 {
		t.Errorf("assert_structural: expected focused_count 1, got %v", sa.FocusedCount)
	}
	if sa.PanelType["nav"] != "nav" {
		t.Errorf("assert_structural: expected panel_type[nav]=nav, got %v", sa.PanelType)
	}
}

func TestParseScript_AssertStructural_SingleFieldShapes(t *testing.T) {
	cases := []struct {
		name   string
		script string
	}{
		{"active_panel", `- assert_structural: {active_panel: nav}`},
		{"rail_active", `- assert_structural: {rail_active: sessions}`},
		{"focused", `- assert_structural: {focused: nav}`},
		{"focused_count", `- assert_structural: {focused_count: 0}`},
		{"panel_type", `- assert_structural: {panel_type: {nav: nav}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps, err := ParseScript([]byte(tc.script))
			if err != nil {
				t.Fatalf("ParseScript: %v", err)
			}
			if len(steps) != 1 || steps[0].Kind != StepAssertStructural {
				t.Fatalf("expected one assert_structural step, got %+v", steps)
			}
		})
	}
}

func TestParseScript_AssertStructural_RejectsEmptyMapping(t *testing.T) {
	_, err := ParseScript([]byte(`- assert_structural: {}`))
	if err == nil {
		t.Fatal("expected error for empty assert_structural, got nil")
	}
	if !contains(err.Error(), "at least one field") {
		t.Errorf("expected 'at least one field' error, got: %v", err)
	}
}

func TestParseScript_AssertStructural_RejectsUnknownField(t *testing.T) {
	_, err := ParseScript([]byte(`- assert_structural: {active_pannel: nav}`))
	if err == nil {
		t.Fatal("expected error for unknown assert_structural field, got nil")
	}
	if !contains(err.Error(), "unknown field") {
		t.Errorf("expected 'unknown field' error, got: %v", err)
	}
}

func TestParseScript_AssertStructural_RejectsNonMapping(t *testing.T) {
	_, err := ParseScript([]byte(`- assert_structural: "nav"`))
	if err == nil {
		t.Fatal("expected error for scalar assert_structural, got nil")
	}
}

func TestParseScript_EmptyIsNoSteps(t *testing.T) {
	for _, in := range []string{"", "   ", "\n", "[]"} {
		steps, err := ParseScript([]byte(in))
		if err != nil {
			t.Fatalf("ParseScript(%q) error: %v", in, err)
		}
		if len(steps) != 0 {
			t.Errorf("ParseScript(%q): expected 0 steps, got %d", in, len(steps))
		}
	}
}

func TestParseScript_UnknownKeyRejected(t *testing.T) {
	script := `
- type: "ok"
- typo_key: "boom"
`
	_, err := ParseScript([]byte(script))
	if err == nil {
		t.Fatal("expected error for unknown step key, got nil")
	}
	if !contains(err.Error(), "unknown step key") {
		t.Errorf("expected 'unknown step key' error, got: %v", err)
	}
}

func TestParseScript_RejectsMultiKeyStep(t *testing.T) {
	script := `
- type: "a"
  snapshot: "b"
`
	_, err := ParseScript([]byte(script))
	if err == nil {
		t.Fatal("expected error for multi-key step, got nil")
	}
}

func TestParseScript_RejectsBadRegexp(t *testing.T) {
	_, err := ParseScript([]byte(`- assert_pattern: "he(llo"`))
	if err == nil {
		t.Fatal("expected error for invalid regexp, got nil")
	}
}

func TestParseScript_RejectsBadDuration(t *testing.T) {
	_, err := ParseScript([]byte(`- wait: {timeout: "notaduration"}`))
	if err == nil {
		t.Fatal("expected error for invalid duration, got nil")
	}
}

func TestParseScript_RejectsUnknownWaitField(t *testing.T) {
	_, err := ParseScript([]byte(`- wait: {timeoutt: 5s}`))
	if err == nil {
		t.Fatal("expected error for unknown wait field, got nil")
	}
}

func TestParseScript_KittyKeyRequiresKeycode(t *testing.T) {
	_, err := ParseScript([]byte(`- kittykey: {panel: "p"}`))
	if err == nil {
		t.Fatal("expected error for missing keycode, got nil")
	}
}

func TestParseScript_KittyKeyRejectsUnknownField(t *testing.T) {
	_, err := ParseScript([]byte(`- kittykey: {keycode: 97, bogus: 1}`))
	if err == nil {
		t.Fatal("expected error for unknown kittykey field, got nil")
	}
}

func TestParseScript_ChordRejectsEmptyKeys(t *testing.T) {
	_, err := ParseScript([]byte(`- chord: ""`))
	if err == nil {
		t.Fatal("expected error for empty chord, got nil")
	}
	if !contains(err.Error(), "non-empty key sequence") {
		t.Errorf("expected 'non-empty key sequence' error, got: %v", err)
	}
}

func TestParseScript_ChordRejectsMapping(t *testing.T) {
	_, err := ParseScript([]byte(`- chord: {keys: "C-g w"}`))
	if err == nil {
		t.Fatal("expected error for mapping chord, got nil")
	}
}

func TestParseScript_SnapshotRequiresLabel(t *testing.T) {
	_, err := ParseScript([]byte(`- snapshot: ""`))
	if err == nil {
		t.Fatal("expected error for empty snapshot label, got nil")
	}
}

func TestParseScript_RejectsNonListRoot(t *testing.T) {
	_, err := ParseScript([]byte(`type: "hello"`))
	if err == nil {
		t.Fatal("expected error for non-list root, got nil")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
