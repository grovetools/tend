package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-core/pkg/workspace"
	"github.com/mattsolo1/grove-tend/internal/tui/runner"
	"github.com/mattsolo1/grove-tend/pkg/recorder"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// newTuiCmd creates the `tui` subcommand.
func newTuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch an interactive TUI for browsing and running tests",
		Long: `Launch an interactive Terminal User Interface to browse, manage,
and run 'tend' test scenarios across the ecosystem.

If no subcommand is given, this will launch the test runner TUI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// This RunE is executed if no subcommand is specified.
			// Determine initial focus path based on current directory
			initialFocusPath := findInitialWorkspacePath()

			m := runner.New(initialFocusPath)
			p := tea.NewProgram(m, tea.WithAltScreen())

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running TUI: %w", err)
			}
			return nil
		},
	}

	// Add record subcommand
	cmd.AddCommand(newRecordCmd())

	return cmd
}

// newRecordCmd creates the `tui record` subcommand.
func newRecordCmd() *cobra.Command {
	var outputFile string
	recordCmd := &cobra.Command{
		Use:   "record [--out file.html] -- <command...>",
		Short: "Record a manual TUI session to an HTML file",
		Long: `Launches a command within a recordable sub-shell. All keystrokes and
terminal output are captured and saved to an interactive HTML report.

This is useful for creating a shareable, replayable recording of a TUI session,
often for providing context to an LLM for writing automated tests.

Use '--' to separate the recorder's flags from the command you want to record.
If no command is provided, it will default to launching your default shell ($SHELL).

Example:
  tend tui record --out my-session.html -- nb tui`,
		RunE: func(cmd *cobra.Command, args []string) error {
			commandToRun := []string{}
			dashDashIndex := cmd.Flags().ArgsLenAtDash()

			if dashDashIndex != -1 {
				commandToRun = args[dashDashIndex:]
			}

			if len(commandToRun) == 0 {
				shell := os.Getenv("SHELL")
				if shell == "" {
					shell = "/bin/sh" // A sensible default
				}
				commandToRun = []string{shell}
				fmt.Printf("No command specified. Recording default shell: %s\n", shell)
			}

			fmt.Printf("Starting recording session. Type 'exit' or press Ctrl+D to stop.\n\n")

			rec := recorder.New()
			frames, err := rec.Run(commandToRun)
			if err != nil {
				return fmt.Errorf("recording session failed: %w", err)
			}

			// Open output file
			file, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file %s: %w", outputFile, err)
			}
			defer file.Close()

			// Generate HTML report
			if err := recorder.GenerateHTMLReport(frames, file); err != nil {
				return fmt.Errorf("failed to generate HTML report: %w", err)
			}

			fmt.Printf("\n\nRecording saved to %s\n", outputFile)
			return nil
		},
	}

	recordCmd.Flags().StringVarP(&outputFile, "out", "o", "tend-recording.html", "Output file for the HTML recording")
	return recordCmd
}

// findInitialWorkspacePath determines which workspace path contains the current directory.
// Returns empty string if no workspace is found (showing all workspaces).
func findInitialWorkspacePath() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	cwd, _ = filepath.Abs(cwd)

	// Discover all workspaces
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	discoveryService := workspace.NewDiscoveryService(logger)

	result, err := discoveryService.DiscoverAll()
	if err != nil {
		return ""
	}

	provider := workspace.NewProvider(result)
	allNodes := provider.All()

	// Find the workspace that contains the current directory
	// Walk up the directory tree to find a match
	currentPath := cwd
	for {
		for _, node := range allNodes {
			// Use case-insensitive comparison for macOS/Windows compatibility
			if strings.EqualFold(node.Path, currentPath) {
				return node.Path
			}
		}

		// Walk up one directory
		parent := filepath.Dir(currentPath)
		if parent == currentPath || parent == "." || parent == "/" {
			// Reached the root without finding a workspace
			break
		}
		currentPath = parent
	}

	// Also check if we're inside a workspace (subdirectory)
	for _, node := range allNodes {
		// Use case-insensitive prefix check
		if strings.HasPrefix(strings.ToLower(cwd), strings.ToLower(node.Path)+string(filepath.Separator)) || strings.EqualFold(cwd, node.Path) {
			return node.Path
		}
	}

	return ""
}
