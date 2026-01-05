package e_runner

import (
	"context"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// startRunCmd starts the parallel ecosystem test execution.
func startRunCmd(projects []*ProjectState, numJobs int) tea.Cmd {
	return func() tea.Msg {
		// Use the specified number of jobs, or default to half the available CPUs
		numWorkers := numJobs
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU() / 2
			if numWorkers < 1 {
				numWorkers = 1
			}
		}
		eventsChan := Run(context.Background(), projects, numWorkers)
		return eventsChan
	}
}

// waitForEventCmd waits for the next event from the runner.
func waitForEventCmd(eventsChan <-chan Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-eventsChan
		if !ok {
			return nil // Channel closed
		}
		return event
	}
}
