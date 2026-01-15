// Package main implements a simple task manager TUI for testing conditional flows.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateMenu state = iota
	stateProcessing
	stateResult
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")). // Blue
			MarginBottom(1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Bright blue
			PaddingLeft(2)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")). // Green
			PaddingLeft(2)

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")). // Yellow
			PaddingLeft(2)

	detailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")). // Cyan
			PaddingLeft(4)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Gray
			Italic(true)
)

type model struct {
	state     state
	choice    string
	result    string
	quitting  bool
	startTime time.Time
}

type processingDoneMsg struct {
	result string
}

func initialModel() model {
	return model{
		state: stateMenu,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "1":
				m.choice = "1"
				m.state = stateProcessing
				m.startTime = time.Now()
				return m, func() tea.Msg {
					time.Sleep(300 * time.Millisecond)
					return processingDoneMsg{
						result: "* Success: All files processed\nFound 15 files modified\nFound 3 files added",
					}
				}
			case "2":
				m.choice = "2"
				m.state = stateProcessing
				m.startTime = time.Now()
				return m, func() tea.Msg {
					time.Sleep(300 * time.Millisecond)
					return processingDoneMsg{
						result: "* Success: All tests passed\nTest Suite: Unit Tests\nTests: 42 passed, 0 failed",
					}
				}
			case "3":
				m.choice = "3"
				m.state = stateProcessing
				m.startTime = time.Now()
				return m, func() tea.Msg {
					time.Sleep(300 * time.Millisecond)
					return processingDoneMsg{
						result: "WARNING: Warning: Low disk space",
					}
				}
			}
		case stateResult:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case processingDoneMsg:
		m.result = msg.result
		m.state = stateResult
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var s string

	switch m.state {
	case stateMenu:
		s += titleStyle.Render("Task Manager v1.0") + "\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("=================") + "\n\n"
		s += "Select action:\n"
		s += menuItemStyle.Render("1) Process files") + "\n"
		s += menuItemStyle.Render("2) Run tests") + "\n"
		s += menuItemStyle.Render("3) Check status") + "\n\n"
		s += promptStyle.Render("Choice: ")

	case stateProcessing:
		s += titleStyle.Render("Task Manager v1.0") + "\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("=================") + "\n\n"

		var action string
		switch m.choice {
		case "1":
			action = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("Processing files...")
		case "2":
			action = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("Running tests...")
		case "3":
			action = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("Checking status...")
		}
		s += action + "\n"

	case stateResult:
		s += titleStyle.Render("Task Manager v1.0") + "\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("=================") + "\n\n"

		// Parse and colorize the result
		if m.choice == "1" {
			s += successStyle.Render("* Success: All files processed") + "\n"
			s += detailStyle.Render("Found 15 files modified") + "\n"
			s += detailStyle.Render("Found 3 files added") + "\n"
		} else if m.choice == "2" {
			s += successStyle.Render("* Success: All tests passed") + "\n"
			s += detailStyle.Render("Test Suite: Unit Tests") + "\n"
			s += detailStyle.Render("Tests: 42 passed, 0 failed") + "\n"
		} else if m.choice == "3" {
			s += warningStyle.Render("WARNING: Warning: Low disk space") + "\n"
		}

		s += "\n" + promptStyle.Render("Press 'q' to quit.") + "\n"
	}

	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
