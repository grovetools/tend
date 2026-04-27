package command

import (
	"fmt"
	"testing"
	"time"
)

func TestProcessStart(t *testing.T) {
	// Test starting a simple command in the background
	cmd := New("echo", "hello from background")
	process, err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Check that we have a PID
	if process.PID == 0 {
		t.Error("Expected non-zero PID")
	}

	// Wait for completion
	result := process.Wait(5 * time.Second)
	if result.Error != nil {
		t.Fatalf("Process failed: %v", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "hello from background\n" {
		t.Errorf("Expected 'hello from background\\n', got %q", result.Stdout)
	}
}

func TestProcessTimeout(t *testing.T) {
	// Test a long-running command that should timeout
	cmd := New("sleep", "10")
	process, err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Wait with a short timeout
	result := process.Wait(100 * time.Millisecond)
	if result.Error == nil {
		t.Error("Expected timeout error")
	}

	if result.ExitCode != -1 {
		t.Errorf("Expected exit code -1 for timeout, got %d", result.ExitCode)
	}
}

func TestProcessKill(t *testing.T) {
	// Test killing a process
	cmd := New("sleep", "10")
	process, err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Kill the process
	err = process.Kill()
	if err != nil {
		t.Fatalf("Failed to kill process: %v", err)
	}

	// Wait should complete quickly with an error
	result := process.Wait(1 * time.Second)
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for killed process")
	}
}

func TestProcessMultipleBackgroundCommands(t *testing.T) {
	// Test running multiple commands in the background
	processes := make([]*Process, 3)

	for i := 0; i < 3; i++ {
		cmd := New("echo", fmt.Sprintf("process %d", i))
		process, err := cmd.Start()
		if err != nil {
			t.Fatalf("Failed to start process %d: %v", i, err)
		}
		processes[i] = process
	}

	// Wait for all processes
	for i, process := range processes {
		result := process.Wait(5 * time.Second)
		if result.Error != nil {
			t.Errorf("Process %d failed: %v", i, result.Error)
		}
		expected := fmt.Sprintf("process %d\n", i)
		if result.Stdout != expected {
			t.Errorf("Process %d: expected %q, got %q", i, expected, result.Stdout)
		}
	}
}
