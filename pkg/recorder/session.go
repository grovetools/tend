package recorder

import "time"

// Frame represents a single cause-and-effect step in the recorded session.
// It links a user's input to the resulting raw ANSI output from the application.
type Frame struct {
	Timestamp time.Duration `json:"timestamp"` // Time since session start.
	Input     string        `json:"input"`     // The user's input (keystrokes).
	Output    string        `json:"output"`    // The raw ANSI output from the application in response to the input.
}
