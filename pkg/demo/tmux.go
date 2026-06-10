package demo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/grovetools/core/pkg/mux"
)

// muxResult records which mux backend a demo was set up with, so the
// metadata can capture everything destroy/status/attach need later.
type muxResult struct {
	backend         mux.MuxType
	tuimuxSocket    string // socket path (tuimux backend only)
	tuimuxDaemonPID int    // daemon PID spawned for this demo (tuimux backend only)
}

// demoBackend decides which mux backend demo setup will use.
// Shared by Preflight and setupMux so they always agree.
func demoBackend() mux.MuxType {
	activeMux := mux.ActiveMux()
	if envMux := os.Getenv(mux.EnvGroveMux); envMux == "tuimux" {
		activeMux = mux.MuxTuimux
	}
	if activeMux == mux.MuxTuimux {
		return mux.MuxTuimux
	}
	return mux.MuxTmux
}

// setupMux routes demo session setup to the appropriate mux backend.
func (g *Generator) setupMux(content *DemoContent) (*muxResult, error) {
	if demoBackend() == mux.MuxTuimux {
		return g.setupTuimux(content)
	}
	return g.setupTmux(content)
}

// setupTuimux creates and configures the demo tuimux session.
func (g *Generator) setupTuimux(content *DemoContent) (*muxResult, error) {
	socketPath := TuimuxSocketPath(g.rootDir)
	sessionName := fmt.Sprintf("grove-demo-%s", g.demoName)

	proc, err := spawnDemoDaemon(socketPath)
	if err != nil {
		return nil, fmt.Errorf("spawning tuimux daemon: %w", err)
	}

	engine, err := mux.NewTuimuxEngineWithSocket(socketPath)
	if err != nil {
		_ = proc.Signal(os.Kill)
		return nil, fmt.Errorf("connecting to tuimux daemon: %w", err)
	}

	ctx := context.Background()
	workDir := filepath.Join(g.ecosystemsDir(), "homelab")
	if err := engine.StartServer(ctx, sessionName, mux.WithWorkDir(workDir)); err != nil {
		_ = proc.Signal(os.Kill)
		return nil, fmt.Errorf("starting tuimux server: %w", err)
	}

	ulog.Info("Tuimux demo session created").
		Field("socket", socketPath).
		Field("session", sessionName).
		Field("daemon_pid", proc.Pid).
		Emit()

	return &muxResult{
		backend:         mux.MuxTuimux,
		tuimuxSocket:    socketPath,
		tuimuxDaemonPID: proc.Pid,
	}, nil
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
func (g *Generator) setupTmux(content *DemoContent) (*muxResult, error) {
	socketName := g.tmuxSocket
	sessionName := fmt.Sprintf("grove-demo-%s", g.demoName)

	engine, err := mux.NewTmuxEngineWithSocket(socketName)
	if err != nil {
		return nil, fmt.Errorf("creating mux engine: %w", err)
	}

	ctx := context.Background()
	env := BuildEnvironment(g.rootDir, g.tmuxSocket)

	// Start in the main ecosystem directory
	workDir := filepath.Join(g.ecosystemsDir(), "homelab")

	// Create a simple session with one empty window
	launchOpts := mux.LaunchOptions{
		SessionName:      sessionName,
		WorkingDirectory: workDir,
		WindowIndex:      -1,
		Panes: []mux.PaneOptions{
			{Env: env},
		},
	}

	if err := engine.Launch(ctx, launchOpts); err != nil {
		return nil, fmt.Errorf("launching session: %w", err)
	}

	// Set environment variables at the server (global) level so that
	// ALL new sessions created on this tmux server inherit them.
	// This ensures tools like `nav` that create new sessions still
	// have access to GROVE_CONFIG_OVERLAY and other demo env vars.
	for key, value := range env {
		cmd := exec.Command("tmux", "-L", socketName, "set-environment", "-g", key, value)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("setting tmux global environment %s: %w\nOutput: %s", key, err, string(output))
		}
	}

	ulog.Info("Tmux session created").
		Field("socket", socketName).
		Field("session", sessionName).
		Emit()

	return &muxResult{backend: mux.MuxTmux}, nil
}
