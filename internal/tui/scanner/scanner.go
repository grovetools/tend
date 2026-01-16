package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grovetools/tend/pkg/harness"
)

// Scenario patterns for regex parsing
var (
	// scenarioStartRegex finds the start of a struct literal scenario
	scenarioStartRegex = regexp.MustCompile(`&harness\.Scenario\{`)
	nameRegex          = regexp.MustCompile(`Name:\s*"(.*?)"`)
	descriptionRegex   = regexp.MustCompile(`Description:\s*"(.*?)"`)
	tagsRegex          = regexp.MustCompile(`Tags:\s*\[\]string\{(.*?)\}`)
	localOnlyRegex     = regexp.MustCompile(`LocalOnly:\s*true`)
	explicitOnlyRegex  = regexp.MustCompile(`ExplicitOnly:\s*true`)
)

// ScanProjectForScenarios walks a project directory and parses Go files to find scenario definitions.
// It returns a map where keys are file paths and values are slices of scenarios found in that file.
func ScanProjectForScenarios(projectPath string) (map[string][]*harness.Scenario, error) {
	scenariosByFile := make(map[string][]*harness.Scenario)
	visitedFiles := make(map[string]bool)

	// Walk through the project, looking for test files.
	// We'll look in common directories like 'tests/e2e' and the project root.
	searchDirs := []string{
		projectPath,
		filepath.Join(projectPath, "tests"),
		filepath.Join(projectPath, "tests", "e2e"),
		filepath.Join(projectPath, "tests", "e2e", "tend"),
	}

	for _, dir := range searchDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Only process Go files
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return nil // Skip if we can't get an absolute path
				}
				if visitedFiles[absPath] {
					return nil // Already processed, skip.
				}
				visitedFiles[absPath] = true

				content, err := os.ReadFile(absPath)
				if err != nil {
					return nil // Skip files we can't read
				}

				foundScenarios := parseScenariosFromFile(string(content))
				if len(foundScenarios) > 0 {
					scenariosByFile[absPath] = foundScenarios
				}
			}
			return nil
		})
		if err != nil {
			// Don't fail, just log a warning if we can't walk a directory
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", dir, err)
		}
	}

	return scenariosByFile, nil
}

// parseScenariosFromFile uses regex to extract scenario definitions from file content.
func parseScenariosFromFile(content string) []*harness.Scenario {
	var scenarios []*harness.Scenario

	// Find struct literal scenarios using balanced brace matching
	matches := scenarioStartRegex.FindAllStringIndex(content, -1)
	for _, match := range matches {
		// Find the opening brace position
		startBrace := match[1] - 1
		endBrace := findMatchingBrace(content, startBrace)
		if endBrace == -1 {
			continue
		}

		scenarioContent := content[startBrace+1 : endBrace]

		nameMatch := nameRegex.FindStringSubmatch(scenarioContent)
		if len(nameMatch) < 2 {
			continue // Name is required
		}

		scenario := &harness.Scenario{
			Name: nameMatch[1],
		}

		if descMatch := descriptionRegex.FindStringSubmatch(scenarioContent); len(descMatch) > 1 {
			scenario.Description = descMatch[1]
		}

		if tagsMatch := tagsRegex.FindStringSubmatch(scenarioContent); len(tagsMatch) > 1 {
			tagsStr := tagsMatch[1]
			tags := strings.Split(tagsStr, ",")
			for _, tag := range tags {
				tag = strings.Trim(tag, ` "`)
				if tag != "" {
					scenario.Tags = append(scenario.Tags, tag)
				}
			}
		}

		if localOnlyRegex.MatchString(scenarioContent) {
			scenario.LocalOnly = true
		}
		if explicitOnlyRegex.MatchString(scenarioContent) {
			scenario.ExplicitOnly = true
		}

		scenarios = append(scenarios, scenario)
	}

	// Also parse scenarios created with constructor functions
	scenarios = append(scenarios, parseFunctionScenarios(content)...)

	return scenarios
}

// parseFunctionScenarios finds scenarios defined with harness.NewScenario...()
func parseFunctionScenarios(content string) []*harness.Scenario {
	var scenarios []*harness.Scenario
	funcRegex := regexp.MustCompile(`harness\.NewScenario(WithOptions)?\(`)
	stringLiteralRegex := regexp.MustCompile(`"([^"]*)"`)
	boolRegex := regexp.MustCompile(`(true|false)`)

	matches := funcRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		// match[0] is start of full match, match[1] is end
		// For finding the opening paren, we need to find it within the matched text
		startParen := match[1] - 1 // The '(' is the last character of the match
		endParen := findMatchingParen(content, startParen)
		if endParen == -1 {
			continue
		}

		// Extract content within the function call parentheses
		argContent := content[startParen+1 : endParen]
		isWithOptions := match[2] != -1 // Check if "WithOptions" was captured

		// Parse arguments - find string literals for name and description
		stringMatches := stringLiteralRegex.FindAllStringSubmatch(argContent, -1)
		if len(stringMatches) < 2 {
			continue // Must have at least name and description
		}

		scenario := &harness.Scenario{
			Name:        stringMatches[0][1],
			Description: stringMatches[1][1],
		}

		if tagsMatch := tagsRegex.FindStringSubmatch(argContent); len(tagsMatch) > 1 {
			tagsStr := tagsMatch[1]
			tags := strings.Split(tagsStr, ",")
			for _, tag := range tags {
				tag = strings.Trim(tag, ` "`)
				if tag != "" {
					scenario.Tags = append(scenario.Tags, tag)
				}
			}
		}

		if isWithOptions {
			// Find the last two boolean values for LocalOnly and ExplicitOnly
			boolMatches := boolRegex.FindAllString(argContent, -1)
			if len(boolMatches) >= 2 {
				scenario.LocalOnly = boolMatches[len(boolMatches)-2] == "true"
				scenario.ExplicitOnly = boolMatches[len(boolMatches)-1] == "true"
			}
		}
		scenarios = append(scenarios, scenario)
	}

	return scenarios
}

// findMatchingParen finds the matching closing parenthesis for an opening one at start index.
func findMatchingParen(content string, start int) int {
	if start >= len(content) || content[start] != '(' {
		return -1
	}
	balance := 1
	for i := start + 1; i < len(content); i++ {
		switch content[i] {
		case '(':
			balance++
		case ')':
			balance--
			if balance == 0 {
				return i
			}
		}
	}
	return -1
}

// findMatchingBrace finds the matching closing brace for an opening one at start index.
func findMatchingBrace(content string, start int) int {
	if start >= len(content) || content[start] != '{' {
		return -1
	}
	balance := 1
	for i := start + 1; i < len(content); i++ {
		switch content[i] {
		case '{':
			balance++
		case '}':
			balance--
			if balance == 0 {
				return i
			}
		}
	}
	return -1
}
