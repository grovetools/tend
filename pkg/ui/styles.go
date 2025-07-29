package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette
	primaryColor   = lipgloss.Color("#00D4AA")
	successColor   = lipgloss.Color("#00D787")
	errorColor     = lipgloss.Color("#FF5F87")
	warningColor   = lipgloss.Color("#FFAF00")
	infoColor      = lipgloss.Color("#5FAFFF")
	mutedColor     = lipgloss.Color("#6C7086")
	backgroundColor = lipgloss.Color("#1E1E2E")
	foregroundColor = lipgloss.Color("#CDD6F4")

	// Base styles
	BaseStyle = lipgloss.NewStyle().
		Foreground(foregroundColor).
		Background(backgroundColor)

	// Header styles
	HeaderStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		MarginBottom(1)

	TitleStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Underline(true).
		MarginBottom(1)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(warningColor).
		Bold(true)

	InfoStyle = lipgloss.NewStyle().
		Foreground(infoColor).
		Bold(true)

	MutedStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	// Container styles
	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		Margin(1, 0)

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