package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-core/tui/theme"
)

var (
	// Color palette - DEPRECATED, use theme.DefaultTheme.Colors
	primaryColor    = theme.DefaultTheme.Colors.Orange
	successColor    = theme.DefaultTheme.Colors.Green
	errorColor      = theme.DefaultTheme.Colors.Red
	warningColor    = theme.DefaultTheme.Colors.Yellow
	infoColor       = theme.DefaultTheme.Colors.Cyan
	mutedColor      = theme.DefaultTheme.Colors.MutedText
	backgroundColor = lipgloss.Color("#1E1E2E") // Keep for now, or adapt
	foregroundColor = theme.DefaultTheme.Colors.LightText

	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			Background(backgroundColor)

	// Header styles
	HeaderStyle = theme.DefaultTheme.Header

	TitleStyle = theme.DefaultTheme.Title

	// Status styles
	SuccessStyle = theme.DefaultTheme.Success
	ErrorStyle   = theme.DefaultTheme.Error
	WarningStyle = theme.DefaultTheme.Warning
	InfoStyle    = theme.DefaultTheme.Info
	MutedStyle   = theme.DefaultTheme.Muted

	// Container styles
	BoxStyle = theme.DefaultTheme.Box

	CodeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#313244")).
			Foreground(lipgloss.Color("#F8F8F2")).
			Padding(0, 1).
			MarginLeft(2)

	// Progress styles
	ProgressBarStyle = lipgloss.NewStyle().
				Background(mutedColor).
				Height(1)

	ProgressFillStyle = lipgloss.NewStyle().
				Background(successColor).
				Height(1)

	// Step styles
	StepNumberStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Width(3).
			Align(lipgloss.Right)

	StepNameStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			MarginLeft(1)

	StepRunningStyle = lipgloss.NewStyle().
				Foreground(infoColor).
				MarginLeft(1)

	StepSuccessStyle = lipgloss.NewStyle().
				Foreground(successColor).
				MarginLeft(1)

	StepErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			MarginLeft(1)

	// List styles
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(0)

	BulletStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
)

// Status icons
const (
	IconPending = "⏳"
	IconRunning = "🔄"
	IconSuccess = "✅"
	IconError   = "❌"
	IconWarning = "⚠️"
	IconInfo    = "ℹ️"
	IconArrow   = "→"
	IconBullet  = "•"
)