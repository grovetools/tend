package runner

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-core/pkg/workspace"
	"github.com/mattsolo1/grove-core/tui/components/help"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// TestStatus represents the run state of a testable node.
type TestStatus int

const (
	StatusNotRun TestStatus = iota
	StatusRunning
	StatusPassed
	StatusFailed
)

// DisplayNode represents a single line in the hierarchical TUI view.
type DisplayNode struct {
	IsEcosystem bool
	IsProject   bool
	IsFile      bool
	IsScenario  bool

	// Use Project for all node types to know the project context
	Project   *workspace.WorkspaceNode
	FilePath  string            // For File and Scenario nodes
	Scenario  *harness.Scenario // For Scenario nodes
	ScenariosInFile []*harness.Scenario // For File nodes

	// Pre-calculated for rendering
	Prefix string
	Depth  int
}

// ID returns a unique identifier for this node, used for tracking collapsed state and test status.
func (n *DisplayNode) ID() string {
	if n.IsEcosystem || n.IsProject {
		return "ws:" + n.Project.Path
	}
	if n.IsFile {
		return "file:" + n.FilePath
	}
	if n.IsScenario {
		// Use file path and scenario name for a unique ID
		return "sc:" + n.FilePath + "::" + n.Scenario.Name
	}
	return ""
}

// Model is the main model for the test runner TUI.
type Model struct {
	isLoading          bool
	workspaces         []*workspace.WorkspaceNode
	scenariosByProject map[string]map[string][]*harness.Scenario // project path -> file path -> scenarios
	displayNodes       []*DisplayNode
	cursor             int
	scrollOffset       int
	help               help.Model
	keys               KeyMap
	width, height      int
	err                error
	ready              bool

	// Focus and navigation
	initialFocusPath string                   // Path of workspace to focus on startup
	focusedProject   *workspace.WorkspaceNode // Currently focused workspace
	collapsedNodes   map[string]bool          // Key is a unique node ID
	lastKey          string                   // For detecting 'gg' and 'z' chords
	statusMessage    string
	statusTimeout    time.Time

	// Filter/search
	filterInput textinput.Model

	// Test execution state
	testStatuses  map[string]TestStatus // Key is DisplayNode.ID()
	outputPane    viewport.Model
	outputContent string
	outputVisible bool
	testRunning   bool
}

// New creates a new TUI model.
// If initialFocusPath is provided (non-empty), the TUI will only show tests from that workspace and its children.
func New(initialFocusPath string) Model {
	helpModel := help.NewBuilder().
		WithKeys(newKeyMap()).
		WithTitle("Tend Test Runner - Help").
		Build()

	ti := textinput.New()
	ti.Placeholder = "Search scenarios..."
	ti.CharLimit = 100
	ti.Width = 50

	return Model{
		isLoading:          true,
		keys:               newKeyMap(),
		help:               helpModel,
		scenariosByProject: make(map[string]map[string][]*harness.Scenario),
		collapsedNodes:     make(map[string]bool),
		initialFocusPath:   initialFocusPath,
		filterInput:        ti,
		testStatuses:       make(map[string]TestStatus),
	}
}

func (m Model) Init() tea.Cmd {
	return loadDataCmd(m.initialFocusPath)
}
