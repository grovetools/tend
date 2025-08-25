package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GroveConfig defines the structure of a grove.yml file.
type GroveConfig struct {
	Binary struct {
		Name string `yaml:"name"`
		Path string `yaml:"path"`
	} `yaml:"binary"`
}

// GetBinaryPath finds the project's main binary by searching for grove.yml
// starting from the given root directory and walking up.
func GetBinaryPath(startDir string) (string, error) {
	configPath, err := findGroveYml(startDir)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("reading grove.yml at %s: %w", configPath, err)
	}

	var config GroveConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("parsing grove.yml at %s: %w", configPath, err)
	}

	if config.Binary.Path == "" {
		return "", fmt.Errorf("binary.path not defined in %s", configPath)
	}

	// The path in grove.yml is relative to the directory containing it.
	binaryFullPath := filepath.Join(filepath.Dir(configPath), config.Binary.Path)

	return filepath.Abs(binaryFullPath)
}

// findGroveYml searches for grove.yml starting from dir and moving upwards.
func findGroveYml(dir string) (string, error) {
	currentDir := dir
	for {
		configPath := filepath.Join(currentDir, "grove.yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Go up one level
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached the root of the filesystem
			break
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf("grove.yml not found in or above %s", dir)
}

// FindTendBinary searches for a binary containing "tend" in its name within the project.
// It first finds grove.yml, then looks in common binary locations relative to the project root.
func FindTendBinary(startDir string) (string, error) {
	configPath, err := findGroveYml(startDir)
	if err != nil {
		return "", err
	}

	projectRoot := filepath.Dir(configPath)
	
	// Common locations to search for binaries
	searchPaths := []string{
		"bin",
		".",
		"cmd",
		"scripts",
	}

	// First pass: look for exact match "tend" or "tend.exe"
	for _, searchPath := range searchPaths {
		binDir := filepath.Join(projectRoot, searchPath)
		
		// Check for exact matches first
		exactMatches := []string{"tend", "tend.exe"}
		for _, exactName := range exactMatches {
			fullPath := filepath.Join(binDir, exactName)
			info, err := os.Stat(fullPath)
			if err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
				return filepath.Abs(fullPath)
			}
		}
	}

	// Second pass: look for binaries starting with "tend-"
	for _, searchPath := range searchPaths {
		binDir := filepath.Join(projectRoot, searchPath)
		
		// Check if directory exists
		if info, err := os.Stat(binDir); err != nil || !info.IsDir() {
			continue
		}

		// Look for files starting with "tend-"
		entries, err := os.ReadDir(binDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Check if filename starts with "tend-"
			if len(name) >= 5 && name[:5] == "tend-" {
				fullPath := filepath.Join(binDir, name)
				
				// Check if it's executable
				info, err := os.Stat(fullPath)
				if err == nil && info.Mode()&0111 != 0 {
					return filepath.Abs(fullPath)
				}
			}
		}
	}

	return "", fmt.Errorf("no 'tend' binary found in project %s", projectRoot)
}