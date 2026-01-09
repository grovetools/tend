package harness

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	grovelogging "github.com/mattsolo1/grove-core/logging"
	"github.com/mattsolo1/grove-core/tui/theme"
)

var ulog = grovelogging.NewUnifiedLogger("grove-tend.harness.ui")

// UI handles all user interface output (basic implementation)
type UI struct {
	interactive bool
	verbose     bool
	veryVerbose bool
	reader      *bufio.Reader
	
	// Container monitoring
	monitor          *ContainerMonitor
	monitorEnabled   bool
	lastContainerIDs map[string]bool // Track container IDs to detect changes
	testID           string          // ID of current test for filtering containers
	baselineContainers map[string]bool // Containers that existed before test started
	mu               sync.Mutex
}

// NewUI creates a new UI instance
func NewUI(interactive, verbose, veryVerbose bool) *UI {
	return &UI{
		interactive:        interactive,
		verbose:            verbose,
		veryVerbose:        veryVerbose,
		reader:             bufio.NewReader(os.Stdin),
		lastContainerIDs:   make(map[string]bool),
		baselineContainers: make(map[string]bool),
	}
}

// ScenarioStart displays the start of a scenario
func (ui *UI) ScenarioStart(name, description string) {
	ctx := context.Background()
	prettyMsg := fmt.Sprintf("\n%s Scenario: %s", theme.IconTestTube, name)
	if description != "" {
		prettyMsg += fmt.Sprintf("\n   %s", description)
	}
	prettyMsg += "\n" + strings.Repeat("-", 60)

	ulog.Info("Scenario started").
		Field("name", name).
		Field("description", description).
		Pretty(prettyMsg).
		Log(ctx)
}

// ScenarioSuccess displays scenario completion
func (ui *UI) ScenarioSuccess(name string, duration time.Duration) {
	ctx := context.Background()
	ulog.Success("Scenario completed").
		Field("name", name).
		Field("duration", duration).
		Pretty(strings.Repeat("-", 60) + fmt.Sprintf("\n%s Scenario completed successfully in %v\n", theme.IconSuccess, duration)).
		Log(ctx)
}

// ScenarioFailed displays scenario failure
func (ui *UI) ScenarioFailed(name string, err error) {
	ctx := context.Background()
	prettyMsg := strings.Repeat("-", 60) + fmt.Sprintf("\n%s Scenario failed: %s", theme.IconError, name)
	if err != nil {
		prettyMsg += fmt.Sprintf("\nError: %v\n", err)
	}

	ulog.Error("Scenario failed").
		Field("name", name).
		Err(err).
		Pretty(prettyMsg).
		Log(ctx)
}

// PhaseStart displays the start of a test phase (e.g., Setup, Test, Teardown).
func (ui *UI) PhaseStart(name string) {
	ctx := context.Background()
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")). // Cyan
		MarginTop(1).
		MarginBottom(1)

	ulog.Info("Phase started").
		Field("phase", name).
		Pretty(style.Render(fmt.Sprintf("--- %s Phase ---", name))).
		Log(ctx)
}

// StepStart displays the start of a step
func (ui *UI) StepStart(current, total int, name string) {
	ctx := context.Background()
	ulog.Progress("Step started").
		Field("current", current).
		Field("total", total).
		Field("name", name).
		Pretty(fmt.Sprintf("\n[%d/%d] %s", current, total, name)).
		Log(ctx)
}

// StepSuccess displays step completion
func (ui *UI) StepSuccess(stepResult StepResult) {
	if ui.verbose {
		ctx := context.Background()
		prettyMsg := fmt.Sprintf("%s %s (Completed in %v)", theme.IconSuccess, stepResult.Name, stepResult.Duration)
		// Print successful assertions
		for _, assertion := range stepResult.Assertions {
			if assertion.Success {
				prettyMsg += fmt.Sprintf("\n  %s %s", theme.IconSuccess, assertion.Description)
			}
		}

		ulog.Success("Step completed").
			Field("name", stepResult.Name).
			Field("duration", stepResult.Duration).
			Field("assertions_count", len(stepResult.Assertions)).
			Pretty(prettyMsg).
			Log(ctx)
	}
}

// StepFailed displays step failure
func (ui *UI) StepFailed(stepResult StepResult) {
	ctx := context.Background()
	prettyMsg := fmt.Sprintf("%s %s (Failed after %v)", theme.IconError, stepResult.Name, stepResult.Duration)

	// Print successful assertions before the failure
	for _, assertion := range stepResult.Assertions {
		if assertion.Success {
			prettyMsg += fmt.Sprintf("\n  %s %s", theme.IconSuccess, assertion.Description)
		}
	}

	// Print the failure details
	if stepResult.Error != nil {
		prettyMsg += fmt.Sprintf("\n  Error: %v", stepResult.Error)
	}

	ulog.Error("Step failed").
		Field("name", stepResult.Name).
		Field("duration", stepResult.Duration).
		Err(stepResult.Error).
		Pretty(prettyMsg).
		Log(ctx)
}

