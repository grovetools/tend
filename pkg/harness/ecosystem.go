package harness

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mattsolo1/grove-tend/pkg/command"
)

var pathRegex = regexp.MustCompile(`\((.*?)\)`)

// FindRealBinary finds the path of a real ecosystem tool. It prioritizes an
// environment variable override before falling back to `grove dev current`.
func FindRealBinary(toolName string) (string, error) {
	// 1. Check for environment variable override (e.g., TEND_REAL_DEP_FLOW)
	envVarKey := fmt.Sprintf("TEND_REAL_DEP_%s", strings.ToUpper(toolName))
	if realBinaryPath := os.Getenv(envVarKey); realBinaryPath != "" {
		if _, err := os.Stat(realBinaryPath); err == nil {
			// Use fmt.Fprintf to stderr to provide debug info without polluting stdout
			fmt.Fprintf(os.Stderr, "INFO: Using real binary for '%s' from env var %s: %s\n", toolName, envVarKey, realBinaryPath)
			return realBinaryPath, nil
		}
		fmt.Fprintf(os.Stderr, "WARN: Env var %s is set to '%s', but file does not exist. Falling back.\n", envVarKey, realBinaryPath)
	}

	// 2. Fallback to using `grove dev current`
	// Assumes 'grove' is in the PATH.
	cmd := command.New("grove", "dev", "current", toolName)
	result := cmd.Run()

	if result.Error != nil {
		return "", fmt.Errorf("failed to run 'grove dev current %s': %w. You can set the %s env var as an override", toolName, result.Error, envVarKey)
	}

	// Output is like: "  flow: main-6b984f8-dirty (/path/to/bin/flow) [dev]"
	// We need to parse out the path in parentheses.
	matches := pathRegex.FindStringSubmatch(result.Stdout)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse path from 'grove dev current' output for '%s': %s. You can set the %s env var as an override", toolName, result.Stdout, envVarKey)
	}

	return strings.TrimSpace(matches[1]), nil
}