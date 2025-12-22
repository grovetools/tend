package harness

// AssertionResult holds the outcome of a single check within a step.
type AssertionResult struct {
	Description string `json:"description"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}
