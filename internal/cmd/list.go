package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/core/tui/components/table"
	"github.com/grovetools/core/tui/theme"
	"github.com/spf13/cobra"

	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/ui"
)

var ulogList = grovelogging.NewUnifiedLogger("grove-tend.cmd.list")

// newListCmd creates the list command with the provided scenarios
func newListCmd(allScenarios []*harness.Scenario) *cobra.Command {
	var keyword string
	
	listCmd := &cobra.Command{
	Use:   "list",
	Short: "List available test scenarios",
	Long: `List all available test scenarios with their descriptions and tags.

This command helps you discover what scenarios are available and understand
their purpose before running them.

Examples:
  tend list                         # List all scenarios
  tend list --tags=smoke            # List scenarios tagged with 'smoke'
  tend list --keyword=git           # List scenarios containing 'git' in name, description, or tags
  tend list --verbose               # List with detailed information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listScenarios(cmd, args, allScenarios, keyword)
	},
	}
	
	listCmd.Flags().StringVarP(&keyword, "keyword", "k", "", "Filter scenarios by keyword (searches name, description, and tags)")
	
	return listCmd
}

func listScenarios(cmd *cobra.Command, args []string, allScenarios []*harness.Scenario, keyword string) error {
	// Create UI renderer
	renderer := ui.NewRenderer(cmd.OutOrStdout(), verbose, 80)
	
	
	// Filter scenarios by tags and keyword if specified
	filteredScenarios := filterScenarios(allScenarios, []string{}, tags)
	
	// Apply keyword filtering if specified
	if keyword != "" {
		filteredScenarios = filterByKeyword(filteredScenarios, keyword)
	}
	
	if len(filteredScenarios) == 0 {
		renderer.RenderInfo("No scenarios found matching the specified criteria")
		return nil
	}
	
	// Display header
	ulogList.Info("Listing scenarios").
		Field("count", len(filteredScenarios)).
		Pretty(fmt.Sprintf("Available scenarios (%d):\n", len(filteredScenarios))).
		Emit()
	
	// Build table data
	headers := []string{"NAME", "DESCRIPTION", "LOCAL", "EXPLICIT", "TAGS", "STEPS"}
	var rows [][]string
	
	for _, scenario := range filteredScenarios {
		// Format tags
		tagStr := "-"
		if len(scenario.Tags) > 0 {
			tagStr = strings.Join(scenario.Tags, ", ")
		}
		
		// Format description - truncate if too long
		description := scenario.Description
		if len(description) > 50 && !verbose {
			description = description[:47] + "..."
		}
		
		// Format local and explicit indicators
		localIndicator := ""
		if scenario.LocalOnly {
			localIndicator = theme.IconSuccess
		}

		explicitIndicator := ""
		if scenario.ExplicitOnly {
			explicitIndicator = theme.IconSuccess
		}
		
		row := []string{
			scenario.Name,
			description,
			localIndicator,
			explicitIndicator,
			tagStr,
			fmt.Sprintf("%d", len(scenario.Steps)),
		}
		rows = append(rows, row)
	}
	
	// Create table renderer
	t := table.NewStyledTable().
		Headers(headers...).
		Rows(rows...)

	// Apply column-specific styling
	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == 0 {
			// Header style is already applied by NewStyledTable
			return lipgloss.NewStyle()
		}
		// Style based on column
		switch col {
		case 0: // Name column
			return theme.DefaultTheme.Title.Bold(true)
		case 2, 3: // Local and Explicit columns (centered)
			return lipgloss.NewStyle().Padding(0, 1).Align(lipgloss.Center)
		case 4: // Tags column
			return theme.DefaultTheme.Info.Padding(0, 1)
		case 5: // Steps column
			return theme.DefaultTheme.Success.Padding(0, 1)
		default:
			return lipgloss.NewStyle().Padding(0, 1)
		}
	})
	
	ulogList.Info("Scenario list table").
		Pretty(t.String()).
		PrettyOnly().
		Emit()

	// If verbose, show detailed step information for each scenario
	if verbose {
		ulogList.Info("Detailed scenario information").
			Pretty("\nDetailed scenario information:").
			PrettyOnly().
			Emit()
		for _, scenario := range filteredScenarios {
			displayScenarioDetails(renderer, scenario)
		}
	}
	
	return nil
}

func displayScenarioDetails(renderer *ui.Renderer, scenario *harness.Scenario) {

	// Scenario name and description
	prettyMsg := fmt.Sprintf("\n%s %s",
		theme.DefaultTheme.Header.Render("●"),
		theme.DefaultTheme.Title.Render(scenario.Name))

	if scenario.Description != "" {
		prettyMsg += fmt.Sprintf("\n  %s", theme.DefaultTheme.Muted.Render(scenario.Description))
	}

	// Show LocalOnly warning
	if scenario.LocalOnly {
		prettyMsg += fmt.Sprintf("\n  %s This scenario is marked as local-only and will be skipped in CI environments",
			theme.DefaultTheme.Warning.Render(theme.IconWarning))
	}

	// Show ExplicitOnly warning
	if scenario.ExplicitOnly {
		prettyMsg += fmt.Sprintf("\n  %s This scenario must be run explicitly by name (skipped during 'tend run')",
			theme.DefaultTheme.Warning.Render(theme.IconWarning))
	}

	// Tags
	if len(scenario.Tags) > 0 {
		tagStr := strings.Join(scenario.Tags, ", ")
		prettyMsg += fmt.Sprintf("\n  %s %s",
			theme.DefaultTheme.Info.Render("Tags:"),
			theme.DefaultTheme.Muted.Render(tagStr))
	}

	// Step count
	prettyMsg += fmt.Sprintf("\n  %s %d step(s)",
		theme.DefaultTheme.Info.Render("Steps:"),
		len(scenario.Steps))

	ulogList.Info("Scenario details").
		Field("name", scenario.Name).
		Field("steps_count", len(scenario.Steps)).
		Pretty(prettyMsg).
		Emit()

	// If verbose, show step details
	if verbose {
		stepNumberStyle := lipgloss.NewStyle().
			Foreground(theme.DefaultTheme.Colors.Orange).
			Bold(true).
			Width(3).
			Align(lipgloss.Right)
		stepNameStyle := lipgloss.NewStyle().
			Foreground(theme.DefaultTheme.Colors.LightText).
			MarginLeft(1)
		for i, step := range scenario.Steps {
			fmt.Printf("    %s %s\n",
				stepNumberStyle.Render(fmt.Sprintf("%d.", i+1)),
				stepNameStyle.Render(step.Name))

			if step.Description != "" {
				fmt.Printf("      %s\n", theme.DefaultTheme.Muted.Render(step.Description))
			}
		}
	}
}

// filterByKeyword filters scenarios based on a keyword search in name, description, and tags
func filterByKeyword(scenarios []*harness.Scenario, keyword string) []*harness.Scenario {
	var filtered []*harness.Scenario
	lowercaseKeyword := strings.ToLower(keyword)
	
	for _, scenario := range scenarios {
		// Check if keyword appears in scenario name
		if strings.Contains(strings.ToLower(scenario.Name), lowercaseKeyword) {
			filtered = append(filtered, scenario)
			continue
		}
		
		// Check if keyword appears in description
		if strings.Contains(strings.ToLower(scenario.Description), lowercaseKeyword) {
			filtered = append(filtered, scenario)
			continue
		}
		
		// Check if keyword appears in any tag
		for _, tag := range scenario.Tags {
			if strings.Contains(strings.ToLower(tag), lowercaseKeyword) {
				filtered = append(filtered, scenario)
				break
			}
		}
	}
	
	return filtered
}