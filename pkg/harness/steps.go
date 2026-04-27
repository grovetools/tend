package harness

import (
	"fmt"
	"runtime"
	"time"
)

// StepFunc is a convenience type for step functions
type StepFunc func(ctx *Context) error

// NewStep creates a new step with the given name and function.
//
// This constructor automatically captures the source location where the step is defined,
// which enables the debug editor feature (--debug flag) to navigate to each step as the
// test progresses.
//
// Using NewStep is recommended over Step{} literals for better debugging support.
func NewStep(name string, fn StepFunc) Step {
	_, file, line, _ := runtime.Caller(1)
	return Step{
		Name: name,
		Func: fn,
		File: file,
		Line: line,
	}
}

// SequentialSteps creates a step that runs multiple sub-steps in sequence
func SequentialSteps(name string, steps ...Step) Step {
	return Step{
		Name: name,
		Func: func(ctx *Context) error {
			for _, step := range steps {
				if err := step.Func(ctx); err != nil {
					return fmt.Errorf("%s: %w", step.Name, err)
				}
			}
			return nil
		},
	}
}

// RetryStep creates a step that retries on failure
func RetryStep(name string, maxAttempts int, delay time.Duration, fn StepFunc) Step {
	return Step{
		Name: name,
		Func: func(ctx *Context) error {
			var lastErr error
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				if err := fn(ctx); err == nil {
					return nil
				} else {
					lastErr = err
					if attempt < maxAttempts {
						time.Sleep(delay)
					}
				}
			}
			return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
		},
	}
}

// ConditionalStep creates a step that only runs if a condition is met
func ConditionalStep(name string, condition func(*Context) bool, fn StepFunc) Step {
	return Step{
		Name: name,
		Func: func(ctx *Context) error {
			if condition(ctx) {
				return fn(ctx)
			}
			return nil
		},
	}
}

// DelayStep creates a step that waits for a duration
func DelayStep(name string, duration time.Duration) Step {
	return Step{
		Name: name,
		Func: func(ctx *Context) error {
			time.Sleep(duration)
			return nil
		},
	}
}
