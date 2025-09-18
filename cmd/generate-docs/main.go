package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattsolo1/grove-core/config"
	"gopkg.in/yaml.v3"
)

// --- New Configuration Structs ---

// DocsConfig mirrors the structure of docs.config.yml
type DocsConfig struct {
	Settings SettingsConfig  `yaml:"settings"`
	Sections []SectionConfig `yaml:"sections"`
}

type SettingsConfig struct {
	RegenerationMode     string `yaml:"regeneration_mode"`
	OutputMarkdownCombined string `yaml:"output_markdown_combined"`
	StructuredOutputFile   string `yaml:"structured_output_file"`
}

type SectionConfig struct {
	Name       string `yaml:"name"`
	Prompt     string `yaml:"prompt"`
	JSONKey    string `yaml:"json_key"`
	OutputFile string `yaml:"output_file"`
}

// FlowConfig defines the structure for the 'flow' section in grove.yml.
type FlowConfig struct {
	OneshotModel string `yaml:"oneshot_model"`
}

// --- Updated Data Structures for JSON Output ---

// TendGuide is the top-level structure for the generated JSON document.
type TendGuide struct {
	Introduction  string     `json:"introduction"`
	CoreConcepts  []Concept  `json:"core_concepts"`
	UsagePatterns []Pattern  `json:"usage_patterns"`
	BestPractices []Practice `json:"best_practices"`
}

type Concept struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

type Pattern struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

type Practice struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("1. Loading docs generation config from docs/generation/docs.config.yml...")
	cfg, err := loadDocsConfig()
	if err != nil {
		return fmt.Errorf("failed to load docs.config.yml: %w", err)
	}

	fmt.Println("2. Building context from docs/generation/examples.cx.rules...")
	if err := buildContext(); err != nil {
		return fmt.Errorf("failed to build context: %w", err)
	}

	fmt.Println("3. Determining LLM model...")
	model, err := getLLMModel()
	if err != nil {
		return fmt.Errorf("failed to get LLM model: %w", err)
	}
	fmt.Printf("   Using model: %s\n", model)

	// --- New Multi-Request Generation Loop ---
	fmt.Println("4. Generating structured JSON content section by section...")
	sectionContents := make(map[string]json.RawMessage)
	for _, section := range cfg.Sections {
		fmt.Printf("   -> Generating section: %s\n", section.Name)
		promptPath := filepath.Join("docs", "generation", section.Prompt)
		jsonSnippet, err := generateSectionContent(model, promptPath, cfg.Settings)
		if err != nil {
			return fmt.Errorf("failed to generate content for section '%s': %w", section.Name, err)
		}
		sectionContents[section.JSONKey] = json.RawMessage(jsonSnippet)
	}

	fmt.Println("5. Assembling and writing final JSON output...")
	finalJSON, err := assembleFinalJSON(sectionContents)
	if err != nil {
		return fmt.Errorf("failed to assemble final JSON: %w", err)
	}
	if err := os.WriteFile(cfg.Settings.StructuredOutputFile, finalJSON, 0644); err != nil {
		return fmt.Errorf("failed to write structured JSON output: %w", err)
	}
	fmt.Printf("   Successfully wrote %s\n", cfg.Settings.StructuredOutputFile)

	fmt.Println("6. Transforming JSON to Markdown outputs...")
	if err := generateMarkdownOutputs(finalJSON, cfg); err != nil {
		return fmt.Errorf("failed to generate markdown outputs: %w", err)
	}

	fmt.Println("\nDocumentation generation complete.")
	return nil
}

