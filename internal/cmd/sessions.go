package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/mattsolo1/grove-core/pkg/tmux"
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

	// Add subcommands
	sessionsCmd.AddCommand(newSessionsListCmd())
	sessionsCmd.AddCommand(newSessionsKillCmd())
	sessionsCmd.AddCommand(newSessionsCaptureCmd())
	sessionsCmd.AddCommand(newSessionsSendKeysCmd())
	sessionsCmd.AddCommand(newSessionsAttachCmd())

	return sessionsCmd
}

// newSessionsListCmd creates the "sessions list" command.
func newSessionsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active tend debug sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionNames, err := sessions.ListTendSessions()
			if err != nil {
				return err
			}
			if len(sessionNames) == 0 {
				fmt.Println("No active tend sessions found.")
				return nil
			}
			for _, name := range sessionNames {
				fmt.Println(name)
			}
			return nil
		},
	}
}

// newSessionsKillCmd creates the "sessions kill" command.
func newSessionsKillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill [session-name...]",
		Short: "Kill one or more tend sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			client, err := tmux.NewClient()
			if err != nil {
				return err
			}

			var sessionsToKill []string
			if all {
				sessionsToKill, err = sessions.ListTendSessions()
				if err != nil {
					return err
				}
				if len(sessionsToKill) == 0 {
					fmt.Println("No active tend sessions to kill.")
					return nil
				}
			} else {
				if len(args) == 0 {
					return fmt.Errorf("at least one session name is required unless --all is specified")
				}
				sessionsToKill = args
			}

			for _, sessionName := range sessionsToKill {
				if err := client.KillSession(cmd.Context(), sessionName); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to kill session %s: %v\n", sessionName, err)
				} else {
					fmt.Printf("Killed session: %s\n", sessionName)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Kill all active tend sessions")
	return cmd
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Match ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}

// newSessionsCaptureCmd creates the "sessions capture" command.
func newSessionsCaptureCmd() *cobra.Command {
	var withAnsi bool
	var waitFor string
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "capture <session-target>",
		Short: "Capture the contents of a tmux pane",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			client, err := tmux.NewClient()
			if err != nil {
				return err
			}

			// If --wait-for is specified, poll until text appears or timeout
			if waitFor != "" {
				deadline := time.Now().Add(timeout)
				for time.Now().Before(deadline) {
					content, err := client.CapturePane(cmd.Context(), target)
					if err != nil {
						return err
					}

					checkContent := content
					if !withAnsi {
						checkContent = stripANSI(content)
					}

					if strings.Contains(checkContent, waitFor) {
						// Found it! Print and return
						if withAnsi {
							fmt.Print(content)
						} else {
							fmt.Print(stripANSI(content))
						}
						return nil
					}

					time.Sleep(200 * time.Millisecond)
				}
				return fmt.Errorf("timeout waiting for text: %q", waitFor)
			}

			// Normal capture without waiting
			content, err := client.CapturePane(cmd.Context(), target)
			if err != nil {
				return err
			}

			if withAnsi {
				fmt.Print(content)
			} else {
				fmt.Print(stripANSI(content))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&withAnsi, "with-ansi", false, "Preserve ANSI escape codes in output (default: strip)")
	cmd.Flags().StringVar(&waitFor, "wait-for", "", "Wait for text to appear before capturing")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "Timeout when using --wait-for")

	return cmd
}

// newSessionsSendKeysCmd creates the "sessions send-keys" command.
func newSessionsSendKeysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send-keys <session-target> -- [keys...]",
		Short: "Send keystrokes to a tmux pane",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			var keys []string
			dashIdx := cmd.Flags().ArgsLenAtDash()
			if dashIdx != -1 {
				keys = args[dashIdx:]
			} else {
				keys = args[1:]
			}

			if len(keys) == 0 {
				return fmt.Errorf("no keys provided to send")
			}

			client, err := tmux.NewClient()
			if err != nil {
				return err
			}
			return client.SendKeys(cmd.Context(), target, keys...)
		},
	}
}

// newSessionsAttachCmd creates the "sessions attach" command.
func newSessionsAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <session-name>",
		Short: "Attach to a running tend session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]
			tmuxPath, err := exec.LookPath("tmux")
			if err != nil {
				return fmt.Errorf("tmux executable not found in PATH")
			}

			var tmuxArgs []string
			if os.Getenv("TMUX") != "" {
				// Inside tmux, switch client
				tmuxArgs = []string{"tmux", "switch-client", "-t", sessionName}
			} else {
				// Outside tmux, attach
				tmuxArgs = []string{"tmux", "attach-session", "-t", sessionName}
			}

			// Replace the current process with tmux
			if err := syscall.Exec(tmuxPath, tmuxArgs, os.Environ()); err != nil {
				return fmt.Errorf("failed to exec tmux: %w", err)
			}
			return nil // This line is never reached
		},
	}
}
