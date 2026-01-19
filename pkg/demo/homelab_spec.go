package demo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grovetools/core/util/delegation"
	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/git"
	"gopkg.in/yaml.v3"
)

// HomelabSpec implements the DemoSpec interface for the homelab demo.
// This demo creates a full-featured environment with 3 ecosystems (13 repos total),
// multiple worktrees, realistic git states, notes, and plans.
type HomelabSpec struct{}

// Name returns the unique identifier for this demo type.
func (h *HomelabSpec) Name() string {
	return "homelab"
}

// Description returns a human-readable description of the demo.
func (h *HomelabSpec) Description() string {
	return "Full-featured demo with 3 ecosystems, 13 repos, worktrees, notes, and plans"
}

// Generate creates the homelab demo content.
func (h *HomelabSpec) Generate(rootDir string) (*DemoContent, error) {
	gen := &homelabGenerator{rootDir: rootDir}

	// Create all three ecosystems
	ecosystems, err := gen.createEcosystems()
	if err != nil {
		return nil, err
	}

	// Update the overlay config with ecosystem/notebook definitions
	// This MUST happen before seeding notebooks so CLI commands use demo paths
	if err := gen.createOverlayConfig(ecosystems); err != nil {
		return nil, err
	}

	// Seed notebooks with notes and plans
	if err := gen.seedNotebooks(); err != nil {
		return nil, err
	}

	return &DemoContent{
		Ecosystems: ecosystems,
		TmuxNeeded: true,
	}, nil
}

// homelabGenerator is the internal generator for homelab demo content.
type homelabGenerator struct {
	rootDir string
}

// Helper methods for directory paths
func (g *homelabGenerator) notebookDir() string {
	return filepath.Join(g.rootDir, "notebooks")
}

func (g *homelabGenerator) ecosystemsDir() string {
	return filepath.Join(g.rootDir, "ecosystems")
}

func (g *homelabGenerator) overlayPath() string {
	return filepath.Join(g.rootDir, "grove-overlay.yml")
}

// createEcosystems creates all three ecosystems.
func (g *homelabGenerator) createEcosystems() ([]EcosystemMeta, error) {
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
func (g *homelabGenerator) createHomelabEcosystem() (*EcosystemMeta, error) {
	ecoDir := filepath.Join(g.ecosystemsDir(), "homelab")
	if err := fs.CreateDir(ecoDir); err != nil {
		return nil, err
	}

	// Write ecosystem grove.yml
	if err := g.writeEcosystemConfig(ecoDir, "homelab", []string{
		"dashboard", "sentinel", "vault", "beacon",
		"guardian", "relay", "chronicle", "shared",
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
		{Name: "shared", Lang: "go", Depth: "skeleton", GitState: "clean"},
	}

	for _, spec := range repos {
		if err := g.createRepo(ecoDir, spec); err != nil {
			return nil, fmt.Errorf("creating repo %s: %w", spec.Name, err)
		}
	}

	return &EcosystemMeta{
		Name:        "homelab",
		Path:        ecoDir,
		RepoCount:   len(repos),
		Description: "Main homelab ecosystem",
	}, nil
}

// createContribEcosystem creates the "contrib" ecosystem with 3 repos.
func (g *homelabGenerator) createContribEcosystem() (*EcosystemMeta, error) {
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
		Name:        "contrib",
		Path:        ecoDir,
		RepoCount:   len(repos),
		Description: "Community contributions",
	}, nil
}

// createInfraEcosystem creates the "infra" ecosystem with 2 repos.
func (g *homelabGenerator) createInfraEcosystem() (*EcosystemMeta, error) {
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
		Name:        "infra",
		Path:        ecoDir,
		RepoCount:   len(repos),
		Description: "Infrastructure and deployment",
	}, nil
}

