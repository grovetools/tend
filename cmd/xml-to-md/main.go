package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

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
	xmlContent, err := os.ReadFile("pkg/docs/tend-examples.xml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading XML file: %v\n", err)
		os.Exit(1)
	}
	
	var guide TendGuide
	if err := xml.Unmarshal(xmlContent, &guide); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
		os.Exit(1)
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

	if err := os.WriteFile("docs/TEND_GUIDE.md", []byte(md.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Markdown file: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Successfully created docs/TEND_GUIDE.md")
}