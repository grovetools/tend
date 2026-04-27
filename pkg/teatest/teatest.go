package teatest

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// HeadlessSession provides a simple headless testing interface for BubbleTea models.
type HeadlessSession struct {
	model tea.Model
	msgs  []tea.Msg
}

// NewHeadlessSession creates a new headless session for a BubbleTea model.
func NewHeadlessSession(m tea.Model) *HeadlessSession {
	_ = m.Init()

	return &HeadlessSession{
		model: m,
		msgs:  []tea.Msg{},
	}
}

// Send sends a message to the model's Update function.
func (s *HeadlessSession) Send(msg tea.Msg) {
	s.msgs = append(s.msgs, msg)
	newModel, cmd := s.model.Update(msg)
	s.model = newModel

	// In a real implementation, we'd handle the Cmd here
	_ = cmd
}

// TypeString sends a string of characters to the model.
func (s *HeadlessSession) TypeString(str string) {
	for _, ch := range str {
		s.Send(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{ch},
		})
	}
}

// Output returns the current string output from the model's View() method.
func (s *HeadlessSession) Output() string {
	return s.model.View()
}

// Wait a short duration for the model to process messages.
func (s *HeadlessSession) Wait() {
	time.Sleep(100 * time.Millisecond) // Allow time for async operations if any
}
