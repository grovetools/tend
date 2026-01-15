package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	grovelogging "github.com/mattsolo1/grove-core/logging"
	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/project"
)


var ulog = grovelogging.NewUnifiedLogger("grove-tend.main")

const childProcessEnvVar = "TEND_IS_CHILD_PROCESS"

// shouldSkipProxy determines if we should skip proxying based on the command being run.
// Commands that don't need project-specific test runners can skip the rebuild.
func shouldSkipProxy() bool {
	// If no args, show help - doesn't need proxy
	if len(os.Args) <= 1 {
		return true
	}

	// Check the first argument (the command)
	cmd := os.Args[1]

	// Commands that don't need project-specific test runners
	skipCommands := map[string]bool{
		"version":    true, // Just shows version info
		"help":       true, // Shows help text
		"completion": true, // Generates shell completions
		"docs":       true, // Prints JSON documentation
		"sessions":   true, // Manages test sessions (operates on existing data)
		"-h":         true, // Help flag
		"--help":     true, // Help flag
		"-v":         true, // Might be version flag
		"--version":  true, // Version flag (if supported)
	}

	return skipCommands[cmd]
}

func main() {
	// CLI output goes to stdout (stderr is for errors only)
	grovelogging.SetGlobalOutput(os.Stdout)

	// Try to proxy to project-specific binary first
	proxyToProjectBinary()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
  ulog.Info("\nReceived interrupt signal, shutting down...").Pretty("\nReceived interrupt signal, shutting down...").PrettyOnly().Emit()
		cancel()
	}()

	// No built-in scenarios - grove-tend is now a pure library
	// Scenarios should be defined in the repositories they test
	var allScenarios []*harness.Scenario

	// Execute the application
	if err := app.Execute(ctx, allScenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func proxyToProjectBinary() {
	// Check if we're already a child process to prevent recursion
	if os.Getenv(childProcessEnvVar) == "true" {
		return
	}

	// Skip proxying for commands that don't need project-specific test runners
	if shouldSkipProxy() {
		return
	}

	// Get the path of the currently executing binary
	currentBinary, err := os.Executable()
	if err != nil {
		// Can't determine current binary, continue without proxying
		return
	}

	// Resolve any symlinks to get the real path
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	// Build the project-specific tend binary (always rebuilds for latest changes)
	projectBinary, err := project.BuildProjectTendBinary(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building project test runner: %v\n", err)
		os.Exit(1)
	}

	if projectBinary == "" {
		// No project-specific binary source found, continue with global binary
		return
	}

	// Style helpers for messages
	arrow := lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Render("→")

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Italic(true)

	// Resolve symlinks for the project binary
	projectBinary, err = filepath.EvalSymlinks(projectBinary)
	if err != nil {
		return
	}

	// Check if the binaries are different
	if currentBinary == projectBinary {
		// Same binary, no need to proxy
		return
	}

	// Print executing message
	execMsg := fmt.Sprintf("%s %s %s",
		arrow,
		style.Render("Executing project test runner:"),
		pathStyle.Render(projectBinary),
	)
	fmt.Fprintln(os.Stderr, execMsg)

	// Prepare the environment with the child process marker
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=true", childProcessEnvVar))

	// Replace the current process with the project binary
	err = syscall.Exec(projectBinary, os.Args, env)
	if err != nil {
		// syscall.Exec should not return, but if it does, it's an error
		fmt.Fprintf(os.Stderr, "Error executing project binary: %v\n", err)
		os.Exit(1)
	}
}