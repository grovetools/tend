package demo

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/grovetools/core/pkg/tmux"
)

// setupTmux creates and configures the demo tmux session.
func (g *Generator) setupTmux() error {
	// Use a named socket (not a path) for tmux -L flag
	socketName := TmuxSocketName

	// Create tmux client with isolated socket
	client, err := tmux.NewClientWithSocket(socketName)
	if err != nil {
		return fmt.Errorf("creating tmux client: %w", err)
	}

	ctx := context.Background()

	// Build environment for the demo session
	env := BuildEnvironment(g.rootDir)

	// Convert env map to slice format
	var envSlice []string
	for k, v := range env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}

	// Main ecosystem directory
	homelabDir := filepath.Join(g.ecosystemsDir(), "homelab")

	// Create the main session with a layout suitable for screenshots
	launchOpts := tmux.LaunchOptions{
		SessionName:      "grove-demo",
		WorkingDirectory: homelabDir,
		WindowIndex:      -1,
		Panes: []tmux.PaneOptions{
			{
				// First pane: Run grove navigator in the main ecosystem
				Command: "echo 'Welcome to Grove Demo Environment' && echo '' && echo 'Ecosystems:' && ls -la " + g.ecosystemsDir(),
				Env:     env,
			},
		},
	}

	if err := client.Launch(ctx, launchOpts); err != nil {
		return fmt.Errorf("launching main session: %w", err)
	}

	// Give tmux a moment to start
	time.Sleep(200 * time.Millisecond)

	// Create additional windows for the demo
	// Window 2: Dashboard worktree (branch name sanitized: feature/gpu-widgets -> feature-gpu-widgets)
	dashboardWorktree := filepath.Join(homelabDir, ".grove-worktrees", "dashboard", "feature-gpu-widgets")
	if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
		Target:     "grove-demo",
		WindowName: "dashboard",
		WorkingDir: dashboardWorktree,
		Env:        envSlice,
	}); err != nil {
		ulog.Warn("Failed to create dashboard window").Err(err).Emit()
	}

	// Window 3: Sentinel
	sentinelDir := filepath.Join(homelabDir, "sentinel")
	if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
		Target:     "grove-demo",
		WindowName: "sentinel",
		WorkingDir: sentinelDir,
		Env:        envSlice,
	}); err != nil {
		ulog.Warn("Failed to create sentinel window").Err(err).Emit()
	}

	// Window 4: Git status overview
	if err := client.NewWindowWithOptions(ctx, tmux.NewWindowOptions{
		Target:     "grove-demo",
		WindowName: "git",
		WorkingDir: homelabDir,
		Env:        envSlice,
	}); err != nil {
		ulog.Warn("Failed to create git window").Err(err).Emit()
	} else {
		// Send git status command
		_ = client.SendKeys(ctx, "grove-demo:git", "git status && echo '' && echo 'Repository structure:' && ls -la")
	}

	// Rename first window
	if err := client.RenameWindow(ctx, "grove-demo:0", "overview"); err != nil {
		ulog.Warn("Failed to rename overview window").Err(err).Emit()
	}

	// Select the first window
	_ = client.SelectWindow(ctx, "grove-demo:overview")

	ulog.Info("Tmux session created").
		Field("socket", socketName).
		Field("session", "grove-demo").
		Pretty(fmt.Sprintf("Tmux session 'grove-demo' created with socket: %s", socketName)).
		Emit()

	// Show instructions in the overview window
	welcomeMsg := fmt.Sprintf(`clear && cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                   Grove Demo Environment                          ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                    ║
║  This is an isolated demo environment for Grove.                   ║
║                                                                    ║
║  Ecosystems:                                                       ║
║    • homelab (main) - 8 repos, 4 worktrees                        ║
║    • contrib        - 3 repos (plugins & themes)                  ║
║    • infra          - 2 repos (deployment & charts)               ║
║                                                                    ║
║  Demo Features:                                                    ║
║    • Realistic repository structure                                ║
║    • Active worktrees with git changes                            ║
║    • Notes and plans with proper frontmatter                      ║
║                                                                    ║
║  Windows:                                                          ║
║    1. overview   - Main ecosystem view                             ║
║    2. dashboard  - Dashboard worktree (feature/gpu-widgets)       ║
║    3. sentinel   - Monitoring service                              ║
║    4. git        - Git status overview                             ║
║                                                                    ║
║  Environment:                                                      ║
║    HOME=%s
║    GROVE_DEMO=1                                                    ║
║                                                                    ║
╚══════════════════════════════════════════════════════════════════╝
EOF
`, g.homeDir())

	_ = client.SendKeys(ctx, "grove-demo:overview", welcomeMsg)

	// Send a command to show the dashboard worktree
	dashboardCmd := fmt.Sprintf("cd %s && echo 'Dashboard worktree: feature/gpu-widgets' && echo '' && ls -la", dashboardWorktree)
	_ = client.SendKeys(ctx, "grove-demo:dashboard", dashboardCmd)

	// Send a command to show sentinel structure
	sentinelCmd := fmt.Sprintf("cd %s && echo 'Sentinel - Metrics & Monitoring' && echo '' && find . -name '*.go' | head -20", sentinelDir)
	_ = client.SendKeys(ctx, "grove-demo:sentinel", sentinelCmd)

	return nil
}
