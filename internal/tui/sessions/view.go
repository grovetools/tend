package sessions

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/grovetools/core/tui/theme"
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

	// Apply left margin to the entire view
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		combined,
		footer,
	)

	return lipgloss.NewStyle().MarginLeft(2).Render(content)
}

// renderEmptyState shows the empty state when no sessions are found.
func (m *Model) renderEmptyState() string {
	emptyStyle := theme.DefaultTheme.Muted.Italic(true).Align(lipgloss.Center)
	content := emptyStyle.Render(
		"\n\n" +
			"No active test sessions\n\n" +
			"Use 'D' in tend tui to launch a debug session\n" +
			"Or run: tend run <scenario> --debug-session\n\n",
	)

	footer := theme.DefaultTheme.Muted.Render("q: quit  •  r: refresh")

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
	title := theme.DefaultTheme.Header.Render("Preview")

	// Get selected session name for title
	if selectedItem, ok := m.list.SelectedItem().(item); ok {
		title = theme.DefaultTheme.Header.Render(fmt.Sprintf("Preview: %s", selectedItem.title))
	}

	content := m.viewport.View()
	if strings.TrimSpace(content) == "" {
		emptyStyle := theme.DefaultTheme.Muted.Italic(true).Align(lipgloss.Center)
		content = emptyStyle.Render("\n\nNo preview available\n\n")
	}

	preview := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		strings.Repeat("─", m.viewport.Width),
		content,
	)

	borderStyle := theme.DefaultTheme.Box.
		Width(m.viewport.Width + 2).
		Height(m.viewport.Height + 4)

	return borderStyle.Render(preview)
}

// renderFooter renders the help footer.
func (m *Model) renderFooter() string {
	var parts []string

	// Status indicators legend
	legend := fmt.Sprintf("%s running  %s paused  %s idle",
		theme.DefaultTheme.Success.Render("●"),
		theme.DefaultTheme.Warning.Render("◐"),
		theme.DefaultTheme.Muted.Render("○"),
	)
	parts = append(parts, legend)

	// Key bindings
	bindings := "enter: switch  •  x: kill  •  r: refresh  •  q: quit"
	parts = append(parts, bindings)

	footer := theme.DefaultTheme.Muted.Render(strings.Join(parts, "  •  "))

	return "\n" + footer
}
