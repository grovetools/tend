// Package demo provides functionality for generating demo environments.
package demo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/pkg/mux"
	"github.com/grovetools/core/pkg/paths"
	"gopkg.in/yaml.v3"

	"github.com/grovetools/tend/pkg/fs"
)

var ulog = grovelogging.NewUnifiedLogger("grove-tend.demo")

// SocketName returns the tmux socket name for a demo environment.
// This is used with tmux's -L flag which creates sockets in the tmux temp directory.
// Each demo gets its own socket (e.g., "grove-demo-homelab") so multiple demos
// can coexist without sharing a tmux server.
func SocketName(demoName string) string {
	return fmt.Sprintf("grove-demo-%s", demoName)
}

// DemosDir returns the standard root directory for all demos.
// Typically ~/.local/share/grove/demos
func DemosDir() string {
	return filepath.Join(paths.DataDir(), "demos")
}

// Generator creates demo environments.
type Generator struct {
	rootDir         string
	demoName        string
	spec            DemoSpec
	tmuxSocket      string
	copyCredentials bool
}

// Option configures a Generator.
type Option func(*Generator)

// WithoutCredentials disables copying the user's API credentials (from
// ~/.config/grove/grove.override.yml) into the demo config. Demo agents
// will then have no API keys and cannot drive real providers.
func WithoutCredentials() Option {
	return func(g *Generator) { g.copyCredentials = false }
}

