package harness

import "runtime"

// NewScenario creates a new scenario and captures its definition location.
//
// Using this constructor enables the debug editor feature (--debug flag) to open
// at the scenario definition when tests start. If you use struct literals instead
// (&harness.Scenario{...}), the editor will fall back to opening at the first step's
// location (assuming steps use harness.NewStep).
//
// Example:
//   scenario := harness.NewScenario(
//       "my-test",
//       "Tests something important",
//       []string{"integration"},
//       []harness.Step{
//           harness.NewStep("First step", func(ctx *harness.Context) error {
//               // ...
//           }),
//       },
//   )
//
// Source location capture is automatic via runtime.Caller and is used by:
// - Debug mode editor navigation (--debug flag)
// - Future debugging and reporting features
func NewScenario(name, description string, tags []string, steps []Step) *Scenario {
	_, file, line, _ := runtime.Caller(1)
	return &Scenario{
		Name:         name,
		Description:  description,
		Tags:         tags,
		Steps:        steps,
		LocalOnly:    false, // Default values
		ExplicitOnly: false,
		File:         file,
		Line:         line,
	}
}

// NewScenarioWithOptions creates a new scenario with explicit control over all fields.
// See NewScenario for documentation on source location capture and debug features.
func NewScenarioWithOptions(name, description string, tags []string, steps []Step, localOnly, explicitOnly bool) *Scenario {
	_, file, line, _ := runtime.Caller(1)
	return &Scenario{
		Name:         name,
		Description:  description,
		Tags:         tags,
		Steps:        steps,
		LocalOnly:    localOnly,
		ExplicitOnly: explicitOnly,
		File:         file,
		Line:         line,
	}
}
