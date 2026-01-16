package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/pkg/workspace"
	"github.com/grovetools/core/tui/theme"
	"github.com/grovetools/tend/internal/tui/scanner"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/sirupsen/logrus"
)

type dataLoadedMsg struct {
	workspaces         []*workspace.WorkspaceNode
	scenariosByProject map[string]map[string][]*harness.Scenario
	initialFocus       *workspace.WorkspaceNode
	err                error
}

// clearStatusMsg is used to clear the status message after a timeout.
type clearStatusMsg struct{}

// statusMsg is used to display a status message to the user.
type statusMsg string

// testOutputMsg carries output from a running test.
type testOutputMsg struct {
	nodeID   string // ID of the node that was run
	output   string
	jsonPath string // Path to JSON results file
	done     bool
	err      error
}

// runTestInPaneCmd runs a test in the background and streams output to the TUI.
func runTestInPaneCmd(node *DisplayNode) tea.Cmd {
	var args []string
	projectPath := node.Project.Path

	nodeID := node.ID()

	switch {
	case node.IsScenario:
		args = []string{"run", node.Scenario.Name}
	case node.IsFile:
		var scenarioNames []string
		for _, s := range node.ScenariosInFile {
			scenarioNames = append(scenarioNames, s.Name)
		}
		if len(scenarioNames) == 0 {
			return func() tea.Msg { return testOutputMsg{nodeID: nodeID, output: "No scenarios to run.", done: true} }
		}
		args = append([]string{"run"}, scenarioNames...)
	case node.IsProject:
		args = []string{"run"}
	case node.IsEcosystem:
		return func() tea.Msg { return testOutputMsg{nodeID: nodeID, output: "Cannot run ecosystem tests.", done: true} }
	default:
		return func() tea.Msg { return testOutputMsg{nodeID: nodeID, output: "Not supported.", done: true} }
	}

	executable, err := os.Executable()
	if err != nil {
		return func() tea.Msg { return testOutputMsg{nodeID: nodeID, output: fmt.Sprintf("Error: %v", err), done: true, err: err} }
	}

	return func() tea.Msg {
		// Create a temp file for JSON output
		jsonFile, err := os.CreateTemp("", "tend-results-*.json")
		if err != nil {
			return testOutputMsg{
				nodeID: nodeID,
				output: fmt.Sprintf("Error creating temp file: %v", err),
				done:   true,
				err:    err,
			}
		}
		jsonPath := jsonFile.Name()
		jsonFile.Close()

		// Add --json flag to write structured results
		args = append(args, "--json", jsonPath)

		cmd := exec.Command(executable, args...)
		cmd.Dir = projectPath
		cmd.Env = append(os.Environ(), "CLICOLOR_FORCE=1", "TERM=xterm-256color")

		// Use CombinedOutput for simpler capture
		output, cmdErr := cmd.CombinedOutput()

		outputStr := string(output)
		if cmdErr != nil {
			return testOutputMsg{
				nodeID:   nodeID,
				output:   outputStr + fmt.Sprintf("\n\n%s Failed: %v", theme.IconError, cmdErr),
				jsonPath: jsonPath,
				done:     true,
				err:      cmdErr,
			}
		}
		return testOutputMsg{
			nodeID:   nodeID,
			output:   outputStr + "\n\n" + theme.IconSuccess + " Completed",
			jsonPath: jsonPath,
			done:     true,
		}
	}
}

