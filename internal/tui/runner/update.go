package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/pkg/workspace"
	"github.com/grovetools/core/tui/keymap"
	"github.com/sirupsen/logrus"

	"github.com/grovetools/tend/pkg/harness/reporters"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetSize(msg.Width, msg.Height)
		paneWidth := m.width/2 - 4
		paneHeight := m.height - 8
		if paneHeight < 5 {
			paneHeight = 5
		}
		m.outputPane = viewport.New(paneWidth, paneHeight)
		m.outputPane.SetContent(m.outputContent)
		m.ready = true
		return m, nil

	case dataLoadedMsg:
		m.isLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.workspaces = msg.workspaces
		m.scenariosByProject = msg.scenariosByProject
		m.focusedProject = msg.initialFocus
		m.buildDisplayTree()
		return m, nil

	case statusMsg:
		m.statusMessage = string(msg)
		m.statusTimeout = time.Now().Add(3 * time.Second)
		return m, clearStatusCmd(3 * time.Second)

	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case testOutputMsg:
		m.outputContent = msg.output
		m.outputPane.SetContent(m.outputContent)
		m.outputPane.GotoBottom()
		if msg.done {
			m.testRunning = false
			// Update status based on result
			status := StatusPassed
			if msg.err != nil {
				status = StatusFailed
			}
			m.updateStatuses(msg.nodeID, status)

			// Parse JSON results to update individual scenario statuses
			if msg.jsonPath != "" {
				m.parseJSONResults(msg.jsonPath, msg.nodeID)
				// Clean up temp file
				os.Remove(msg.jsonPath)
			}

			m.statusMessage = "Test finished."
			m.statusTimeout = time.Now().Add(3 * time.Second)
			return m, clearStatusCmd(3 * time.Second)
		}
		return m, nil

	case tea.KeyMsg:
		if m.help.ShowAll {
			m.help.Toggle()
			return m, nil
		}

		// Handle input while filtering
		if m.filterInput.Focused() {
			var cmd tea.Cmd
			switch msg.String() {
			case "enter", "esc":
				m.filterInput.Blur()
				return m, nil
			}

			// Update the filter input and re-build the display tree
			prevValue := m.filterInput.Value()
			m.filterInput, cmd = m.filterInput.Update(msg)
			if m.filterInput.Value() != prevValue {
				m.buildDisplayTree()
				m.cursor = 0 // Reset cursor on new filter
			}
			return m, cmd
		}

		// Close output pane with Escape
		if m.outputVisible && !m.testRunning && msg.String() == "esc" {
			m.outputVisible = false
			return m, nil
		}

		// Process sequence state for multi-key commands
		sequenceBindings := keymap.CommonSequenceBindings(m.keys.Base)
		result, _ := m.sequence.Process(msg, sequenceBindings...)
		buffer := m.sequence.Buffer()

		// Handle 'gg' sequence - go to top
		if result == keymap.SequenceMatch && keymap.Matches(buffer, m.keys.Top) {
			m.cursor = 0
			m.adjustScrollOffset()
			m.sequence.Clear()
			return m, nil
		}

		// Handle z* fold commands
		if result == keymap.SequenceMatch {
			if keymap.Matches(buffer, m.keys.FoldToggle) {
				m.toggleFold()
				m.sequence.Clear()
				return m, nil
			} else if keymap.Matches(buffer, m.keys.FoldClose) {
				m.closeFold()
				m.sequence.Clear()
				return m, nil
			} else if keymap.Matches(buffer, m.keys.FoldOpen) {
				m.openFold()
				m.sequence.Clear()
				return m, nil
			} else if keymap.Matches(buffer, m.keys.FoldOpenAll) {
				m.openAllFolds()
				m.sequence.Clear()
				return m, nil
			} else if keymap.Matches(buffer, m.keys.FoldCloseAll) {
				m.closeAllFolds()
				m.sequence.Clear()
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			m.adjustScrollOffset()
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.displayNodes)-1 {
				m.cursor++
			}
			m.adjustScrollOffset()
		case key.Matches(msg, m.keys.PageUp):
			halfPage := m.getVisibleNodeCount() / 2
			if halfPage < 1 {
				halfPage = 1
			}
			m.cursor -= halfPage
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustScrollOffset()
		case key.Matches(msg, m.keys.PageDown):
			halfPage := m.getVisibleNodeCount() / 2
			if halfPage < 1 {
				halfPage = 1
			}
			m.cursor += halfPage
			if m.cursor >= len(m.displayNodes) {
				m.cursor = len(m.displayNodes) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustScrollOffset()
		case key.Matches(msg, m.keys.Bottom):
			if len(m.displayNodes) > 0 {
				m.cursor = len(m.displayNodes) - 1
			}
			m.adjustScrollOffset()
		case key.Matches(msg, m.keys.Left):
			m.closeFold()
		case key.Matches(msg, m.keys.Right):
			m.openFold()
		case key.Matches(msg, m.keys.FocusSelected):
			if m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				if node.IsProject || node.IsEcosystem {
					m.focusedProject = node.Project
					m.buildDisplayTree()
					m.cursor = 0
				}
			}
		case key.Matches(msg, m.keys.FocusEcosystem):
			if m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				if node.IsProject || node.IsEcosystem {
					m.focusedProject = node.Project
					m.buildDisplayTree()
					m.cursor = 0
				}
			}
		case key.Matches(msg, m.keys.ClearFocus):
			if m.focusedProject != nil {
				m.focusedProject = nil
				m.buildDisplayTree()
				m.cursor = 0
			}
		case key.Matches(msg, m.keys.Run):
			if m.cursor < len(m.displayNodes) && !m.testRunning {
				node := m.displayNodes[m.cursor]
				if node.IsEcosystem {
					m.statusMessage = "Cannot run tests for an entire ecosystem from the TUI."
					m.statusTimeout = time.Now().Add(3 * time.Second)
					return m, clearStatusCmd(3 * time.Second)
				}
				m.outputContent = "Running test...\n"
				m.outputVisible = true
				m.testRunning = true
				m.outputPane.SetContent(m.outputContent)
				m.updateStatuses(node.ID(), StatusRunning)
				return m, runTestInPaneCmd(node)
			}
		case key.Matches(msg, m.keys.DebugRun):
			if m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				// Running tests for an entire ecosystem from the TUI is disabled for now
				if node.IsEcosystem {
					m.statusMessage = "Cannot run tests for an entire ecosystem from the TUI."
					m.statusTimeout = time.Now().Add(3 * time.Second)
					return m, clearStatusCmd(3 * time.Second)
				}
				return m, runTestCmd(node)
			}
		case key.Matches(msg, m.keys.DebugSession):
			if m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				if node.IsEcosystem || node.IsProject {
					m.statusMessage = "Select a scenario or file for debug session."
					m.statusTimeout = time.Now().Add(3 * time.Second)
					return m, clearStatusCmd(3 * time.Second)
				}
				return m, runTestDebugSessionCmd(node)
			}
		case key.Matches(msg, m.keys.Search):
			m.filterInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, m.keys.Help):
			m.help.Toggle()
		default:
			// Clear sequence buffer for keys that aren't part of sequences
			// unless we're in the middle of a potential sequence
			if result != keymap.SequencePending {
				m.sequence.Clear()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// buildDisplayTree constructs the flat list of nodes for rendering.
func (m *Model) buildDisplayTree() {
	logger := logrus.New()
	logPath := filepath.Join(os.TempDir(), "tend-tui.log")
	// Silently ignore errors, but fallback to discarding logs to prevent UI corruption.
	if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666); err == nil {
		logger.SetOutput(logFile)
	} else {
		logger.SetOutput(io.Discard)
	}
	logger.SetLevel(logrus.DebugLevel)

	logger.Debugf("[TEND TUI buildDisplayTree] Starting, focusedProject=%v, total workspaces=%d", m.focusedProject != nil, len(m.workspaces))

	var nodes []*DisplayNode
	filter := strings.ToLower(m.filterInput.Value())

	var projectsToDisplay []*workspace.WorkspaceNode
	if m.focusedProject != nil {
		logger.Debugf("[TEND TUI buildDisplayTree] In focused mode: %q", m.focusedProject.Name)
		projectsToDisplay = []*workspace.WorkspaceNode{m.focusedProject}
		for _, p := range m.workspaces {
			if p.IsChildOf(m.focusedProject.Path) {
				projectsToDisplay = append(projectsToDisplay, p)
			}
		}
	} else {
		logger.Debugf("[TEND TUI buildDisplayTree] In global view mode")
		projectsToDisplay = m.workspaces
	}

	logger.Debugf("[TEND TUI buildDisplayTree] Projects to display: %d", len(projectsToDisplay))

	for _, p := range projectsToDisplay {
		// Skip workspaces with empty names (defensive check)
		if p == nil || p.Name == "" {
			logger.Warnf("[TEND TUI buildDisplayTree] Skipping workspace with empty name")
			continue
		}

		logger.Debugf("[TEND TUI buildDisplayTree] Processing: Name=%q Path=%q Depth=%d Prefix=%q", p.Name, p.Path, p.Depth, p.TreePrefix)

		if _, hasScenarios := m.scenariosByProject[p.Path]; !hasScenarios && !p.IsEcosystem() {
			if m.focusedProject == nil && p.Depth > 0 { // In global view, only show top-level or ecosystems
				logger.Debugf("[TEND TUI buildDisplayTree] Skipping (global view, depth>0, no scenarios): %q", p.Name)
				continue
			}
			if m.focusedProject != nil && p.Path != m.focusedProject.Path { // In focus view, only show projects with tests
				logger.Debugf("[TEND TUI buildDisplayTree] Skipping (focus view, not focused, no scenarios): %q", p.Name)
				continue
			}
		}

		node := &DisplayNode{
			IsProject: true,
			Project:   p,
			Prefix:    p.TreePrefix,
			Depth:     p.Depth,
		}
		if p.IsEcosystem() {
			node.IsEcosystem = true
		}
		logger.Debugf("[TEND TUI buildDisplayTree] Adding project node: Name=%q IsEco=%v Prefix=%q", p.Name, node.IsEcosystem, node.Prefix)
		nodes = append(nodes, node)

		projectNodeID := node.ID()
		if m.collapsedNodes[projectNodeID] {
			continue
		}

		if scenariosByFile, ok := m.scenariosByProject[p.Path]; ok {
			var filePaths []string
			for path := range scenariosByFile {
				filePaths = append(filePaths, path)
			}
			sort.Strings(filePaths)

			for i, filePath := range filePaths {
				isLastFile := i == len(filePaths)-1

				var filePrefixBuilder strings.Builder
				indentPrefix := strings.ReplaceAll(p.TreePrefix, "├─", "│ ")
				indentPrefix = strings.ReplaceAll(indentPrefix, "└─", "  ")
				filePrefixBuilder.WriteString(indentPrefix)
				if p.Depth > 0 || p.TreePrefix != "" {
					filePrefixBuilder.WriteString("  ")
				}
				if isLastFile {
					filePrefixBuilder.WriteString("└─ ")
				} else {
					filePrefixBuilder.WriteString("├─ ")
				}

				fileNode := &DisplayNode{
					IsFile:          true,
					Project:         p,
					FilePath:        filePath,
					ScenariosInFile: scenariosByFile[filePath],
					Prefix:          filePrefixBuilder.String(),
					Depth:           p.Depth + 1,
				}
				nodes = append(nodes, fileNode)

				fileNodeID := fileNode.ID()
				if m.collapsedNodes[fileNodeID] {
					continue
				}

				scenarios := scenariosByFile[filePath]
				for j, scenario := range scenarios {
					isLastScenario := j == len(scenarios)-1

					var scenarioPrefixBuilder strings.Builder
					scenarioIndent := strings.ReplaceAll(filePrefixBuilder.String(), "├─", "│ ")
					scenarioIndent = strings.ReplaceAll(scenarioIndent, "└─", "  ")
					scenarioPrefixBuilder.WriteString(scenarioIndent)
					if isLastScenario {
						scenarioPrefixBuilder.WriteString("└─ ")
					} else {
						scenarioPrefixBuilder.WriteString("├─ ")
					}

					scenarioNode := &DisplayNode{
						IsScenario: true,
						Project:    p,
						FilePath:   filePath,
						Scenario:   scenario,
						Prefix:     scenarioPrefixBuilder.String(),
						Depth:      p.Depth + 2,
					}
					nodes = append(nodes, scenarioNode)
				}
			}
		}
	}

	// Apply filtering if there's a search term
	if filter != "" {
		logger.Debugf("[TEND TUI buildDisplayTree] Applying filter: %q", filter)
		nodesToKeep := make(map[string]bool)

		// First pass: identify matching nodes and mark them
		for i, node := range nodes {
			var name string
			if node.IsEcosystem || node.IsProject {
				name = node.Project.Name
			} else if node.IsFile {
				name = filepath.Base(node.FilePath)
			} else if node.IsScenario && node.Scenario != nil {
				name = node.Scenario.Name
			}

			// Check if this node matches the filter
			if strings.Contains(strings.ToLower(name), filter) {
				logger.Debugf("[TEND TUI buildDisplayTree] Match found: %q at index %d", name, i)
				// Mark this node
				if id := node.ID(); id != "" {
					nodesToKeep[id] = true
				}
				// For scenarios (which don't have IDs), we track by index temporarily
				if node.IsScenario {
					nodesToKeep[fmt.Sprintf("idx:%d", i)] = true
				}

				// Mark all ancestors by traversing backwards
				for j := i - 1; j >= 0; j-- {
					ancestor := nodes[j]
					// An ancestor is a node with lower depth
					if ancestor.Depth < node.Depth {
						if ancestorID := ancestor.ID(); ancestorID != "" {
							if !nodesToKeep[ancestorID] {
								logger.Debugf("[TEND TUI buildDisplayTree] Marking ancestor: %q at index %d", ancestor.Project.Name, j)
								nodesToKeep[ancestorID] = true
							}
						}
						// Stop at the immediate parent's depth level
						if ancestor.Depth == node.Depth-1 {
							break
						}
					}
				}
			}
		}

		// Second pass: build filtered list
		var filteredNodes []*DisplayNode
		for i, node := range nodes {
			shouldKeep := false
			if id := node.ID(); id != "" {
				shouldKeep = nodesToKeep[id]
			} else if node.IsScenario {
				shouldKeep = nodesToKeep[fmt.Sprintf("idx:%d", i)]
			}

			if shouldKeep {
				filteredNodes = append(filteredNodes, node)
			}
		}
		nodes = filteredNodes
		logger.Debugf("[TEND TUI buildDisplayTree] After filtering: %d nodes remain", len(nodes))
	}

	m.displayNodes = nodes

	logger.Debugf("[TEND TUI buildDisplayTree] Created %d display nodes", len(m.displayNodes))
	for i, node := range m.displayNodes {
		if node.IsProject || node.IsEcosystem {
			logger.Debugf("[TEND TUI buildDisplayTree]   [%d] Project: Name=%q Prefix=%q", i, node.Project.Name, node.Prefix)
		} else if node.IsFile {
			logger.Debugf("[TEND TUI buildDisplayTree]   [%d] File: %q Prefix=%q", i, filepath.Base(node.FilePath), node.Prefix)
		} else if node.IsScenario {
			scenarioName := "<nil>"
			if node.Scenario != nil {
				scenarioName = node.Scenario.Name
			}
			logger.Debugf("[TEND TUI buildDisplayTree]   [%d] Scenario: %q Prefix=%q", i, scenarioName, node.Prefix)
		}
	}

	// Clamp cursor
	if m.cursor >= len(m.displayNodes) {
		m.cursor = len(m.displayNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) toggleFold() {
	if m.cursor >= len(m.displayNodes) {
		return
	}
	node := m.displayNodes[m.cursor]
	if !node.IsFile && !node.IsProject && !node.IsEcosystem {
		return
	}

	nodeID := node.ID()
	if m.collapsedNodes[nodeID] {
		delete(m.collapsedNodes, nodeID)
	} else {
		m.collapsedNodes[nodeID] = true
	}
	m.buildDisplayTree()
}

func (m *Model) openFold() {
	if m.cursor >= len(m.displayNodes) {
		return
	}
	node := m.displayNodes[m.cursor]
	if !node.IsFile && !node.IsProject && !node.IsEcosystem {
		return
	}

	delete(m.collapsedNodes, node.ID())
	m.buildDisplayTree()
}

func (m *Model) closeFold() {
	if m.cursor >= len(m.displayNodes) {
		return
	}
	node := m.displayNodes[m.cursor]
	if !node.IsFile && !node.IsProject && !node.IsEcosystem {
		return
	}

	m.collapsedNodes[node.ID()] = true
	m.buildDisplayTree()
}

func (m *Model) openAllFolds() {
	m.collapsedNodes = make(map[string]bool)
	m.buildDisplayTree()
}

func (m *Model) closeAllFolds() {
	for _, node := range m.displayNodes {
		if node.IsFile || node.IsProject || node.IsEcosystem {
			m.collapsedNodes[node.ID()] = true
		}
	}
	m.buildDisplayTree()
}

// getVisibleNodeCount returns how many nodes can be displayed in the viewport
func (m *Model) getVisibleNodeCount() int {
	// Adjust for header, footer, and other UI elements
	const uiChromeHeight = 8
	visibleHeight := m.height - uiChromeHeight
	if visibleHeight < 1 {
		return 1
	}
	return visibleHeight
}

// adjustScrollOffset ensures the cursor is visible
func (m *Model) adjustScrollOffset() {
	visibleCount := m.getVisibleNodeCount()

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+visibleCount {
		m.scrollOffset = m.cursor - visibleCount + 1
	}
}

// clearStatusCmd returns a command that sends a clearStatusMsg after a delay.
func clearStatusCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// updateStatuses sets the status for a given node ID.
// For scenarios, it only updates the scenario.
// For files/projects when setting to Running, it updates all child scenarios to Running.
// For files/projects when setting to Passed/Failed, it only updates the parent node, not children.
func (m *Model) updateStatuses(nodeID string, status TestStatus) {
	m.testStatuses[nodeID] = status

	// Find the node in the display tree
	var parentNode *DisplayNode
	for _, n := range m.displayNodes {
		if n.ID() == nodeID {
			parentNode = n
			break
		}
	}

	if parentNode == nil {
		return // Should not happen
	}

	// Only propagate StatusRunning to children, not Passed/Failed
	// This allows individual scenario results to show through
	if (parentNode.IsProject || parentNode.IsFile) && status == StatusRunning {
		for _, n := range m.displayNodes {
			// A child scenario will have a deeper depth and a matching project/file path
			if n.IsScenario && n.Depth > parentNode.Depth {
				if parentNode.IsProject && n.Project.Path == parentNode.Project.Path {
					m.testStatuses[n.ID()] = StatusRunning
				}
				if parentNode.IsFile && n.FilePath == parentNode.FilePath {
					m.testStatuses[n.ID()] = StatusRunning
				}
			}
		}
	}
}

// parseJSONResults reads the JSON results file and updates scenario statuses
func (m *Model) parseJSONResults(jsonPath, parentNodeID string) {
	logger := logrus.New()
	logPath := filepath.Join(os.TempDir(), "tend-tui.log")
	if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666); err == nil {
		logger.SetOutput(logFile)
	} else {
		logger.SetOutput(io.Discard)
	}
	logger.SetLevel(logrus.DebugLevel)

	logger.Infof("[PARSE] Parsing JSON results from: %s", jsonPath)

	// Find the parent node to determine which scenarios to update
	var parentNode *DisplayNode
	for _, n := range m.displayNodes {
		if n.ID() == parentNodeID {
			parentNode = n
			break
		}
	}

	if parentNode == nil || parentNode.IsScenario {
		logger.Debugf("[PARSE] Skipping: parentNode is nil or is a scenario")
		return // Only parse for file/project runs
	}

	logger.Debugf("[PARSE] Parent node: IsFile=%v, IsProject=%v, Path=%s", parentNode.IsFile, parentNode.IsProject, parentNode.FilePath)

	// Read and parse JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		logger.Errorf("[PARSE] Failed to read JSON file: %v", err)
		return
	}

	var report reporters.JSONReport
	if err := json.Unmarshal(data, &report); err != nil {
		logger.Errorf("[PARSE] Failed to parse JSON: %v", err)
		return
	}

	logger.Infof("[PARSE] Parsed JSON: %d results, %d passed, %d failed", report.TotalTests, report.Passed, report.Failed)

	// Update individual scenario statuses from JSON results
	matchCount := 0
	for _, result := range report.Results {
		scenarioName := result.Name
		var status TestStatus
		if result.Success {
			status = StatusPassed
		} else {
			status = StatusFailed
		}

		logger.Debugf("[PARSE] JSON result: Scenario=%q, Success=%v", scenarioName, result.Success)

		// Find the scenario node and update its status
		found := false
		for _, n := range m.displayNodes {
			if n.IsScenario && n.Scenario != nil && n.Scenario.Name == scenarioName {
				// Make sure it's a child of the parent node that was run
				if (parentNode.IsFile && n.FilePath == parentNode.FilePath) ||
					(parentNode.IsProject && n.Project.Path == parentNode.Project.Path) {
					logger.Debugf("[PARSE] Updating status for scenario %q (ID: %s) to %d", scenarioName, n.ID(), status)
					m.testStatuses[n.ID()] = status
					matchCount++
					found = true
				}
			}
		}
		if !found {
			logger.Warnf("[PARSE] Could not find matching display node for scenario: %q", scenarioName)
		}
	}
	logger.Infof("[PARSE] Updated %d scenario statuses from JSON", matchCount)
}
