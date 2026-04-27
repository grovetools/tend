package command

import (
	"fmt"
	"strings"

	"github.com/grovetools/tend/pkg/assert"
)

// AssertStdoutContains asserts that stdout contains all the specified patterns.
// Returns the first assertion error encountered, or nil if all patterns are found.
func (r *Result) AssertStdoutContains(patterns ...string) error {
	for _, pattern := range patterns {
		if err := assert.Contains(r.Stdout, pattern,
			fmt.Sprintf("stdout missing pattern: %s", pattern)); err != nil {
			return err
		}
	}
	return nil
}

// AssertStdoutNotContains asserts that stdout does not contain any of the specified patterns.
func (r *Result) AssertStdoutNotContains(patterns ...string) error {
	for _, pattern := range patterns {
		if err := assert.NotContains(r.Stdout, pattern,
			fmt.Sprintf("stdout should not contain pattern: %s", pattern)); err != nil {
			return err
		}
	}
	return nil
}

// AssertStderrContains asserts that stderr contains all the specified patterns.
func (r *Result) AssertStderrContains(patterns ...string) error {
	for _, pattern := range patterns {
		if err := assert.Contains(r.Stderr, pattern,
			fmt.Sprintf("stderr missing pattern: %s", pattern)); err != nil {
			return err
		}
	}
	return nil
}

// AssertStderrNotContains asserts that stderr does not contain any of the specified patterns.
func (r *Result) AssertStderrNotContains(patterns ...string) error {
	for _, pattern := range patterns {
		if err := assert.NotContains(r.Stderr, pattern,
			fmt.Sprintf("stderr should not contain pattern: %s", pattern)); err != nil {
			return err
		}
	}
	return nil
}

// AssertSuccess asserts that the command succeeded (exit code 0, no error).
func (r *Result) AssertSuccess() error {
	if r.Error != nil {
		return fmt.Errorf("command failed: %w\nStderr: %s", r.Error, r.Stderr)
	}
	if r.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d\nStderr: %s", r.ExitCode, r.Stderr)
	}
	return nil
}

// AssertFailure asserts that the command failed (non-zero exit code or error).
func (r *Result) AssertFailure() error {
	if r.Error == nil && r.ExitCode == 0 {
		return fmt.Errorf("expected command to fail, but it succeeded")
	}
	return nil
}

// AssertExitCode asserts that the command exited with the specified code.
func (r *Result) AssertExitCode(expectedCode int) error {
	return assert.Equal(expectedCode, r.ExitCode,
		"command exit code mismatch")
}

// AssertStdoutEmpty asserts that stdout is empty.
func (r *Result) AssertStdoutEmpty() error {
	trimmed := strings.TrimSpace(r.Stdout)
	if trimmed != "" {
		return fmt.Errorf("expected empty stdout, got: %s", trimmed)
	}
	return nil
}

// AssertStderrEmpty asserts that stderr is empty.
func (r *Result) AssertStderrEmpty() error {
	trimmed := strings.TrimSpace(r.Stderr)
	if trimmed != "" {
		return fmt.Errorf("expected empty stderr, got: %s", trimmed)
	}
	return nil
}
