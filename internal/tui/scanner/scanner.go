package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// Scenario patterns for regex parsing
var (
	scenarioBlockRegex = regexp.MustCompile(`(?s)&harness\.Scenario\{(.*?)\}`)
	nameRegex          = regexp.MustCompile(`Name:\s*"(.*?)"`)
	descriptionRegex   = regexp.MustCompile(`Description:\s*"(.*?)"`)
	tagsRegex          = regexp.MustCompile(`Tags:\s*\[\]string\{(.*?)\}`)
)

// ScanProjectForScenarios walks a project directory and parses Go files to find scenario definitions.
// It returns a map where keys are file paths and values are slices of scenarios found in that file.
func ScanProjectForScenarios(projectPath string) (map[string][]*harness.Scenario, error) {
	scenariosByFile := make(map[string][]*harness.Scenario)

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
				content, err := os.ReadFile(path)
				if err != nil {
					return nil // Skip files we can't read
				}

				foundScenarios := parseScenariosFromFile(string(content))
				if len(foundScenarios) > 0 {
					scenariosByFile[path] = append(scenariosByFile[path], foundScenarios...)
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

	blocks := scenarioBlockRegex.FindAllStringSubmatch(content, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		scenarioContent := block[1]

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
		scenarios = append(scenarios, scenario)
	}

	return scenarios
}
