// Package demo provides functionality for generating demo environments.
package demo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/pkg/paths"
	"gopkg.in/yaml.v3"

	"github.com/grovetools/tend/pkg/fs"
)

var ulog = grovelogging.NewUnifiedLogger("grove-tend.demo")

// TmuxSocketName is the name of the tmux socket used for demo environments.
// This is used with tmux's -L flag which creates sockets in the tmux temp directory.
// For multiple demos, this will be parameterized per demo (e.g., "grove-demo-homelab").
const TmuxSocketName = "grove-demo"

// DemosDir returns the standard root directory for all demos.
// Typically ~/.local/share/grove/demos
func DemosDir() string {
	return filepath.Join(paths.DataDir(), "demos")
}

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

	// Create an empty/minimal config first so that CLI commands
	// (grove, nb, flow) work during spec.Generate(). We'll update it after.
	if err := g.createEmptyConfig(); err != nil {
		return fmt.Errorf("creating initial config: %w", err)
	}

	// Copy user's grove.override.yml if it exists (for API keys)
	if err := g.copyUserOverride(); err != nil {
		// Log but don't fail, this is optional
		ulog.Warn("Failed to copy user overrides").Field("error", err).Emit()
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
		ConfigPath:      g.configPath(),
		NotebookDir:     g.notebookDir(),
	}
	return SaveMetadata(g.rootDir, meta)
}

// createEmptyConfig creates a minimal config so CLI commands work during generation.
func (g *Generator) createEmptyConfig() error {
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
	return writeFile(g.configPath(), data)
}

// createDirectoryStructure creates the base directory structure.
// Creates the XDG-compliant directory structure for isolated demo environment.
func (g *Generator) createDirectoryStructure() error {
	dirs := []string{
		g.notebookDir(),
		g.ecosystemsDir(),
		filepath.Join(g.rootDir, "config", "grove"),
		filepath.Join(g.rootDir, "data", "grove"),
		filepath.Join(g.rootDir, "state", "grove"),
		filepath.Join(g.rootDir, "cache", "grove"),
	}

	for _, dir := range dirs {
		if err := createDir(dir); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	return nil
}

// createGlobalConfig creates the grove.yml file in the demo's config directory.
func (g *Generator) createGlobalConfig(content *DemoContent) error {
	config := map[string]interface{}{
		"name":    "grove-demo-" + g.demoName,
		"version": "1.0",
		"groves":  make(map[string]interface{}),
		"notebooks": map[string]interface{}{
			"definitions": make(map[string]interface{}),
		},
		// No need to override explicit_projects or context as we are creating a fresh config
	}

	groves := config["groves"].(map[string]interface{})
	notebooks := config["notebooks"].(map[string]interface{})["definitions"].(map[string]interface{})

	// Standard workspace categories that nb expects
	workspaceCategories := []string{
		"inbox", "issues", "plans", "in_progress", "review",
		"learn", "concepts", "docgen", "icebox", "llm",
		"quick", "todos", "completed", "templates", "recipes",
	}

	for _, eco := range content.Ecosystems {
		groves[eco.Name] = map[string]interface{}{
			"path":        eco.Path,
			"enabled":     true,
			"description": eco.Description,
			"notebook":    eco.Name,
		}

		// Build explicit workspace paths for the ecosystem
		notebookRoot := filepath.Join(g.notebookDir(), eco.Name)
		workspaceRoot := filepath.Join(notebookRoot, "workspaces", eco.Name)

		paths := make(map[string]string)
		for _, cat := range workspaceCategories {
			paths[cat] = filepath.Join(workspaceRoot, cat)
		}

		notebooks[eco.Name] = map[string]interface{}{
			"root_dir": notebookRoot,
			"workspaces": map[string]interface{}{
				eco.Name: map[string]interface{}{
					"paths": paths,
				},
			},
		}
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return writeFile(g.configPath(), data)
}

// copyUserOverride copies filtered settings from the user's override file to the demo.
// Only copies API keys and agent config - excludes groves, explicit_projects, notebooks
// which would break demo isolation.
func (g *Generator) copyUserOverride() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Check standard locations
	sources := []string{
		filepath.Join(home, ".config", "grove", "grove.override.yml"),
		filepath.Join(home, ".config", "grove", "grove.override.yaml"),
	}

	var sourcePath string
	for _, p := range sources {
		if _, err := os.Stat(p); err == nil {
			sourcePath = p
			break
		}
	}

	if sourcePath == "" {
		return nil // No override to copy
	}

	// Read and parse the source file
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	var fullConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &fullConfig); err != nil {
		return err
	}

	// Only keep safe keys that don't affect workspace discovery
	safeKeys := map[string]bool{
		"anthropic": true,
		"gemini":    true,
		"openai":    true,
		"agent":     true,
	}

	filteredConfig := make(map[string]interface{})
	for key, value := range fullConfig {
		if safeKeys[key] {
			filteredConfig[key] = value
		}
	}

	// If nothing to copy, skip creating the file
	if len(filteredConfig) == 0 {
		return nil
	}

	// Write filtered config
	filteredData, err := yaml.Marshal(filteredConfig)
	if err != nil {
		return err
	}

	destPath := filepath.Join(g.rootDir, "config", "grove", "grove.override.yml")
	return os.WriteFile(destPath, filteredData, 0o644) //nolint:gosec // config output file
}

// Helper methods for directory paths
func (g *Generator) notebookDir() string {
	return filepath.Join(g.rootDir, "notebooks")
}

func (g *Generator) ecosystemsDir() string {
	return filepath.Join(g.rootDir, "ecosystems")
}

func (g *Generator) configPath() string {
	return filepath.Join(g.rootDir, "config", "grove", "grove.yml")
}

// BuildEnvironment returns the environment variables for the demo.
// Uses GROVE_HOME for full XDG isolation (config, data, state, cache all isolated)
// and GROVE_BIN to preserve access to the real installed binaries for delegation.
func BuildEnvironment(demoDir, tmuxSocket string) map[string]string {
	// Capture the real bin directory BEFORE GROVE_HOME would affect it
	realBinDir := paths.BinDir()

	return map[string]string{
		"GROVE_HOME":        demoDir,    // Isolates config/data/state/cache
		"GROVE_BIN":         realBinDir, // Preserves binary delegation
		"GROVE_DEMO":        "1",
		"GROVE_TMUX_SOCKET": tmuxSocket,
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
