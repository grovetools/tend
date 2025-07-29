package tend_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/grovepm/grove-tend/internal/harness"
	"github.com/grovepm/grove-tend/pkg/fs"
	"github.com/grovepm/grove-tend/pkg/git"
)

func TestCoreAbstractions(t *testing.T) {
	// Test Context
	ctx := &harness.Context{
		RootDir:     "/tmp/test",
		GroveBinary: "grove",
	}

	// Test directory management
	mainDir := ctx.NewDir("main")
	if mainDir != "/tmp/test/main" {
		t.Errorf("Expected /tmp/test/main, got %s", mainDir)
	}

	if ctx.Dir("main") != mainDir {
		t.Errorf("Dir() returned different path")
	}

	// Test value storage
	ctx.Set("test-key", "test-value")
	if ctx.GetString("test-key") != "test-value" {
		t.Errorf("GetString() failed")
	}
}

func TestFilesystemHelpers(t *testing.T) {
	// Create temp directory manager
	tempMgr, err := fs.NewTempDirManager("grove-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir manager: %v", err)
	}
	defer tempMgr.Cleanup()

	// Test file operations
	testDir, err := tempMgr.CreateDir("test")
	if err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Write a file
	testFile := filepath.Join(testDir, "test.txt")
	if err := fs.WriteString(testFile, "Hello, Grove!"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Check file exists
	if !fs.Exists(testFile) {
		t.Error("File should exist")
	}

	if !fs.IsFile(testFile) {
		t.Error("Should be a file")
	}

	// Test Grove config
	if err := fs.WriteBasicGroveConfig(testDir); err != nil {
		t.Fatalf("Failed to write grove config: %v", err)
	}

	groveYml := filepath.Join(testDir, "grove.yml")
	if !fs.Exists(groveYml) {
		t.Error("grove.yml should exist")
	}
}

func TestGitHelpers(t *testing.T) {
	// Skip if git is not installed
	if !git.IsGitInstalled() {
		t.Skip("Git is not installed")
	}

	// Create temp directory
	tempMgr, err := fs.NewTempDirManager("grove-git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer tempMgr.Cleanup()

	repoDir, err := tempMgr.CreateDir("repo")
	if err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Create test repository
	files := map[string]string{
		"README.md":   "# Test Repo",
		"main.go":     "package main\n",
		".gitignore":  "*.tmp\n",
	}

	repo, err := git.CreateTestRepo(repoDir, files)
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	// Check current branch
	branch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("Expected main or master branch, got %s", branch)
	}

	// Check status is clean
	hasChanges, err := repo.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to check changes: %v", err)
	}

	if hasChanges {
		t.Error("Should have no uncommitted changes")
	}
}

func ExampleScenario() {
	// Example of how to define a scenario
	scenario := &harness.Scenario{
		Name:        "example-test",
		Description: "An example test scenario",
		Tags:        []string{"example", "demo"},
		Steps: []harness.Step{
			{
				Name: "Setup test environment",
				Func: func(ctx *harness.Context) error {
					// Create test directory
					testDir := ctx.NewDir("test")
					fmt.Printf("Created test directory: %s\n", testDir)
					return nil
				},
			},
			{
				Name: "Create grove.yml",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.Dir("test")
					return fs.WriteBasicGroveConfig(testDir)
				},
			},
		},
	}

	// This would be executed by the harness
	h := harness.New(harness.Options{
		Verbose: true,
	})

	// In real usage:
	// result, err := h.Run(context.Background(), scenario)
	_ = h
	_ = scenario
}