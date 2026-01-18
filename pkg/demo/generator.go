// Package demo provides functionality for generating demo environments.
package demo

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/git"
	"gopkg.in/yaml.v3"
)

var ulog = grovelogging.NewUnifiedLogger("grove-tend.demo")

// Generator creates demo environments.
type Generator struct {
	rootDir string
}

// NewGenerator creates a new demo generator.
func NewGenerator(rootDir string) *Generator {
	return &Generator{rootDir: rootDir}
}

// Generate creates the complete demo environment.
func (g *Generator) Generate() error {
	ulog.Info("Creating demo environment").
		Field("path", g.rootDir).
		Pretty(fmt.Sprintf("Creating demo environment at: %s", g.rootDir)).
		Emit()

	// Create directory structure
	if err := g.createDirectoryStructure(); err != nil {
		return fmt.Errorf("creating directory structure: %w", err)
	}

	// Create ecosystems
	ecosystems, err := g.createEcosystems()
	if err != nil {
		return fmt.Errorf("creating ecosystems: %w", err)
	}

	// Create global grove config
	if err := g.createGlobalConfig(); err != nil {
		return fmt.Errorf("creating global config: %w", err)
	}

	// Seed notebooks with content
	if err := g.seedNotebooks(); err != nil {
		return fmt.Errorf("seeding notebooks: %w", err)
	}

	// Start tmux session
	if err := g.setupTmux(); err != nil {
		return fmt.Errorf("setting up tmux: %w", err)
	}

	// Write metadata
	meta := &Metadata{
		CreatedAt:   time.Now(),
		TmuxSocket:  TmuxSocketName,
		Ecosystems:  ecosystems,
		OverlayPath: g.overlayPath(),
		NotebookDir: g.notebookDir(),
	}
	if err := SaveMetadata(g.rootDir, meta); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	return nil
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
		if err := fs.CreateDir(dir); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	return nil
}

// createEcosystems creates all three ecosystems.
func (g *Generator) createEcosystems() ([]EcosystemMeta, error) {
	var ecosystems []EcosystemMeta

	// Main ecosystem: homelab (8 repos, 4-6 worktrees)
	homeLab, err := g.createHomelabEcosystem()
	if err != nil {
		return nil, fmt.Errorf("creating homelab ecosystem: %w", err)
	}
	ecosystems = append(ecosystems, *homeLab)

	// Secondary ecosystem: contrib (3 repos)
	contrib, err := g.createContribEcosystem()
	if err != nil {
		return nil, fmt.Errorf("creating contrib ecosystem: %w", err)
	}
	ecosystems = append(ecosystems, *contrib)

	// Secondary ecosystem: infra (2 repos)
	infra, err := g.createInfraEcosystem()
	if err != nil {
		return nil, fmt.Errorf("creating infra ecosystem: %w", err)
	}
	ecosystems = append(ecosystems, *infra)

	return ecosystems, nil
}

// createHomelabEcosystem creates the main "homelab" ecosystem with 8 repos.
func (g *Generator) createHomelabEcosystem() (*EcosystemMeta, error) {
	ecoDir := filepath.Join(g.ecosystemsDir(), "homelab")
	if err := fs.CreateDir(ecoDir); err != nil {
		return nil, err
	}

	// Write ecosystem grove.yml
	if err := g.writeEcosystemConfig(ecoDir, "homelab", []string{
		"dashboard", "sentinel", "vault", "beacon",
		"guardian", "relay", "chronicle", "core",
	}); err != nil {
		return nil, err
	}

	repos := []RepoSpec{
		{Name: "dashboard", Lang: "typescript", Depth: "hero", Worktree: "feature/gpu-widgets", GitState: "dirty-staged"},
		{Name: "sentinel", Lang: "go", Depth: "hero", Worktree: "feature/container-stats", GitState: "clean"},
		{Name: "vault", Lang: "go", Depth: "skeleton", Worktree: "fix/s3-timeout", GitState: "dirty-unstaged"},
		{Name: "beacon", Lang: "go", Depth: "skeleton", Worktree: "feature/passkey-login", GitState: "dirty-unstaged"},
		{Name: "guardian", Lang: "python", Depth: "skeleton", GitState: "clean"},
		{Name: "relay", Lang: "go", Depth: "skeleton", GitState: "clean"},
		{Name: "chronicle", Lang: "python", Depth: "skeleton", GitState: "untracked"},
		{Name: "core", Lang: "go", Depth: "skeleton", GitState: "clean"},
	}

	for _, spec := range repos {
		if err := g.createRepo(ecoDir, spec); err != nil {
			return nil, fmt.Errorf("creating repo %s: %w", spec.Name, err)
		}
	}

	return &EcosystemMeta{
		Name:      "homelab",
		Path:      ecoDir,
		RepoCount: len(repos),
	}, nil
}