// createRepo creates a single repository with the given specification.
// Uses CLI delegation for base structure, Go for synthetic states.
func (g *homelabGenerator) createRepo(ecoDir string, spec RepoSpec) error {
	// Step 1: Use CLI to create repo with proper grove.yml
	cmd := g.delegatedCommand("grove", "repo", "add", spec.Name,
		"--description", fmt.Sprintf("Demo %s service", spec.Name))
	cmd.Dir = ecoDir
	if err := g.runDelegatedCmd(cmd, "Creating repo "+spec.Name); err != nil {
		return err
	}

	repoDir := filepath.Join(ecoDir, spec.Name)

	// Step 2: Add language-specific files (Go layer)
	files := g.getLanguageFiles(spec)
	for path, content := range files {
		fullPath := filepath.Join(repoDir, path)
		if err := fs.WriteString(fullPath, content); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	// Commit additional files if any were added
	if len(files) > 0 {
		repo := git.New(repoDir)
		if err := repo.Add(); err != nil {
			return fmt.Errorf("staging files: %w", err)
		}
		if err := repo.Commit("Add project structure"); err != nil {
			return fmt.Errorf("committing project structure: %w", err)
		}
	}

	// Step 3: Create worktree if specified (Go layer)
	if spec.Worktree != "" {
		safeBranchName := strings.ReplaceAll(spec.Worktree, "/", "-")
		worktreeDir := filepath.Join(repoDir, ".grove-worktrees", safeBranchName)
		repo := git.New(repoDir)
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
func (g *homelabGenerator) applyGitState(dir, state string) error {
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
		// Modify an existing file without staging it
		// This will show as "modified" in git status
		file := filepath.Join(dir, "README.md")
		existingContent, err := fs.ReadString(file)
		if err != nil {
			// If README doesn't exist, create a modification to grove.yml
			file = filepath.Join(dir, "grove.yml")
			existingContent, _ = fs.ReadString(file)
		}
		return fs.WriteString(file, existingContent+"\n# TODO: Fix this issue\n")

	case "untracked":
		// Create untracked files
		file := filepath.Join(dir, "notes.txt")
		return fs.WriteString(file, "Quick notes about this project...\n")

	default:
		return nil
	}
}

// getLanguageFiles returns the language-specific file contents for a repository.
// Note: grove.yml and README.md are created by the CLI, so we only return
// language-specific files here.
func (g *homelabGenerator) getLanguageFiles(spec RepoSpec) map[string]string {
	files := make(map[string]string)

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
func (g *homelabGenerator) writeEcosystemConfig(ecoDir, name string, workspaces []string) error {
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

// createOverlayConfig creates the GROVE_CONFIG_OVERLAY file with ecosystem definitions.
// This must be called after ecosystems are created but before seeding notebooks.
func (g *homelabGenerator) createOverlayConfig(ecosystems []EcosystemMeta) error {
	config := map[string]interface{}{
		"version": "1.0",
		"groves":  make(map[string]interface{}),
		"notebooks": map[string]interface{}{
			"definitions": make(map[string]interface{}),
		},
	}

	groves := config["groves"].(map[string]interface{})
	notebooks := config["notebooks"].(map[string]interface{})["definitions"].(map[string]interface{})

	for _, eco := range ecosystems {
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
	return fs.WriteFile(g.overlayPath(), data)
}

// seedNotebooks creates notebook content for the ecosystems.
func (g *homelabGenerator) seedNotebooks() error {
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

// runDelegatedCmd executes a delegated CLI command with proper environment.
func (g *homelabGenerator) runDelegatedCmd(cmd *exec.Cmd, description string) error {
	ulog.Debug(description).Field("cmd", cmd.String()).Emit()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\nOutput:\n%s", description, err, string(output))
	}
	return nil
}

// buildCmdEnv returns the environment slice for delegated commands.
func (g *homelabGenerator) buildCmdEnv() []string {
	demoEnv := BuildEnvironment(g.rootDir, TmuxSocketName)
	env := os.Environ()
	for k, v := range demoEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// delegatedCommand creates a delegated command with proper environment.
func (g *homelabGenerator) delegatedCommand(tool string, args ...string) *exec.Cmd {
	cmd := delegation.Command(tool, args...)
	cmd.Env = g.buildCmdEnv()
	return cmd
}

// generateGoMain generates a Go main.go file.
func (g *homelabGenerator) generateGoMain(spec RepoSpec) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/homelab/%s/pkg/%s"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("%s v1.0.0")
	return %s.Start()
}
`, spec.Name, spec.Name, spec.Name, spec.Name)
}

// generateGoPackage generates a Go package file.
func (g *homelabGenerator) generateGoPackage(spec RepoSpec) string {
	return fmt.Sprintf(`// Package %s provides the core functionality.
package %s

// Start initializes and runs the service.
func Start() error {
	return nil
}

// Version returns the current version.
func Version() string {
	return "1.0.0"
}
`, spec.Name, spec.Name)
}

// generatePackageJSON generates a package.json file.
func (g *homelabGenerator) generatePackageJSON(spec RepoSpec) string {
	return fmt.Sprintf(`{
  "name": "@homelab/%s",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint src --ext ts,tsx",
    "test": "vitest"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "@vitejs/plugin-react": "^4.2.0",
    "typescript": "^5.3.0",
    "vite": "^5.0.0",
    "vitest": "^1.0.0"
  }
}
`, spec.Name)
}

// generateThemePackageJSON generates a package.json for CSS theme.
func (g *homelabGenerator) generateThemePackageJSON(spec RepoSpec) string {
	return fmt.Sprintf(`{
  "name": "@homelab/%s",
  "version": "1.0.0",
  "main": "src/theme.css",
  "files": ["src"],
  "keywords": ["catppuccin", "theme", "css"]
}
`, spec.Name)
}

// generatePyProject generates a pyproject.toml file.
func (g *homelabGenerator) generatePyProject(spec RepoSpec) string {
	return fmt.Sprintf(`[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "%s"
version = "1.0.0"
description = "Homelab %s service"
readme = "README.md"
requires-python = ">=3.11"
dependencies = [
    "httpx>=0.25.0",
    "pydantic>=2.5.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.4.0",
    "pytest-asyncio>=0.21.0",
    "ruff>=0.1.0",
]

[tool.ruff]
line-length = 88
target-version = "py311"
`, spec.Name, spec.Name)
}

// addGoHeroFiles adds full-depth Go project files.
func (g *homelabGenerator) addGoHeroFiles(spec RepoSpec, files map[string]string) {
	// cmd files
	files["cmd/root.go"] = fmt.Sprintf(`package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "%s",
	Short: "Homelab %s service",
}

func Execute() error {
	return rootCmd.Execute()
}
`, spec.Name, spec.Name)

	files["cmd/start.go"] = fmt.Sprintf(`package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the %s service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting %s...")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
`, spec.Name, spec.Name)

	// pkg files for sentinel (Go hero)
	if spec.Name == "sentinel" {
		files["pkg/collector/cpu.go"] = `package collector

import "runtime"

// CPUCollector gathers CPU metrics.
type CPUCollector struct{}

// Collect returns CPU usage metrics.
func (c *CPUCollector) Collect() (float64, error) {
	return float64(runtime.NumCPU()), nil
}
`
		files["pkg/collector/memory.go"] = `package collector

import "runtime"

// MemoryCollector gathers memory metrics.
type MemoryCollector struct{}

// Collect returns memory usage metrics.
func (c *MemoryCollector) Collect() (uint64, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc, nil
}
`
		files["pkg/collector/disk.go"] = `package collector

// DiskCollector gathers disk usage metrics.
type DiskCollector struct {
	paths []string
}

// NewDiskCollector creates a disk collector for the given paths.
func NewDiskCollector(paths []string) *DiskCollector {
	return &DiskCollector{paths: paths}
}

// Collect returns disk usage for monitored paths.
func (c *DiskCollector) Collect() (map[string]uint64, error) {
	result := make(map[string]uint64)
	for _, p := range c.paths {
		result[p] = 0 // Placeholder
	}
	return result, nil
}
`
		files["pkg/exporter/prometheus.go"] = `package exporter

import (
	"fmt"
	"net/http"
)

// PrometheusExporter exposes metrics in Prometheus format.
type PrometheusExporter struct {
	port int
}

// NewPrometheusExporter creates a new exporter.
func NewPrometheusExporter(port int) *PrometheusExporter {
	return &PrometheusExporter{port: port}
}

// Start begins serving metrics.
func (e *PrometheusExporter) Start() error {
	http.HandleFunc("/metrics", e.handleMetrics)
	return http.ListenAndServe(fmt.Sprintf(":%d", e.port), nil)
}

func (e *PrometheusExporter) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "# HELP sentinel_up Service availability")
	fmt.Fprintln(w, "# TYPE sentinel_up gauge")
	fmt.Fprintln(w, "sentinel_up 1")
}
`
		files["pkg/config/config.go"] = `package config

// Config holds the sentinel configuration.
type Config struct {
	Port           int      ` + "`yaml:\"port\"`" + `
	Interval       int      ` + "`yaml:\"interval\"`" + `
	EnabledMetrics []string ` + "`yaml:\"enabled_metrics\"`" + `
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Port:     9090,
		Interval: 15,
		EnabledMetrics: []string{"cpu", "memory", "disk"},
	}
}
`
	}

	// internal files
	files["internal/server/server.go"] = fmt.Sprintf(`package server

import (
	"context"
	"net/http"
)

// Server handles HTTP requests.
type Server struct {
	srv *http.Server
}

// New creates a new server.
func New(addr string) *Server {
	return &Server{
		srv: &http.Server{Addr: addr},
	}
}

// Start begins serving.
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
`)
}

// addTypeScriptHeroFiles adds full-depth TypeScript project files.
func (g *homelabGenerator) addTypeScriptHeroFiles(spec RepoSpec, files map[string]string) {
	files["vite.config.ts"] = `import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000
  }
})
`

	files["src/main.tsx"] = `import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
`

	files["src/App.tsx"] = `import { useState } from 'react'
import { Header } from './components/layout/Header'
import { Sidebar } from './components/layout/Sidebar'
import { Dashboard } from './components/Dashboard'

function App() {
  const [sidebarOpen, setSidebarOpen] = useState(true)

  return (
    <div className="app">
      <Header onMenuClick={() => setSidebarOpen(!sidebarOpen)} />
      <div className="app-body">
        {sidebarOpen && <Sidebar />}
        <main className="main-content">
          <Dashboard />
        </main>
      </div>
    </div>
  )
}

export default App
`

	files["src/index.css"] = `* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.app-body {
  display: flex;
  flex: 1;
}

.main-content {
  flex: 1;
  padding: 1rem;
}
`

	files["src/components/Dashboard.tsx"] = `import { CpuWidget } from './widgets/CpuWidget'
import { MemoryWidget } from './widgets/MemoryWidget'
import { DiskWidget } from './widgets/DiskWidget'

export function Dashboard() {
  return (
    <div className="dashboard">
      <h1>System Overview</h1>
      <div className="widget-grid">
        <CpuWidget />
        <MemoryWidget />
        <DiskWidget />
      </div>
    </div>
  )
}
`

	files["src/components/layout/Header.tsx"] = `interface HeaderProps {
  onMenuClick: () => void
}

export function Header({ onMenuClick }: HeaderProps) {
  return (
    <header className="header">
      <button onClick={onMenuClick} className="menu-button">
        Menu
      </button>
      <h1>Homelab Dashboard</h1>
    </header>
  )
}
`

	files["src/components/layout/Sidebar.tsx"] = `export function Sidebar() {
  return (
    <aside className="sidebar">
      <nav>
        <ul>
          <li><a href="#overview">Overview</a></li>
          <li><a href="#services">Services</a></li>
          <li><a href="#containers">Containers</a></li>
          <li><a href="#settings">Settings</a></li>
        </ul>
      </nav>
    </aside>
  )
}
`

	files["src/components/widgets/CpuWidget.tsx"] = `import { useMetrics } from '../../hooks/useMetrics'

export function CpuWidget() {
  const { data, loading } = useMetrics('cpu')

  if (loading) return <div className="widget loading">Loading...</div>

  return (
    <div className="widget cpu-widget">
      <h3>CPU Usage</h3>
      <div className="metric-value">{data?.usage ?? 0}%</div>
    </div>
  )
}
`

	files["src/components/widgets/MemoryWidget.tsx"] = `import { useMetrics } from '../../hooks/useMetrics'

export function MemoryWidget() {
  const { data, loading } = useMetrics('memory')

  if (loading) return <div className="widget loading">Loading...</div>

  return (
    <div className="widget memory-widget">
      <h3>Memory</h3>
      <div className="metric-value">{data?.used ?? 0} / {data?.total ?? 0} GB</div>
    </div>
  )
}
`

	files["src/components/widgets/DiskWidget.tsx"] = `import { useMetrics } from '../../hooks/useMetrics'

export function DiskWidget() {
  const { data, loading } = useMetrics('disk')

  if (loading) return <div className="widget loading">Loading...</div>

  return (
    <div className="widget disk-widget">
      <h3>Disk Usage</h3>
      <div className="metric-value">{data?.usedPercent ?? 0}%</div>
    </div>
  )
}
`

	files["src/hooks/useMetrics.ts"] = `import { useState, useEffect } from 'react'

interface MetricsData {
  [key: string]: number | string
}

export function useMetrics(type: string) {
  const [data, setData] = useState<MetricsData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Simulate API call
    const timer = setTimeout(() => {
      setData({
        usage: Math.floor(Math.random() * 100),
        used: Math.floor(Math.random() * 16),
        total: 16,
        usedPercent: Math.floor(Math.random() * 100),
      })
      setLoading(false)
    }, 500)

    return () => clearTimeout(timer)
  }, [type])

  return { data, loading }
}
`

	files["src/lib/api.ts"] = `const API_BASE = '/api/v1'

export async function fetchMetrics(type: string) {
  const response = await fetch(` + "`${API_BASE}/metrics/${type}`" + `)
  if (!response.ok) throw new Error('Failed to fetch metrics')
  return response.json()
}

export async function fetchServices() {
  const response = await fetch(` + "`${API_BASE}/services`" + `)
  if (!response.ok) throw new Error('Failed to fetch services')
  return response.json()
}
`

	files["src/types/index.ts"] = `export interface Service {
  id: string
  name: string
  status: 'running' | 'stopped' | 'error'
  port?: number
}

export interface Metric {
  name: string
  value: number
  unit: string
  timestamp: Date
}

export interface Container {
  id: string
  name: string
  image: string
  status: string
  ports: string[]
}
`
}

// noteSpec defines a note to be created.
type noteSpec struct {
	title    string
	noteType string
	body     string
}

// seedHomelabNotes creates notes for the homelab ecosystem using CLI delegation.
func (g *homelabGenerator) seedHomelabNotes() error {
	// Set working directory to dashboard repo for git context
	dashboardDir := filepath.Join(g.ecosystemsDir(), "homelab", "dashboard")

	notes := []noteSpec{
		{
			title:    "Component library evaluation - shadcn vs Radix",
			noteType: "research",
			body: `# Component Library Evaluation

## Options Considered

### shadcn/ui
- Pros: Copy-paste components, full control, Tailwind-based
- Cons: More manual setup required

### Radix UI
- Pros: Unstyled primitives, accessibility built-in
- Cons: Need to style everything

## Decision

Going with shadcn/ui for the dashboard. Better DX and we already use Tailwind.
`,
		},
		{
			title:    "Dark mode implementation notes",
			noteType: "research",
			body: `# Dark Mode Implementation

## Approach

Using CSS custom properties with a data-theme attribute on the root.

## Theme Toggle

- Store preference in localStorage
- Respect system preference via prefers-color-scheme
- Add toggle button in header

## Colors

Using Catppuccin Mocha as the dark theme base.
`,
		},
		{
			title:    "Widget drag-and-drop research",
			noteType: "research",
			body: `# Drag and Drop Research

## Libraries Evaluated

1. **react-dnd** - Most flexible, good ecosystem
2. **@dnd-kit/core** - Modern, accessible, lightweight
3. **react-beautiful-dnd** - Deprecated by Atlassian

## Recommendation

Use @dnd-kit/core - best balance of features and bundle size.
`,
		},
		{
			title:    "Prometheus vs custom metrics format",
			noteType: "research",
			body: `# Metrics Format Decision

## Prometheus Format

Standard, well-supported, compatible with Grafana.

## Custom JSON Format

More flexible, easier to extend, but less ecosystem support.

## Decision

Use Prometheus format for sentinel. Standard tooling is worth it.
`,
		},
		{
			title:    "Container runtime detection",
			noteType: "research",
			body: `# Container Runtime Detection

Need to support multiple container runtimes:

- Docker
- Podman
- containerd (via nerdctl)

## Detection Strategy

1. Check for socket files in standard locations
2. Try to connect and query version
3. Fall back to CLI detection

## Implementation

Use a runtime interface with specific implementations for each.
`,
		},
	}

	for _, note := range notes {
		// Create note via CLI with stdin
		cmd := g.delegatedCommand("nb", "new", note.title,
			"-t", note.noteType,
			"--no-edit",
			"--stdin")
		cmd.Dir = dashboardDir
		cmd.Stdin = strings.NewReader(note.body)

		if err := g.runDelegatedCmd(cmd, "Creating note: "+note.title); err != nil {
			return fmt.Errorf("creating note %s: %w", note.title, err)
		}
	}

	return nil
}

// jobSpec defines a job to be created in a plan.
type jobSpec struct {
	title     string
	jobType   string
	prompt    string
	status    string // target status after creation
	dependsOn int    // index of dependency job (0-based), -1 for no dependency
}

// seedHomelabPlans creates plans for the homelab ecosystem using CLI delegation.
func (g *homelabGenerator) seedHomelabPlans() error {
	// Run from dashboard repo for workspace context (like notes)
	dashboardDir := filepath.Join(g.ecosystemsDir(), "homelab", "dashboard")

	// Create GPU Monitoring plan - realistic multi-job plan
	if err := g.createPlanWithJobs(dashboardDir, "gpu-monitoring", []jobSpec{
		{
			title:     "Define GPU Monitoring Spec",
			jobType:   "oneshot",
			prompt:    "Research NVIDIA and AMD GPU monitoring APIs. Define metrics to collect including utilization, memory, temperature, and power consumption.",
			status:    "completed",
			dependsOn: -1,
		},
		{
			title:     "Implement Sentinel Collector",
			jobType:   "headless_agent",
			prompt:    "Add GPU metrics collector to sentinel using nvidia-smi. Implement the Collector interface and add GPU-specific metrics.",
			status:    "completed",
			dependsOn: 0, // depends on job 0
		},
		{
			title:     "Add Dashboard Widget",
			jobType:   "headless_agent",
			prompt:    "Create GPU widget component in the dashboard. Display real-time GPU utilization, memory usage, and temperature graphs.",
			status:    "running",
			dependsOn: 1, // depends on job 1
		},
		{
			title:     "Write Integration Tests",
			jobType:   "oneshot",
			prompt:    "Write e2e tests for GPU monitoring using tend. Test collector data flow and dashboard rendering.",
			status:    "pending",
			dependsOn: 2, // depends on job 2
		},
	}); err != nil {
		return fmt.Errorf("creating gpu-monitoring plan: %w", err)
	}

	// Create Security Hardening plan - completed plan
	if err := g.createPlanWithJobs(dashboardDir, "security-hardening", []jobSpec{
		{
			title:     "Research WebAuthn Passkey",
			jobType:   "oneshot",
			prompt:    "Research WebAuthn/Passkey implementation options. Evaluate libraries and browser support.",
			status:    "completed",
			dependsOn: -1,
		},
		{
			title:     "Add Passkey Support to Beacon",
			jobType:   "headless_agent",
			prompt:    "Implement passkey authentication in the beacon service. Add registration and login flows.",
			status:    "completed",
			dependsOn: 0,
		},
		{
			title:     "Implement Audit Logging",
			jobType:   "headless_agent",
			prompt:    "Add comprehensive audit logging for authentication events. Log to structured JSON format.",
			status:    "completed",
			dependsOn: 1,
		},
	}); err != nil {
		return fmt.Errorf("creating security-hardening plan: %w", err)
	}

	// Create v2 Roadmap plan - pending plan
	if err := g.createPlanWithJobs(dashboardDir, "v2-roadmap", []jobSpec{
		{
			title:     "Gather Community Feedback",
			jobType:   "oneshot",
			prompt:    "Collect and organize feature requests from GitHub issues and discussions.",
			status:    "pending",
			dependsOn: -1,
		},
		{
			title:     "Define Feature Priorities",
			jobType:   "oneshot",
			prompt:    "Prioritize features based on community feedback and technical feasibility.",
			status:    "pending",
			dependsOn: 0,
		},
		{
			title:     "Create Technical Design Docs",
			jobType:   "headless_agent",
			prompt:    "Write technical design documents for the top priority features.",
			status:    "pending",
			dependsOn: 1,
		},
	}); err != nil {
		return fmt.Errorf("creating v2-roadmap plan: %w", err)
	}

	return nil
}

// createPlanWithJobs creates a plan directory and adds jobs using CLI delegation.
// repoDir is the repo to run from (for workspace context).
func (g *homelabGenerator) createPlanWithJobs(repoDir, planName string, jobs []jobSpec) error {
	// Step 1: Initialize plan directory via CLI (no recipe = minimal plan)
	// Running from repoDir gives us workspace context
	cmd := g.delegatedCommand("flow", "plan", "init", planName)
	cmd.Dir = repoDir
	if err := g.runDelegatedCmd(cmd, "Creating "+planName+" plan"); err != nil {
		return err
	}

	// Find the created plan directory
	planDir, err := g.findPlanDir(repoDir, planName)
	if err != nil {
		return fmt.Errorf("finding plan directory: %w", err)
	}

	// Track created job filenames for dependency resolution
	jobFilenames := make([]string, 0, len(jobs))

	// Step 2: Add jobs via CLI
	for i, job := range jobs {
		args := []string{"plan", "add", planDir,
			"-t", job.jobType,
			"--title", job.title,
			"-p", job.prompt,
		}
		if job.dependsOn >= 0 && job.dependsOn < len(jobFilenames) {
			args = append(args, "-d", jobFilenames[job.dependsOn])
		}

		cmd := g.delegatedCommand("flow", args...)
		if err := g.runDelegatedCmd(cmd, "Adding job: "+job.title); err != nil {
			return err
		}

		// Find the created job file (it's the most recent one with the expected number prefix)
		jobFile, err := g.findJobFile(planDir, i+1)
		if err != nil {
			return fmt.Errorf("finding job file for %s: %w", job.title, err)
		}
		jobFilenames = append(jobFilenames, filepath.Base(jobFile))

		// Step 3: Update job status via CLI
		if job.status == "completed" {
			// Use flow plan complete CLI
			cmd := g.delegatedCommand("flow", "plan", "complete", jobFile)
			if err := g.runDelegatedCmd(cmd, "Completing job: "+job.title); err != nil {
				return fmt.Errorf("completing job %s: %w", job.title, err)
			}
		} else if job.status == "running" {
			// "running" is a synthetic demo state - use manual update
			if err := g.updateJobStatusFile(jobFile, job.status); err != nil {
				return fmt.Errorf("updating job status for %s: %w", job.title, err)
			}
		}
		// "pending" is the default, no action needed
	}

	return nil
}

// findPlanDir finds the plan directory created by flow plan init.
// It searches in the notebook workspace for the given plan name.
func (g *homelabGenerator) findPlanDir(repoDir, planName string) (string, error) {
	// Plans are created in notebooks/<ecosystem>/workspaces/<repo>/plans/<planName>
	// We need to search for the plan directory
	pattern := filepath.Join(g.notebookDir(), "*", "workspaces", "*", "plans", planName)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("plan directory not found: %s", planName)
	}
	return matches[0], nil
}

// findJobFile finds the job file with the given number prefix.
func (g *homelabGenerator) findJobFile(planDir string, jobNum int) (string, error) {
	prefix := fmt.Sprintf("%02d-", jobNum)
	pattern := filepath.Join(planDir, prefix+"*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no job file found matching %s", pattern)
	}
	return matches[0], nil
}

// updateJobStatusFile modifies a job file's status field.
// Used only for synthetic "running" status - completed jobs use flow plan complete CLI.
func (g *homelabGenerator) updateJobStatusFile(filePath, status string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Update the status field in frontmatter
	statusRe := regexp.MustCompile(`(?m)^status:\s*.*$`)
	newContent := statusRe.ReplaceAllString(string(content), "status: "+status)

	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// Register the homelab spec with the registry on package initialization.
func init() {
	Register(&HomelabSpec{})
}
