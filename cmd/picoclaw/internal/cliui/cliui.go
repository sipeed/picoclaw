// Package cliui renders human-oriented CLI output: bordered panels and columns
// on wide interactive terminals. Layout (boxes/columns) is independent of ANSI
// color: use --no-color or NO_COLOR to disable colors only; narrow or non-TTY
// stdout falls back to plain line-oriented output.
package cliui

import (
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// Minimum terminal width (columns) for bordered / structured layout.
// Below this, plain line-oriented output is used so boxes do not wrap badly.
const minWidthFancy = 88

// Minimum width to lay out some views in two columns (e.g. status providers).
const minWidthColumns = 104

var initMu sync.Mutex

// Init configures lipgloss for this process. When disableAnsiColors is true
// (e.g. --no-color, NO_COLOR, or TERM=dumb), only color is turned off; Unicode
// borders still render when UseFancyLayout() is true.
func Init(disableAnsiColors bool) {
	initMu.Lock()
	defer initMu.Unlock()
	if disableAnsiColors {
		lipgloss.SetColorProfile(termenv.Ascii)
		return
	}
	lipgloss.SetColorProfile(termenv.EnvColorProfile())
}

// StdoutWidth returns the terminal width or a sane default if unknown.
func StdoutWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 20 {
		return 80
	}
	return w
}

// UseFancyLayout is true when styled boxes/columns should be used.
func UseFancyLayout() bool {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	return StdoutWidth() >= minWidthFancy
}

// UseColumnLayout is true when a second content column is viable.
func UseColumnLayout() bool {
	return UseFancyLayout() && StdoutWidth() >= minWidthColumns
}

// InnerWidth is the target content width inside borders/margins.
func InnerWidth() int {
	w := StdoutWidth()
	// Rounded border + horizontal padding (lipgloss borders ~= 2 cols each side + padding).
	const borderBudget = 8
	if w > borderBudget+48 {
		return w - borderBudget
	}
	return 48
}

// StderrWidth returns stderr terminal width or a sane default.
func StderrWidth() int {
	w, _, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || w < 20 {
		return 80
	}
	return w
}

// UseFancyStderr is true when stderr can show boxed errors without ugly wraps.
func UseFancyStderr() bool {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return false
	}
	return StderrWidth() >= minWidthFancy
}

// InnerStderrWidth mirrors InnerWidth but for stderr.
func InnerStderrWidth() int {
	w := StderrWidth()
	const borderBudget = 8
	if w > borderBudget+48 {
		return w - borderBudget
	}
	return 48
}

var (
	accentBlue = lipgloss.Color("#3E5DB9")
	accentRed  = lipgloss.Color("#D54646")
	colorMuted = lipgloss.Color("#6B6B6B")
	colorOK    = lipgloss.Color("#2E7D32")
)

func borderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentBlue).
		Padding(0, 1)
}

func titleBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(accentRed).
		Bold(true)
}

func mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorMuted)
}

func bodyStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}

func kvKeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentBlue).Bold(true)
}

func kvValStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}

// helpIntroStyle is the top tagline (PicoClaw blue, matches ASCII banner left side).
func helpIntroStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentBlue).Bold(true)
}

// helpIdentStyle is the left column for commands and flags (blue identifiers).
func helpIdentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentBlue).Bold(true)
}

// helpPlaceholderStyle highlights <placeholders> in usage lines (red accent).
func helpPlaceholderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accentRed).Bold(true)
}
