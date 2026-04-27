package harness

import "fmt"

// StepError wraps an error with step context
type StepError struct {
	StepName string
	Err      error
}

func (e *StepError) Error() string {
	return fmt.Sprintf("step '%s' failed: %v", e.StepName, e.Err)
}

func (e *StepError) Unwrap() error {
	return e.Err
}

// ScenarioError indicates a scenario-level failure
type ScenarioError struct {
	ScenarioName string
	Err          error
}

func (e *ScenarioError) Error() string {
	return fmt.Sprintf("scenario '%s' failed: %v", e.ScenarioName, e.Err)
}

func (e *ScenarioError) Unwrap() error {
	return e.Err
}
