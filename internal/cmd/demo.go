package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/pkg/tmux"
	"github.com/grovetools/tend/pkg/demo"
	"github.com/spf13/cobra"
)

var ulogDemo = grovelogging.NewUnifiedLogger("grove-tend.demo")

// newDemoCmd creates the demo command.
func newDemoCmd() *cobra.Command {
	demoCmd := &cobra.Command{
		Use:   "demo",
		Short: "Manage demo environments for screenshots and demos",
		Long: `Create and manage isolated demo environments for screenshots and demonstrations.

The demo environment creates a complete, isolated Grove ecosystem with:
  - 3 ecosystems: homelab (main), contrib, and infra
  - 8 repositories in the main ecosystem with realistic structure
  - Active worktrees simulating in-progress work
  - Git states (dirty, unstaged, untracked files)
  - Notes and plans with proper frontmatter
  - Isolated tmux session with preconfigured layout

Commands:
  tend demo create    Create a new demo environment
  tend demo attach    Attach to the demo tmux session
  tend demo destroy   Remove the demo environment
  tend demo status    Show demo environment status`,
	}

	demoCmd.AddCommand(newDemoCreateCmd())
	demoCmd.AddCommand(newDemoAttachCmd())
	demoCmd.AddCommand(newDemoDestroyCmd())
	demoCmd.AddCommand(newDemoStatusCmd())

	return demoCmd
}

// newDemoCreateCmd creates the "demo create" command.
func newDemoCreateCmd() *cobra.Command {
	var outputDir string
	var attach bool
	var force bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new demo environment",
		Long: `Create a new demo environment with isolated ecosystems, repositories, and notebooks.

The environment is created at ~/.grove-demo by default, or at the path specified
with --output-dir. Use --force to overwrite an existing demo environment.

After creation, use 'tend demo attach' to connect to the demo tmux session.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default output directory
			if outputDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("getting home directory: %w", err)
				}
				outputDir = filepath.Join(homeDir, ".grove-demo")
			}

			// Check if already exists
			if _, err := os.Stat(outputDir); err == nil && !force {
				return fmt.Errorf("demo environment already exists at %s (use --force to overwrite)", outputDir)
			}

			// If force, remove existing
			if force {
				if err := os.RemoveAll(outputDir); err != nil {
					return fmt.Errorf("removing existing demo: %w", err)
				}
			}

			// Create the demo environment
			gen := demo.NewGenerator(outputDir)
			if err := gen.Generate(); err != nil {
				return fmt.Errorf("generating demo environment: %w", err)
			}

			ulogDemo.Success("Demo environment created successfully").
				Field("path", outputDir).
				Pretty(fmt.Sprintf("Demo environment created at: %s", outputDir)).
				Emit()

			// Optionally attach immediately
			if attach {
				return attachToDemo(outputDir)
			}

			ulogDemo.Info("Attach with: tend demo attach").
				Pretty("\nTo connect to the demo environment:\n  tend demo attach").
				PrettyOnly().
				Emit()

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for demo environment (default: ~/.grove-demo)")
	cmd.Flags().BoolVarP(&attach, "attach", "a", false, "Attach to the demo session immediately after creation")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing demo environment")

	return cmd
}

// newDemoAttachCmd creates the "demo attach" command.
func newDemoAttachCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "attach",
		Short: "Attach to the demo tmux session",
		Long: `Attach to the demo environment's isolated tmux session.

This spawns a shell with the demo's environment variables (HOME, XDG_CONFIG_HOME, etc.)
and connects to the demo's tmux server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default output directory
			if outputDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("getting home directory: %w", err)
				}
				outputDir = filepath.Join(homeDir, ".grove-demo")
			}

			// Check if demo exists
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				return fmt.Errorf("demo environment not found at %s (run 'tend demo create' first)", outputDir)
			}

			return attachToDemo(outputDir)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Demo environment directory (default: ~/.grove-demo)")

	return cmd
}

