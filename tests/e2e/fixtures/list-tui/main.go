// Package main implements a simple bubbletea list navigator for testing.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	items    []string
	cursor   int
	selected string
	quitting bool
}

type tickMsg time.Time

var allItems = []string{
	"README.md",
	"main.go",
	"docs/guide.md",
}

func initialModel() model {
	return model{
		items: []string{},
	}
}

func (m model) Init() tea.Cmd {
	// Start the loading process
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.items) {
				m.selected = m.items[m.cursor]
			}
		}

	case tickMsg:
		if len(m.items) < len(allItems) {
			m.items = append(m.items, allItems[len(m.items)])
			return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	s := "File Browser\n"
	s += "============\n\n"

	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, item)
	}

	if m.selected != "" {
		s += fmt.Sprintf("\nSelected: %s\n", m.selected)
	}

	s += "\nUse arrow keys to navigate. Press 'enter' to select. Press 'q' to quit.\n"
	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
