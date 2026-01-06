package e_runner

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-core/tui/theme"
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
	s, ok := item.(*ProjectState)
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
		if s.result != nil {
			durationStr = theme.DefaultTheme.Muted.Render(
				fmt.Sprintf("(%v | %s %d %s %d)", s.duration.Round(time.Millisecond), theme.IconSuccess, s.result.Passed, theme.IconError, s.result.Failed),
			)
		} else {
			durationStr = theme.DefaultTheme.Muted.Render(fmt.Sprintf("(%v)", s.duration.Round(time.Millisecond)))
		}
	case StatusFailure:
		statusIcon = theme.DefaultTheme.Error.Render(theme.IconError)
		if s.result != nil {
			durationStr = theme.DefaultTheme.Muted.Render(
				fmt.Sprintf("(%v | %s %d %s %d)", s.duration.Round(time.Millisecond), theme.IconSuccess, s.result.Passed, theme.IconError, s.result.Failed),
			)
		} else {
			durationStr = theme.DefaultTheme.Muted.Render(fmt.Sprintf("(%v)", s.duration.Round(time.Millisecond)))
		}
	}

	line := fmt.Sprintf(" %s %s %s", statusIcon, s.projectName, durationStr)
	fmt.Fprint(w, line)
}
