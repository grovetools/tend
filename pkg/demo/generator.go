// Package demo provides functionality for generating demo environments.
package demo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/pkg/mux"
	"github.com/grovetools/core/pkg/paths"
	"github.com/pelletier/go-toml/v2"

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
	rootDir    string
	demoName   string
	spec       DemoSpec
	tmuxSocket string
	withoutMux bool
	progress   func(step string)
}

// Option configures a Generator.
type Option func(*Generator)

// WithoutMux skips the mux setup step entirely: no tmux server or tuimux
// daemon is spawned for the demo, and Preflight stops requiring a mux
// binary. For embedding callers (treemux's splash) the host terminal IS
// the demo's terminal — it execs into the demo env itself, so a demo mux
// session would only be an orphaned process to tear down later.
func WithoutMux() Option {
	return func(g *Generator) { g.withoutMux = true }
}

// WithProgress registers a callback invoked at the start of each Generate
// step with a short human-readable label. It is called from Generate's
// goroutine (whatever goroutine the caller runs Generate on), so UI
// callers must marshal it onto their event loop themselves.
func WithProgress(fn func(step string)) Option {
	return func(g *Generator) { g.progress = fn }
}

// NewGenerator creates a new demo generator for the specified demo type.
func NewGenerator(rootDir, demoName string, opts ...Option) (*Generator, error) {
	spec, err := Get(demoName)
	if err != nil {
		return nil, err
	}

	g := &Generator{
		rootDir:    rootDir,
		demoName:   demoName,
		spec:       spec,
		tmuxSocket: SocketName(demoName),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g, nil
}

// reportStep surfaces a generation step to the WithProgress callback.
func (g *Generator) reportStep(step string) {
	if g.progress != nil {
		g.progress(step)
	}
}

// Generate creates the complete demo environment.
func (g *Generator) Generate() error {
	// Fail fast with a single clear error if required tools are missing,
	// rather than failing opaquely mid-generation.
	g.reportStep("checking required tools")
	if err := g.Preflight(); err != nil {
		return err
	}

	ulog.Info("Creating demo environment").
		Field("path", g.rootDir).
		Field("demo", g.demoName).
		Pretty(fmt.Sprintf("Creating %s demo environment at: %s", g.demoName, g.rootDir)).
		Emit()

	// Create directory structure
	g.reportStep("creating directory structure")
	if err := g.createDirectoryStructure(); err != nil {
		return fmt.Errorf("creating directory structure: %w", err)
	}

	// Create an empty/minimal config first so that CLI commands
	// (grove, nb, flow) work during spec.Generate(). We'll update it after.
	g.reportStep("writing initial config")
	if err := g.createEmptyConfig(); err != nil {
		return fmt.Errorf("creating initial config: %w", err)
	}

	// Sync the user's real [tui] choices (theme, keybindings, focus) into the
	// demo so it feels native. One-way (real → demo); non-fatal on failure so
	// a missing theme never blocks generation.
	g.reportStep("syncing tui config")
	if err := SyncUserTUIConfig(g.rootDir); err != nil {
		ulog.Warn("Failed to sync user TUI config into demo").Field("error", err).Emit()
	}

	// Let spec generate content (ecosystems, repos, notes, plans)
	g.reportStep("generating ecosystems, repos, notes, and plans")
	content, err := g.spec.Generate(g.rootDir)
	if err != nil {
		return fmt.Errorf("generating demo content: %w", err)
	}

	// Update overlay config with full ecosystem information
	g.reportStep("writing global config")
	if err := g.createGlobalConfig(content); err != nil {
		return fmt.Errorf("creating global config: %w", err)
	}

	// Setup mux session if needed (skipped with WithoutMux — the embedding
	// caller is the demo's terminal, so a mux server would just be an
	// orphan to tear down later).
	var muxRes *muxResult
	if content.TmuxNeeded && !g.withoutMux {
		g.reportStep("starting mux session")
		var err error
		muxRes, err = g.setupMux(content)
		if err != nil {
			return fmt.Errorf("setting up mux: %w", err)
		}
	}

	// Save metadata
	g.reportStep("saving metadata")
	meta := &Metadata{
		DemoName:        g.demoName,
		CreatedAt:       time.Now(),
		TmuxSocket:      g.tmuxSocket,
		TmuxSessionName: fmt.Sprintf("grove-demo-%s", g.demoName),
		Ecosystems:      content.Ecosystems,
		ConfigPath:      g.configPath(),
		NotebookDir:     g.notebookDir(),
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
	// With WithoutMux there is no mux step, so no mux binary is required.
	if !g.withoutMux {
		if demoBackend() == mux.MuxTuimux {
			required = append(required, "tuimux")
		} else {
			required = append(required, "tmux")
		}
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
		"groves": make(map[string]interface{}),
		"notebooks": map[string]interface{}{
			"definitions": make(map[string]interface{}),
		},
	}

	data, err := toml.Marshal(config)
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

// createGlobalConfig creates the grove.toml file in the demo's config directory.
func (g *Generator) createGlobalConfig(content *DemoContent) error {
	config := demoGlobalConfig(g.notebookDir(), content.Ecosystems)
	config["name"] = "grove-demo-" + g.demoName
	// No need to override explicit_projects or context as we are creating a fresh config

	data, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	return writeFile(g.configPath(), data)
}

// demoGlobalConfig builds the modern grove.toml payload for the demo's global
// config layer: one [groves.<eco>] source and one [notebooks.definitions.<eco>]
// per ecosystem. Only root_dir is set on each notebook definition — the demo
// seeds notes under <root_dir>/workspaces/<eco>/<category>, which is exactly
// what core's default path templates resolve to (see
// core/pkg/workspace/notebook_locator.go, defaultNotesPathTemplate et al.), so
// no explicit path templates are needed. The legacy YAML emission additionally
// wrote a notebooks.definitions.<eco>.workspaces.<eco>.paths block; that shape
// has no counterpart in the modern config.Notebook struct and was silently
// dropped by config parsing.
func demoGlobalConfig(notebookDir string, ecosystems []EcosystemMeta) map[string]interface{} {
	groves := make(map[string]interface{})
	definitions := make(map[string]interface{})

	for _, eco := range ecosystems {
		groves[eco.Name] = map[string]interface{}{
			"path":        eco.Path,
			"enabled":     true,
			"description": eco.Description,
			"notebook":    eco.Name,
		}
		definitions[eco.Name] = map[string]interface{}{
			"root_dir": filepath.Join(notebookDir, eco.Name),
		}
	}

	return map[string]interface{}{
		"groves": groves,
		"notebooks": map[string]interface{}{
			"definitions": definitions,
		},
	}
}

// Helper methods for directory paths
func (g *Generator) notebookDir() string {
	return filepath.Join(g.rootDir, "notebooks")
}

func (g *Generator) ecosystemsDir() string {
	return filepath.Join(g.rootDir, "ecosystems")
}

func (g *Generator) configPath() string {
	return filepath.Join(g.rootDir, "config", "grove", "grove.toml")
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
