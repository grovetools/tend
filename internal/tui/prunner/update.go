package prunner

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type quitMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case <-chan Event:
		m.eventsChan = msg
		return m, waitForEventCmd(m.eventsChan)

	case Event:
		switch msg.Type {
		case "start":
			m.scenarios[msg.Index].status = StatusRunning
			m.running++
		case "finish":
			state := m.scenarios[msg.Index]
			state.result = msg.Result
			state.duration = msg.Result.Duration
			state.output = msg.Output
			m.running--
			if msg.Result.Success {
				state.status = StatusSuccess
				m.success++
			} else {
				state.status = StatusFailure
				m.failed++
			}
			m.results = append(m.results, msg.Result)
		}
		items := m.list.Items()
		items[msg.Index] = m.scenarios[msg.Index]
		m.list.SetItems(items)

		if m.success+m.failed == len(m.scenarios) {
			m.finished = true
			// Wait a moment for the final view to render before quitting
			return m, func() tea.Msg {
				time.Sleep(100 * time.Millisecond)
				return quitMsg{}
			}
		}
		return m, waitForEventCmd(m.eventsChan)

	case nil: // Channel closed
		m.finished = true
		// Wait a moment for the final view to render before quitting
		return m, func() tea.Msg {
			time.Sleep(100 * time.Millisecond)
			return quitMsg{}
		}

	case quitMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
