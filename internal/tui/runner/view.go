package runner

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/grovetools/core/tui/components/table"
	"github.com/grovetools/core/tui/theme"
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

			// 1. Status Icon
			status, exists := m.testStatuses[node.ID()]
			if !exists {
				status = StatusNotRun
			}
			statusIcon := "  " // Default for not-runnable
			switch {
			case node.IsScenario, node.IsFile, node.IsProject && !node.IsEcosystem:
				switch status {
				case StatusRunning:
					statusIcon = theme.DefaultTheme.Warning.Render("~ ")
				case StatusPassed:
					statusIcon = theme.DefaultTheme.Success.Render("* ")
				case StatusFailed:
					statusIcon = theme.DefaultTheme.Error.Render("x ")
				case StatusNotRun:
					statusIcon = "  "
				}
			}

			// Add indicators for scenarios
			var indicatorStr string
			if node.IsScenario && node.Scenario != nil {
				var indicators []string
				if node.Scenario.LocalOnly {
					indicators = append(indicators, theme.DefaultTheme.Muted.Render("[L]"))
				}
				if node.Scenario.ExplicitOnly {
					indicators = append(indicators, theme.DefaultTheme.Warning.Render("[E]"))
				}
				if len(indicators) > 0 {
					indicatorStr = strings.Join(indicators, " ") + " "
				}
			}

			// Apply highlighting if there's a filter
			var highlightedName string
			if m.filterInput.Value() != "" {
				// Highlight just the name part, not the prefix
				highlightedName = highlightMatch(name, m.filterInput.Value())
			} else {
				highlightedName = name
			}

			// Apply styling, but don't re-style the indicators which already have styles
			styledName := statusIcon + style.Render(node.Prefix) + indicatorStr + style.Render(highlightedName)

			listRows = append(listRows, []string{styledName})
		}
	}

	// Create the selectable table for the list
	// Width depends on whether output pane is visible
	listWidth := m.width - 4
	if m.outputVisible {
		listWidth = m.width/2 - 4
	}

	// Adjust cursor for SelectableTable, which only sees the visible slice
	relativeCursor := m.cursor - m.scrollOffset

	listView := lipgloss.NewStyle().Width(listWidth).Render(
		table.SelectableTable(nil, listRows, relativeCursor),
	)

	// Build main content area (potentially with output pane on right)
	var mainContent string
	if m.outputVisible {
		// Output pane title
		outputTitle := "Test Output"
		if m.testRunning {
			outputTitle = "Test Output (running...)"
		} else {
			outputTitle = "Test Output (press 'esc' to close)"
		}

		outputPane := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(m.width/2 - 2).
			Height(m.height - 10).
			Render(theme.DefaultTheme.Muted.Render(outputTitle) + "\n" + m.outputPane.View())

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, listView, outputPane)
	} else {
		mainContent = listView
	}

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
		mainContent,
		"",
		footer,
	)

	// Add top margin to prevent border cutoff
	return "\n" + fullView
}
