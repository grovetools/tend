package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-core/pkg/workspace"
	"github.com/mattsolo1/grove-tend/internal/tui/runner"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
