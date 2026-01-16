package fs

import (
	"fmt"

	"github.com/grovetools/tend/pkg/assert"
)

// AssertExists asserts that a file or directory exists at the given path.
func AssertExists(path string) error {
	if !Exists(path) {
		return fmt.Errorf("expected file or directory to exist: %s", path)
	}
	return nil
}

// AssertNotExists asserts that a file or directory does not exist at the given path.
func AssertNotExists(path string) error {
	if Exists(path) {
		return fmt.Errorf("expected file or directory to not exist: %s", path)
	}
	return nil
}

// AssertContains asserts that a file contains the specified substring.
func AssertContains(path, substr string) error {
	content, err := ReadString(path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", path, err)
	}
	return assert.Contains(content, substr, fmt.Sprintf("file %s does not contain expected content", path))
}

// AssertNotContains asserts that a file does not contain the specified substring.
func AssertNotContains(path, substr string) error {
	content, err := ReadString(path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", path, err)
	}
	return assert.NotContains(content, substr, fmt.Sprintf("file %s should not contain content", path))
}

// AssertEmpty asserts that a file is empty or contains only whitespace.
func AssertEmpty(path string) error {
	content, err := ReadString(path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", path, err)
	}
	return assert.Empty(content, fmt.Sprintf("file %s should be empty", path))
}

// AssertNotEmpty asserts that a file is not empty.
func AssertNotEmpty(path string) error {
	content, err := ReadString(path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", path, err)
	}
	if len(content) == 0 {
		return fmt.Errorf("expected file %s to not be empty", path)
	}
	return nil
}
