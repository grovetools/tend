package sessions

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/pkg/tmux"
)

// sessionsListedMsg contains the list of tend sessions found.
type sessionsListedMsg struct {
	sessions []string
	err      error
}

// previewCapturedMsg contains the preview content for a session.
type previewCapturedMsg struct {
	sessionName string
	content     string
	err         error
}

// sessionKilledMsg indicates a session was killed.
type sessionKilledMsg struct {
	sessionName string
	err         error
}

// ListTendSessions fetches all tend debug sessions from tmux.
func ListTendSessions() ([]string, error) {
	// Try main server first
	client, err := tmux.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux client: %w", err)
	}

	allSessions, err := client.ListSessions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Filter for tend sessions (those starting with "tend_")
	var tendSessions []string
	for _, sessionName := range allSessions {
		if strings.HasPrefix(sessionName, "tend_") {
			tendSessions = append(tendSessions, sessionName)
		}
	}

	// TODO: Also check dedicated server (tend-debug) for sessions
	// This would require checking if the dedicated server exists and listing its sessions

	return tendSessions, nil
}

// listTendSessionsCmd fetches all tend debug sessions from tmux.
func listTendSessionsCmd() tea.Msg {
	sessions, err := ListTendSessions()
	if err != nil {
		return sessionsListedMsg{err: err}
	}
	return sessionsListedMsg{sessions: sessions, err: nil}
}

// capturePaneCmd captures the content of a session's runner window.
func capturePaneCmd(sessionName string) tea.Cmd {
	return func() tea.Msg {
		client, err := tmux.NewClient()
		if err != nil {
			return previewCapturedMsg{
				sessionName: sessionName,
				err:         fmt.Errorf("failed to create tmux client: %w", err),
			}
		}

		// Capture the runner window (assumed to be window 0 or named "runner")
		target := fmt.Sprintf("%s:runner", sessionName)
		content, err := client.CapturePane(context.Background(), target)
		if err != nil {
			// Try window 0 if named window doesn't exist
			target = fmt.Sprintf("%s:0", sessionName)
			content, err = client.CapturePane(context.Background(), target)
			if err != nil {
				return previewCapturedMsg{
					sessionName: sessionName,
					err:         fmt.Errorf("failed to capture pane: %w", err),
				}
			}
		}

		return previewCapturedMsg{
			sessionName: sessionName,
			content:     content,
			err:         nil,
		}
	}
}

// killSessionCmd kills a tmux session.
func killSessionCmd(sessionName string) tea.Cmd {
	return func() tea.Msg {
		client, err := tmux.NewClient()
		if err != nil {
			return sessionKilledMsg{
				sessionName: sessionName,
				err:         fmt.Errorf("failed to create tmux client: %w", err),
			}
		}

		err = client.KillSession(context.Background(), sessionName)
		return sessionKilledMsg{
			sessionName: sessionName,
			err:         err,
		}
	}
}
