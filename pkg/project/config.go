package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/command"
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

// BuildProjectTendBinary finds the project's tend test runner source, builds it, and returns the path to the executable.
// It always rebuilds to ensure the latest changes are included.
func BuildProjectTendBinary(startDir string) (string, error) {
	configPath, err := findGroveYml(startDir)
	if err != nil {
		// Not a grove project, or no config found, so no project-specific binary.
		return "", nil
	}

	projectRoot := filepath.Dir(configPath)

	// 1. Find the source directory for the test runner.
	sourceDirs := []string{
		filepath.Join(projectRoot, "tests", "e2e", "tend"),
		filepath.Join(projectRoot, "tests", "e2e"),
	}
	var sourceDir string
	for _, dir := range sourceDirs {
		if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
			sourceDir = dir
			break
		}
	}

	if sourceDir == "" {
		// This is not an error, it just means the project doesn't have a tend test suite.
		return "", nil
	}

	// 2. Handle mock-building prerequisites.
	makefile := filepath.Join(projectRoot, "Makefile")
	if _, err := os.Stat(makefile); err == nil {
		content, err := os.ReadFile(makefile)
		if err != nil {
			return "", fmt.Errorf("failed to read Makefile at %s: %w", makefile, err)
		}

		mockTargets := []string{"build-e2e-mocks", "build-mocks"}
		for _, target := range mockTargets {
			if strings.Contains(string(content), target+":") {
				cmd := command.New("make", target).Dir(projectRoot).Timeout(2 * time.Minute)
				result := cmd.Run()
				if result.Error != nil {
					return "", fmt.Errorf("failed to run prerequisite 'make %s': %w\nStdout: %s\nStderr: %s",
						target, result.Error, result.Stdout, result.Stderr)
				}
				break // Only run the first mock target found.
			}
		}
	}

	// 3. Compile the test runner.
	binDir := filepath.Join(projectRoot, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory at %s: %w", binDir, err)
	}
	outputPath := filepath.Join(binDir, "tend")

	buildCommand := command.New("go", "build", "-o", outputPath, ".").Dir(sourceDir).Timeout(2 * time.Minute)
	buildResult := buildCommand.Run()
	if buildResult.Error != nil {
		return "", fmt.Errorf("failed to build test runner from %s: %w\nStdout: %s\nStderr: %s",
			sourceDir, buildResult.Error, buildResult.Stdout, buildResult.Stderr)
	}

	return outputPath, nil
}

// FindTendBinary searches for a binary containing "tend" in its name within the project.
// It first finds grove.yml, then looks in common binary locations relative to the project root.
// Deprecated: Use BuildProjectTendBinary instead, which always rebuilds for latest changes.
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