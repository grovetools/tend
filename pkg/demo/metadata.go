package demo

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Metadata stores information about a demo environment.
type Metadata struct {
	DemoName        string          `yaml:"demo_name"`
	CreatedAt       time.Time       `yaml:"created_at"`
	TmuxSocket      string          `yaml:"tmux_socket"`
	TmuxSessionName string          `yaml:"tmux_session_name,omitempty"`
	Ecosystems      []EcosystemMeta `yaml:"ecosystems"`
	ConfigPath      string          `yaml:"config_path"`
	NotebookDir     string          `yaml:"notebook_dir"`
}

// EcosystemMeta contains metadata about a single ecosystem.
type EcosystemMeta struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	RepoCount   int    `yaml:"repo_count"`
	Description string `yaml:"description"`
}

// RepoSpec defines the specification for creating a repository.
type RepoSpec struct {
	Name     string // Repository name
	Lang     string // Primary language (go, typescript, python, css, hcl, yaml)
	Depth    string // "hero" for full depth, "skeleton" for minimal
	Worktree string // Worktree branch name (empty for none)
	GitState string // Git state: clean, dirty-staged, dirty-unstaged, untracked
}

// SaveMetadata saves the demo metadata to disk.
func SaveMetadata(demoDir string, meta *Metadata) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}

	path := filepath.Join(demoDir, ".grove-demo.yml")
	return os.WriteFile(path, data, 0644)
}

// LoadMetadata loads the demo metadata from disk.
func LoadMetadata(demoDir string) (*Metadata, error) {
	path := filepath.Join(demoDir, ".grove-demo.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta Metadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
