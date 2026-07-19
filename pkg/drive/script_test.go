package drive

import (
	"testing"
	"time"
)

func TestParseScript_ValidAllStepTypes(t *testing.T) {
	script := `
- type: "hello"
- kittykey: {panel: "shell-0", keycode: 97, mods: 4}
- wait: {}
- wait: {timeout: 5s}
- assert_contains: "hello"
- assert_pattern: "he[l]+o"
- snapshot: "after-type"
`
	steps, err := ParseScript([]byte(script))
	if err != nil {
		t.Fatalf("ParseScript returned error: %v", err)
	}
	if len(steps) != 7 {
		t.Fatalf("expected 7 steps, got %d", len(steps))
	}

	want := []StepKind{
		StepType, StepKittyKey, StepWait, StepWait,
		StepAssertContains, StepAssertPattern, StepSnapshot,
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
	if steps[2].Timeout != 0 {
		t.Errorf("empty wait: expected zero timeout, got %v", steps[2].Timeout)
	}
	if steps[3].Timeout != 5*time.Second {
		t.Errorf("wait timeout: expected 5s, got %v", steps[3].Timeout)
	}
	if steps[5].pattern == nil || !steps[5].pattern.MatchString("hello") {
		t.Errorf("assert_pattern: pattern not compiled/usable")
	}
	if steps[6].Text != "after-type" {
		t.Errorf("snapshot: expected label %q, got %q", "after-type", steps[6].Text)
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