// WaitForUser prompts the user to continue
// Returns a string indicating the user's choice: "continue", "quit", or "attach"
func (ui *UI) WaitForUser() string {
	if !ui.interactive {
		return "continue"
	}

	ctx := context.Background()
	ulog.Info("Waiting for user input").
		Pretty(theme.IconSelect + " Press ENTER to continue, 'a' to attach, 'q' to quit: ").
		PrettyOnly().
		Log(ctx)

	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return "quit"
	}

	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "a", "attach":
		return "attach"
	case "q", "quit":
		return "quit"
	default:
		return "continue"
	}
}

// RenderTUICapture displays the captured content of a TUI session
func (ui *UI) RenderTUICapture(content string) {
	if ui.verbose {
		ctx := context.Background()
		ulog.Info("TUI capture").
			Pretty(content).
			PrettyOnly().
			Log(ctx)
	}
}

// Cleanup displays cleanup message
func (ui *UI) Cleanup() {
	if ui.verbose {
		ctx := context.Background()
		ulog.Info("Cleaning up").
			Pretty(theme.IconFolderRemove + " Cleaning up temporary files...").
			Log(ctx)
	}
}

// SetTestID sets the test ID for container filtering
func (ui *UI) SetTestID(testID string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.testID = testID
}

// EnableMonitoring starts container monitoring
func (ui *UI) EnableMonitoring(filter string) {
	ui.mu.Lock()
	
	if ui.monitor != nil {
		ui.monitor.Stop()
	}
	
	ui.monitor = NewContainerMonitor(filter, 2*time.Second)
	ui.monitorEnabled = true
	
	// Get baseline containers before test starts
	firstUpdate := true
	ui.monitor.Start(func(containers []ContainerInfo) {
		ui.mu.Lock()
		defer ui.mu.Unlock()
		
		if firstUpdate {
			// First update - capture baseline
			for _, c := range containers {
				if strings.Contains(c.Names, "grove") && strings.Contains(c.Names, "agent") {
					ui.baselineContainers[c.Names] = true
				}
			}
			firstUpdate = false
		} else {
			// Subsequent updates
			ui.handleContainerUpdate(containers)
		}
	})
	
	ui.mu.Unlock()
	
	// Give it a moment to capture baseline
	time.Sleep(100 * time.Millisecond)
}

// DisableMonitoring stops container monitoring
func (ui *UI) DisableMonitoring() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	
	if ui.monitor != nil {
		ui.monitor.Stop()
		ui.monitor = nil
		ui.monitorEnabled = false
	}
}

// handleContainerUpdate processes container updates and prints changes
func (ui *UI) handleContainerUpdate(containers []ContainerInfo) {
	// Note: caller already holds the lock
	
	if !ui.monitorEnabled {
		return
	}
	
	// Filter only grove agent containers created by this test
	var agentContainers []ContainerInfo
	currentIDs := make(map[string]bool)
	
	for _, c := range containers {
		if strings.Contains(c.Names, "grove") && strings.Contains(c.Names, "agent") {
			// Skip containers that existed before the test started
			if !ui.baselineContainers[c.Names] {
				agentContainers = append(agentContainers, c)
				currentIDs[c.Names] = true
			}
		}
	}
	
	// Check if there are changes
	changed := len(currentIDs) != len(ui.lastContainerIDs)
	if !changed {
		for id := range currentIDs {
			if !ui.lastContainerIDs[id] {
				changed = true
				break
			}
		}
	}
	if !changed {
		for id := range ui.lastContainerIDs {
			if !currentIDs[id] {
				changed = true
				break
			}
		}
	}
	
	// Print table only if there are changes
	if changed {
		ui.lastContainerIDs = currentIDs
		ui.printDockerTable(agentContainers)
	}
}