func loadDataCmd(initialFocusPath string) tea.Cmd {
	return func() tea.Msg {
		// 1. Discover all projects using workspace discovery service
		logger := logrus.New()
		logPath := filepath.Join(os.TempDir(), "tend-tui.log")
		// Silently ignore errors, but fallback to discarding logs to prevent UI corruption.
		if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			logger.SetOutput(logFile)
		} else {
			logger.SetOutput(io.Discard)
		}
		logger.SetLevel(logrus.DebugLevel)
		discoveryService := workspace.NewDiscoveryService(logger)

		logger.Debugf("[TEND TUI] Starting workspace discovery, initialFocusPath=%s", initialFocusPath)

		result, err := discoveryService.DiscoverAll()
		if err != nil {
			logger.Errorf("[TEND TUI] Discovery failed: %v", err)
			return dataLoadedMsg{err: err}
		}

		// 2. Create a provider to get the workspace nodes
		provider := workspace.NewProvider(result)
		allNodes := provider.All()

		logger.Debugf("[TEND TUI] Discovered %d workspaces before tree build", len(allNodes))
		for i, node := range allNodes {
			logger.Debugf("[TEND TUI]   [%d] Name=%q Path=%q Kind=%s Depth=%d", i, node.Name, node.Path, node.Kind, node.Depth)
		}

		// 3. Build the workspace tree to set up tree structure and prefixes
		allNodes = workspace.BuildWorkspaceTree(allNodes)

		logger.Debugf("[TEND TUI] After tree build: %d workspaces", len(allNodes))
		for i, node := range allNodes {
			logger.Debugf("[TEND TUI]   [%d] Name=%q Path=%q TreePrefix=%q Depth=%d", i, node.Name, node.Path, node.TreePrefix, node.Depth)
		}

		// 4. Find the initial focus workspace from the discovered nodes
		var initialFocus *workspace.WorkspaceNode
		if initialFocusPath != "" {
			logger.Debugf("[TEND TUI] Looking for initial focus: %q", initialFocusPath)
			for _, node := range allNodes {
				// Use case-insensitive comparison for macOS/Windows compatibility
				if strings.EqualFold(node.Path, initialFocusPath) {
					initialFocus = node
					logger.Debugf("[TEND TUI] Found initial focus: Name=%q Path=%q", node.Name, node.Path)
					break
				}
			}
			if initialFocus == nil {
				logger.Warnf("[TEND TUI] Initial focus path not found: %q", initialFocusPath)
				logger.Debugf("[TEND TUI] Available paths:")
				for i, node := range allNodes {
					if i < 10 { // Only log first 10 to avoid spam
						logger.Debugf("[TEND TUI]   %q", node.Path)
					}
				}
			}
		}

		// 5. Filter nodes based on initial focus
		// If we have an initial focus, only show that workspace and its children
		if initialFocus != nil {
			logger.Debugf("[TEND TUI] Filtering to focus workspace and children")
			var filteredNodes []*workspace.WorkspaceNode
			filteredNodes = append(filteredNodes, initialFocus)
			for _, node := range allNodes {
				if node.IsChildOf(initialFocus.Path) {
					logger.Debugf("[TEND TUI]   Adding child: Name=%q Path=%q", node.Name, node.Path)
					filteredNodes = append(filteredNodes, node)
				}
			}
			allNodes = filteredNodes
			logger.Debugf("[TEND TUI] After filtering: %d workspaces", len(allNodes))

			// Note: We don't rebuild the tree here because BuildWorkspaceTree would
			// remove nodes whose parents aren't in the filtered list. The tree
			// structure (TreePrefix, Depth) was already correctly set before filtering.

			logger.Debugf("[TEND TUI] Filtered nodes to display:")
			for i, node := range allNodes {
				logger.Debugf("[TEND TUI]   [%d] Name=%q TreePrefix=%q Depth=%d", i, node.Name, node.TreePrefix, node.Depth)
			}
		}

		// 6. Concurrently scan each project for scenarios.
		var wg sync.WaitGroup
		var mu sync.Mutex
		scenariosByProject := make(map[string]map[string][]*harness.Scenario)

		for _, node := range allNodes {
			// Only scan actual projects, not ecosystems
			if node.IsEcosystem() {
				continue
			}
			wg.Add(1)
			go func(project *workspace.WorkspaceNode) {
				defer wg.Done()
				scenarios, _ := scanner.ScanProjectForScenarios(project.Path)
				if len(scenarios) > 0 {
					mu.Lock()
					scenariosByProject[project.Path] = scenarios
					mu.Unlock()
				}
			}(node)
		}
		wg.Wait()

		logger.Debugf("[TEND TUI] Scanned scenarios from %d projects", len(scenariosByProject))
		for path, fileMap := range scenariosByProject {
			scenarioCount := 0
			for _, scenarios := range fileMap {
				scenarioCount += len(scenarios)
			}
			logger.Debugf("[TEND TUI]   %s: %d scenarios", path, scenarioCount)
		}

		logger.Debugf("[TEND TUI] Returning %d workspaces, initialFocus=%v", len(allNodes), initialFocus != nil)

		return dataLoadedMsg{
			workspaces:         allNodes,
			scenariosByProject: scenariosByProject,
			initialFocus:       initialFocus,
		}
	}
}

