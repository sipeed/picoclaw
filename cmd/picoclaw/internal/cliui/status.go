package cliui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProviderRow holds one provider's display name and status value.
type ProviderRow struct {
	Name string
	Val  string
}

// StatusReport is a structured status view for PrintStatus.
type StatusReport struct {
	Logo          string
	Version       string
	Build         string
	ConfigPath    string
	ConfigOK      bool
	WorkspacePath string
	WorkspaceOK   bool
	Model         string
	Providers     []ProviderRow
	OAuthLines    []string // each full line "provider (method): state"
}

// PrintStatus renders picoclaw status (plain or fancy).
func PrintStatus(r StatusReport) {
	if !UseFancyLayout() {
		printStatusPlain(r)
		return
	}
	printStatusFancy(r)
}

func printStatusPlain(r StatusReport) {
	fmt.Printf("%s picoclaw Status\n", r.Logo)
	fmt.Printf("Version: %s\n", r.Version)
	if r.Build != "" {
		fmt.Printf("Build: %s\n", r.Build)
	}
	fmt.Println()

	printPathLine("Config", r.ConfigPath, r.ConfigOK)
	printPathLine("Workspace", r.WorkspacePath, r.WorkspaceOK)

	if r.ConfigOK {
		fmt.Printf("Model: %s\n", r.Model)
		for _, p := range r.Providers {
			fmt.Printf("%s: %s\n", p.Name, p.Val)
		}
		if len(r.OAuthLines) > 0 {
			fmt.Println("\nOAuth/Token Auth:")
			for _, line := range r.OAuthLines {
				fmt.Printf("  %s\n", line)
			}
		}
	}
}

func printPathLine(label, path string, ok bool) {
	mark := "✗"
	if ok {
		mark = "✓"
	}
	fmt.Println(label+":", path, mark)
}

func printStatusFancy(r StatusReport) {
	inner := InnerWidth()
	topBox := borderStyle().Width(inner)

	var head strings.Builder
	head.WriteString(titleBarStyle().Render(r.Logo + " picoclaw Status"))
	head.WriteString("\n\n")
	head.WriteString(kvKeyStyle().Render("Version") + "  " + kvValStyle().Render(r.Version))
	if r.Build != "" {
		head.WriteString("\n")
		head.WriteString(kvKeyStyle().Render("Build") + "     " + kvValStyle().Render(r.Build))
	}
	fmt.Println(topBox.Render(head.String()))
	fmt.Println()

	if UseColumnLayout() && len(r.Providers) > 0 && r.ConfigOK {
		leftW := (inner - 2) / 2
		rightW := inner - leftW - 2
		pathsNarrow := pathStatusPanel(r, leftW)
		prov := providerTablePanel(r, rightW)
		gap := strings.Repeat(" ", 2)
		fmt.Println(lipgloss.JoinHorizontal(lipgloss.Top, pathsNarrow, gap, prov))
	} else {
		fmt.Println(pathStatusPanel(r, inner))
		if len(r.Providers) > 0 && r.ConfigOK {
			fmt.Println(providerTablePanel(r, inner))
		}
	}

	if len(r.OAuthLines) > 0 && r.ConfigOK {
		var ob strings.Builder
		ob.WriteString(titleBarStyle().Render("OAuth / token auth") + "\n\n")
		for _, line := range r.OAuthLines {
			ob.WriteString("  • " + line + "\n")
		}
		fmt.Println()
		fmt.Println(borderStyle().Width(inner).Render(ob.String()))
	}
}

func pathStatusPanel(r StatusReport, inner int) string {
	cfgMark := statusMark(r.ConfigOK)
	wsMark := statusMark(r.WorkspaceOK)
	var b strings.Builder
	b.WriteString(kvKeyStyle().Render("Config") + "\n")
	b.WriteString(mutedStyle().Render(r.ConfigPath))
	b.WriteString(" " + cfgMark + "\n\n")
	b.WriteString(kvKeyStyle().Render("Workspace") + "\n")
	b.WriteString(mutedStyle().Render(r.WorkspacePath))
	b.WriteString(" " + wsMark + "\n")
	if r.ConfigOK {
		b.WriteString("\n")
		b.WriteString(kvKeyStyle().Render("Model") + "  " + kvValStyle().Render(r.Model))
	}
	return borderStyle().Width(inner).Render(b.String())
}

func statusMark(ok bool) string {
	if ok {
		return lipgloss.NewStyle().Foreground(colorOK).Render("✓")
	}
	return lipgloss.NewStyle().Foreground(accentRed).Render("✗")
}

func providerTablePanel(r StatusReport, colW int) string {
	if len(r.Providers) == 0 {
		return ""
	}
	keyW := min(22, colW/3)
	if keyW < 14 {
		keyW = 14
	}
	valW := colW - keyW - 3
	if valW < 12 {
		valW = 12
	}

	var b strings.Builder
	b.WriteString(titleBarStyle().Render("Providers & local") + "\n\n")
	for _, p := range r.Providers {
		k := lipgloss.NewStyle().Foreground(accentBlue).Bold(true).Width(keyW).Render(p.Name)
		v := styleProviderVal(p.Val).Width(valW).Render(p.Val)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, k, "  ", v))
		b.WriteString("\n")
	}
	return borderStyle().Width(colW).Render(strings.TrimRight(b.String(), "\n"))
}

func styleProviderVal(s string) lipgloss.Style {
	if s == "✓" || strings.HasPrefix(s, "✓ ") {
		return lipgloss.NewStyle().Foreground(colorOK)
	}
	if s == "not set" {
		return mutedStyle()
	}
	return lipgloss.NewStyle()
}
