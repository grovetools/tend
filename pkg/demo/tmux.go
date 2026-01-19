package demo

import (
	"context"
	"fmt"
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

	ulog.Info("Tmux session created").
		Field("socket", socketName).
		Field("session", sessionName).
		Emit()

	return nil
}
