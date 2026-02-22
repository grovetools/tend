package sessions

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/config"
	"github.com/grovetools/core/tui/theme"
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
	keyMap        KeyMap
}

// NewModel creates a new sessions TUI model.
func NewModel() (*Model, error) {
	// Load user-configurable keybindings
	cfg, _ := config.LoadDefault() // Ignore error - newKeyMap handles nil config gracefully
	keyMap := newKeyMap(cfg)

	// Create list with themed delegate
	delegate := list.NewDefaultDelegate()

	// Apply grove-core theme to the delegate
	delegate.Styles.SelectedTitle = theme.DefaultTheme.Selected.Copy()
	delegate.Styles.SelectedDesc = theme.DefaultTheme.Selected.Copy()
	delegate.Styles.NormalTitle = theme.DefaultTheme.Normal.Copy()
	delegate.Styles.NormalDesc = theme.DefaultTheme.Muted.Copy()
	delegate.Styles.DimmedTitle = theme.DefaultTheme.Muted.Copy()
	delegate.Styles.DimmedDesc = theme.DefaultTheme.Muted.Copy()
	delegate.Styles.FilterMatch = theme.DefaultTheme.Highlight.Copy()

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Test Sessions"
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)

	// Apply theme to list title and status bar
	l.Styles.Title = theme.DefaultTheme.Header.Copy()
	l.Styles.TitleBar = theme.DefaultTheme.Normal.Copy()
	l.Styles.StatusBar = theme.DefaultTheme.Muted.Copy()
	l.Styles.FilterPrompt = theme.DefaultTheme.Accent.Copy()
	l.Styles.FilterCursor = theme.DefaultTheme.Cursor.Copy()

	// Create viewport for preview pane
	vp := viewport.New(0, 0)

	m := &Model{
		list:     l,
		viewport: vp,
		sessions: []string{},
		keyMap:   keyMap,
	}

	return m, nil
}

// Init initializes the model and fetches sessions.
func (m *Model) Init() tea.Cmd {
	return listTendSessionsCmd
}
