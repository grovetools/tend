package sessions

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	// Border styles
	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	// Header style
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Padding(0, 1)

	// Status indicator styles
	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	pausedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
	idleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // Gray

	// Empty state style
	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Align(lipgloss.Center)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)
)

// View renders the sessions TUI.
func (m *Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\n", m.err)
	}

	if !m.ready {
		return "\n  Loading sessions...\n"
	}

	if len(m.sessions) == 0 {
		return m.renderEmptyState()
	}

	// Split view: list on left, preview on right
	listView := m.list.View()
	previewView := m.renderPreview()

	// Combine list and preview side by side
	combined := lipgloss.JoinHorizontal(
		lipgloss.Top,
		listView,
		previewView,
	)

	// Add footer with help text
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		combined,
		footer,
	)
}

// renderEmptyState shows the empty state when no sessions are found.
func (m *Model) renderEmptyState() string {
	content := emptyStyle.Render(
		"\n\n" +
			"No active test sessions\n\n" +
			"Use 'D' in tend tui to launch a debug session\n" +
			"Or run: tend run <scenario> --debug-session\n\n",
	)

	footer := helpStyle.Render("q: quit  •  r: refresh")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		"\n\n",
		content,
		"\n",
		footer,
	)
}

// renderPreview renders the preview pane.
func (m *Model) renderPreview() string {
	title := headerStyle.Render("Preview")

	// Get selected session name for title
	if selectedItem, ok := m.list.SelectedItem().(item); ok {
		title = headerStyle.Render(fmt.Sprintf("Preview: %s", selectedItem.title))
	}

	content := m.viewport.View()
	if strings.TrimSpace(content) == "" {
		content = emptyStyle.Render("\n\nNo preview available\n\n")
	}

	preview := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		strings.Repeat("─", m.viewport.Width),
		content,
	)

	return borderStyle.
		Width(m.viewport.Width + 2).
		Height(m.viewport.Height + 4).
		Render(preview)
}

// renderFooter renders the help footer.
func (m *Model) renderFooter() string {
	var parts []string

	// Status indicators legend
	legend := fmt.Sprintf("%s running  %s paused  %s idle",
		runningStyle.Render("●"),
		pausedStyle.Render("◐"),
		idleStyle.Render("○"),
	)
	parts = append(parts, legend)

	// Key bindings
	bindings := "enter: switch  •  k: kill  •  r: refresh  •  q: quit"
	parts = append(parts, bindings)

	footer := helpStyle.Render(strings.Join(parts, "  •  "))

	return "\n" + footer
}