func loadDocsConfig() (*DocsConfig, error) {
	configPath := "docs/generation/docs.config.yml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config DocsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func buildContext() error {
	// First, ensure .grove directory exists
	if err := os.MkdirAll(".grove", 0755); err != nil {
		return fmt.Errorf("failed to create .grove directory: %w", err)
	}
	
	// Copy rules file to .grove/rules
	rulesContent, err := os.ReadFile("docs/generation/examples.cx.rules")
	if err != nil {
		return fmt.Errorf("failed to read docs/generation/examples.cx.rules: %w", err)
	}
	if err := os.WriteFile(".grove/rules", rulesContent, 0644); err != nil {
		return fmt.Errorf("failed to write .grove/rules: %w", err)
	}
	
	// Generate context
	cmd := exec.Command("cx", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getLLMModel() (string, error) {
	// First, try to load grove.yml from current directory
	coreCfg, err := config.LoadFrom(".")
	if err != nil {
		// If no grove.yml, return with a default
		return "gemini-2.0-flash", nil
	}
	
	// Try to get model from flow.oneshot_model config
	var flowCfg FlowConfig
	if err := coreCfg.UnmarshalExtension("flow", &flowCfg); err == nil && flowCfg.OneshotModel != "" {
		return flowCfg.OneshotModel, nil
	}
	
	// Default fallback
	return "gemini-2.0-flash", nil
}

// generateSectionContent replaces the old generateXML function
func generateSectionContent(model, promptPath string, settings SettingsConfig) ([]byte, error) {
	promptContent, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
	}

	// Handle regeneration_mode
	finalPrompt := string(promptContent)
	if settings.RegenerationMode == "reference" && settings.OutputMarkdownCombined != "" {
		if existingDocs, err := os.ReadFile(settings.OutputMarkdownCombined); err == nil {
			finalPrompt = "For your reference, here is the previous version of the documentation:\n\n<reference_docs>\n" +
				string(existingDocs) + "\n</reference_docs>\n\n---\n\n" + finalPrompt
		}
	}

	// Create a temporary file for the prompt
	promptFile, err := os.CreateTemp("", "tend-docs-prompt-*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	defer os.Remove(promptFile.Name())

	if _, err := promptFile.WriteString(finalPrompt); err != nil {
		return nil, fmt.Errorf("failed to write to temp prompt file: %w", err)
	}
	if err := promptFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp prompt file: %w", err)
	}

	// Use gemapi to make the request
	args := []string{
		"request",
		"--model", model,
		"--file", promptFile.Name(),
		"--yes",
	}
	cmd := exec.Command("gemapi", args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gemapi request failed: %w", err)
	}
	
	// Clean up the JSON output
	jsonContent := string(output)
	jsonContent = strings.TrimSpace(jsonContent)
	if strings.HasPrefix(jsonContent, "```json") {
		jsonContent = strings.TrimPrefix(jsonContent, "```json")
		jsonContent = strings.TrimSuffix(jsonContent, "```")
		jsonContent = strings.TrimSpace(jsonContent)
	}
	
	// The LLM should return a JSON object like {"introduction": ...} or {"core_concepts": ...}
	// We want to extract just the value part
	var temp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonContent), &temp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON output: %w. Raw output:\n%s", err, jsonContent)
	}
	
	// Return the first (and should be only) value
	for _, v := range temp {
		return v, nil
	}
	
	return nil, fmt.Errorf("LLM output was valid JSON but empty")
}

func assembleFinalJSON(sections map[string]json.RawMessage) ([]byte, error) {
	// Create a new map to hold the final structure
	finalMap := make(map[string]json.RawMessage)
	for key, value := range sections {
		finalMap[key] = value
	}

	// Marshal with indentation for readability
	return json.MarshalIndent(finalMap, "", "  ")
}

