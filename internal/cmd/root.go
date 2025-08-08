package cmd

import (
	"fmt"
	"os"

	"github.com/mattsolo1/grove-core/cli"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/spf13/cobra"
)

var (
	verbose       bool
	veryVerbose   bool
	interactive   bool
	tags          []string
	rootDir       string
	groveBinary   string
	monitorDocker bool
	dockerFilter  string
)

// NewRootCmd creates the root command for a tend application, configured with the provided scenarios.
func NewRootCmd(allScenarios []*harness.Scenario) *cobra.Command {
	rootCmd := cli.NewStandardCommand(
		"tend",
		"End-to-end scenario testing",
	)
	
	rootCmd.Long = `A modern, Go-based end-to-end testing framework for Grove.

This tool provides structured, maintainable testing capabilities to replace
ad-hoc bash scripts with proper error handling, cleanup, and beautiful output.

Features:
  • Interactive step-through testing mode
  • Parallel test execution
  • Beautiful terminal output with progress bars
  • Comprehensive logging and error reporting
  • Git worktree support for multi-branch testing
  • Docker container management
  • Grove-specific command helpers`
	
	// Initialize configuration
	cobra.OnInitialize(func() {
		// Get verbose flag value from cobra
		if flag := rootCmd.Flag("verbose"); flag != nil {
			verbose, _ = rootCmd.Flags().GetBool("verbose")
		}
		
		if rootDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
				os.Exit(1)
			}
			rootDir = wd
		}
	})

	// Global flags
	// Note: verbose flag is already defined by grove-core's NewStandardCommand
	rootCmd.PersistentFlags().BoolVar(&veryVerbose, "very-verbose", false, "Enable very verbose output (includes command details)")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "Enable interactive mode")
	rootCmd.PersistentFlags().StringSliceVarP(&tags, "tags", "t", []string{}, "Filter scenarios by tags")
	rootCmd.PersistentFlags().StringVarP(&rootDir, "root", "r", "", "Root directory for tests (default: current directory)")
	rootCmd.PersistentFlags().StringVarP(&groveBinary, "grove", "g", "grove", "Path to Grove binary")
	rootCmd.PersistentFlags().BoolVarP(&monitorDocker, "monitor", "m", false, "Show live Docker container updates during tests")
	rootCmd.PersistentFlags().StringVar(&dockerFilter, "docker-filter", "name=grove", "Docker filter for container monitoring")

	// Add subcommands with scenarios
	rootCmd.AddCommand(newRunCmd(allScenarios))
	rootCmd.AddCommand(newListCmd(allScenarios))
	rootCmd.AddCommand(newValidateCmd(allScenarios))
	rootCmd.AddCommand(newVersionCmd())
	
	return rootCmd
}