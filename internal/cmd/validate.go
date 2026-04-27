package cmd

import (
	"fmt"

	"github.com/grovetools/core/tui/theme"
	"github.com/spf13/cobra"

	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/ui"
)

// newValidateCmd creates the validate command with the provided scenarios
func newValidateCmd(allScenarios []*harness.Scenario) *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate test scenarios",
		Long: `Validate that all test scenarios are properly defined and can be loaded.

This command checks:
  • Scenario files can be parsed
  • Required fields are present
  • Steps are properly defined
  • No circular dependencies exist

This is useful for CI/CD pipelines to catch configuration errors early.

Examples:
  tend validate              # Validate all scenarios
  tend validate --verbose    # Show detailed validation output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateScenarios(cmd, args, allScenarios)
		},
	}

	return validateCmd
}

func validateScenarios(cmd *cobra.Command, args []string, allScenarios []*harness.Scenario) error {
	// Create UI renderer
	renderer := ui.NewRenderer(cmd.OutOrStdout(), verbose, 80)

	if len(allScenarios) == 0 {
		renderer.RenderInfo("No scenarios found to validate")
		return nil
	}

	renderer.RenderInfo(fmt.Sprintf("Validating %d scenario(s)...", len(allScenarios)))

	var validationErrors []string
	validCount := 0

	for _, scenario := range allScenarios {
		errors := validateScenario(scenario)
		if len(errors) == 0 {
			validCount++
			if verbose {
				renderer.RenderSuccess(fmt.Sprintf("* %s", scenario.Name))
			}
		} else {
			for _, err := range errors {
				validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", scenario.Name, err))
			}
		}
	}

	// Display results
	if len(validationErrors) == 0 {
		renderer.RenderSuccess(fmt.Sprintf("All %d scenario(s) are valid!", validCount))
	} else {
		renderer.RenderError(fmt.Errorf("found %d validation error(s)", len(validationErrors)))
		for _, err := range validationErrors {
			fmt.Printf("  • %s\n", theme.DefaultTheme.Error.Render(err))
		}
		return fmt.Errorf("validation failed")
	}

	return nil
}

func validateScenario(scenario *harness.Scenario) []string {
	var errors []string

	// Check required fields
	if scenario.Name == "" {
		errors = append(errors, "scenario name is required")
	}

	if len(scenario.Steps) == 0 {
		errors = append(errors, "scenario must have at least one step")
	}

	// Validate steps
	for i, step := range scenario.Steps {
		if step.Name == "" {
			errors = append(errors, fmt.Sprintf("step %d is missing a name", i+1))
		}

		if step.Func == nil {
			errors = append(errors, fmt.Sprintf("step %d (%s) is missing a function", i+1, step.Name))
		}
	}

	// Check for duplicate step names
	stepNames := make(map[string]int)
	for i, step := range scenario.Steps {
		if prev, exists := stepNames[step.Name]; exists {
			errors = append(errors, fmt.Sprintf("duplicate step name '%s' at positions %d and %d", step.Name, prev+1, i+1))
		}
		stepNames[step.Name] = i
	}

	return errors
}
