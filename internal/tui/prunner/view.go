package prunner

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var header string
	if m.finished {
		header = fmt.Sprintf("Finished! Success: %d, Failed: %d, Total: %d",
			m.success, m.failed, len(m.scenarios))
	} else {
		header = fmt.Sprintf("Running %d scenarios... Running: %d, Success: %d, Failed: %d",
			len(m.scenarios), m.running, m.success, m.failed)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.list.View(),
		"q to quit",
	)
}
