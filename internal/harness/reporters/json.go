package reporters

import (
	"encoding/json"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/grovepm/grove-tend/internal/harness"
)

// JSONReport represents the overall test report
type JSONReport struct {
	Timestamp   time.Time         `json:"timestamp"`
	Duration    string            `json:"duration"`
	TotalTests  int               `json:"total_tests"`
	Passed      int               `json:"passed"`
	Failed      int               `json:"failed"`
	SuccessRate float64           `json:"success_rate"`
	Results     []JSONTestResult  `json:"results"`
	Environment JSONEnvironment   `json:"environment"`
}

// JSONTestResult represents a single test result
type JSONTestResult struct {
	Name       string           `json:"name"`
	Success    bool             `json:"success"`
	Duration   string           `json:"duration"`
	StartTime  time.Time        `json:"start_time"`
	EndTime    time.Time        `json:"end_time"`
	FailedStep string           `json:"failed_step,omitempty"`
	Error      string           `json:"error,omitempty"`
	Steps      []JSONTestStep   `json:"steps,omitempty"`
}

// JSONTestStep represents a test step
type JSONTestStep struct {
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// JSONEnvironment represents test environment info
type JSONEnvironment struct {
	GoVersion    string            `json:"go_version"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	GroveBinary  string            `json:"grove_binary"`
	Variables    map[string]string `json:"variables,omitempty"`
}

// JSONReporter generates JSON reports
type JSONReporter struct {
	pretty      bool
	includeSteps bool
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter(pretty, includeSteps bool) *JSONReporter {
	return &JSONReporter{
		pretty:      pretty,
		includeSteps: includeSteps,
	}
}

// WriteReport writes JSON report for test results
func (r *JSONReporter) WriteReport(w io.Writer, results []*harness.Result) error {
	report := JSONReport{
		Timestamp:  time.Now(),
		TotalTests: len(results),
		Environment: JSONEnvironment{
			GoVersion:   runtime.Version(),
			OS:          runtime.GOOS,
			Arch:        runtime.GOARCH,
			GroveBinary: os.Getenv("GROVE_BINARY"),
			Variables: map[string]string{
				"CI":               os.Getenv("CI"),
				"GITHUB_ACTIONS":   os.Getenv("GITHUB_ACTIONS"),
				"GITHUB_RUN_ID":    os.Getenv("GITHUB_RUN_ID"),
				"GITHUB_SHA":       os.Getenv("GITHUB_SHA"),
			},
		},
	}

	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Success {
			report.Passed++
		} else {
			report.Failed++
		}

		totalDuration += result.Duration

		jsonResult := JSONTestResult{
			Name:      result.ScenarioName,
			Success:   result.Success,
			Duration:  result.Duration.String(),
			StartTime: result.StartTime,
			EndTime:   result.EndTime,
		}

		if !result.Success {
			jsonResult.FailedStep = result.FailedStep
			if result.Error != nil {
				jsonResult.Error = result.Error.Error()
			}
		}

		// Add step details if requested
		if r.includeSteps && result.StepResults != nil {
			for _, step := range result.StepResults {
				jsonStep := JSONTestStep{
					Name:      step.Name,
					StartTime: step.StartTime,
					EndTime:   step.EndTime,
					Duration:  step.Duration.String(),
					Success:   step.Success,
				}
				if step.Error != nil {
					jsonStep.Error = step.Error.Error()
				}
				jsonResult.Steps = append(jsonResult.Steps, jsonStep)
			}
		}

		report.Results = append(report.Results, jsonResult)
	}

	report.Duration = totalDuration.String()
	if report.TotalTests > 0 {
		report.SuccessRate = float64(report.Passed) / float64(report.TotalTests) * 100
	}

	// Write JSON
	encoder := json.NewEncoder(w)
	if r.pretty {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(report)
}