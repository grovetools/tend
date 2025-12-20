// Package main implements a simple file saver TUI for testing filesystem interactions.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	saved    bool
	message  string
	quitting bool
}

func initialModel() model {
	return model{
		message: "Press 's' to save a file.",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "s":
			if !m.saved {
				// Save a file
				content := fmt.Sprintf("saved at %s", time.Now().Format(time.RFC3339))
				if err := os.WriteFile("output.txt", []byte(content), 0644); err != nil {
					m.message = fmt.Sprintf("Error saving file: %v", err)
				} else {
					m.saved = true
					m.message = "File saved to output.txt"
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	s := "File Saver\n"
	s += "==========\n\n"
	s += m.message + "\n\n"

	if m.saved {
		s += "Press 'q' to quit.\n"
	} else {
		s += "Press 's' to save, 'q' to quit.\n"
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