// This is the new markdown generation logic, replacing transformXMLToMarkdown
func generateMarkdownOutputs(jsonContent []byte, cfg *DocsConfig) error {
	var guide TendGuide
	if err := json.Unmarshal(jsonContent, &guide); err != nil {
		return fmt.Errorf("failed to parse final JSON for markdown generation: %w", err)
	}

	// Generate combined guide if configured
	if cfg.Settings.OutputMarkdownCombined != "" {
		fmt.Printf("   -> Generating combined guide: %s\n", cfg.Settings.OutputMarkdownCombined)
		var md strings.Builder
		md.WriteString("# Grove Tend Testing Library - Comprehensive Guide\n\n")
		
		// Render all sections
		if guide.Introduction != "" {
			md.WriteString(renderIntroductionMarkdown(&guide))
		}
		if len(guide.CoreConcepts) > 0 {
			md.WriteString(renderCoreConceptsMarkdown(&guide))
		}
		if len(guide.UsagePatterns) > 0 {
			md.WriteString(renderUsagePatternsMarkdown(&guide))
		}
		if len(guide.BestPractices) > 0 {
			md.WriteString(renderBestPracticesMarkdown(&guide))
		}

		if err := os.WriteFile(cfg.Settings.OutputMarkdownCombined, []byte(md.String()), 0644); err != nil {
			return fmt.Errorf("failed to write combined markdown file: %w", err)
		}
	}

	// Generate section-specific files
	for _, section := range cfg.Sections {
		if section.OutputFile != "" {
			fmt.Printf("   -> Generating section file: %s\n", section.OutputFile)
			var content string
			switch section.JSONKey {
			case "introduction":
				content = renderIntroductionMarkdown(&guide)
			case "core_concepts":
				content = renderCoreConceptsMarkdown(&guide)
			case "usage_patterns":
				content = renderUsagePatternsMarkdown(&guide)
			case "best_practices":
				content = renderBestPracticesMarkdown(&guide)
			}
			
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(section.OutputFile), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", section.OutputFile, err)
			}
			if err := os.WriteFile(section.OutputFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write section file %s: %w", section.OutputFile, err)
			}
		}
	}

	return nil
}

// --- New Markdown Rendering Helper Functions ---

func renderIntroductionMarkdown(guide *TendGuide) string {
	if guide.Introduction == "" {
		return ""
	}
	// The introduction already contains markdown formatting
	return strings.TrimSpace(guide.Introduction) + "\n\n"
}

func renderCoreConceptsMarkdown(guide *TendGuide) string {
	if len(guide.CoreConcepts) == 0 {
		return ""
	}
	
	var md strings.Builder
	md.WriteString("## Core Concepts\n\n")
	for _, c := range guide.CoreConcepts {
		md.WriteString(fmt.Sprintf("### %s\n\n", c.Name))
		md.WriteString(strings.TrimSpace(c.Description) + "\n\n")
		if c.Example != "" {
			md.WriteString("```go\n")
			md.WriteString(strings.TrimSpace(c.Example) + "\n")
			md.WriteString("```\n\n")
		}
	}
	return md.String()
}

func renderUsagePatternsMarkdown(guide *TendGuide) string {
	if len(guide.UsagePatterns) == 0 {
		return ""
	}
	
	var md strings.Builder
	md.WriteString("## Usage Patterns\n\n")
	for _, p := range guide.UsagePatterns {
		md.WriteString(fmt.Sprintf("### %s\n\n", p.Name))
		md.WriteString(strings.TrimSpace(p.Description) + "\n\n")
		if p.Example != "" {
			// Check if example is shell commands (starts with # or ./)
			if strings.Contains(p.Example, "#") || strings.HasPrefix(strings.TrimSpace(p.Example), "./") {
				md.WriteString("```bash\n")
			} else {
				md.WriteString("```go\n")
			}
			md.WriteString(strings.TrimSpace(p.Example) + "\n")
			md.WriteString("```\n\n")
		}
	}
	return md.String()
}

func renderBestPracticesMarkdown(guide *TendGuide) string {
	if len(guide.BestPractices) == 0 {
		return ""
	}
	
	var md strings.Builder
	md.WriteString("## Best Practices\n\n")
	for _, p := range guide.BestPractices {
		md.WriteString(fmt.Sprintf("### %s\n\n", p.Title))
		md.WriteString(strings.TrimSpace(p.Text) + "\n\n")
	}
	return md.String()
}