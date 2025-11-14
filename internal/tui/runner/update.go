package runner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-core/pkg/workspace"
	"github.com/sirupsen/logrus"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetSize(msg.Width, msg.Height)
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

	case tea.KeyMsg:
		if m.help.ShowAll {
			m.help.Toggle()
			return m, nil
		}

		// Handle 'gg' and 'z' chords
		if m.lastKey == "g" && msg.String() == "g" {
			m.cursor = 0
			m.adjustScrollOffset()
			m.lastKey = ""
			return m, nil
		}
		if m.lastKey == "z" {
			switch msg.String() {
			case "a": // za - toggle fold
				m.toggleFold()
			case "c": // zc - close fold
				m.closeFold()
			case "o": // zo - open fold
				m.openFold()
			case "R": // zR - open all folds
				m.openAllFolds()
			case "M": // zM - close all folds
				m.closeAllFolds()
			}
			m.lastKey = ""
			return m, nil
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
		case key.Matches(msg, m.keys.GoToTop):
			m.lastKey = "g" // Wait for second 'g'
		case key.Matches(msg, m.keys.GoToBottom):
			if len(m.displayNodes) > 0 {
				m.cursor = len(m.displayNodes) - 1
			}
			m.adjustScrollOffset()
		case key.Matches(msg, m.keys.Fold):
			m.closeFold()
		case key.Matches(msg, m.keys.Unfold):
			m.openFold()
		case key.Matches(msg, m.keys.FoldPrefix):
			m.lastKey = "z"
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
		case key.Matches(msg, m.keys.Help):
			m.help.Toggle()
		default:
			m.lastKey = "" // Reset chord on any other key
		}
	}

	return m, tea.Batch(cmds...)
}

// buildDisplayTree constructs the flat list of nodes for rendering.
func (m *Model) buildDisplayTree() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetOutput(os.Stderr)

	logger.Debugf("[TEND TUI buildDisplayTree] Starting, focusedProject=%v, total workspaces=%d", m.focusedProject != nil, len(m.workspaces))

	var nodes []*DisplayNode

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
		logger.Debugf("[TEND TUI buildDisplayTree] Processing: Name=%q Path=%q Depth=%d Prefix=%q", p.Name, p.Path, p.Depth, p.TreePrefix)

		// Skip workspaces with empty names (defensive check)
		if p == nil || p.Name == "" {
			logger.Warnf("[TEND TUI buildDisplayTree] Skipping workspace with empty name: Path=%q", p.Path)
			continue
		}

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
