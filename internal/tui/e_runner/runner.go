package e_runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/grovetools/tend/pkg/harness/reporters"
)

// ProjectResult holds the outcome of a single project's test run.
type ProjectResult struct {
	ProjectName string
	Success     bool
	ReportPath  string // Path to the generated JSON report
	Error       error
	Duration    time.Duration
	Passed      int
	Failed      int
}

// Event represents an update from the ecosystem test runner.
type Event struct {
	Type       string // "start", "finish", "output"
	Index      int
	Result     *ProjectResult
	Output     string // Full output for "finish" event
	OutputLine string // Single line for "output" event
}

// Run executes project tests in parallel and sends events to a channel.
func Run(ctx context.Context, projects []*ProjectState, numWorkers int) <-chan Event {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	jobsChan := make(chan struct {
		project *ProjectState
		index   int
	}, len(projects))
	eventsChan := make(chan Event, len(projects)*2)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				// Send start event
				eventsChan <- Event{Type: "start", Index: job.index}

				// Create temp file for JSON report
				jsonFile, err := os.CreateTemp("", "tend-eco-report-*.json")
				if err != nil {
					eventsChan <- Event{Type: "finish", Index: job.index, Result: &ProjectResult{Success: false, Error: err, ProjectName: job.project.projectName}}
					continue
				}
				jsonPath := jsonFile.Name()
				jsonFile.Close()
				defer os.Remove(jsonPath)

				// Execute `make <target>` for the project
				makeArgs := fmt.Sprintf("ARGS=--json %s", jsonPath)
				cmd := exec.CommandContext(ctx, "make", job.project.makeTarget)
				cmd.Dir = job.project.projectPath
				cmd.Env = append(os.Environ(), makeArgs)

				// Get pipes for stdout and stderr
				stdoutPipe, _ := cmd.StdoutPipe()
				stderrPipe, _ := cmd.StderrPipe()

				// Buffer to capture full output
				var outputBuf bytes.Buffer

				startTime := time.Now()
				cmdErr := cmd.Start()
				if cmdErr != nil {
					// Send finish event with start error
					eventsChan <- Event{Type: "finish", Index: job.index, Result: &ProjectResult{Success: false, Error: cmdErr, ProjectName: job.project.projectName}}
					continue
				}

				// Stream output from both pipes concurrently
				var streamWg sync.WaitGroup
				streamOutput := func(r io.Reader) {
					defer streamWg.Done()
					scanner := bufio.NewScanner(r)
					for scanner.Scan() {
						line := scanner.Text()
						outputBuf.WriteString(line)
						outputBuf.WriteString("\n")
						eventsChan <- Event{
							Type:       "output",
							Index:      job.index,
							OutputLine: line,
						}
					}
				}

				streamWg.Add(2)
				go streamOutput(stdoutPipe)
				go streamOutput(stderrPipe)

				// Wait for streaming to finish, then for command to exit
				streamWg.Wait()
				cmdErr = cmd.Wait()

				// Parse the JSON report
				var finalResult *ProjectResult
				if reportData, readErr := os.ReadFile(jsonPath); readErr == nil {
					var report reporters.JSONReport
					if json.Unmarshal(reportData, &report) == nil {
						// CRITICAL FIX: Success is determined by zero failures
						success := report.Failed == 0
						finalResult = &ProjectResult{
							ProjectName: job.project.projectName,
							Success:     success,
							ReportPath:  jsonPath,
							Duration:    time.Since(startTime),
							Passed:      report.Passed,
							Failed:      report.Failed,
						}
						if !success {
							finalResult.Error = fmt.Errorf("%d test scenarios failed", report.Failed)
						}
					}
				}

				// If report parsing failed, create a synthetic result from the command error
				if finalResult == nil {
					finalResult = &ProjectResult{
						ProjectName: job.project.projectName,
						Success:     cmdErr == nil,
						Error:       cmdErr,
						Duration:    time.Since(startTime),
						ReportPath:  jsonPath, // still useful for debugging
					}
				}

				// Send finish event
				eventsChan <- Event{
					Type:   "finish",
					Index:  job.index,
					Result: finalResult,
					Output: outputBuf.String(),
				}
			}
		}()
	}

	// Feed jobs to workers
	for i, p := range projects {
		jobsChan <- struct {
			project *ProjectState
			index   int
		}{project: p, index: i}
	}
	close(jobsChan)

	// Close events channel when all workers are done
	go func() {
		wg.Wait()
		close(eventsChan)
	}()

	return eventsChan
}
