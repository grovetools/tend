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

Demo environments provide complete, isolated Grove ecosystems for testing and demos.
Each demo type creates different content (repositories, notes, plans, etc.) and
can have its own tmux session with a preconfigured layout.

Available demos:
  homelab   - Full-featured demo with 3 ecosystems, 13 repos, worktrees, notes, and plans

Commands:
  tend demo create <name>     Create a new demo environment
  tend demo attach <name>     Attach to the demo tmux session
  tend demo destroy <name>    Remove the demo environment
  tend demo status <name>     Show demo environment status`,
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
		Use:   "create <name>",
		Short: "Create a new demo environment",
		Long: `Create a new demo environment with isolated ecosystems, repositories, and notebooks.

Available demos:
  homelab   - Full ecosystem with 3 ecosystems, 13 repos, worktrees, notes, and plans

The environment is created at ~/.grove-demos/<name> by default, or at the path
specified with --output-dir. Use --force to overwrite an existing demo environment.

After creation, use 'tend demo attach <name>' to connect to the demo tmux session.`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return demo.List(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			demoName := args[0]

			// Verify demo exists
			if _, err := demo.Get(demoName); err != nil {
				return fmt.Errorf("unknown demo '%s'. Available: %v", demoName, demo.List())
			}

			// Set default output directory
			if outputDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("getting home directory: %w", err)
				}
				outputDir = filepath.Join(homeDir, ".grove-demos", demoName)
			}

			// Check if already exists
			if _, err := os.Stat(outputDir); err == nil && !force {
				return fmt.Errorf("demo exists at %s (use --force to overwrite)", outputDir)
			}

			// If force, remove existing
			if force {
				if err := os.RemoveAll(outputDir); err != nil {
					return fmt.Errorf("removing existing demo: %w", err)
				}
			}

			// Create the demo environment
			gen, err := demo.NewGenerator(outputDir, demoName)
			if err != nil {
				return err
			}

			if err := gen.Generate(); err != nil {
				return fmt.Errorf("generating demo environment: %w", err)
			}

			ulogDemo.Success("Demo environment created successfully").
				Field("demo", demoName).
				Field("path", outputDir).
				Pretty(fmt.Sprintf("Demo created at: %s", outputDir)).
				Emit()

			// Optionally attach immediately
			if attach {
				return attachToDemo(outputDir)
			}

			ulogDemo.Info("Attach with command").
				Pretty(fmt.Sprintf("\nTo connect to the demo environment:\n  tend demo attach %s", demoName)).
				PrettyOnly().
				Emit()

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory (default: ~/.grove-demos/<name>)")
	cmd.Flags().BoolVarP(&attach, "attach", "a", false, "Attach to the demo session immediately after creation")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing demo environment")

	return cmd
}

// newDemoAttachCmd creates the "demo attach" command.
func newDemoAttachCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "attach <name>",
		Short: "Attach to a demo tmux session",
		Long: `Attach to the demo environment's isolated tmux session.

This spawns a shell with the demo's environment variables and connects
to the demo's tmux server.`,
		Args: cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return listExistingDemos(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine directory
			if outputDir == "" {
				if len(args) == 0 {
					// Check for legacy ~/.grove-demo
					homeDir, _ := os.UserHomeDir()
					legacyDir := filepath.Join(homeDir, ".grove-demo")
					if _, err := os.Stat(legacyDir); err == nil {
						outputDir = legacyDir
					} else {
						return fmt.Errorf("no demo name provided")
					}
				} else {
					homeDir, _ := os.UserHomeDir()
					outputDir = filepath.Join(homeDir, ".grove-demos", args[0])
				}
			}

			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				return fmt.Errorf("demo not found at %s", outputDir)
			}

			return attachToDemo(outputDir)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Demo directory")

	return cmd
}

// listExistingDemos returns a list of existing demo directories.
func listExistingDemos() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	demosDir := filepath.Join(homeDir, ".grove-demos")
	entries, err := os.ReadDir(demosDir)
	if err != nil {
		return nil
	}

	var demos []string
	for _, entry := range entries {
		if entry.IsDir() {
			demos = append(demos, entry.Name())
		}
	}
	return demos
}

