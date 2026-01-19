// Package demo provides functionality for generating demo environments.
package demo

import (
	"fmt"
	"path/filepath"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/tend/pkg/fs"
	"gopkg.in/yaml.v3"
)

var ulog = grovelogging.NewUnifiedLogger("grove-tend.demo")

// TmuxSocketName is the name of the tmux socket used for demo environments.
// This is used with tmux's -L flag which creates sockets in the tmux temp directory.
// For multiple demos, this will be parameterized per demo (e.g., "grove-demo-homelab").
const TmuxSocketName = "grove-demo"

// Generator creates demo environments.
type Generator struct {
	rootDir    string
	demoName   string
	spec       DemoSpec
	tmuxSocket string
}

// NewGenerator creates a new demo generator for the specified demo type.
func NewGenerator(rootDir, demoName string) (*Generator, error) {
	spec, err := Get(demoName)
	if err != nil {
		return nil, err
	}

	return &Generator{
		rootDir:    rootDir,
		demoName:   demoName,
		spec:       spec,
		tmuxSocket: fmt.Sprintf("grove-demo-%s", demoName),
	}, nil
}

// Generate creates the complete demo environment.
func (g *Generator) Generate() error {
	ulog.Info("Creating demo environment").
		Field("path", g.rootDir).
		Field("demo", g.demoName).
		Pretty(fmt.Sprintf("Creating %s demo environment at: %s", g.demoName, g.rootDir)).
		Emit()

	// Create directory structure
	if err := g.createDirectoryStructure(); err != nil {
		return fmt.Errorf("creating directory structure: %w", err)
	}

	// Create an empty/minimal overlay config first so that CLI commands
	// (grove, nb, flow) work during spec.Generate(). We'll update it after.
	if err := g.createEmptyOverlay(); err != nil {
		return fmt.Errorf("creating initial overlay: %w", err)
	}

	// Let spec generate content (ecosystems, repos, notes, plans)
	content, err := g.spec.Generate(g.rootDir)
	if err != nil {
		return fmt.Errorf("generating demo content: %w", err)
	}

	// Update overlay config with full ecosystem information
	if err := g.createGlobalConfig(content); err != nil {
		return fmt.Errorf("creating global config: %w", err)
	}

	// Setup tmux if needed
	if content.TmuxNeeded {
		if err := g.setupTmux(content); err != nil {
			return fmt.Errorf("setting up tmux: %w", err)
		}
	}

	// Save metadata
	meta := &Metadata{
		DemoName:        g.demoName,
		CreatedAt:       time.Now(),
		TmuxSocket:      g.tmuxSocket,
		TmuxSessionName: fmt.Sprintf("grove-demo-%s", g.demoName),
		Ecosystems:      content.Ecosystems,
		OverlayPath:     g.overlayPath(),
		NotebookDir:     g.notebookDir(),
	}
	return SaveMetadata(g.rootDir, meta)
}

// createEmptyOverlay creates a minimal overlay config so CLI commands work during generation.
func (g *Generator) createEmptyOverlay() error {
	config := map[string]interface{}{
		"version": "1.0",
		"groves":  make(map[string]interface{}),
		"notebooks": map[string]interface{}{
			"definitions": make(map[string]interface{}),
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return writeFile(g.overlayPath(), data)
}

// createDirectoryStructure creates the base directory structure.
// Note: We no longer create a sandbox home since we use GROVE_CONFIG_OVERLAY
// to isolate grove config while using the real HOME.
func (g *Generator) createDirectoryStructure() error {
	dirs := []string{
		g.notebookDir(),
		g.ecosystemsDir(),
	}

	for _, dir := range dirs {
		if err := createDir(dir); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	return nil
}

// createGlobalConfig creates the GROVE_CONFIG_OVERLAY file.
// This overlay is merged on top of the user's real grove config,
// overriding workspaces/groves while preserving API keys and other settings.
func (g *Generator) createGlobalConfig(content *DemoContent) error {
	config := map[string]interface{}{
		"version": "1.0",
		"groves":  make(map[string]interface{}),
		"notebooks": map[string]interface{}{
			"definitions": make(map[string]interface{}),
		},
	}

	groves := config["groves"].(map[string]interface{})
	notebooks := config["notebooks"].(map[string]interface{})["definitions"].(map[string]interface{})

	for _, eco := range content.Ecosystems {
		groves[eco.Name] = map[string]interface{}{
			"path":        eco.Path,
			"enabled":     true,
			"description": eco.Description,
			"notebook":    eco.Name,
		}
		notebooks[eco.Name] = map[string]interface{}{
			"root_dir": filepath.Join(g.notebookDir(), eco.Name),
		}
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return writeFile(g.overlayPath(), data)
}

// Helper methods for directory paths
func (g *Generator) notebookDir() string {
	return filepath.Join(g.rootDir, "notebooks")
}

func (g *Generator) ecosystemsDir() string {
	return filepath.Join(g.rootDir, "ecosystems")
}

func (g *Generator) overlayPath() string {
	return filepath.Join(g.rootDir, "grove-overlay.yml")
}

// BuildEnvironment returns the environment variables for the demo.
// Uses GROVE_CONFIG_OVERLAY to isolate grove config discovery while keeping
// the real HOME, so nvim, LSPs, shell config, etc. all work normally.
func BuildEnvironment(demoDir, tmuxSocket string) map[string]string {
	return map[string]string{
		"GROVE_CONFIG_OVERLAY": filepath.Join(demoDir, "grove-overlay.yml"),
		"GROVE_DEMO":           "1",
		"GROVE_TMUX_SOCKET":    tmuxSocket,
	}
}

// Helper functions for file operations
func createDir(path string) error {
	// Import fs package for directory creation
	return fs.CreateDir(path)
}

func writeFile(path string, data []byte) error {
	// Import fs package for file writing
	return fs.WriteFile(path, data)
}
