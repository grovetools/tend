package runner

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-core/tui/components/table"
	"github.com/mattsolo1/grove-core/tui/theme"
)

// highlightMatch highlights the filter text within the name
func highlightMatch(text, filter string) string {
	if filter == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerFilter := strings.ToLower(filter)
	idx := strings.Index(lowerText, lowerFilter)

	if idx == -1 {
		return text
	}

	// Highlight the matched portion
	before := text[:idx]
	match := text[idx : idx+len(filter)]
	after := text[idx+len(filter):]

	highlightStyle := theme.DefaultTheme.Success.Copy().Reverse(true)
	return before + highlightStyle.Render(match) + after
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	if m.isLoading {
		return "Loading tests..."
	}
	if !m.ready {
		return "Initializing..."
	}
	if m.help.ShowAll {
		return m.help.View()
	}

	// Header
	header := "Tend Test Runner"
	if m.focusedProject != nil {
		header = fmt.Sprintf("Focus: %s", m.focusedProject.Name)
	}
	headerView := theme.DefaultTheme.Header.Render(header)

	// Search input
	searchView := m.filterInput.View()

	// Main content: full-width scrollable list
	// Calculate viewport boundaries
	viewHeight := m.getVisibleNodeCount()
	start := m.scrollOffset
	end := m.scrollOffset + viewHeight
	if end > len(m.displayNodes) {
		end = len(m.displayNodes)
	}

	// Render only visible nodes
	var listRows [][]string
	if start < end {
		for i := start; i < end; i++ {
			node := m.displayNodes[i]
			var name string
			var style lipgloss.Style
			if node.IsEcosystem {
				name = node.Project.Name
				style = theme.DefaultTheme.Header
			} else if node.IsProject {
				name = node.Project.Name
				style = theme.DefaultTheme.Bold
			} else if node.IsFile {
				name = filepath.Base(node.FilePath)
				style = theme.DefaultTheme.Muted
			} else if node.IsScenario {
				if node.Scenario != nil {
					name = node.Scenario.Name
				}
				style = lipgloss.NewStyle()
			}

			// Skip nodes with empty names (shouldn't happen, but defensive)
			if name == "" || name == "." {
				name = "<unnamed>"
			}

			// Apply prefix for tree structure
			fullName := node.Prefix + name

			// Apply highlighting if there's a filter
			if m.filterInput.Value() != "" {
				// Highlight just the name part, not the prefix
				highlightedName := highlightMatch(name, m.filterInput.Value())
				fullName = node.Prefix + highlightedName
			}

			// Apply styling
			styledName := style.Render(fullName)

			listRows = append(listRows, []string{styledName})
		}
	}

	// Create the selectable table for the list (full width)
	listWidth := m.width - 4 // Use full width minus some padding

	// Adjust cursor for SelectableTable, which only sees the visible slice
	relativeCursor := m.cursor - m.scrollOffset

	listView := lipgloss.NewStyle().Width(listWidth).Render(
		table.SelectableTable(nil, listRows, relativeCursor),
	)

	// Footer
	var footer string
	if m.statusMessage != "" && time.Now().Before(m.statusTimeout) {
		footer = theme.DefaultTheme.Success.Render(m.statusMessage)
	} else {
		footer = m.help.View()
	}

	fullView := lipgloss.JoinVertical(lipgloss.Left,
		headerView,
		searchView,
		"",
		listView,
		"",
		footer,
	)

	// Add top margin to prevent border cutoff
	return "\n" + fullView
}
