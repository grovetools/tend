package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	grovelogging "github.com/grovetools/core/logging"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/pkg/tmux"
	"github.com/grovetools/tend/internal/tui/sessions"
	"github.com/spf13/cobra"
)

var ulogSessions = grovelogging.NewUnifiedLogger("grove-tend.sessions")

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
	sessionsCmd.AddCommand(newSessionsCleanupCmd())

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
				ulogSessions.Info("No active tend sessions found.").Pretty("No active tend sessions found.").PrettyOnly().Emit()
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
					ulogSessions.Info("No active tend sessions to kill.").Pretty("No active tend sessions to kill.").PrettyOnly().Emit()
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

// newSessionsCleanupCmd creates the "sessions cleanup" command to remove orphaned tmux servers.
func newSessionsCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up orphaned tend test tmux servers",
		Long: `Clean up orphaned tmux servers created by tend tests.

Tend tests create isolated tmux servers using sockets named "tend-test-*".
These servers are normally cleaned up automatically, but may remain if tests
are interrupted (Ctrl+C) or crash. This command finds and kills all orphaned
tend test tmux servers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			removeStale, _ := cmd.Flags().GetBool("remove-stale")

			// Get the tmux socket directory. Respect TMUX_TMPDIR/TMPDIR — the tmux
			// client uses the same precedence and on macOS the system TMPDIR is
			// not /tmp.
			tmpDir := os.Getenv("TMUX_TMPDIR")
			if tmpDir == "" {
				tmpDir = os.TempDir()
			}
			socketDir := filepath.Join(tmpDir, fmt.Sprintf("tmux-%d", os.Getuid()))

			// Check if directory exists
			entries, err := os.ReadDir(socketDir)
			if err != nil {
				if os.IsNotExist(err) {
					ulogSessions.Info("No tmux socket directory found.").
						Pretty("No tmux socket directory found.").
						PrettyOnly().Emit()
					return nil
				}
				return fmt.Errorf("failed to read socket directory: %w", err)
			}

			// Find all tend-test sockets
			// NOTE: We cannot reliably check if servers are running without potentially
			// auto-starting them (tmux auto-starts servers when you connect).
			// So we just collect all tend-test sockets and kill them.
			var tendSockets []string

			for _, entry := range entries {
				// Match both legacy "tend-test-" sockets and the new short "tt-" prefix.
				name := entry.Name()
				if strings.HasPrefix(name, "tend-test-") || strings.HasPrefix(name, "tt-") {
					tendSockets = append(tendSockets, name)
				}
			}

			if len(tendSockets) == 0 {
				ulogSessions.Info("No tend test servers or sockets found.").
					Pretty("No tend test servers or sockets found.").
					PrettyOnly().Emit()
				return nil
			}

			ulogSessions.Info("Found tend test sockets").
				Field("count", len(tendSockets)).
				Pretty(fmt.Sprintf("Found %d tend test sockets to clean up:", len(tendSockets))).
				Emit()

			// Kill servers and optionally remove socket files
			var killedCount, removedCount int
			for _, socketName := range tendSockets {
				socketPath := filepath.Join(socketDir, socketName)

				if dryRun {
					fmt.Printf("  [dry-run] Would kill server and remove socket: %s\n", socketName)
					continue
				}

				// Try to kill the server (if running)
				client, err := tmux.NewClientWithSocket(socketName)
				if err == nil {
					if err := client.KillServer(cmd.Context()); err == nil {
						killedCount++
						fmt.Printf("  Killed server: %s\n", socketName)
					}
				}

				// Remove the socket file if requested
				if removeStale {
					if err := os.Remove(socketPath); err == nil {
						removedCount++
						fmt.Printf("  Removed socket file: %s\n", socketName)
					}
				}
			}

			if !dryRun {
				if removeStale {
					ulogSessions.Success("Cleanup complete").
						Field("killed", killedCount).
						Field("removed", removedCount).
						Pretty(fmt.Sprintf("Cleanup complete: killed %d servers, removed %d socket files", killedCount, removedCount)).
						PrettyOnly().Emit()
				} else {
					ulogSessions.Success("Cleanup complete").
						Field("killed", killedCount).
						Pretty(fmt.Sprintf("Cleanup complete: killed %d servers. Use --remove-stale to also remove socket files.", killedCount)).
						PrettyOnly().Emit()
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("dry-run", false, "Show what would be cleaned up without actually doing it")
	cmd.Flags().Bool("remove-stale", false, "Also remove stale socket files where servers are not running")
	return cmd
}
