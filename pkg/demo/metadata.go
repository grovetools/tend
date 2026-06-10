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
	Backend         string          `yaml:"backend,omitempty"` // mux backend: "tmux" or "tuimux"
	TmuxSocket      string          `yaml:"tmux_socket"`
	TmuxSessionName string          `yaml:"tmux_session_name,omitempty"`
	TuimuxSocket    string          `yaml:"tuimux_socket,omitempty"`     // tuimux daemon socket path (tuimux backend)
	TuimuxDaemonPID int             `yaml:"tuimux_daemon_pid,omitempty"` // PID of the tuimux daemon spawned for this demo
	Ecosystems      []EcosystemMeta `yaml:"ecosystems"`
	ConfigPath      string          `yaml:"config_path"`
	NotebookDir     string          `yaml:"notebook_dir"`
	Credentials     *CredentialCopy `yaml:"credentials,omitempty"` // credentials copied into the demo config (names only)
}

// CredentialCopy records which credential keys were copied into the demo
// config and from where. Only key names are recorded, never values.
type CredentialCopy struct {
	SourcePath string   `yaml:"source_path"` // user config file the keys came from
	Keys       []string `yaml:"keys"`        // copied key paths, e.g. "gemini.api_key"
}

// UsesTuimux reports whether this demo was created on the tuimux backend.
// Falls back to checking the tuimux socket for metadata written before the
// backend field existed.
func (m *Metadata) UsesTuimux() bool {
	return m.Backend == "tuimux" || m.TuimuxSocket != "" || m.TuimuxDaemonPID > 0
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
	return os.WriteFile(path, data, 0o644) //nolint:gosec // metadata output file
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
