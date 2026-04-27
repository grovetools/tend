package prunner

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/tend/pkg/harness"
)

// Status represents the execution status of a scenario.
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusSuccess
	StatusFailure
)

// ScenarioState holds the state for a single scenario.
type ScenarioState struct {
	scenario *harness.Scenario
	status   Status
	duration time.Duration
	output   string
	result   *harness.Result
}

// Implement the list.Item interface
func (s ScenarioState) Title() string       { return s.scenario.Name }
func (s ScenarioState) Description() string { return "" }
func (s ScenarioState) FilterValue() string { return s.scenario.Name }

// Getters for accessing scenario state fields
func (s *ScenarioState) Scenario() *harness.Scenario { return s.scenario }
func (s *ScenarioState) Status() Status              { return s.status }
func (s *ScenarioState) Duration() time.Duration     { return s.duration }
func (s *ScenarioState) Output() string              { return s.output }

// Model is the main Bubble Tea model for the parallel runner.
type Model struct {
	scenarios     []*ScenarioState
	list          list.Model
	spinner       spinner.Model
	width, height int
	running       int
	success       int
	failed        int
	finished      bool
	results       []*harness.Result
	eventsChan    <-chan Event
	projectRoot   string
	numJobs       int
}

// New creates a new parallel runner TUI model.
func New(scenarios []*harness.Scenario, projectRoot string, numJobs int) Model {
	var states []*ScenarioState
	items := make([]list.Item, len(scenarios))
	for i, s := range scenarios {
		state := &ScenarioState{scenario: s, status: StatusPending}
		states = append(states, state)
		items[i] = state
	}

	s := spinner.New()
	s.Spinner = spinner.Dot

	l := list.New(items, newDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)

	return Model{
		scenarios:   states,
		list:        l,
		spinner:     s,
		projectRoot: projectRoot,
		numJobs:     numJobs,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		startRunCmd(m.scenarios, m.projectRoot, m.numJobs),
	)
}

// Results returns the final test results.
func (m Model) Results() []*harness.Result {
	return m.results
}

// ScenarioStates returns the final states of all scenarios.
func (m Model) ScenarioStates() []*ScenarioState {
	return m.scenarios
}
