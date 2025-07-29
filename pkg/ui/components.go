package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Header renders a styled header with title and description
func Header(title, description string) string {
	var parts []string
	
	if title != "" {
		parts = append(parts, TitleStyle.Render(title))
	}
	
	if description != "" {
		parts = append(parts, MutedStyle.Render(description))
	}
	
	return strings.Join(parts, "\n")
}

// Box renders content in a styled box
func Box(title, content string) string {
	style := BoxStyle
	if title != "" {
		style = style.Copy().BorderTop(true).BorderTopForeground(primaryColor)
		titleLine := lipgloss.JoinHorizontal(lipgloss.Left,
			" ",
			HeaderStyle.Render(title),
			" ",
		)
		content = titleLine + "\n" + content
	}
	return style.Render(content)
}

// ProgressBar renders a progress bar
func ProgressBar(current, total int, width int) string {
	if width <= 0 {
		width = 40
	}
	
	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))
	
	if filled > width {
		filled = width
	}
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	
	return lipgloss.JoinHorizontal(lipgloss.Left,
		ProgressBarStyle.Copy().Width(width).Render(bar),
		fmt.Sprintf(" %d/%d (%.1f%%)", current, total, percentage*100),
	)
}

// StepStatus renders a step with its status
func StepStatus(number int, name string, status StepStatusType, duration time.Duration) string {
	var icon string
	var style lipgloss.Style
	var statusText string
	
	switch status {
	case StatusPending:
		icon = IconPending
		style = StepNameStyle
		statusText = name
	case StatusRunning:
		icon = IconRunning
		style = StepRunningStyle
		statusText = name + " (running...)"
	case StatusSuccess:
		icon = IconSuccess
		style = StepSuccessStyle
		if duration > 0 {
			statusText = fmt.Sprintf("%s (%v)", name, duration.Round(time.Millisecond))
		} else {
			statusText = name
		}
	case StatusError:
		icon = IconError
		style = StepErrorStyle
		if duration > 0 {
			statusText = fmt.Sprintf("%s (failed after %v)", name, duration.Round(time.Millisecond))
		} else {
			statusText = name + " (failed)"
		}
	}
	
	numberStr := StepNumberStyle.Render(fmt.Sprintf("%d.", number))
	iconStr := style.Copy().Width(2).Render(icon)
	nameStr := style.Render(statusText)
	
	return lipgloss.JoinHorizontal(lipgloss.Left, numberStr, iconStr, nameStr)
}

// ErrorBox renders an error in a styled box
func ErrorBox(err error) string {
	if err == nil {
		return ""
	}
	
	content := ErrorStyle.Render("Error: ") + err.Error()
	return BoxStyle.Copy().
		BorderForeground(errorColor).
		Render(content)
}

// SuccessBox renders a success message in a styled box
func SuccessBox(message string) string {
	content := SuccessStyle.Render("Success: ") + message
	return BoxStyle.Copy().
		BorderForeground(successColor).
		Render(content)
}

// InfoBox renders an info message in a styled box
func InfoBox(message string) string {
	content := InfoStyle.Render("Info: ") + message
	return BoxStyle.Copy().
		BorderForeground(infoColor).
		Render(content)
}

// CodeBlock renders code in a styled block
func CodeBlock(code string) string {
	return CodeStyle.Render(code)
}

// List renders a bulleted list
func List(items []string) string {
	var parts []string
	for _, item := range items {
		bullet := BulletStyle.Render(IconBullet)
		text := ListItemStyle.Render(item)
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
		statusStyle = SuccessStyle
		statusText = "PASSED"
	} else {
		statusIcon = IconError
		statusStyle = ErrorStyle
		statusText = "FAILED"
	}
	
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		statusStyle.Render(statusIcon+" "+statusText),
		" ",
		HeaderStyle.Render(name),
	)
	
	details := MutedStyle.Render(fmt.Sprintf(
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