// createContribEcosystem creates the "contrib" ecosystem with 3 repos.
func (g *Generator) createContribEcosystem() (*EcosystemMeta, error) {
	ecoDir := filepath.Join(g.ecosystemsDir(), "contrib")
	if err := fs.CreateDir(ecoDir); err != nil {
		return nil, err
	}

	// Write ecosystem grove.yml
	if err := g.writeEcosystemConfig(ecoDir, "contrib", []string{
		"plugin-plex", "plugin-unifi", "theme-catppuccin",
	}); err != nil {
		return nil, err
	}

	repos := []RepoSpec{
		{Name: "plugin-plex", Lang: "python", Depth: "skeleton", GitState: "clean"},
		{Name: "plugin-unifi", Lang: "typescript", Depth: "skeleton", GitState: "clean"},
		{Name: "theme-catppuccin", Lang: "css", Depth: "skeleton", GitState: "clean"},
	}

	for _, spec := range repos {
		if err := g.createRepo(ecoDir, spec); err != nil {
			return nil, fmt.Errorf("creating repo %s: %w", spec.Name, err)
		}
	}

	return &EcosystemMeta{
		Name:      "contrib",
		Path:      ecoDir,
		RepoCount: len(repos),
	}, nil
}

// createInfraEcosystem creates the "infra" ecosystem with 2 repos.
func (g *Generator) createInfraEcosystem() (*EcosystemMeta, error) {
	ecoDir := filepath.Join(g.ecosystemsDir(), "infra")
	if err := fs.CreateDir(ecoDir); err != nil {
		return nil, err
	}

	// Write ecosystem grove.yml
	if err := g.writeEcosystemConfig(ecoDir, "infra", []string{
		"deploy", "charts",
	}); err != nil {
		return nil, err
	}

	repos := []RepoSpec{
		{Name: "deploy", Lang: "hcl", Depth: "skeleton", GitState: "clean"},
		{Name: "charts", Lang: "yaml", Depth: "skeleton", GitState: "clean"},
	}

	for _, spec := range repos {
		if err := g.createRepo(ecoDir, spec); err != nil {
			return nil, fmt.Errorf("creating repo %s: %w", spec.Name, err)
		}
	}

	return &EcosystemMeta{
		Name:      "infra",
		Path:      ecoDir,
		RepoCount: len(repos),
	}, nil
}

// createRepo creates a single repository with the given specification.
func (g *Generator) createRepo(ecoDir string, spec RepoSpec) error {
	repoDir := filepath.Join(ecoDir, spec.Name)
	if err := fs.CreateDir(repoDir); err != nil {
		return err
	}

	// Create files based on depth
	files := g.getRepoFiles(spec)
	for path, content := range files {
		fullPath := filepath.Join(repoDir, path)
		if err := fs.WriteString(fullPath, content); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	// Initialize git repo
	repo, err := git.SetupTestRepo(repoDir)
	if err != nil {
		return fmt.Errorf("initializing git: %w", err)
	}

	// Create initial commit
	if err := repo.AddCommit("Initial commit"); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	// Create worktree if specified
	if spec.Worktree != "" {
		// Sanitize branch name for directory path (replace / with -)
		safeBranchName := strings.ReplaceAll(spec.Worktree, "/", "-")
		// Worktrees go inside the repo's .grove-worktrees directory (EcosystemSubProjectWorktree pattern)
		worktreeDir := filepath.Join(repoDir, ".grove-worktrees", safeBranchName)
		if err := repo.CreateWorktree(worktreeDir, spec.Worktree); err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}

		// Apply git state to worktree
		if err := g.applyGitState(worktreeDir, spec.GitState); err != nil {
			return fmt.Errorf("applying git state: %w", err)
		}
	} else {
		// Apply git state to main repo
		if err := g.applyGitState(repoDir, spec.GitState); err != nil {
			return fmt.Errorf("applying git state: %w", err)
		}
	}

	return nil
}

// applyGitState applies the specified git state to a directory.
func (g *Generator) applyGitState(dir, state string) error {
	switch state {
	case "clean":
		// Nothing to do
		return nil

	case "dirty-staged":
		// Create a file and stage it
		file := filepath.Join(dir, "CHANGELOG.md")
		if err := fs.WriteString(file, "# Changelog\n\n## Unreleased\n- Work in progress\n"); err != nil {
			return err
		}
		repo := git.New(dir)
		return repo.Add("CHANGELOG.md")

	case "dirty-unstaged":
		// Create a modification without staging
		file := filepath.Join(dir, "TODO.md")
		return fs.WriteString(file, "# TODO\n\n- [ ] Fix this issue\n- [ ] Add tests\n")

	case "untracked":
		// Create untracked files
		file := filepath.Join(dir, "notes.txt")
		return fs.WriteString(file, "Quick notes about this project...\n")

	default:
		return nil
	}
}

