package sessions

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// item represents a single session in our list.
type item struct {
	sessionName string
	title       string
	desc        string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Model represents the state of the sessions TUI.
type Model struct {
	list          list.Model
	viewport      viewport.Model
	sessions      []string
	width         int
	height        int
	ready         bool
	err           error
	previewActive bool
}

// NewModel creates a new sessions TUI model.
func NewModel() (*Model, error) {
	// Create list with default delegate
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Test Sessions"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)

	// Create viewport for preview pane
	vp := viewport.New(0, 0)

	m := &Model{
		list:     l,
		viewport: vp,
		sessions: []string{},
	}

	return m, nil
}

// Init initializes the model and fetches sessions.
func (m *Model) Init() tea.Cmd {
	return listTendSessionsCmd
}
