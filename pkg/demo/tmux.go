package demo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/grovetools/core/pkg/mux"
	"github.com/grovetools/core/pkg/tmux"
)

// setupMux routes demo session setup to the appropriate mux backend.
func (g *Generator) setupMux(content *DemoContent) error {
	activeMux := mux.ActiveMux()
	if envMux := os.Getenv(mux.EnvGroveMux); envMux == "tuimux" {
		activeMux = mux.MuxTuimux
	}
	if activeMux == mux.MuxTuimux {
		return g.setupTuimux(content)
	}
	return g.setupTmux(content)
}

// setupTuimux creates and configures the demo tuimux session.
func (g *Generator) setupTuimux(content *DemoContent) error {
	socketPath := filepath.Join(g.rootDir, "state", "tuimux-demo.sock")
	sessionName := fmt.Sprintf("grove-demo-%s", g.demoName)

	proc, err := spawnDemoDaemon(socketPath)
	if err != nil {
		return fmt.Errorf("spawning tuimux daemon: %w", err)
	}

	engine, err := mux.NewTuimuxEngineWithSocket(socketPath)
	if err != nil {
		_ = proc.Signal(os.Kill)
		return fmt.Errorf("connecting to tuimux daemon: %w", err)
	}

	ctx := context.Background()
	workDir := filepath.Join(g.ecosystemsDir(), "homelab")
	if err := engine.CreateSession(ctx, sessionName, mux.WithWorkDir(workDir)); err != nil {
		return fmt.Errorf("creating tuimux session: %w", err)
	}

	ulog.Info("Tuimux demo session created").
		Field("socket", socketPath).
		Field("session", sessionName).
		Emit()

	return nil
}

// spawnDemoDaemon starts a tuimux daemon for the demo environment.
func spawnDemoDaemon(socketPath string) (*os.Process, error) {
	tuimuxBin, err := exec.LookPath("tuimux")
	if err != nil {
		return nil, fmt.Errorf("tuimux binary not found: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return nil, err
	}
	cmd := exec.Command(tuimuxBin, "daemon", "--socket", socketPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start tuimux daemon: %w", err)
	}
	// Wait for readiness
	for range 25 {
		time.Sleep(200 * time.Millisecond)
		if mux.PingTuimuxSocket(socketPath) == nil {
			return cmd.Process, nil
		}
	}
	_ = cmd.Process.Signal(os.Kill)
	return nil, fmt.Errorf("tuimux daemon did not become ready at %s", socketPath)
}

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
