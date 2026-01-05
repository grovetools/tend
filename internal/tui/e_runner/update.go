package e_runner

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type quitMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		listWidth := msg.Width / 2
		logWidth := msg.Width - listWidth
		listHeight := msg.Height - 4 // Account for header/footer

		m.list.SetSize(listWidth, listHeight)
		m.logViewport.Width = logWidth - 2 // for border
		m.logViewport.Height = listHeight
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
			m.projects[msg.Index].status = StatusRunning
			m.running++
		case "output":
			// Prepend the project name to the incoming line
			line := fmt.Sprintf("[%s] %s", m.projects[msg.Index].projectName, msg.OutputLine)
			m.logLines = append(m.logLines, line)
			if len(m.logLines) > m.maxLogLines {
				m.logLines = m.logLines[len(m.logLines)-m.maxLogLines:]
			}
			// Only update live logs if a running/pending item is selected
			if selected, ok := m.list.SelectedItem().(*ProjectState); ok {
				if selected.status == StatusRunning || selected.status == StatusPending {
					m.logViewport.SetContent(strings.Join(m.logLines, "\n"))
					m.logViewport.GotoBottom()
				}
			}
		case "finish":
			state := m.projects[msg.Index]
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
			// If the finished project is the one currently selected, show its full output
			if m.list.Index() == msg.Index {
				m.logViewport.SetContent(state.output)
			}
		}
		items := m.list.Items()
		items[msg.Index] = m.projects[msg.Index]
		m.list.SetItems(items)

		if m.success+m.failed == len(m.projects) {
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

	// Store previous index to check for selection changes
	prevIndex := m.list.Index()

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	// When selection changes, update the log viewport
	if m.list.Index() != prevIndex {
		if selected, ok := m.list.SelectedItem().(*ProjectState); ok {
			if selected.status == StatusSuccess || selected.status == StatusFailure {
				m.logViewport.SetContent(selected.output)
			} else {
				m.logViewport.SetContent(strings.Join(m.logLines, "\n"))
				m.logViewport.GotoBottom()
			}
		}
	}

	return m, cmd
}
