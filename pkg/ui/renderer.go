package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// Renderer handles rendering test output with beautiful styling
type Renderer struct {
	writer  io.Writer
	verbose bool
	width   int
}

// NewRenderer creates a new UI renderer
func NewRenderer(writer io.Writer, verbose bool, width int) *Renderer {
	if width <= 0 {
		width = 80
	}
	return &Renderer{
		writer:  writer,
		verbose: verbose,
		width:   width,
	}
}

// RenderScenarioStart renders the start of a scenario
func (r *Renderer) RenderScenarioStart(scenario *harness.Scenario) {
	output := Header(scenario.Name, scenario.Description)

	if len(scenario.Tags) > 0 {
		tags := "Tags: " + strings.Join(scenario.Tags, ", ")
		output += "\n" + theme.DefaultTheme.Muted.Render(tags)
	}

	output += "\n" + strings.Repeat("─", r.width) + "\n"

	r.write(output)
}

// RenderScenarioEnd renders the end of a scenario
func (r *Renderer) RenderScenarioEnd(result *harness.Result) {
	output := "\n" + strings.Repeat("─", r.width) + "\n"
	output += Summary(result.ScenarioName, result.Success, result.Duration, len(result.StepResults))
	
	if !result.Success && result.Error != nil {
		output += "\n" + ErrorBox(result.Error)
	}
	
	r.write(output + "\n")
}

// RenderStepStart renders the start of a step
func (r *Renderer) RenderStepStart(stepNumber int, step harness.Step) {
	if r.verbose {
		output := StepStatus(stepNumber, step.Name, StatusRunning, 0)
		r.write(output + "\n")
	}
}

// RenderStepEnd renders the end of a step
func (r *Renderer) RenderStepEnd(stepNumber int, step harness.Step, result harness.StepResult) {
	var status StepStatusType
	if result.Success {
		status = StatusSuccess
	} else {
		status = StatusError
	}
	
	output := StepStatus(stepNumber, step.Name, status, result.Duration)
	
	if !result.Success && result.Error != nil && r.verbose {
		output += "\n" + ErrorBox(result.Error)
	}
	
	r.write(output + "\n")
}

// RenderProgress renders overall progress
func (r *Renderer) RenderProgress(current, total int) {
	if r.verbose {
		progress := ProgressBar(current, total, 40)
		r.write("\n" + progress + "\n")
	}
}

// RenderInteractivePrompt renders a prompt for interactive mode
func (r *Renderer) RenderInteractivePrompt(stepNumber int, step harness.Step) {
	prompt := fmt.Sprintf("\n%s Ready to execute step %d: %s",
		theme.DefaultTheme.Info.Render(IconInfo),
		stepNumber,
		theme.DefaultTheme.Header.Render(step.Name))

	if step.Description != "" {
		prompt += "\n" + theme.DefaultTheme.Muted.Render(step.Description)
	}

	prompt += "\n" + theme.DefaultTheme.Muted.Render("Press Enter to continue, 's' to skip, 'q' to quit: ")

	r.write(prompt)
}

// RenderCommandOutput renders command output
func (r *Renderer) RenderCommandOutput(stdout, stderr string) {
	if r.verbose {
		if stdout != "" {
			r.write("\n" + InfoBox("Command Output:"))
			r.write(CodeBlock(stdout) + "\n")
		}

		if stderr != "" {
			r.write("\n" + theme.DefaultTheme.Warning.Render("Command Stderr:"))
			r.write(CodeBlock(stderr) + "\n")
		}
	}
}

// RenderError renders a standalone error
func (r *Renderer) RenderError(err error) {
	if err != nil {
		r.write(ErrorBox(err) + "\n")
	}
}

// RenderInfo renders an info message
func (r *Renderer) RenderInfo(message string) {
	r.write(InfoBox(message) + "\n")
}

// RenderSuccess renders a success message
func (r *Renderer) RenderSuccess(message string) {
	r.write(SuccessBox(message) + "\n")
}

// RenderList renders a list of items
func (r *Renderer) RenderList(title string, items []string) {
	output := theme.DefaultTheme.Header.Render(title) + "\n"
	output += List(items)
	r.write(output + "\n")
}

// write outputs text to the writer
func (r *Renderer) write(text string) {
	fmt.Fprint(r.writer, text)
}