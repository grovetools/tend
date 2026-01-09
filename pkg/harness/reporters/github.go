package reporters

import (
	grovelogging "github.com/mattsolo1/grove-core/logging"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)


var ulog = grovelogging.NewUnifiedLogger("grove-tend.reporters/github")

// GitHubReporter outputs GitHub Actions annotations
type GitHubReporter struct {
	summaryFile string
}

// NewGitHubReporter creates a new GitHub reporter
func NewGitHubReporter() *GitHubReporter {
	return &GitHubReporter{
		summaryFile: os.Getenv("GITHUB_STEP_SUMMARY"),
	}
}

// WriteReport writes GitHub Actions annotations and summary
func (r *GitHubReporter) WriteReport(w io.Writer, results []*harness.Result) error {
	passed := 0
	failed := 0

	// Process results and emit annotations
	for _, result := range results {
		if result.Success {
			passed++
			fmt.Fprintf(w, "::notice title=%s::Test passed in %v\n",
				result.ScenarioName, result.Duration)
		} else {
			failed++

			// Emit error annotation
			message := fmt.Sprintf("Test failed at step: %s", result.FailedStep)
			if result.Error != nil {
				message += fmt.Sprintf(" - %v", result.Error)
			}

			fmt.Fprintf(w, "::error title=%s::%s\n",
				result.ScenarioName, message)
		}
	}

	// Write summary if in GitHub Actions
	if r.summaryFile != "" {
		return r.writeSummary(results, passed, failed)
	}

	return nil
}

// writeSummary writes a markdown summary for GitHub Actions
func (r *GitHubReporter) writeSummary(results []*harness.Result, passed, failed int) error {
	file, err := os.OpenFile(r.summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening summary file: %w", err)
	}
	defer file.Close()

	// Write summary header
	fmt.Fprintln(file, "## Grove Tend Test Results")
	fmt.Fprintln(file)

	// Summary stats
	total := passed + failed
	successRate := 0.0
	if total > 0 {
		successRate = float64(passed) / float64(total) * 100
	}

	// Status icon
	statusIcon := theme.IconSuccess
	if failed > 0 {
		statusIcon = theme.IconError
	}

	fmt.Fprintf(file, "%s **%d/%d tests passed** (%.1f%% success rate)\n\n",
		statusIcon, passed, total, successRate)

	// Results table
	fmt.Fprintln(file, "### Test Results")
	fmt.Fprintln(file)
	fmt.Fprintln(file, "| Test | Status | Duration | Details |")
	fmt.Fprintln(file, "|------|--------|----------|---------|")

	for _, result := range results {
		status := theme.IconSuccess + " Passed"
		details := "-"

		if !result.Success {
			status = theme.IconError + " Failed"
			details = result.FailedStep
			if result.Error != nil {
				// Escape markdown special characters
				errMsg := strings.ReplaceAll(result.Error.Error(), "|", "\\|")
				errMsg = strings.ReplaceAll(errMsg, "\n", " ")
				if len(errMsg) > 50 {
					errMsg = errMsg[:50] + "..."
				}
				details = fmt.Sprintf("%s: %s", result.FailedStep, errMsg)
			}
		}

		fmt.Fprintf(file, "| %s | %s | %v | %s |\n",
			result.ScenarioName, status, result.Duration, details)
	}

	fmt.Fprintln(file)

	// Add run information
	if runID := os.Getenv("GITHUB_RUN_ID"); runID != "" {
		fmt.Fprintf(file, "%s [View full logs](https://github.com/%s/actions/runs/%s)\n",
			theme.IconGithubAction, os.Getenv("GITHUB_REPOSITORY"), runID)
	}

	return nil
}

// EmitGroupStart emits a group start marker
func EmitGroupStart(name string) {
	fmt.Printf("::group::%s\n", name)
}

// EmitGroupEnd emits a group end marker
func EmitGroupEnd() {
 ulog.Info("::endgroup::").Pretty("::endgroup::").PrettyOnly().Emit()
}