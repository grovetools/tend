package e_runner

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/grovetools/core/tui/theme"
)

func (m Model) View() string {
	var header string
	if m.finished {
		header = fmt.Sprintf("Finished! Success: %d, Failed: %d, Total: %d",
			m.success, m.failed, len(m.projects))
	} else {
		header = fmt.Sprintf("Running ecosystem tests for %d projects... Running: %d, Success: %d, Failed: %d",
			len(m.projects), m.running, m.success, m.failed)
	}

	logViewStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(theme.DefaultTheme.Muted.GetForeground()).
		PaddingLeft(1)

	logView := logViewStyle.Render(m.logViewport.View())

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		m.list.View(),
		logView,
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		mainContent,
		"q to quit",
	)
}