// printDockerTable prints the Docker container table inline
func (ui *UI) printDockerTable(containers []ContainerInfo) {
	ctx := context.Background()
	prettyMsg := fmt.Sprintf("\n%s Docker Container Update:\n", theme.IconSync)

	if len(containers) == 0 {
		prettyMsg += "  No agent containers running"
	} else {
		// Table header
		prettyMsg += fmt.Sprintf("  %-40s %-30s %-20s\n", "NAMES", "IMAGE", "CREATED")
		prettyMsg += fmt.Sprintf("  %s\n", strings.Repeat("-", 92))

		// Container rows
		for _, c := range containers {
			// Truncate long names
			name := c.Names
			if len(name) > 40 {
				name = name[:37] + "..."
			}

			image := c.Image
			if len(image) > 30 {
				image = image[:27] + "..."
			}

			// Format created time to be shorter
			created := c.Created
			if len(created) > 20 {
				// Try to extract just the relative time part
				parts := strings.Fields(created)
				if len(parts) >= 2 {
					created = parts[0] + " " + parts[1]
				}
			}
			if len(created) > 20 {
				created = created[:17] + "..."
			}

			prettyMsg += fmt.Sprintf("  %-40s %-30s %-20s\n", name, image, created)
		}
	}

	ulog.Info("Docker container update").
		Field("container_count", len(containers)).
		Pretty(prettyMsg).
		Log(ctx)
}

// ShowDockerStatus displays current Docker container status
func (ui *UI) ShowDockerStatus() {
	ctx := context.Background()
	containers, err := GetContainerSnapshot("name=grove")
	if err != nil {
		return
	}

	if len(containers) == 0 {
		return
	}

	prettyMsg := fmt.Sprintf("\n%s Docker Containers (grove-related):\n", theme.IconFolder)
	prettyMsg += fmt.Sprintf("%-30s %-20s %s\n", "IMAGE", "CREATED", "NAMES")
	prettyMsg += strings.Repeat("-", 70) + "\n"

	for _, c := range containers {
		if strings.Contains(c.Names, "grove") {
			prettyMsg += fmt.Sprintf("%-30s %-20s %s\n", c.Image, c.Created, c.Names)
		}
	}

	ulog.Info("Docker status").
		Field("container_count", len(containers)).
		Pretty(prettyMsg).
		Log(ctx)
}

// Info displays an info message
func (ui *UI) Info(title, message string) {
	ctx := context.Background()
	prettyMsg := theme.IconInfo + " " + title
	if message != "" {
		prettyMsg += ": " + message
	}

	ulog.Info(title).
		Field("message", message).
		Pretty(prettyMsg).
		Log(ctx)
}

// Error displays an error message
func (ui *UI) Error(title string, err error) {
	ctx := context.Background()
	prettyMsg := theme.IconError + " " + title
	if err != nil {
		prettyMsg += fmt.Sprintf(": %v", err)
	}

	ulog.Error(title).
		Err(err).
		Pretty(prettyMsg).
		Log(ctx)
}

// CommandOutput displays command output in verbose mode, mimicking terminal experience
func (ui *UI) CommandOutput(command, stdout, stderr string) {
	if !ui.verbose {
		return
	}

	// Only show output if there's something to display
	if stdout == "" && stderr == "" && !ui.veryVerbose {
		return
	}

	ctx := context.Background()

	// ANSI color codes
	cyan := "\033[36m"    // Cyan for box borders
	green := "\033[32m"   // Green for command prompt
	red := "\033[31m"     // Red for stderr
	reset := "\033[0m"    // Reset colors

	// Terminal-like separator with colored borders
	prettyMsg := fmt.Sprintf("  %s┌─────────────────────────────────────────────%s\n", cyan, reset)

	// Show command as if typed in terminal (only in very verbose mode)
	if command != "" && ui.veryVerbose {
		// Special handling for PATH logging to make it less noisy
		if strings.HasPrefix(command, "PATH for") {
			prettyMsg += fmt.Sprintf("  %s│%s %sDebug:%s %s\n", cyan, reset, green, reset, command)
		} else {
			prettyMsg += fmt.Sprintf("  %s│%s %s$%s %s\n", cyan, reset, green, reset, command)
		}
	}

	// Show stdout exactly as user would see it
	if stdout != "" {
		lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
		for _, line := range lines {
			// Special handling for PATH logging to make it less noisy
			if strings.HasPrefix(command, "PATH for") {
				prettyMsg += fmt.Sprintf("  %s│%s   %s\n", cyan, reset, line)
			} else {
				prettyMsg += fmt.Sprintf("  %s│%s %s\n", cyan, reset, line)
			}
		}
	}

	// Show stderr with different styling
	if stderr != "" {
		lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
		for _, line := range lines {
			prettyMsg += fmt.Sprintf("  %s│%s %s%s%s\n", cyan, reset, red, line, reset)
		}
	}

	prettyMsg += fmt.Sprintf("  %s└─────────────────────────────────────────────%s", cyan, reset)

	ulog.Debug("Command output").
		Field("command", command).
		Field("has_stdout", stdout != "").
		Field("has_stderr", stderr != "").
		Pretty(prettyMsg).
		Log(ctx)
}