// newDemoDestroyCmd creates the "demo destroy" command.
func newDemoDestroyCmd() *cobra.Command {
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Remove the demo environment",
		Long: `Remove the demo environment and all its contents.

This kills the demo tmux server and removes all demo files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default output directory
			if outputDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("getting home directory: %w", err)
				}
				outputDir = filepath.Join(homeDir, ".grove-demo")
			}

			// Check if demo exists
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				ulogDemo.Info("Demo environment not found").
					Pretty(fmt.Sprintf("Demo environment not found at %s", outputDir)).
					PrettyOnly().
					Emit()
				return nil
			}

			// Confirm unless forced
			if !force {
				ulogDemo.Warn("This will delete the entire demo environment").
					Pretty(fmt.Sprintf("About to delete: %s\nRun with --force to confirm", outputDir)).
					PrettyOnly().
					Emit()
				return nil
			}

			// Kill the tmux server first
			client, err := tmux.NewClientWithSocket(demo.TmuxSocketName)
			if err == nil {
				_ = client.KillServer(cmd.Context())
			}

			// Remove the directory
			// Note: os.RemoveAll is safe with symlinks - it removes the link, not the target.
			// The demo may contain symlinks to real user configs (~/.config/*), but those
			// real files will NOT be deleted.
			if err := os.RemoveAll(outputDir); err != nil {
				return fmt.Errorf("removing demo directory: %w", err)
			}

			ulogDemo.Success("Demo environment destroyed").
				Pretty(fmt.Sprintf("Demo environment removed: %s", outputDir)).
				Emit()

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Demo environment directory (default: ~/.grove-demo)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// newDemoStatusCmd creates the "demo status" command.
func newDemoStatusCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show demo environment status",
		Long:  `Display the current state of the demo environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default output directory
			if outputDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("getting home directory: %w", err)
				}
				outputDir = filepath.Join(homeDir, ".grove-demo")
			}

			// Check if demo exists
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				ulogDemo.Info("Demo environment not found").
					Pretty(fmt.Sprintf("Demo environment not found at %s", outputDir)).
					PrettyOnly().
					Emit()
				return nil
			}

			// Load metadata
			meta, err := demo.LoadMetadata(outputDir)
			if err != nil {
				return fmt.Errorf("loading demo metadata: %w", err)
			}

			// Check tmux server status
			tmuxRunning := false
			client, err := tmux.NewClientWithSocket(demo.TmuxSocketName)
			if err == nil {
				sessions, _ := client.ListSessions(cmd.Context())
				tmuxRunning = len(sessions) > 0
			}

			// Display status
			fmt.Printf("Demo Environment Status\n")
			fmt.Printf("=======================\n")
			fmt.Printf("Location:     %s\n", outputDir)
			fmt.Printf("Created:      %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Tmux Socket:  %s\n", meta.TmuxSocket)
			fmt.Printf("Tmux Running: %v\n", tmuxRunning)
			fmt.Printf("\nEcosystems:\n")
			for _, eco := range meta.Ecosystems {
				fmt.Printf("  - %s (%d repos)\n", eco.Name, eco.RepoCount)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Demo environment directory (default: ~/.grove-demo)")

	return cmd
}

// attachToDemo attaches to the demo tmux session.
func attachToDemo(demoDir string) error {
	// Build environment for the demo
	env := demo.BuildEnvironment(demoDir)

	// Get current environment and add demo overrides
	fullEnv := os.Environ()
	for k, v := range env {
		fullEnv = append(fullEnv, fmt.Sprintf("%s=%s", k, v))
	}

	// Find tmux binary
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found in PATH")
	}

	// Check if demo tmux session exists
	client, err := tmux.NewClientWithSocket(demo.TmuxSocketName)
	if err != nil {
		return fmt.Errorf("failed to create tmux client: %w", err)
	}

	ctx := context.Background()
	sessions, err := client.ListSessions(ctx)
	if err != nil || len(sessions) == 0 {
		return fmt.Errorf("demo tmux session not running (run 'tend demo create' first)")
	}

	// Attach or switch to the session
	// Use -L for socket name (not -S which is for socket path)
	var tmuxArgs []string
	if os.Getenv("TMUX") != "" {
		// Inside tmux, switch client
		tmuxArgs = []string{"tmux", "-L", demo.TmuxSocketName, "switch-client", "-t", "grove-demo"}
	} else {
		// Outside tmux, attach
		tmuxArgs = []string{"tmux", "-L", demo.TmuxSocketName, "attach-session", "-t", "grove-demo"}
	}

	// Replace the current process with tmux
	if err := syscall.Exec(tmuxPath, tmuxArgs, fullEnv); err != nil {
		return fmt.Errorf("failed to exec tmux: %w", err)
	}

	return nil
}