// NewGenerator creates a new demo generator for the specified demo type.
func NewGenerator(rootDir, demoName string, opts ...Option) (*Generator, error) {
	spec, err := Get(demoName)
	if err != nil {
		return nil, err
	}

	g := &Generator{
		rootDir:         rootDir,
		demoName:        demoName,
		spec:            spec,
		tmuxSocket:      SocketName(demoName),
		copyCredentials: true,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g, nil
}

// Generate creates the complete demo environment.
func (g *Generator) Generate() error {
	// Fail fast with a single clear error if required tools are missing,
	// rather than failing opaquely mid-generation.
	if err := g.Preflight(); err != nil {
		return err
	}

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

	// Copy the user's API credentials (from grove.override.yml) so demo
	// agents can talk to real providers, unless disabled. Whatever is
	// copied is disclosed loudly below (key names only, never values).
	var credCopy *CredentialCopy
	if g.copyCredentials {
		var err error
		credCopy, err = g.copyUserOverride()
		if err != nil {
			// Log but don't fail, this is optional
			ulog.Warn("Failed to copy user credentials").Field("error", err).Emit()
		}
		if credCopy != nil {
			ulog.Info("Copied credentials into the demo config").
				Field("keys", strings.Join(credCopy.Keys, ", ")).
				Field("source", credCopy.SourcePath).
				Pretty(fmt.Sprintf("Copied credentials into the demo config: %s (from %s)\nOnly key names are shown; values are never printed. Use --no-credentials to skip copying.",
					strings.Join(credCopy.Keys, ", "), credCopy.SourcePath)).
				Emit()
		}
	} else {
		ulog.Info("Skipping credential copy").
			Pretty("Skipping credential copy (--no-credentials); demo agents will have no API keys.").
			Emit()
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

	// Setup mux session if needed
	var muxRes *muxResult
	if content.TmuxNeeded {
		var err error
		muxRes, err = g.setupMux(content)
		if err != nil {
			return fmt.Errorf("setting up mux: %w", err)
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
		Credentials:     credCopy,
	}
	if muxRes != nil {
		meta.Backend = string(muxRes.backend)
		meta.TuimuxSocket = muxRes.tuimuxSocket
		meta.TuimuxDaemonPID = muxRes.tuimuxDaemonPID
	}
	return SaveMetadata(g.rootDir, meta)
}

// Preflight verifies that all external tools required to generate the demo
// are available, returning a single error that lists every missing binary.
//
// Demo generation delegates to grove, nb, and flow CLIs (see homelab_spec.go)
// and needs a mux backend (tmux or tuimux) for the demo session. Delegated
// tools are resolved either from PATH or from the grove bin directory.
func (g *Generator) Preflight() error {
	required := []string{"grove", "nb", "flow"}
	// The mux backend binary depends on which backend setupMux will pick.
	if demoBackend() == mux.MuxTuimux {
		required = append(required, "tuimux")
	} else {
		required = append(required, "tmux")
	}

	binDir := paths.BinDir()
	var missing []string
	for _, tool := range required {
		if _, err := exec.LookPath(tool); err == nil {
			continue
		}
		// Delegated commands also resolve binaries from the grove bin dir.
		if info, err := os.Stat(filepath.Join(binDir, tool)); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			continue
		}
		missing = append(missing, tool)
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools for demo creation: %s (install them or make them available in PATH or %s)",
			strings.Join(missing, ", "), binDir)
	}
	return nil
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

// overrideProviderSections are the top-level provider sections from which
// credential keys are copied into the demo override config.
var overrideProviderSections = []string{"anthropic", "gemini", "openai"}

// overrideCredentialKeys are the only keys copied from each provider section.
// Anything else in a provider section stays out of the demo.
var overrideCredentialKeys = []string{"api_key", "api_key_command"}

// filterOverrideConfig extracts the subset of the user's override config that
// the demo needs: provider API credentials and agent settings. Everything
// else (groves, explicit_projects, notebooks, extra provider settings) is
// excluded so it cannot leak into or break the isolated demo. It returns the
// filtered config plus the list of copied key paths (names only, never
// values) so callers can disclose exactly what was copied.
func filterOverrideConfig(fullConfig map[string]interface{}) (map[string]interface{}, []string) {
	filtered := make(map[string]interface{})
	var copied []string

	for _, section := range overrideProviderSections {
		sectionMap, ok := fullConfig[section].(map[string]interface{})
		if !ok {
			continue
		}
		narrowed := make(map[string]interface{})
		for _, key := range overrideCredentialKeys {
			if v, ok := sectionMap[key]; ok {
				narrowed[key] = v
				copied = append(copied, section+"."+key)
			}
		}
		if len(narrowed) > 0 {
			filtered[section] = narrowed
		}
	}

	// Agent settings are not credentials, but demo agents need them to run
	// the same way the user's agents do.
	if agent, ok := fullConfig["agent"]; ok {
		filtered["agent"] = agent
		if agentMap, isMap := agent.(map[string]interface{}); isMap {
			keys := make([]string, 0, len(agentMap))
			for k := range agentMap {
				keys = append(keys, "agent."+k)
			}
			sort.Strings(keys)
			copied = append(copied, keys...)
		} else {
			copied = append(copied, "agent")
		}
	}

	return filtered, copied
}

// copyUserOverride copies filtered settings from the user's override file to
// the demo: provider API credentials (api_key/api_key_command) and agent
// config only. It returns a record of what was copied (key names and source
// path, never values) for disclosure, or nil if nothing was copied.
func (g *Generator) copyUserOverride() (*CredentialCopy, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
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
		return nil, nil // No override to copy
	}

	// Read and parse the source file
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}

	var fullConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &fullConfig); err != nil {
		return nil, err
	}

	filteredConfig, copiedKeys := filterOverrideConfig(fullConfig)

	// If nothing to copy, skip creating the file
	if len(filteredConfig) == 0 {
		return nil, nil
	}

	// Write filtered config
	filteredData, err := yaml.Marshal(filteredConfig)
	if err != nil {
		return nil, err
	}

	destPath := filepath.Join(g.rootDir, "config", "grove", "grove.override.yml")
	// 0600: this file contains API credentials.
	if err := os.WriteFile(destPath, filteredData, 0o600); err != nil {
		return nil, err
	}
	return &CredentialCopy{SourcePath: sourcePath, Keys: copiedKeys}, nil
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

// TuimuxSocketPath returns the path of the demo's tuimux daemon socket.
func TuimuxSocketPath(demoDir string) string {
	return filepath.Join(demoDir, "state", "tuimux-demo.sock")
}

// BuildEnvironment returns the environment variables for the demo.
// Uses GROVE_HOME for full XDG isolation (config, data, state, cache all isolated)
// and GROVE_BIN to preserve access to the real installed binaries for delegation.
func BuildEnvironment(demoDir, tmuxSocket string) map[string]string {
	// Capture the real bin directory BEFORE GROVE_HOME would affect it
	realBinDir := paths.BinDir()

	env := map[string]string{
		"GROVE_HOME":        demoDir,    // Isolates config/data/state/cache
		"GROVE_BIN":         realBinDir, // Preserves binary delegation
		"GROVE_DEMO":        "1",
		"GROVE_TMUX_SOCKET": tmuxSocket,
	}
	// Include tuimux socket if a demo daemon is running
	tuimuxSocket := TuimuxSocketPath(demoDir)
	if _, err := os.Stat(tuimuxSocket); err == nil {
		env["GROVE_TUIMUX_SOCKET"] = tuimuxSocket
	}
	return env
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
