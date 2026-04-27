package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/pkg/workspace"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/grovetools/tend/internal/tui/runner"
	"github.com/grovetools/tend/pkg/recorder"
)

// newTuiCmd creates the `tui` subcommand.
func newTuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch an interactive TUI for browsing and running tests",
		Long: `Launch an interactive Terminal User Interface to browse, manage,
and run 'tend' test scenarios across the ecosystem.`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	return cmd
}

// newRecordCmd creates the `record` command.
func newRecordCmd() *cobra.Command {
	var outputFile string
	recordCmd := &cobra.Command{
		Use:   "record [--out basename] -- <command...>",
		Short: "Record a manual TUI session to multiple output formats",
		Long: `Launches a command within a recordable sub-shell. All keystrokes and
terminal output are captured and saved to five formats:
  - .html:       Interactive HTML report (for human review)
  - .md:         Markdown report (for LLM consumption, plain text)
  - .ansi.md:    Markdown with ANSI codes (for color debugging)
  - .xml:        XML report (for LLM consumption, plain text)
  - .ansi.xml:   XML with ANSI codes (for color debugging)

This is useful for creating shareable, replayable recordings of a TUI session,
often for providing context to an LLM for writing automated tests.

Use '--' to separate the recorder's flags from the command you want to record.
If no command is provided, it will default to launching your default shell ($SHELL).

Example:
  tend record --out my-session -- nb tui
  # Creates: my-session.{html,md,ansi.md,xml,ansi.xml}`,
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

			// Determine base filename (remove extension if provided)
			baseName := outputFile
			if ext := filepath.Ext(baseName); ext != "" {
				baseName = baseName[:len(baseName)-len(ext)]
			}

			// Generate all five formats
			formats := []struct {
				ext       string
				generator func([]recorder.Frame, io.Writer) error
				name      string
			}{
				{".html", recorder.GenerateHTMLReport, "HTML"},
				{".md", recorder.GenerateMarkdownReport, "Markdown"},
				{".ansi.md", recorder.GenerateMarkdownReportWithANSI, "Markdown (ANSI)"},
				{".xml", recorder.GenerateXMLReport, "XML"},
				{".ansi.xml", recorder.GenerateXMLReportWithANSI, "XML (ANSI)"},
			}

			var savedFiles []string
			for _, format := range formats {
				filename := baseName + format.ext
				file, err := os.Create(filename)
				if err != nil {
					return fmt.Errorf("failed to create %s file %s: %w", format.name, filename, err)
				}

				if err := format.generator(frames, file); err != nil {
					file.Close()
					return fmt.Errorf("failed to generate %s report: %w", format.name, err)
				}
				file.Close()
				savedFiles = append(savedFiles, filename)
			}

			fmt.Printf("\n\nRecordings saved:\n")
			for _, file := range savedFiles {
				fmt.Printf("  - %s\n", file)
			}
			return nil
		},
	}

	recordCmd.Flags().StringVarP(&outputFile, "out", "o", "tend-recording", "Base filename for output files (without extension)")
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
