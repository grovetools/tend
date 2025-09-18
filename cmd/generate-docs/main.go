package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mattsolo1/grove-core/config"
)

// loadPrompt reads the prompt from the file or returns a default
func loadPrompt() (string, error) {
	promptPath := "docs/generation/tend-docs.prompt.md"
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
	}
	return string(content), nil
}

// FlowConfig defines the structure for the 'flow' section in grove.yml.
type FlowConfig struct {
	OneshotModel string `yaml:"oneshot_model"`
}

// Data structures for parsing tend-examples.xml
type TendGuide struct {
	XMLName        xml.Name       `xml:"tend_guide"`
	Introduction   string         `xml:"introduction"`
	CoreConcepts   []Concept      `xml:"core_concepts>concept"`
	UsagePatterns  []Pattern      `xml:"usage_patterns>pattern"`
	BestPractices  []Practice     `xml:"best_practices>practice"`
}
type Concept struct {
	Name        string `xml:"name,attr"`
	Description string `xml:"description"`
	Example     string `xml:"example"`
}
type Pattern struct {
	Name        string `xml:"name,attr"`
	Description string `xml:"description"`
	Example     string `xml:"example"`
}
type Practice struct {
	Title string `xml:"title,attr"`
	Text  string `xml:",chardata"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("1. Building context from docs/examples.cx.rules...")
	if err := buildContext(); err != nil {
		return fmt.Errorf("failed to build context: %w", err)
	}

	fmt.Println("2. Determining LLM model...")
	model, err := getLLMModel()
	if err != nil {
		return fmt.Errorf("failed to get LLM model: %w", err)
	}
	fmt.Printf("   Using model: %s\n", model)

	fmt.Println("3. Generating tend-examples.xml with LLM...")
	xmlContent, err := generateXML(model)
	if err != nil {
		return fmt.Errorf("failed to generate XML: %w", err)
	}

	xmlOutputPath := "pkg/docs/tend-examples.xml"
	if err := os.WriteFile(xmlOutputPath, []byte(xmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write XML output: %w", err)
	}
	fmt.Printf("   Successfully wrote %s\n", xmlOutputPath)

	fmt.Println("4. Transforming XML to Markdown...")
	if err := transformXMLToMarkdown(xmlContent); err != nil {
		return fmt.Errorf("failed to transform XML to Markdown: %w", err)
	}
	fmt.Printf("   Successfully wrote docs/TEND_GUIDE.md\n")

	fmt.Println("\nDocumentation generation complete.")
	return nil
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

func generateXML(model string) (string, error) {
	// Load the prompt from file
	promptContent, err := loadPrompt()
	if err != nil {
		return "", fmt.Errorf("failed to load prompt: %w", err)
	}
	
	promptFile, err := os.CreateTemp("", "tend-docs-prompt-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	defer os.Remove(promptFile.Name())

	if _, err := promptFile.WriteString(promptContent); err != nil {
		return "", fmt.Errorf("failed to write to temp prompt file: %w", err)
	}
	if err := promptFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp prompt file: %w", err)
	}

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
		return "", fmt.Errorf("gemapi request failed: %w", err)
	}

	xmlContent := string(output)
	
	// Clean up the XML content - remove any markdown code fences
	xmlContent = strings.TrimSpace(xmlContent)
	if strings.HasPrefix(xmlContent, "```xml") {
		xmlContent = strings.TrimPrefix(xmlContent, "```xml")
		xmlContent = strings.TrimSuffix(xmlContent, "```")
		xmlContent = strings.TrimSpace(xmlContent)
	}
	if strings.HasPrefix(xmlContent, "```") {
		xmlContent = strings.TrimPrefix(xmlContent, "```")
		xmlContent = strings.TrimSuffix(xmlContent, "```")
		xmlContent = strings.TrimSpace(xmlContent)
	}
	
	// Fix common XML escaping issues
	// Replace unescaped <tool> with &lt;tool&gt;
	xmlContent = strings.ReplaceAll(xmlContent, "`grove dev current <tool>`", "`grove dev current &lt;tool&gt;`")
	
	// Ensure XML header if missing
	if !strings.HasPrefix(xmlContent, "<?xml") {
		xmlContent = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + xmlContent
	}
	
	return xmlContent, nil
}

func transformXMLToMarkdown(xmlContent string) error {
	var guide TendGuide
	if err := xml.Unmarshal([]byte(xmlContent), &guide); err != nil {
		return fmt.Errorf("failed to parse generated XML: %w", err)
	}

	var md strings.Builder
	md.WriteString("# Grove Tend Testing Library - Comprehensive Guide\n\n")
	md.WriteString(strings.TrimSpace(guide.Introduction) + "\n\n")

	md.WriteString("## Core Concepts\n\n")
	for _, c := range guide.CoreConcepts {
		md.WriteString(fmt.Sprintf("### %s\n\n", c.Name))
		md.WriteString(strings.TrimSpace(c.Description) + "\n\n")
		md.WriteString("```go\n")
		md.WriteString(strings.TrimSpace(c.Example) + "\n")
		md.WriteString("```\n\n")
	}

	md.WriteString("## Usage Patterns\n\n")
	for _, p := range guide.UsagePatterns {
		md.WriteString(fmt.Sprintf("### %s\n\n", p.Name))
		md.WriteString(strings.TrimSpace(p.Description) + "\n\n")
		// Check if example is shell commands (starts with # or ./)
		if strings.Contains(p.Example, "#") || strings.HasPrefix(strings.TrimSpace(p.Example), "./") {
			md.WriteString("```bash\n")
		} else {
			md.WriteString("```go\n")
		}
		md.WriteString(strings.TrimSpace(p.Example) + "\n")
		md.WriteString("```\n\n")
	}

	md.WriteString("## Best Practices\n\n")
	for _, p := range guide.BestPractices {
		md.WriteString(fmt.Sprintf("### %s\n\n", p.Title))
		md.WriteString(strings.TrimSpace(p.Text) + "\n\n")
	}

	mdOutputPath := "docs/TEND_GUIDE.md"
	return os.WriteFile(mdOutputPath, []byte(md.String()), 0644)
}