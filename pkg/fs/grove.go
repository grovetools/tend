package fs

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GroveConfig represents a simplified grove.yml structure for testing
type GroveConfig struct {
	WorkspaceRoot string                 `yaml:"workspace_root,omitempty"`
	Services      map[string]ServiceSpec `yaml:"services,omitempty"`
	Agent         *AgentConfig           `yaml:"agent,omitempty"`
}

// ServiceSpec represents a service definition
type ServiceSpec struct {
	Image   string            `yaml:"image,omitempty"`
	Port    int               `yaml:"port,omitempty"`
	Command []string          `yaml:"command,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// AgentConfig represents agent configuration
type AgentConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Port    int    `yaml:"port,omitempty"`
	Image   string `yaml:"image,omitempty"`
}

// WriteGroveConfig writes a grove.yml file to the specified directory
func WriteGroveConfig(dir string, config *GroveConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := filepath.Join(dir, "grove.yml")
	if err := WriteFile(path, data); err != nil {
		return fmt.Errorf("writing grove.yml: %w", err)
	}

	return nil
}

// WriteBasicGroveConfig writes a minimal grove.yml for testing
func WriteBasicGroveConfig(dir string) error {
	config := &GroveConfig{
		WorkspaceRoot: ".",
		Services: map[string]ServiceSpec{
			"test-service": {
				Image:   "alpine:latest",
				Port:    8080,
				Command: []string{"sleep", "infinity"},
			},
		},
	}

	return WriteGroveConfig(dir, config)
}