package demo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/grovetools/tend/pkg/fs"
)

// demoNote represents a note to be created in the demo notebook.
type demoNote struct {
	filename string
	title    string
	noteType string
	tags     []string
}

// seedRichNotebook creates a rich notebook structure with many notes
// across different categories to showcase the TUI display.
func (g *homelabGenerator) seedRichNotebook() error {
	// Get the homelab workspace directory (notebooks/homelab/workspaces/homelab/)
	// This matches the structure expected by nb for workspace paths
	homelabNotebookRoot := filepath.Join(g.notebookDir(), "homelab")
	workspaceDir := filepath.Join(homelabNotebookRoot, "workspaces", "homelab")

	// Create all the category directories
	categories := []string{
		"inbox", "issues", "plans", "in_progress", "review",
		"learn", "concepts", "docgen", "icebox", "llm",
		"quick", "todos", "completed",
	}

	for _, cat := range categories {
		if err := fs.CreateDir(filepath.Join(workspaceDir, cat)); err != nil {
			return fmt.Errorf("creating %s directory: %w", cat, err)
		}
	}

	// Create plans subfolders
	planNames := []string{
		"api-v2", "auth-rework", "mobile-app", "caching",
		"migration", "testing", "dashboard", "rollback", "metrics",
	}
	for _, plan := range planNames {
		planDir := filepath.Join(workspaceDir, "plans", plan)
		if err := fs.CreateDir(planDir); err != nil {
			return fmt.Errorf("creating plan %s directory: %w", plan, err)
		}
		// Add plan files
		if err := g.createPlanFiles(planDir, plan); err != nil {
			return err
		}
	}

	// Create concepts subfolder
	conceptsArch := filepath.Join(workspaceDir, "concepts", "architecture")
	if err := fs.CreateDir(conceptsArch); err != nil {
		return fmt.Errorf("creating concepts/architecture directory: %w", err)
	}

	// Seed all the notes
	if err := g.seedInboxNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedIssuesNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedInProgressNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedReviewNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedLearnNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedConceptsNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedDocgenNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedIceboxNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedLLMNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedQuickNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedTodosNotes(workspaceDir); err != nil {
		return err
	}
	if err := g.seedCompletedNotes(workspaceDir); err != nil {
		return err
	}

	return nil
}

// createSimpleNote creates a simple demo note with frontmatter.
func (g *homelabGenerator) createSimpleNote(dir, filename, title, noteType string, tags []string) error {
	tagsStr := "[]"
	if len(tags) > 0 {
		tagsStr = "[" + strings.Join(tags, ", ") + "]"
	}

	content := fmt.Sprintf(`---
title: %s
type: %s
tags: %s
created: 2026-01-15T10:00:00Z
---

# %s

Demo note content.
`, title, noteType, tagsStr, title)

	return fs.WriteString(filepath.Join(dir, filename), content)
}

// createPlanFiles creates the standard plan files for a plan folder.
func (g *homelabGenerator) createPlanFiles(planDir, planName string) error {
	files := []struct {
		filename string
		title    string
	}{
		{"01-cx.md", "Context"},
		{"02-spec.md", "Specification"},
		{"03-impl.md", "Implementation"},
	}

	for _, f := range files {
		content := fmt.Sprintf(`---
title: %s
type: plan
tags: [plan, %s]
created: 2026-01-15T10:00:00Z
---

# %s for %s

Plan content.
`, f.title, planName, f.title, planName)

		if err := fs.WriteString(filepath.Join(planDir, f.filename), content); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedInboxNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "inbox")
	notes := []demoNote{
		{"api-ideas.md", "api-ideas", "inbox", []string{"inbox", "api"}},
		{"cli-flags.md", "cli-flags", "inbox", []string{"inbox", "cli"}},
		{"auth-flow.md", "auth-flow", "inbox", []string{"inbox", "auth"}},
		{"ui-refresh.md", "ui-refresh", "inbox", []string{"inbox", "ui"}},
		{"logging.md", "logging", "inbox", []string{"inbox", "infra"}},
		{"perf-notes.md", "perf-notes", "inbox", []string{"inbox", "perf"}},
		{"docs-update.md", "docs-update", "inbox", []string{"inbox", "docs"}},
		{"testing-gaps.md", "testing-gaps", "inbox", []string{"inbox", "testing"}},
		{"mobile-ideas.md", "mobile-ideas", "inbox", []string{"inbox", "mobile"}},
		{"security.md", "security", "inbox", []string{"inbox", "security"}},
		{"ci-setup.md", "ci-setup", "inbox", []string{"inbox", "ci"}},
		{"deps-audit.md", "deps-audit", "inbox", []string{"inbox", "deps"}},
		{"db-schema.md", "db-schema", "inbox", []string{"inbox", "database"}},
	}

	for _, note := range notes {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedIssuesNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "issues")
	issues := []string{
		"login-crash", "slow-query", "mem-leak", "api-timeout", "ui-glitch",
		"auth-error", "cache-stale", "404-pages", "upload-fail", "email-delay",
		"search-empty", "mobile-crash", "sync-fail", "export-bug", "dark-mode",
		"date-format", "scroll-jump", "hotkey-broken", "filter-reset", "csv-parse",
		"undo-broken", "drag-drop", "print-layout", "zoom-issue", "clipboard",
		"tooltip-stuck", "redirect-loop", "sort-order", "spinner-stuck", "truncation",
	}

	for _, issue := range issues {
		filename := issue + ".md"
		if err := g.createSimpleNote(dir, filename, issue, "issues", []string{"issues", "bug"}); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedInProgressNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "in_progress")
	notes := []demoNote{
		{"auth-fix.md", "auth-fix", "in_progress", []string{"in_progress", "auth"}},
		{"api-perf.md", "api-perf", "in_progress", []string{"in_progress", "api"}},
	}

	for _, note := range notes {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedReviewNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "review")
	return g.createSimpleNote(dir, "pr-review.md", "pr-review", "review", []string{"review", "pr"})
}

