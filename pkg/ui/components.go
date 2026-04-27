package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/grovetools/core/tui/theme"
)

// Status icons - using grove-core theme icons
var (
	IconPending = theme.IconPending
	IconRunning = theme.IconRunning
	IconSuccess = theme.IconSuccess
	IconError   = theme.IconError
	IconWarning = theme.IconWarning
	IconInfo    = theme.IconInfo
	IconArrow   = theme.IconArrow
	IconBullet  = theme.IconBullet
)

// Header renders a styled header with title and description
func Header(title, description string) string {
	var parts []string

	if title != "" {
		parts = append(parts, theme.DefaultTheme.Title.Render(title))
	}

	if description != "" {
		parts = append(parts, theme.DefaultTheme.Muted.Render(description))
	}

	return strings.Join(parts, "\n")
}

// Box renders content in a styled box
func Box(title, content string) string {
	style := theme.DefaultTheme.Box
	if title != "" {
		style = style.BorderTop(true).BorderTopForeground(theme.DefaultTheme.Colors.Orange)
		titleLine := lipgloss.JoinHorizontal(lipgloss.Left,
			" ",
			theme.DefaultTheme.Header.Render(title),
			" ",
		)
		content = titleLine + "\n" + content
	}
	return style.Render(content)
}

// ProgressBar renders a progress bar
func ProgressBar(current, total, width int) string {
	if width <= 0 {
		width = 40
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))

	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	progressBarStyle := lipgloss.NewStyle().
		Background(theme.DefaultTheme.Colors.MutedText).
		Height(1)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		progressBarStyle.Width(width).Render(bar),
		fmt.Sprintf(" %d/%d (%.1f%%)", current, total, percentage*100),
	)
}

// StepStatus renders a step with its status
func StepStatus(number int, name string, status StepStatusType, duration time.Duration) string {
	var icon string
	var style lipgloss.Style
	var statusText string

	stepNameStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.LightText).
		MarginLeft(1)
	stepRunningStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Cyan).
		MarginLeft(1)
	stepSuccessStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Green).
		MarginLeft(1)
	stepErrorStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Red).
		MarginLeft(1)

	switch status {
	case StatusPending:
		icon = IconPending
		style = stepNameStyle
		statusText = name
	case StatusRunning:
		icon = IconRunning
		style = stepRunningStyle
		statusText = name + " (running...)"
	case StatusSuccess:
		icon = IconSuccess
		style = stepSuccessStyle
		if duration > 0 {
			statusText = fmt.Sprintf("%s (%v)", name, duration.Round(time.Millisecond))
		} else {
			statusText = name
		}
	case StatusError:
		icon = IconError
		style = stepErrorStyle
		if duration > 0 {
			statusText = fmt.Sprintf("%s (failed after %v)", name, duration.Round(time.Millisecond))
		} else {
			statusText = name + " (failed)"
		}
	}

	stepNumberStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Orange).
		Bold(true).
		Width(3).
		Align(lipgloss.Right)

	numberStr := stepNumberStyle.Render(fmt.Sprintf("%d.", number))
	iconStr := style.Width(2).Render(icon)
	nameStr := style.Render(statusText)

	return lipgloss.JoinHorizontal(lipgloss.Left, numberStr, iconStr, nameStr)
}

// ErrorBox renders an error in a styled box
func ErrorBox(err error) string {
	if err == nil {
		return ""
	}

	content := theme.DefaultTheme.Error.Render("Error: ") + err.Error()
	return theme.DefaultTheme.Box.
		BorderForeground(theme.DefaultTheme.Colors.Red).
		Render(content)
}

// SuccessBox renders a success message in a styled box
func SuccessBox(message string) string {
	content := theme.DefaultTheme.Success.Render("Success: ") + message
	return theme.DefaultTheme.Box.
		BorderForeground(theme.DefaultTheme.Colors.Green).
		Render(content)
}

// InfoBox renders an info message in a styled box
func InfoBox(message string) string {
	content := theme.DefaultTheme.Info.Render("Info: ") + message
	return theme.DefaultTheme.Box.
		BorderForeground(theme.DefaultTheme.Colors.Cyan).
		Render(content)
}

// CodeBlock renders code in a styled block
func CodeBlock(code string) string {
	codeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#313244")).
		Foreground(lipgloss.Color("#F8F8F2")).
		Padding(0, 1).
		MarginLeft(2)
	return codeStyle.Render(code)
}

// List renders a bulleted list
func List(items []string) string {
	var parts []string
	bulletStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Orange).
		Bold(true)
	listItemStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		MarginBottom(0)
	for _, item := range items {
		bullet := bulletStyle.Render(IconBullet)
		text := listItemStyle.Render(item)
		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Left, bullet, text))
	}
	return strings.Join(parts, "\n")
}

// Summary renders a scenario summary
func Summary(name string, success bool, duration time.Duration, stepCount int) string {
	var statusIcon string
	var statusStyle lipgloss.Style
	var statusText string

	if success {
		statusIcon = IconSuccess
		statusStyle = theme.DefaultTheme.Success
		statusText = "PASSED"
	} else {
		statusIcon = IconError
		statusStyle = theme.DefaultTheme.Error
		statusText = "FAILED"
	}

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		statusStyle.Render(statusIcon+" "+statusText),
		" ",
		theme.DefaultTheme.Header.Render(name),
	)

	details := theme.DefaultTheme.Muted.Render(fmt.Sprintf(
		"Duration: %v | Steps: %d",
		duration.Round(time.Millisecond),
		stepCount,
	))

	return lipgloss.JoinVertical(lipgloss.Left, header, details)
}

// StepStatusType represents the status of a step
type StepStatusType int

const (
	StatusPending StepStatusType = iota
	StatusRunning
	StatusSuccess
	StatusError
)
