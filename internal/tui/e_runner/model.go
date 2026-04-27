package e_runner

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Status represents the execution status of a project's test suite.
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusSuccess
	StatusFailure
)

// ProjectState holds the state for a single project's test run.
type ProjectState struct {
	projectName string
	projectPath string
	makeTarget  string
	status      Status
	duration    time.Duration
	output      string
	result      *ProjectResult
}

// Implement the list.Item interface
func (s ProjectState) Title() string       { return s.projectName }
func (s ProjectState) Description() string { return "" }
func (s ProjectState) FilterValue() string { return s.projectName }

// Accessor methods for ProjectState
func (s *ProjectState) Status() Status          { return s.status }
func (s *ProjectState) Duration() time.Duration { return s.duration }
func (s *ProjectState) Output() string          { return s.output }

// Model is the main Bubble Tea model for the ecosystem parallel runner.
type Model struct {
	projects      []*ProjectState
	list          list.Model
	spinner       spinner.Model
	logViewport   viewport.Model
	logLines      []string
	maxLogLines   int
	width, height int
	running       int
	success       int
	failed        int
	finished      bool
	results       []*ProjectResult
	eventsChan    <-chan Event
	numJobs       int
}

// New creates a new parallel runner TUI model for the ecosystem.
func New(testSuites map[string]string, numJobs int) Model {
	var states []*ProjectState
	items := make([]list.Item, 0, len(testSuites))
	for path, target := range testSuites {
		state := &ProjectState{
			projectName: filepath.Base(path),
			projectPath: path,
			makeTarget:  target,
			status:      StatusPending,
		}
		states = append(states, state)
		items = append(items, state)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot

	l := list.New(items, newDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)

	logViewport := viewport.New(0, 0)
	logViewport.SetContent("Waiting for test output...")

	return Model{
		projects:    states,
		list:        l,
		spinner:     s,
		logViewport: logViewport,
		maxLogLines: 200,
		numJobs:     numJobs,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		startRunCmd(m.projects, m.numJobs),
	)
}

// Results returns the final test results.
func (m Model) Results() []*ProjectResult {
	return m.results
}

// ProjectStates returns the final states of all projects.
func (m Model) ProjectStates() []*ProjectState {
	return m.projects
}
