package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grovetools/core/config"
	"github.com/grovetools/tend/pkg/command"
)

// BinaryConfig holds the binary configuration from the grove config's [binary] section.
type BinaryConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// GetBinaryPath finds the project's main binary by searching for grove config
// (grove.toml or grove.yml) starting from the given root directory and walking up.
func GetBinaryPath(startDir string) (string, error) {
	configPath, err := config.FindConfigFile(startDir)
	if err != nil {
		return "", fmt.Errorf("config file not found in or above %s: %w", startDir, err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return "", fmt.Errorf("loading config at %s: %w", configPath, err)
	}

	var binaryCfg BinaryConfig
	if err := cfg.UnmarshalExtension("binary", &binaryCfg); err != nil {
		return "", fmt.Errorf("parsing binary config at %s: %w", configPath, err)
	}

	if binaryCfg.Path == "" {
		return "", fmt.Errorf("binary.path not defined in %s", configPath)
	}

	// The path in config is relative to the directory containing it.
	binaryFullPath := filepath.Join(filepath.Dir(configPath), binaryCfg.Path)

	return filepath.Abs(binaryFullPath)
}

// gitInfo holds version information from git.
type gitInfo struct {
	Commit    string
	Branch    string
	IsDirty   bool
	BuildDate string
}

// getGitInfo retrieves git version information from the project directory.
func getGitInfo(projectRoot string) gitInfo {
	info := gitInfo{
		Commit:    "unknown",
		Branch:    "unknown",
		BuildDate: time.Now().UTC().Format(time.RFC3339),
	}

	// Get commit hash
	commitCmd := command.New("git", "rev-parse", "--short", "HEAD").Dir(projectRoot).Timeout(5 * time.Second)
	if result := commitCmd.Run(); result.Error == nil {
		info.Commit = strings.TrimSpace(result.Stdout)
	}

	// Get branch name
	branchCmd := command.New("git", "rev-parse", "--abbrev-ref", "HEAD").Dir(projectRoot).Timeout(5 * time.Second)
	if result := branchCmd.Run(); result.Error == nil {
		info.Branch = strings.TrimSpace(result.Stdout)
	}

	// Check if dirty
	statusCmd := command.New("git", "status", "--porcelain").Dir(projectRoot).Timeout(5 * time.Second)
	if result := statusCmd.Run(); result.Error == nil && strings.TrimSpace(result.Stdout) != "" {
		info.IsDirty = true
	}

	return info
}

// buildLDFlags constructs the -ldflags string for injecting version info.
func buildLDFlags(info gitInfo) string {
	versionPkg := "github.com/grovetools/core/version"
	version := info.Branch + "-" + info.Commit
	if info.IsDirty {
		version += "-dirty"
	}

	return fmt.Sprintf("-X '%s.Version=%s' -X '%s.Commit=%s' -X '%s.Branch=%s' -X '%s.BuildDate=%s'",
		versionPkg, version,
		versionPkg, info.Commit,
		versionPkg, info.Branch,
		versionPkg, info.BuildDate,
	)
}

// buildMocksByConvention builds mock binaries from conventional locations.
// It looks for directories under <sourceDir>/mocks/src/ or <sourceDir>/tend/mocks/src/
// and builds each one that contains a main.go to the corresponding mocks/bin/mock-<dirname>.
func buildMocksByConvention(sourceDir string) error {
	// Check both conventional locations
	mockLocations := []struct {
		srcDir string
		binDir string
	}{
		{filepath.Join(sourceDir, "mocks", "src"), filepath.Join(sourceDir, "mocks", "bin")},
		{filepath.Join(sourceDir, "tend", "mocks", "src"), filepath.Join(sourceDir, "tend", "mocks", "bin")},
	}

	var mockSrcDir, mockBinDir string
	for _, loc := range mockLocations {
		if _, err := os.Stat(loc.srcDir); err == nil {
			mockSrcDir = loc.srcDir
			mockBinDir = loc.binDir
			break
		}
	}

	if mockSrcDir == "" {
		// No mocks directory found, nothing to build
		return nil
	}
	if err := os.MkdirAll(mockBinDir, 0755); err != nil {
		return fmt.Errorf("failed to create mock bin directory: %w", err)
	}

	entries, err := os.ReadDir(mockSrcDir)
	if err != nil {
		return fmt.Errorf("failed to read mock source directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		mockName := entry.Name()
		mockSrc := filepath.Join(mockSrcDir, mockName)

		// Check if this directory has a main.go
		if _, err := os.Stat(filepath.Join(mockSrc, "main.go")); os.IsNotExist(err) {
			continue
		}

		// Build the mock
		outputPath := filepath.Join(mockBinDir, "mock-"+mockName)
		buildCmd := command.New("go", "build", "-o", outputPath, ".").Dir(mockSrc).Timeout(30 * time.Second)
		result := buildCmd.Run()
		if result.Error != nil {
			return fmt.Errorf("failed to build mock %s: %w\nStderr: %s", mockName, result.Error, result.Stderr)
		}
	}

	return nil
}

// BuildProjectTendBinary finds the project's tend test runner source, builds it, and returns the path to the executable.
// It always rebuilds to ensure the latest changes are included.
func BuildProjectTendBinary(startDir string) (string, error) {
	configPath, err := config.FindConfigFile(startDir)
	if err != nil {
		// Not a grove project, or no config found, so no project-specific binary.
		return "", nil
	}

	projectRoot := filepath.Dir(configPath)

	// Load the config to get the binary name
	cfg, err := config.Load(configPath)
	if err != nil {
		return "", fmt.Errorf("loading config at %s: %w", configPath, err)
	}

	var binaryCfg BinaryConfig
	if err := cfg.UnmarshalExtension("binary", &binaryCfg); err != nil {
		return "", fmt.Errorf("parsing binary config at %s: %w", configPath, err)
	}

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

	// 2. Build mocks by convention (tests/e2e/tend/mocks/src/<name>/ → mocks/bin/mock-<name>)
	if err := buildMocksByConvention(sourceDir); err != nil {
		return "", fmt.Errorf("failed to build mocks: %w", err)
	}

	// 3. Get git info for version injection.
	gitInfo := getGitInfo(projectRoot)
	ldflags := buildLDFlags(gitInfo)

	// 4. Compile the test runner with version info.
	binDir := filepath.Join(projectRoot, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory at %s: %w", binDir, err)
	}

	// Determine the output binary name.
	// If the project's main binary is named "tend", use a different name for the test runner
	// to avoid overwriting the library binary (important for grove-tend itself).
	outputName := "tend"
	if binaryCfg.Name == "tend" {
		outputName = "tend-e2e"
	}
	outputPath := filepath.Join(binDir, outputName)

	buildCommand := command.New("go", "build", "-ldflags", ldflags, "-o", outputPath, ".").Dir(sourceDir).Timeout(2 * time.Minute)
	buildResult := buildCommand.Run()
	if buildResult.Error != nil {
		return "", fmt.Errorf("failed to build test runner from %s: %w\nStdout: %s\nStderr: %s",
			sourceDir, buildResult.Error, buildResult.Stdout, buildResult.Stderr)
	}

	return outputPath, nil
}

// FindTendBinary searches for a binary containing "tend" in its name within the project.
// It first finds grove config, then looks in common binary locations relative to the project root.
// Deprecated: Use BuildProjectTendBinary instead, which always rebuilds for latest changes.
func FindTendBinary(startDir string) (string, error) {
	configPath, err := config.FindConfigFile(startDir)
	if err != nil {
		return "", fmt.Errorf("config file not found in or above %s: %w", startDir, err)
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