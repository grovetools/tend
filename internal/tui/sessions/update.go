package sessions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.ready = true
		}

		// Update list size (60% of width for list, 40% for preview)
		listWidth := int(float64(msg.Width) * 0.6)
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetSize(listWidth-h, msg.Height-v)

		// Update viewport size
		m.viewport.Width = msg.Width - listWidth - 4
		m.viewport.Height = msg.Height - 4

		return m, nil

	case sessionsListedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}

		m.sessions = msg.sessions

		// Convert sessions to list items
		var items []list.Item
		for _, sessionName := range msg.sessions {
			// Extract scenario name from session name
			// Format: tend_<workspace-id>_<scenario-name>
			parts := strings.Split(sessionName, "_")
			title := sessionName
			if len(parts) >= 2 {
				title = parts[len(parts)-1] // Last part is scenario name
			}

			desc := fmt.Sprintf("Session: %s", sessionName)

			items = append(items, item{
				sessionName: sessionName,
				title:       title,
				desc:        desc,
			})
		}

		m.list.SetItems(items)

		// If we have sessions, fetch preview for the first one
		if len(items) > 0 {
			return m, capturePaneCmd(msg.sessions[0])
		}

		return m, nil

	case previewCapturedMsg:
		if msg.err != nil {
			m.viewport.SetContent(fmt.Sprintf("Error capturing preview: %v", msg.err))
		} else {
			m.viewport.SetContent(msg.content)
		}
		return m, nil

	case sessionKilledMsg:
		if msg.err != nil {
			// Show error somehow, for now just refresh the list
			return m, listTendSessionsCmd
		}
		// Refresh the session list after killing
		return m, listTendSessionsCmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			// Switch to the selected session
			if selectedItem, ok := m.list.SelectedItem().(item); ok {
				sessionName := selectedItem.sessionName

				// Use tmux switch-client to switch to the session
				cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
				if err := cmd.Run(); err != nil {
					// If switch-client fails (not in tmux), try attach
					cmd = exec.Command("tmux", "attach", "-t", sessionName)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					_ = cmd.Run()
				}
				return m, tea.Quit
			}

		case "k", "K":
			// Kill the selected session
			if selectedItem, ok := m.list.SelectedItem().(item); ok {
				sessionName := selectedItem.sessionName
				return m, killSessionCmd(sessionName)
			}

		case "r":
			// Refresh session list
			return m, listTendSessionsCmd
		}
	}

	// Update list and handle selection changes
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	// When selection changes, update preview
	if _, ok := msg.(tea.KeyMsg); ok {
		if selectedItem, ok := m.list.SelectedItem().(item); ok {
			cmds = append(cmds, capturePaneCmd(selectedItem.sessionName))
		}
	}

	return m, tea.Batch(cmds...)
}