// getRepoFiles returns the file contents for a repository based on its specification.
func (g *Generator) getRepoFiles(spec RepoSpec) map[string]string {
	files := make(map[string]string)

	// Always include grove.yml and README
	files["grove.yml"] = g.generateGroveYML(spec)
	files["README.md"] = g.generateREADME(spec)

	// Add language-specific files
	switch spec.Lang {
	case "go":
		files["go.mod"] = fmt.Sprintf("module github.com/homelab/%s\n\ngo 1.21\n", spec.Name)
		files["main.go"] = g.generateGoMain(spec)
		if spec.Depth == "hero" {
			g.addGoHeroFiles(spec, files)
		} else {
			files[fmt.Sprintf("pkg/%s/%s.go", spec.Name, spec.Name)] = g.generateGoPackage(spec)
		}

	case "typescript":
		files["package.json"] = g.generatePackageJSON(spec)
		files["tsconfig.json"] = tsConfigContent
		if spec.Depth == "hero" {
			g.addTypeScriptHeroFiles(spec, files)
		} else {
			files["src/index.ts"] = fmt.Sprintf("// %s entry point\nexport const version = '1.0.0';\n", spec.Name)
		}

	case "python":
		files["pyproject.toml"] = g.generatePyProject(spec)
		files[fmt.Sprintf("%s/__init__.py", spec.Name)] = fmt.Sprintf("\"\"\"%s package.\"\"\"\n\n__version__ = \"1.0.0\"\n", spec.Name)

	case "css":
		files["package.json"] = g.generateThemePackageJSON(spec)
		files["src/theme.css"] = catppuccinTheme

	case "hcl":
		files["main.tf"] = terraformMain
		files["variables.tf"] = terraformVars
		files["ansible/playbooks/site.yml"] = ansiblePlaybook

	case "yaml":
		files["Chart.yaml"] = helmChart
		files["values.yaml"] = helmValues
	}

	return files
}

// writeEcosystemConfig writes the grove.yml for an ecosystem.
func (g *Generator) writeEcosystemConfig(ecoDir, name string, workspaces []string) error {
	config := map[string]interface{}{
		"name":       name,
		"version":    "1.0",
		"workspaces": workspaces,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return fs.WriteFile(filepath.Join(ecoDir, "grove.yml"), data)
}

// createGlobalConfig creates the GROVE_CONFIG_OVERLAY file.
// This overlay is merged on top of the user's real grove config,
// overriding workspaces/groves while preserving API keys and other settings.
func (g *Generator) createGlobalConfig() error {
	config := map[string]interface{}{
		"version": "1.0",
		"groves": map[string]interface{}{
			"homelab": map[string]interface{}{
				"path":        filepath.Join(g.ecosystemsDir(), "homelab"),
				"enabled":     true,
				"description": "Main homelab ecosystem",
				"notebook":    "homelab",
			},
			"contrib": map[string]interface{}{
				"path":        filepath.Join(g.ecosystemsDir(), "contrib"),
				"enabled":     true,
				"description": "Community contributions",
				"notebook":    "contrib",
			},
			"infra": map[string]interface{}{
				"path":        filepath.Join(g.ecosystemsDir(), "infra"),
				"enabled":     true,
				"description": "Infrastructure and deployment",
				"notebook":    "infra",
			},
		},
		"notebooks": map[string]interface{}{
			"definitions": map[string]interface{}{
				"homelab": map[string]interface{}{
					"root_dir": filepath.Join(g.notebookDir(), "homelab"),
				},
				"contrib": map[string]interface{}{
					"root_dir": filepath.Join(g.notebookDir(), "contrib"),
				},
				"infra": map[string]interface{}{
					"root_dir": filepath.Join(g.notebookDir(), "infra"),
				},
			},
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return fs.WriteFile(g.overlayPath(), data)
}

// seedNotebooks creates notebook content for the ecosystems.
func (g *Generator) seedNotebooks() error {
	// Create notebook directories
	notebookDirs := []string{
		filepath.Join(g.notebookDir(), "homelab"),
		filepath.Join(g.notebookDir(), "contrib"),
		filepath.Join(g.notebookDir(), "infra"),
	}

	for _, dir := range notebookDirs {
		if err := fs.CreateDir(dir); err != nil {
			return err
		}
	}

	// Seed homelab notes
	if err := g.seedHomelabNotes(); err != nil {
		return err
	}

	// Seed homelab plans
	if err := g.seedHomelabPlans(); err != nil {
		return err
	}

	return nil
}

// TmuxSocketName is the name of the tmux socket used for demo environments.
// This is used with tmux's -L flag which creates sockets in the tmux temp directory.
const TmuxSocketName = "grove-demo"

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
func BuildEnvironment(demoDir string) map[string]string {
	return map[string]string{
		"GROVE_CONFIG_OVERLAY": filepath.Join(demoDir, "grove-overlay.yml"),
		"GROVE_DEMO":           "1",
		"GROVE_TMUX_SOCKET":    TmuxSocketName,
	}
}
