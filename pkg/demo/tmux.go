package demo

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/grovetools/core/pkg/tmux"
)

// setupTmux creates and configures the demo tmux session.
func (g *Generator) setupTmux(content *DemoContent) error {
	socketName := g.tmuxSocket
	sessionName := fmt.Sprintf("grove-demo-%s", g.demoName)

	client, err := tmux.NewClientWithSocket(socketName)
	if err != nil {
		return fmt.Errorf("creating tmux client: %w", err)
	}

	ctx := context.Background()
	env := BuildEnvironment(g.rootDir, g.tmuxSocket)

	// Start in the main ecosystem directory
	workDir := filepath.Join(g.ecosystemsDir(), "homelab")

	// Create a simple session with one empty window
	launchOpts := tmux.LaunchOptions{
		SessionName:      sessionName,
		WorkingDirectory: workDir,
		WindowIndex:      -1,
		Panes: []tmux.PaneOptions{
			{Env: env},
		},
	}

	if err := client.Launch(ctx, launchOpts); err != nil {
		return fmt.Errorf("launching session: %w", err)
	}

	// Set environment variables at the server (global) level so that
	// ALL new sessions created on this tmux server inherit them.
	// This ensures tools like `nav` that create new sessions still
	// have access to GROVE_CONFIG_OVERLAY and other demo env vars.
	for key, value := range env {
		cmd := exec.Command("tmux", "-L", socketName, "set-environment", "-g", key, value)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("setting tmux global environment %s: %w\nOutput: %s", key, err, string(output))
		}
	}

	ulog.Info("Tmux session created").
		Field("socket", socketName).
		Field("session", sessionName).
		Emit()

	return nil
}
