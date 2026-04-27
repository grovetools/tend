package recorder

import (
	"strings"
	"testing"
	"time"
)

func TestRecorderWithSimpleCommand(t *testing.T) {
	// This test will fail when run in CI/non-TTY environments
	// but serves as a manual test
	t.Skip("Skipping test that requires TTY")

	rec := New()

	// Record a simple echo command
	frames, err := rec.Run([]string{"sh", "-c", "echo hello && sleep 0.5 && echo world"})
	if err != nil {
		t.Fatalf("Recording failed: %v", err)
	}

	if len(frames) == 0 {
		t.Fatal("Expected at least one frame")
	}

	// Check that we captured some output
	var allOutput strings.Builder
	for _, frame := range frames {
		allOutput.WriteString(frame.Output)
	}

	output := allOutput.String()
	if !strings.Contains(output, "hello") || !strings.Contains(output, "world") {
		t.Errorf("Expected output to contain 'hello' and 'world', got: %q", output)
	}
}

func TestFrameStructure(t *testing.T) {
	frame := Frame{
		Timestamp: 100 * time.Millisecond,
		Input:     "test input",
		Output:    "test output",
	}

	if frame.Input != "test input" {
		t.Errorf("Expected input 'test input', got %q", frame.Input)
	}

	if frame.Output != "test output" {
		t.Errorf("Expected output 'test output', got %q", frame.Output)
	}

	if frame.Timestamp != 100*time.Millisecond {
		t.Errorf("Expected timestamp 100ms, got %v", frame.Timestamp)
	}
}
