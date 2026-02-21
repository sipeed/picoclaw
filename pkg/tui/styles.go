// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("#FF6B35") // Lobster orange
	secondaryColor = lipgloss.Color("#7B8794")
	errorColor     = lipgloss.Color("#E74C3C")
	successColor   = lipgloss.Color("#2ECC71")
	warningColor   = lipgloss.Color("#F39C12")

	userLabelStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3498DB")).
				Bold(true)

	toolRunningStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	toolDoneStyle = lipgloss.NewStyle().
			Foreground(successColor)

	toolErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2C3E50")).
			Foreground(lipgloss.Color("#ECF0F1")).
			Padding(0, 1)

	thinkingStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34495E"))
)