// newDemoDestroyCmd creates the "demo destroy" command.
func newDemoDestroyCmd() *cobra.Command {
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "destroy <name>",
		Short: "Remove a demo environment",
		Long: `Remove the demo environment and all its contents.

This kills the demo tmux server and removes all demo files.`,
		Args: cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return listExistingDemos(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine directory
			if outputDir == "" {
				if len(args) == 0 {
					// Check for legacy ~/.grove-demo
					homeDir, _ := os.UserHomeDir()
					legacyDir := filepath.Join(homeDir, ".grove-demo")
					if _, err := os.Stat(legacyDir); err == nil {
						outputDir = legacyDir
					} else {
						return fmt.Errorf("no demo name provided")
					}
				} else {
					homeDir, _ := os.UserHomeDir()
					outputDir = filepath.Join(homeDir, ".grove-demos", args[0])
				}
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

			// Kill the tmux server first using metadata
			meta, err := demo.LoadMetadata(outputDir)
			if err == nil && meta.TmuxSocket != "" {
				client, err := tmux.NewClientWithSocket(meta.TmuxSocket)
				if err == nil {
					_ = client.KillServer(cmd.Context())
				}
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

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Demo directory")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// newDemoStatusCmd creates the "demo status" command.
func newDemoStatusCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Show demo environment status",
		Long:  `Display the current state of the demo environment.`,
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return listExistingDemos(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine directory
			if outputDir == "" {
				if len(args) == 0 {
					// Check for legacy ~/.grove-demo
					homeDir, _ := os.UserHomeDir()
					legacyDir := filepath.Join(homeDir, ".grove-demo")
					if _, err := os.Stat(legacyDir); err == nil {
						outputDir = legacyDir
					} else {
						return fmt.Errorf("no demo name provided")
					}
				} else {
					homeDir, _ := os.UserHomeDir()
					outputDir = filepath.Join(homeDir, ".grove-demos", args[0])
				}
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
			client, err := tmux.NewClientWithSocket(meta.TmuxSocket)
			if err == nil {
				sessions, _ := client.ListSessions(cmd.Context())
				tmuxRunning = len(sessions) > 0
			}

			// Display status
			fmt.Printf("Demo Environment Status\n")
			fmt.Printf("=======================\n")
			fmt.Printf("Demo:         %s\n", meta.DemoName)
			fmt.Printf("Location:     %s\n", outputDir)
			fmt.Printf("Created:      %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Tmux Socket:  %s\n", meta.TmuxSocket)
			if meta.TmuxSessionName != "" {
				fmt.Printf("Tmux Session: %s\n", meta.TmuxSessionName)
			}
			fmt.Printf("Tmux Running: %v\n", tmuxRunning)
			fmt.Printf("\nEcosystems:\n")
			for _, eco := range meta.Ecosystems {
				fmt.Printf("  - %s (%d repos", eco.Name, eco.RepoCount)
				if eco.Description != "" {
					fmt.Printf(", %s", eco.Description)
				}
				fmt.Printf(")\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Demo directory")

	return cmd
}

// attachToDemo attaches to the demo tmux session.
func attachToDemo(demoDir string) error {
	// Load metadata
	meta, err := demo.LoadMetadata(demoDir)
	if err != nil {
		return fmt.Errorf("loading demo metadata: %w", err)
	}

	// Check if this demo has a tmux session
	if meta.TmuxSessionName == "" {
		return fmt.Errorf("demo '%s' does not have a tmux session", meta.DemoName)
	}

	// Build environment for the demo
	env := demo.BuildEnvironment(demoDir, meta.TmuxSocket)

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
	client, err := tmux.NewClientWithSocket(meta.TmuxSocket)
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
		tmuxArgs = []string{"tmux", "-L", meta.TmuxSocket, "switch-client", "-t", meta.TmuxSessionName}
	} else {
		// Outside tmux, attach
		tmuxArgs = []string{"tmux", "-L", meta.TmuxSocket, "attach-session", "-t", meta.TmuxSessionName}
	}

	// Replace the current process with tmux
	if err := syscall.Exec(tmuxPath, tmuxArgs, fullEnv); err != nil {
		return fmt.Errorf("failed to exec tmux: %w", err)
	}

	return nil
}
