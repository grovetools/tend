package prunner

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/tui/theme"
)

type delegate struct {
	spinner spinner.Model
}

func newDelegate() delegate {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.DefaultTheme.Highlight
	return delegate{spinner: s}
}

func (d delegate) Height() int                               { return 1 }
func (d delegate) Spacing() int                              { return 0 }
func (d delegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	s, ok := item.(*ScenarioState)
	if !ok {
		return
	}

	var statusIcon, durationStr string

	switch s.status {
	case StatusPending:
		statusIcon = theme.DefaultTheme.Muted.Render(theme.IconPending)
	case StatusRunning:
		statusIcon = d.spinner.View()
	case StatusSuccess:
		statusIcon = theme.DefaultTheme.Success.Render(theme.IconSuccess)
		durationStr = theme.DefaultTheme.Muted.Render(fmt.Sprintf("(%v)", s.duration.Round(time.Millisecond)))
	case StatusFailure:
		statusIcon = theme.DefaultTheme.Error.Render(theme.IconError)
		durationStr = theme.DefaultTheme.Muted.Render(fmt.Sprintf("(%v)", s.duration.Round(time.Millisecond)))
	}

	line := fmt.Sprintf(" %s %s %s", statusIcon, s.scenario.Name, durationStr)
	fmt.Fprint(w, line)
}
