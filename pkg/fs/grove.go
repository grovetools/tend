package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattsolo1/grove-core/config"
	"gopkg.in/yaml.v3"
)

// WriteGroveConfig writes a grove.yml file to the specified directory
func WriteGroveConfig(dir string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := filepath.Join(dir, "grove.yml")
	if err := WriteFile(path, data); err != nil {
		return fmt.Errorf("writing grove.yml: %w", err)
	}

	return nil
}

// WriteBasicGroveConfig writes a minimal, valid grove.yml for testing a project.
func WriteBasicGroveConfig(dir string) error {
	projectName := filepath.Base(dir)
	cfg := &config.Config{
		Name:    projectName,
		Version: "1.0",
		Extensions: map[string]interface{}{
			"description": "A test project for tend scenarios",
			"settings": map[string]interface{}{
				"project_name": projectName,
			},
		},
	}
	return WriteGroveConfig(dir, cfg)
}

// WriteGlobalGroveConfig writes a global grove.yml with search paths into a sandboxed home directory.
// This is useful for testing workspace discovery scenarios.
func WriteGlobalGroveConfig(homeDir string, searchPaths map[string]config.SearchPathConfig) error {
	configDir := filepath.Join(homeDir, ".config", "grove")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating global config dir: %w", err)
	}

	cfg := &config.Config{
		Version:     "1.0",
		SearchPaths: searchPaths,
	}

	return WriteGroveConfig(configDir, cfg)
}