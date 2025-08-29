package harness

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattsolo1/grove-tend/pkg/command"
)

var pathRegex = regexp.MustCompile(`\((.*?)\)`)

// FindRealBinary uses `grove dev current` to find the path of a real ecosystem tool.
func FindRealBinary(toolName string) (string, error) {
	// Assumes 'grove' is in the PATH.
	cmd := command.New("grove", "dev", "current", toolName)
	result := cmd.Run()

	if result.Error != nil {
		return "", fmt.Errorf("failed to run 'grove dev current %s': %w", toolName, result.Error)
	}

	// Output is like: "  flow: main-6b984f8-dirty (/path/to/bin/flow) [dev]"
	// We need to parse out the path in parentheses.
	matches := pathRegex.FindStringSubmatch(result.Stdout)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse path from 'grove dev current' output for '%s': %s", toolName, result.Stdout)
	}

	return strings.TrimSpace(matches[1]), nil
}