// runTestCmd creates a command to run a test scenario in debug mode.
func runTestCmd(node *DisplayNode) tea.Cmd {
	var args []string
	projectPath := node.Project.Path // All nodes have a project context

	switch {
	case node.IsScenario:
		// Correct args for a single scenario: tend run <scenario-name> --debug
		args = []string{"run", node.Scenario.Name, "--debug"}

	case node.IsFile:
		var scenarioNames []string
		for _, s := range node.ScenariosInFile {
			scenarioNames = append(scenarioNames, s.Name)
		}
		if len(scenarioNames) == 0 {
			return func() tea.Msg { return statusMsg("No scenarios to run in this file.") }
		}
		// Correct args for a file: tend run <s1> <s2> ... --debug
		args = append([]string{"run"}, scenarioNames...)
		args = append(args, "--debug")

	case node.IsProject:
		// Correct args for a project: tend run --debug
		args = []string{"run", "--debug"}

	case node.IsEcosystem:
		return func() tea.Msg { return statusMsg("Cannot run tests for an entire ecosystem from the TUI.") }

	default:
		return func() tea.Msg { return statusMsg("This action is not supported for the selected item.") }
	}

	// Get the current executable path to ensure we run the same tend binary.
	executable, err := os.Executable()
	if err != nil {
		return func() tea.Msg {
			return statusMsg(fmt.Sprintf("Error finding tend binary: %v", err))
		}
	}

	cmd := exec.Command(executable, args...)
	cmd.Dir = projectPath // Run the command in the project's directory

	// tea.ExecProcess suspends the TUI and hands terminal control to the child process.
	// The tend harness will then manage the interactive session and its single tmux split.
	// The pane ID is persisted in a cache file (/tmp/tend-debug-pane-id) and reused across runs.
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return statusMsg(fmt.Sprintf("Test run failed: %v", err))
		}
		return statusMsg("Test run finished. Press 'R' to refresh if needed.")
	})
}

// runTestDebugSessionCmd creates debug sessions for scenarios.
// For a single scenario, creates one session. For a file, creates a session per scenario.
func runTestDebugSessionCmd(node *DisplayNode) tea.Cmd {
	projectPath := node.Project.Path

	// Get the current executable path to ensure we run the same tend binary.
	executable, err := os.Executable()
	if err != nil {
		return func() tea.Msg {
			return statusMsg(fmt.Sprintf("Error finding tend binary: %v", err))
		}
	}

	var scenarios []string
	if node.IsScenario {
		scenarios = []string{node.Scenario.Name}
	} else if node.IsFile {
		for _, s := range node.ScenariosInFile {
			scenarios = append(scenarios, s.Name)
		}
	}

	if len(scenarios) == 0 {
		return func() tea.Msg {
			return statusMsg("No scenarios to run.")
		}
	}

	// Run --debug-session for each scenario
	return func() tea.Msg {
		for _, scenarioName := range scenarios {
			args := []string{"run", scenarioName, "--debug-session"}
			cmd := exec.Command(executable, args...)
			cmd.Dir = projectPath
			// Run synchronously so all sessions get created
			_ = cmd.Run()
		}
		if len(scenarios) == 1 {
			return statusMsg(fmt.Sprintf("Debug session created: tend_%s. Run 'tend sessions' to view.", scenarios[0]))
		}
		return statusMsg(fmt.Sprintf("Created %d debug sessions. Run 'tend sessions' to view.", len(scenarios)))
	}
}
