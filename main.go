package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/project"
)

const childProcessEnvVar = "TEND_IS_CHILD_PROCESS"

func main() {
	// Try to proxy to project-specific binary first
	proxyToProjectBinary()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
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