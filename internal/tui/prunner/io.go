package prunner

import (
	"context"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// startRunCmd starts the parallel test execution.
func startRunCmd(scenarios []*ScenarioState, projectRoot string, numJobs int) tea.Cmd {
	return func() tea.Msg {
		var s []*harness.Scenario
		for _, state := range scenarios {
			s = append(s, state.scenario)
		}
		// Use the specified number of jobs, or default to half the available CPUs
		numWorkers := numJobs
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU() / 2
			if numWorkers < 1 {
				numWorkers = 1
			}
		}
		// Debug: print the number of workers
		// fmt.Fprintf(os.Stderr, "DEBUG: Starting parallel runner with %d workers (numJobs=%d)\n", numWorkers, numJobs)
		eventsChan := Run(context.Background(), s, projectRoot, numWorkers)
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
