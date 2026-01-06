package harness

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-core/tui/theme"
)

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
	fmt.Printf("\n%s Scenario: %s\n", theme.IconTestTube, name)
	if description != "" {
		fmt.Printf("   %s\n", description)
	}
	fmt.Println(strings.Repeat("-", 60))
}

// ScenarioSuccess displays scenario completion
func (ui *UI) ScenarioSuccess(name string, duration time.Duration) {
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%s Scenario completed successfully in %v\n\n", theme.IconSuccess, duration)
}

// ScenarioFailed displays scenario failure
func (ui *UI) ScenarioFailed(name string, err error) {
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%s Scenario failed: %s\n", theme.IconError, name)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
	}
}

// PhaseStart displays the start of a test phase (e.g., Setup, Test, Teardown).
func (ui *UI) PhaseStart(name string) {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")). // Cyan
		MarginTop(1).
		MarginBottom(1)
	fmt.Println(style.Render(fmt.Sprintf("--- %s Phase ---", name)))
}

// StepStart displays the start of a step
func (ui *UI) StepStart(current, total int, name string) {
	fmt.Printf("\n[%d/%d] %s\n", current, total, name)
}

// StepSuccess displays step completion
func (ui *UI) StepSuccess(stepResult StepResult) {
	if ui.verbose {
		fmt.Printf("%s %s (Completed in %v)\n", theme.IconSuccess, stepResult.Name, stepResult.Duration)
		// Print successful assertions
		for _, assertion := range stepResult.Assertions {
			if assertion.Success {
				fmt.Printf("  %s %s\n", theme.IconSuccess, assertion.Description)
			}
		}
	}
}

// StepFailed displays step failure
func (ui *UI) StepFailed(stepResult StepResult) {
	fmt.Printf("%s %s (Failed after %v)\n", theme.IconError, stepResult.Name, stepResult.Duration)

	// Print successful assertions before the failure
	for _, assertion := range stepResult.Assertions {
		if assertion.Success {
			fmt.Printf("  %s %s\n", theme.IconSuccess, assertion.Description)
		}
	}

	// Print the failure details
	if stepResult.Error != nil {
		fmt.Printf("  Error: %v\n", stepResult.Error)
	}
}

// WaitForUser prompts the user to continue
// Returns a string indicating the user's choice: "continue", "quit", or "attach"
func (ui *UI) WaitForUser() string {
	if !ui.interactive {
		return "continue"
	}

	fmt.Printf("%s Press ENTER to continue, 'a' to attach, 'q' to quit: ", theme.IconSelect)
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
		fmt.Println(content)
	}
}

// Cleanup displays cleanup message
func (ui *UI) Cleanup() {
	if ui.verbose {
		fmt.Printf("%s Cleaning up temporary files...\n", theme.IconFolderRemove)
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
	fmt.Printf("\n%s Docker Container Update:\n", theme.IconSync)
	
	if len(containers) == 0 {
		fmt.Println("  No agent containers running")
	} else {
		// Table header
		fmt.Printf("  %-40s %-30s %-20s\n", "NAMES", "IMAGE", "CREATED")
		fmt.Printf("  %s\n", strings.Repeat("-", 92))
		
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
			
			fmt.Printf("  %-40s %-30s %-20s\n", name, image, created)
		}
	}
	fmt.Println()
}

// ShowDockerStatus displays current Docker container status
func (ui *UI) ShowDockerStatus() {
	containers, err := GetContainerSnapshot("name=grove")
	if err != nil {
		return
	}
	
	if len(containers) == 0 {
		return
	}
	
	fmt.Printf("\n%s Docker Containers (grove-related):\n", theme.IconFolder)
	fmt.Printf("%-30s %-20s %s\n", "IMAGE", "CREATED", "NAMES")
	fmt.Println(strings.Repeat("-", 70))
	
	for _, c := range containers {
		if strings.Contains(c.Names, "grove") {
			fmt.Printf("%-30s %-20s %s\n", c.Image, c.Created, c.Names)
		}
	}
	fmt.Println()
}

// Info displays an info message
func (ui *UI) Info(title, message string) {
	fmt.Printf("%s %s", theme.IconInfo, title)
	if message != "" {
		fmt.Printf(": %s", message)
	}
	fmt.Println()
}

// Error displays an error message
func (ui *UI) Error(title string, err error) {
	fmt.Printf("%s %s", theme.IconError, title)
	if err != nil {
		fmt.Printf(": %v", err)
	}
	fmt.Println()
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
	
	// ANSI color codes
	cyan := "\033[36m"    // Cyan for box borders
	green := "\033[32m"   // Green for command prompt
	red := "\033[31m"     // Red for stderr
	reset := "\033[0m"    // Reset colors
	
	// Terminal-like separator with colored borders
	fmt.Printf("  %s┌─────────────────────────────────────────────%s\n", cyan, reset)
	
	// Show command as if typed in terminal (only in very verbose mode)
	if command != "" && ui.veryVerbose {
		// Special handling for PATH logging to make it less noisy
		if strings.HasPrefix(command, "PATH for") {
			fmt.Printf("  %s│%s %sDebug:%s %s\n", cyan, reset, green, reset, command)
		} else {
			fmt.Printf("  %s│%s %s$%s %s\n", cyan, reset, green, reset, command)
		}
	}
	
	// Show stdout exactly as user would see it
	if stdout != "" {
		lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
		for _, line := range lines {
			// Special handling for PATH logging to make it less noisy
			if strings.HasPrefix(command, "PATH for") {
				fmt.Printf("  %s│%s   %s\n", cyan, reset, line)
			} else {
				fmt.Printf("  %s│%s %s\n", cyan, reset, line)
			}
		}
	}
	
	// Show stderr with different styling
	if stderr != "" {
		lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
		for _, line := range lines {
			fmt.Printf("  %s│%s %s%s%s\n", cyan, reset, red, line, reset)
		}
	}
	
	fmt.Printf("  %s└─────────────────────────────────────────────%s\n", cyan, reset)
}