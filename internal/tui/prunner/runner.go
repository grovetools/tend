package prunner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/harness/reporters"
)

// Event represents an update from the test runner.
type Event struct {
	Type   string // "start", "finish"
	Index  int
	Result *harness.Result
	Output string
}

// Run executes scenarios in parallel and sends events to a channel.
func Run(ctx context.Context, scenarios []*harness.Scenario, projectRoot string, numWorkers int) <-chan Event {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	jobsChan := make(chan struct {
		scenario *harness.Scenario
		index    int
	}, len(scenarios))
	eventsChan := make(chan Event, len(scenarios)*2)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				// Send start event
				eventsChan <- Event{Type: "start", Index: job.index}

				// Create temp file for JSON report
				jsonFile, err := os.CreateTemp("", "tend-report-*.json")
				if err != nil {
					eventsChan <- Event{Type: "finish", Index: job.index, Result: &harness.Result{Success: false, Error: err}}
					continue
				}
				jsonPath := jsonFile.Name()
				jsonFile.Close()
				defer os.Remove(jsonPath)

				// Get path to current executable
				executable, err := os.Executable()
				if err != nil {
					eventsChan <- Event{Type: "finish", Index: job.index, Result: &harness.Result{Success: false, Error: err}}
					continue
				}

				// Execute tend run for a single scenario
				cmd := exec.CommandContext(ctx, executable, "run", job.scenario.Name, "--json", jsonPath, "--no-cleanup")
				cmd.Dir = projectRoot

				startTime := time.Now()
				output, cmdErr := cmd.CombinedOutput()

				// Parse the JSON report and convert to harness.Result
				var finalResult *harness.Result
				if reportData, readErr := os.ReadFile(jsonPath); readErr == nil {
					var report reporters.JSONReport
					if json.Unmarshal(reportData, &report) == nil && len(report.Results) > 0 {
						jsonResult := report.Results[0]
						// Convert JSONTestResult to harness.Result
						var resultErr error
						if jsonResult.Error != "" {
							resultErr = fmt.Errorf("%s", jsonResult.Error)
						}
						finalResult = &harness.Result{
							ScenarioName: jsonResult.Name,
							Success:      jsonResult.Success,
							FailedStep:   jsonResult.FailedStep,
							Error:        resultErr,
							StartTime:    jsonResult.StartTime,
							EndTime:      jsonResult.EndTime,
							Duration:     jsonResult.EndTime.Sub(jsonResult.StartTime),
						}
					}
				}

				// If we couldn't parse the report, create a synthetic result
				if finalResult == nil {
					finalResult = &harness.Result{
						ScenarioName: job.scenario.Name,
						Success:      cmdErr == nil,
						Error:        cmdErr,
						StartTime:    startTime,
						EndTime:      time.Now(),
						Duration:     time.Since(startTime),
					}
				}

				// Send finish event
				eventsChan <- Event{
					Type:   "finish",
					Index:  job.index,
					Result: finalResult,
					Output: string(output),
				}
			}
		}()
	}

	// Feed jobs to workers
	for i, s := range scenarios {
		jobsChan <- struct {
			scenario *harness.Scenario
			index    int
		}{scenario: s, index: i}
	}
	close(jobsChan)

	// Close events channel when all workers are done
	go func() {
		wg.Wait()
		close(eventsChan)
	}()

	return eventsChan
}
