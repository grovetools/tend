package sessions

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/pkg/mux"
)

type sessionsListedMsg struct {
	sessions []string
	err      error
}

type previewCapturedMsg struct {
	sessionName string
	content     string
	err         error
}

type sessionKilledMsg struct {
	sessionName string
	err         error
}

// ListTendSessions fetches all tend debug sessions via the active mux engine.
func ListTendSessions() ([]string, error) {
	engine, err := mux.DetectMuxEngine(context.Background())
	if err != nil {
		return nil, fmt.Errorf("detecting mux engine: %w", err)
	}

	allSessions, err := engine.ListSessions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	var tendSessions []string
	for _, s := range allSessions {
		if strings.HasPrefix(s.Name, "tend_") || strings.HasPrefix(s.Name, "tend-tui-") {
			tendSessions = append(tendSessions, s.Name)
		}
	}

	return tendSessions, nil
}

func listTendSessionsCmd() tea.Msg {
	sessions, err := ListTendSessions()
	if err != nil {
		return sessionsListedMsg{err: err}
	}
	return sessionsListedMsg{sessions: sessions, err: nil}
}

func capturePaneCmd(sessionName string) tea.Cmd {
	return func() tea.Msg {
		engine, err := mux.DetectMuxEngine(context.Background())
		if err != nil {
			return previewCapturedMsg{
				sessionName: sessionName,
				err:         fmt.Errorf("detecting mux engine: %w", err),
			}
		}

		target := fmt.Sprintf("%s:runner", sessionName)
		content, err := engine.CapturePane(context.Background(), target)
		if err != nil {
			target = fmt.Sprintf("%s:0", sessionName)
			content, err = engine.CapturePane(context.Background(), target)
			if err != nil {
				return previewCapturedMsg{
					sessionName: sessionName,
					err:         fmt.Errorf("capture pane: %w", err),
				}
			}
		}

		return previewCapturedMsg{
			sessionName: sessionName,
			content:     content,
		}
	}
}

func killSessionCmd(sessionName string) tea.Cmd {
	return func() tea.Msg {
		engine, err := mux.DetectMuxEngine(context.Background())
		if err != nil {
			return sessionKilledMsg{
				sessionName: sessionName,
				err:         fmt.Errorf("detecting mux engine: %w", err),
			}
		}

		err = engine.KillSession(context.Background(), sessionName)
		return sessionKilledMsg{
			sessionName: sessionName,
			err:         err,
		}
	}
}
