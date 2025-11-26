package cmd

import (
	"fmt"
	"os"

	"github.com/mattsolo1/grove-tend/internal/tui/sessions"
	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
)

// newSessionsCmd creates the sessions command.
func newSessionsCmd() *cobra.Command {
	sessionsCmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage tend test sessions",
		Long: `Manage and navigate tend debug test sessions.

This command launches a TUI for listing, previewing, and managing tend debug sessions
created with 'tend run --debug-session'.

Examples:
  tend sessions              # List sessions for current workspace
  tend sessions --all        # List sessions for all workspaces`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := sessions.NewModel()
			if err != nil {
				return fmt.Errorf("failed to initialize sessions model: %w", err)
			}

			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
				return err
			}
			return nil
		},
	}

	return sessionsCmd
}
