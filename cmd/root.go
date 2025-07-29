package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mattsolo1/grove-core/cli"
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = cli.NewStandardCommand(
	"tend",
	"End-to-end scenario testing",
)

func init() {
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
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func initializeFlags() {
	cobra.OnInitialize(initConfig)

	// Global flags
	// Note: verbose flag is already defined by grove-core's NewStandardCommand
	rootCmd.PersistentFlags().BoolVar(&veryVerbose, "very-verbose", false, "Enable very verbose output (includes command details)")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "Enable interactive mode")
	rootCmd.PersistentFlags().StringSliceVarP(&tags, "tags", "t", []string{}, "Filter scenarios by tags")
	rootCmd.PersistentFlags().StringVarP(&rootDir, "root", "r", "", "Root directory for tests (default: current directory)")
	rootCmd.PersistentFlags().StringVarP(&groveBinary, "grove", "g", "grove", "Path to Grove binary")
	rootCmd.PersistentFlags().BoolVarP(&monitorDocker, "monitor", "m", false, "Show live Docker container updates during tests")
	rootCmd.PersistentFlags().StringVar(&dockerFilter, "docker-filter", "name=grove", "Docker filter for container monitoring")

	// Add subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(validateCmd)
}

// Initialize flags once
func init() {
	initializeFlags()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
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
}