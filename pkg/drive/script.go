// Package drive implements the `tend drive` scripted TUI driver: it attaches to
// an already-running app's debug socket (no spawn), replays an ordered list of
// steps, and emits a deterministic evidence bundle.
//
// The package is split into three concerns:
//   - script.go   — the YAML step schema and its strict parser
//   - runner.go   — the step executor and its Driver abstraction over a Session
//   - manifest.go — the schema-versioned evidence bundle writer
//
// v1 is attach-only. Because both attach paths (tuimux daemon socket and a
// treemux debug socket) leave the mux engine unset, the runner talks exclusively
// over the debug-socket plane (GetDebugState / Panel.SendKeys / SendKittyKey)
// rather than the engine-plane Session methods (Type/Capture/WaitStable/…),
// which require a tmux engine that attach mode does not have.
package drive

import (
	"fmt"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// StepKind enumerates the supported step types. Each kind maps 1:1 onto a single
// debug-socket operation; the schema deliberately has no loops, variables, or
// conditionals.
type StepKind string

const (
	// StepType injects keystrokes into the active panel (Session.Panel.SendKeys).
	StepType StepKind = "type"
	// StepKittyKey injects a synthesized CSI-u key event (Session.SendKittyKey).
	StepKittyKey StepKind = "kittykey"
	// StepWait blocks until the rendered debug state stabilizes.
	StepWait StepKind = "wait"
	// StepAssertContains asserts the rendered debug state contains a substring.
	StepAssertContains StepKind = "assert_contains"
	// StepAssertPattern asserts the rendered debug state matches a regexp.
	StepAssertPattern StepKind = "assert_pattern"
	// StepAssertStructural asserts fields of the parsed structural debug state
	// (active panel, rail, focus, panel types) against one snapshot.
	StepAssertStructural StepKind = "assert_structural"
	// StepSnapshot writes a labelled visual + structural snapshot to the bundle.
	StepSnapshot StepKind = "snapshot"
)

// Step is a single parsed instruction. Exactly one kind is set per step; the
// other fields carry that kind's argument.
type Step struct {
	Kind StepKind

	// Text carries the argument for type / assert_contains / assert_pattern /
	// snapshot steps (respectively: keys, substring, pattern source, label).
	Text string

	// Timeout is the optional per-step override for wait steps. Zero means use
	// the runner's default timeout.
	Timeout time.Duration

	// Kitty carries the argument for kittykey steps.
	Kitty KittyKey

	// Structural carries the argument for assert_structural steps.
	Structural StructuralAssert

	// pattern is the compiled form of an assert_pattern step's Text.
	pattern *regexp.Regexp
}

// KittyKey is the structured argument of a kittykey step. It mirrors
// Session.SendKittyKey(panelID, keycode, mods). Panel defaults to the active
// panel when empty; Mods defaults to 0.
type KittyKey struct {
	Panel   string
	Keycode int
	Mods    int
}

// StructuralAssert is the structured argument of an assert_structural step. It
// is a one-shot assertion over a single Driver.State() snapshot: all fields are
// optional (at least one must be present), and every present field must pass.
type StructuralAssert struct {
	// ActivePanel asserts snap.ActivePanelID equals this value.
	ActivePanel string
	// RailActive asserts some rail item with IsActive has this ID or Label.
	RailActive string
	// Focused asserts the named panel exists and is focused.
	Focused string
	// FocusedCount asserts the number of focused panels. Nil means not
	// asserted (zero is a legal expectation).
	FocusedCount *int
	// PanelType asserts, per entry, that the named panel exists with this Type.
	PanelType map[string]string
}

// knownStepKeys is the closed set of recognized top-level step keys. Anything
// outside this set is a hard parse error (see decision: never half-execute a
// typo'd script).
var knownStepKeys = map[string]bool{
	string(StepType):             true,
	string(StepKittyKey):         true,
	string(StepWait):             true,
	string(StepAssertContains):   true,
	string(StepAssertPattern):    true,
	string(StepAssertStructural): true,
	string(StepSnapshot):         true,
}

// ParseScript parses a YAML step list into an ordered slice of Steps. It fails
// on the first structural problem (unknown key, wrong value shape, bad regexp,
// bad duration) before any step is returned, so a malformed script never runs
// partially. An empty document yields zero steps and no error.
func ParseScript(data []byte) ([]Step, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse script yaml: %w", err)
	}

	// Empty file (nothing decoded) → no steps.
	if root.Kind == 0 || len(root.Content) == 0 {
		return nil, nil
	}

	seq := root.Content[0]
	if seq.Kind == yaml.ScalarNode && seq.Tag == "!!null" {
		return nil, nil
	}
	if seq.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("script must be a YAML list of steps, got %s", nodeKindName(seq.Kind))
	}

	steps := make([]Step, 0, len(seq.Content))
	for i, item := range seq.Content {
		step, err := parseStep(item)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

// parseStep parses a single mapping node holding exactly one recognized key.
func parseStep(node *yaml.Node) (Step, error) {
	if node.Kind != yaml.MappingNode {
		return Step{}, fmt.Errorf("each step must be a mapping with one key, got %s", nodeKindName(node.Kind))
	}
	if len(node.Content) != 2 {
		return Step{}, fmt.Errorf("each step must have exactly one key (found %d)", len(node.Content)/2)
	}

	keyNode, valNode := node.Content[0], node.Content[1]
	key := keyNode.Value
	if !knownStepKeys[key] {
		return Step{}, fmt.Errorf("unknown step key %q", key)
	}

	switch StepKind(key) {
	case StepType:
		s, err := scalarString(valNode, key)
		if err != nil {
			return Step{}, err
		}
		return Step{Kind: StepType, Text: s}, nil

	case StepAssertContains:
		s, err := scalarString(valNode, key)
		if err != nil {
			return Step{}, err
		}
		return Step{Kind: StepAssertContains, Text: s}, nil

	case StepSnapshot:
		s, err := scalarString(valNode, key)
		if err != nil {
			return Step{}, err
		}
		if s == "" {
			return Step{}, fmt.Errorf("snapshot requires a non-empty label")
		}
		return Step{Kind: StepSnapshot, Text: s}, nil

	case StepAssertPattern:
		s, err := scalarString(valNode, key)
		if err != nil {
			return Step{}, err
		}
		re, err := regexp.Compile(s)
		if err != nil {
			return Step{}, fmt.Errorf("assert_pattern: invalid regexp %q: %w", s, err)
		}
		return Step{Kind: StepAssertPattern, Text: s, pattern: re}, nil

	case StepAssertStructural:
		return parseAssertStructural(valNode)

	case StepWait:
		return parseWait(valNode)

	case StepKittyKey:
		return parseKittyKey(valNode)

	default:
		// Unreachable: knownStepKeys gates the switch.
		return Step{}, fmt.Errorf("unhandled step key %q", key)
	}
}

// parseWait accepts `wait:`, `wait: {}`, or `wait: {timeout: 5s}`.
func parseWait(val *yaml.Node) (Step, error) {
	step := Step{Kind: StepWait}
	// `wait:` with no value decodes as a null scalar → empty wait.
	if val.Kind == yaml.ScalarNode && (val.Tag == "!!null" || val.Value == "") {
		return step, nil
	}
	if val.Kind != yaml.MappingNode {
		return Step{}, fmt.Errorf("wait must be empty or a mapping, got %s", nodeKindName(val.Kind))
	}
	for i := 0; i < len(val.Content); i += 2 {
		k := val.Content[i].Value
		v := val.Content[i+1]
		switch k {
		case "timeout":
			s, err := scalarString(v, "timeout")
			if err != nil {
				return Step{}, err
			}
			d, err := time.ParseDuration(s)
			if err != nil {
				return Step{}, fmt.Errorf("wait: invalid timeout %q: %w", s, err)
			}
			step.Timeout = d
		default:
			return Step{}, fmt.Errorf("wait: unknown field %q", k)
		}
	}
	return step, nil
}

// parseKittyKey accepts `kittykey: {panel: <id>, keycode: <int>, mods: <int>}`.
// panel and mods are optional; keycode is required.
func parseKittyKey(val *yaml.Node) (Step, error) {
	if val.Kind != yaml.MappingNode {
		return Step{}, fmt.Errorf("kittykey must be a mapping {panel, keycode, mods}, got %s", nodeKindName(val.Kind))
	}
	var kk KittyKey
	seenKeycode := false
	for i := 0; i < len(val.Content); i += 2 {
		k := val.Content[i].Value
		v := val.Content[i+1]
		switch k {
		case "panel":
			s, err := scalarString(v, "panel")
			if err != nil {
				return Step{}, err
			}
			kk.Panel = s
		case "keycode":
			n, err := scalarInt(v, "keycode")
			if err != nil {
				return Step{}, err
			}
			kk.Keycode = n
			seenKeycode = true
		case "mods":
			n, err := scalarInt(v, "mods")
			if err != nil {
				return Step{}, err
			}
			kk.Mods = n
		default:
			return Step{}, fmt.Errorf("kittykey: unknown field %q", k)
		}
	}
	if !seenKeycode {
		return Step{}, fmt.Errorf("kittykey: keycode is required")
	}
	return Step{Kind: StepKittyKey, Kitty: kk}, nil
}

// parseAssertStructural accepts `assert_structural: {active_panel, rail_active,
// focused, focused_count, panel_type}`. Every field is optional, but at least
// one must be present — an empty assertion would silently pass.
func parseAssertStructural(val *yaml.Node) (Step, error) {
	if val.Kind != yaml.MappingNode {
		return Step{}, fmt.Errorf("assert_structural must be a mapping {active_panel, rail_active, focused, focused_count, panel_type}, got %s", nodeKindName(val.Kind))
	}
	var sa StructuralAssert
	fields := 0
	for i := 0; i < len(val.Content); i += 2 {
		k := val.Content[i].Value
		v := val.Content[i+1]
		switch k {
		case "active_panel":
			s, err := scalarString(v, "active_panel")
			if err != nil {
				return Step{}, err
			}
			if s == "" {
				return Step{}, fmt.Errorf("assert_structural: active_panel must be non-empty")
			}
			sa.ActivePanel = s
		case "rail_active":
			s, err := scalarString(v, "rail_active")
			if err != nil {
				return Step{}, err
			}
			if s == "" {
				return Step{}, fmt.Errorf("assert_structural: rail_active must be non-empty")
			}
			sa.RailActive = s
		case "focused":
			s, err := scalarString(v, "focused")
			if err != nil {
				return Step{}, err
			}
			if s == "" {
				return Step{}, fmt.Errorf("assert_structural: focused must be non-empty")
			}
			sa.Focused = s
		case "focused_count":
			n, err := scalarInt(v, "focused_count")
			if err != nil {
				return Step{}, err
			}
			if n < 0 {
				return Step{}, fmt.Errorf("assert_structural: focused_count must be >= 0")
			}
			sa.FocusedCount = &n
		case "panel_type":
			if v.Kind != yaml.MappingNode {
				return Step{}, fmt.Errorf("assert_structural: panel_type must be a mapping of panel ID to type, got %s", nodeKindName(v.Kind))
			}
			if len(v.Content) == 0 {
				return Step{}, fmt.Errorf("assert_structural: panel_type must have at least one entry")
			}
			sa.PanelType = make(map[string]string, len(v.Content)/2)
			for j := 0; j < len(v.Content); j += 2 {
				id := v.Content[j].Value
				typ, err := scalarString(v.Content[j+1], "panel_type."+id)
				if err != nil {
					return Step{}, err
				}
				sa.PanelType[id] = typ
			}
		default:
			return Step{}, fmt.Errorf("assert_structural: unknown field %q", k)
		}
		fields++
	}
	if fields == 0 {
		return Step{}, fmt.Errorf("assert_structural requires at least one field")
	}
	return Step{Kind: StepAssertStructural, Structural: sa}, nil
}

// scalarString decodes a scalar node into a string, rejecting non-scalar shapes.
func scalarString(node *yaml.Node, field string) (string, error) {
	if node.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("%s must be a scalar value, got %s", field, nodeKindName(node.Kind))
	}
	var s string
	if err := node.Decode(&s); err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	return s, nil
}

// scalarInt decodes a scalar node into an int, rejecting non-scalar shapes.
func scalarInt(node *yaml.Node, field string) (int, error) {
	if node.Kind != yaml.ScalarNode {
		return 0, fmt.Errorf("%s must be an integer, got %s", field, nodeKindName(node.Kind))
	}
	var n int
	if err := node.Decode(&n); err != nil {
		return 0, fmt.Errorf("%s: must be an integer: %w", field, err)
	}
	return n, nil
}

// nodeKindName renders a yaml.Kind for error messages.
func nodeKindName(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "list"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return "unknown"
	}
}