func (g *homelabGenerator) seedLearnNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "learn")
	notes := []demoNote{
		{"golang-tips.md", "golang-tips", "learn", []string{"learn", "go"}},
		{"docker-best.md", "docker-best", "learn", []string{"learn", "docker"}},
		{"k8s-notes.md", "k8s-notes", "learn", []string{"learn", "k8s"}},
		{"postgres.md", "postgres", "learn", []string{"learn", "database"}},
	}

	for _, note := range notes {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedConceptsNotes(notebookDir string) error {
	// Top-level concepts
	dir := filepath.Join(notebookDir, "concepts")
	topLevel := []demoNote{
		{"patterns.md", "patterns", "concepts", []string{"concepts", "design"}},
		{"testing.md", "testing", "concepts", []string{"concepts", "testing"}},
		{"security.md", "security", "concepts", []string{"concepts", "security"}},
		{"naming.md", "naming", "concepts", []string{"concepts", "conventions"}},
		{"logging.md", "logging", "concepts", []string{"concepts", "logging"}},
		{"metrics.md", "metrics", "concepts", []string{"concepts", "metrics"}},
	}

	for _, note := range topLevel {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}

	// Architecture subfolder concepts
	archDir := filepath.Join(dir, "architecture")
	archNotes := []demoNote{
		{"api-design.md", "api-design", "concepts", []string{"concepts", "architecture"}},
		{"data-flow.md", "data-flow", "concepts", []string{"concepts", "architecture"}},
		{"auth-model.md", "auth-model", "concepts", []string{"concepts", "architecture"}},
		{"cache-layer.md", "cache-layer", "concepts", []string{"concepts", "architecture"}},
		{"events.md", "events", "concepts", []string{"concepts", "architecture"}},
		{"db-schema.md", "db-schema", "concepts", []string{"concepts", "architecture"}},
		{"error-handling.md", "error-handling", "concepts", []string{"concepts", "architecture"}},
	}

	for _, note := range archNotes {
		if err := g.createSimpleNote(archDir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedDocgenNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "docgen")
	notes := []demoNote{
		{"api-docs.md", "api-docs", "docgen", []string{"docgen", "api"}},
		{"user-guide.md", "user-guide", "docgen", []string{"docgen", "guide"}},
	}

	for _, note := range notes {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedIceboxNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "icebox")
	notes := []demoNote{
		{"v3-ideas.md", "v3-ideas", "icebox", []string{"icebox", "future"}},
		{"nice-to-have.md", "nice-to-have", "icebox", []string{"icebox", "low-priority"}},
	}

	for _, note := range notes {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedLLMNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "llm")
	return g.createSimpleNote(dir, "prompts.md", "prompts", "llm", []string{"llm", "prompts"})
}

func (g *homelabGenerator) seedQuickNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "quick")
	notes := []demoNote{
		{"mtg-notes.md", "mtg-notes", "quick", []string{"quick", "meeting"}},
		{"idea.md", "idea", "quick", []string{"quick", "idea"}},
		{"link.md", "link", "quick", []string{"quick", "bookmark"}},
	}

	for _, note := range notes {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}
	return nil
}

func (g *homelabGenerator) seedTodosNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "todos")
	return g.createSimpleNote(dir, "sprint.md", "sprint", "todos", []string{"todos", "sprint"})
}

func (g *homelabGenerator) seedCompletedNotes(notebookDir string) error {
	dir := filepath.Join(notebookDir, "completed")

	// Create a few named completed notes
	named := []demoNote{
		{"auth-v1.md", "auth-v1", "completed", []string{"completed", "auth"}},
		{"api-init.md", "api-init", "completed", []string{"completed", "api"}},
		{"db-setup.md", "db-setup", "completed", []string{"completed", "database"}},
		{"ci-setup.md", "ci-setup", "completed", []string{"completed", "ci"}},
		{"docker-config.md", "docker-config", "completed", []string{"completed", "docker"}},
	}

	for _, note := range named {
		if err := g.createSimpleNote(dir, note.filename, note.title, note.noteType, note.tags); err != nil {
			return err
		}
	}

	// Create numbered completed notes to reach ~94 total
	for i := 6; i <= 94; i++ {
		filename := fmt.Sprintf("task-%d.md", i)
		title := fmt.Sprintf("task-%d", i)
		if err := g.createSimpleNote(dir, filename, title, "completed", []string{"completed"}); err != nil {
			return err
		}
	}

	return nil
}
