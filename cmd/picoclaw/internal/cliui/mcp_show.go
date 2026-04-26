package cliui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// MCPShowServer holds the server metadata for PrintMCPShow.
type MCPShowServer struct {
	Name              string
	Type              string
	Target            string
	Enabled           bool
	EffectiveDeferred bool     // resolved value (per-server override or global default)
	DeferredExplicit  bool     // true = per-server override set, false = inherited from global
	EnvKeys           []string // sorted env var names (values intentionally omitted)
	EnvFile           string
	Headers           []string // sorted header names
}

// MCPShowTool holds one tool's info for PrintMCPShow.
type MCPShowTool struct {
	Name        string
	Description string
	Parameters  []MCPShowParam
}

// MCPShowParam is one parameter entry.
type MCPShowParam struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// PrintMCPShow renders the mcp show output (plain or fancy).
// w is where the output is written; pass cmd.OutOrStdout() from cobra commands.
func PrintMCPShow(w io.Writer, server MCPShowServer, tools []MCPShowTool, disabled bool) {
	if !UseFancyLayout() {
		printMCPShowPlain(w, server, tools, disabled)
		return
	}
	printMCPShowFancy(w, server, tools, disabled)
}

// ── plain (narrow / non-TTY) ────────────────────────────────────────────────

func printMCPShowPlain(w io.Writer, server MCPShowServer, tools []MCPShowTool, disabled bool) {
	fmt.Fprintf(w, "Server: %s\n", server.Name)
	fmt.Fprintf(w, "Type:   %s\n", server.Type)
	fmt.Fprintf(w, "Target: %s\n", server.Target)
	fmt.Fprintf(w, "Enabled: %s\n", boolWord(server.Enabled))
	deferredLabel := boolWord(server.EffectiveDeferred)
	if !server.DeferredExplicit {
		deferredLabel += " (default)"
	}
	fmt.Fprintf(w, "Deferred: %s\n", deferredLabel)
	if len(server.EnvKeys) > 0 {
		fmt.Fprintf(w, "Env vars: %s\n", strings.Join(server.EnvKeys, ", "))
	}
	if server.EnvFile != "" {
		fmt.Fprintf(w, "Env file: %s\n", server.EnvFile)
	}
	if len(server.Headers) > 0 {
		fmt.Fprintf(w, "Headers: %s\n", strings.Join(server.Headers, ", "))
	}
	fmt.Fprintln(w)

	if disabled {
		fmt.Fprintln(w, "Server is disabled; skipping tool discovery.")
		return
	}
	if len(tools) == 0 {
		fmt.Fprintln(w, "No tools exposed by this server.")
		return
	}

	fmt.Fprintf(w, "Tools (%d):\n", len(tools))
	for _, tool := range tools {
		fmt.Fprintf(w, "  %s\n", tool.Name)
		if tool.Description != "" {
			fmt.Fprintf(w, "    %s\n", truncateDescription(tool.Description, 120))
		}
		if len(tool.Parameters) == 0 {
			fmt.Fprintln(w, "    Parameters: none")
			continue
		}
		for _, p := range tool.Parameters {
			line := fmt.Sprintf("    - %s", p.Name)
			if p.Type != "" {
				line += fmt.Sprintf(" (%s", p.Type)
				if p.Required {
					line += ", required"
				}
				line += ")"
			} else if p.Required {
				line += " (required)"
			}
			if p.Description != "" {
				line += ": " + truncateDescription(p.Description, 80)
			}
			fmt.Fprintln(w, line)
		}
	}
}

// ── fancy (wide TTY) ────────────────────────────────────────────────────────

var (
	mcpToolNameStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(accentBlue).Bold(true)
	}
	mcpParamNameStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(accentRed).Bold(true)
	}
	mcpTagStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	}
	mcpRequiredStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#D54646")).Bold(true)
	}
	mcpOptionalStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B"))
	}
	mcpDescStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
	}
)

func printMCPShowFancy(w io.Writer, server MCPShowServer, tools []MCPShowTool, disabled bool) {
	inner := InnerWidth()
	box := borderStyle().Width(inner)

	var b strings.Builder

	// ── server header ──
	b.WriteString(titleBarStyle().Render("⬡  " + server.Name))
	b.WriteString("\n\n")

	keyW := 10
	writeKV := func(key, val string) {
		k := kvKeyStyle().Width(keyW).Render(key)
		b.WriteString(k + "  " + val + "\n")
	}

	writeKV("Type", server.Type)
	writeKV("Target", server.Target)
	writeKV("Enabled", coloredBool(server.Enabled))
	deferredVal := coloredBool(server.EffectiveDeferred)
	if !server.DeferredExplicit {
		deferredVal += "  " + mcpTagStyle().Render("(default)")
	}
	writeKV("Deferred", deferredVal)
	if len(server.EnvKeys) > 0 {
		writeKV("Env vars", mutedStyle().Render(strings.Join(server.EnvKeys, ", ")))
	}
	if server.EnvFile != "" {
		writeKV("Env file", mutedStyle().Render(server.EnvFile))
	}
	if len(server.Headers) > 0 {
		writeKV("Headers", mutedStyle().Render(strings.Join(server.Headers, ", ")))
	}

	if disabled {
		b.WriteString("\n")
		b.WriteString(mutedStyle().Render("Server is disabled; skipping tool discovery."))
		fmt.Fprintln(w, box.Render(b.String()))
		return
	}

	if len(tools) == 0 {
		b.WriteString("\n")
		b.WriteString(mutedStyle().Render("No tools exposed by this server."))
		fmt.Fprintln(w, box.Render(b.String()))
		return
	}

	// ── tools section ──
	b.WriteString("\n")
	b.WriteString(kvKeyStyle().Render(fmt.Sprintf("Tools (%d)", len(tools))))
	b.WriteString("\n")

	contentW := inner - 4 // account for box padding
	for i, tool := range tools {
		if i > 0 {
			b.WriteString(strings.Repeat("─", contentW) + "\n")
		}
		b.WriteString("\n")

		// Tool name + index badge
		badge := mcpTagStyle().Render(fmt.Sprintf("[%d/%d]", i+1, len(tools)))
		b.WriteString("  " + mcpToolNameStyle().Render(tool.Name) + "  " + badge + "\n")

		// Description (wrapped to content width)
		if tool.Description != "" {
			desc := truncateDescription(tool.Description, 160)
			b.WriteString("  " + mcpDescStyle().Render(desc) + "\n")
		}

		// Parameters
		if len(tool.Parameters) == 0 {
			b.WriteString("  " + mcpTagStyle().Render("no parameters") + "\n")
			continue
		}

		b.WriteString("\n")
		for _, p := range tool.Parameters {
			// name
			pName := mcpParamNameStyle().Render(p.Name)

			// type tag
			typeTag := ""
			if p.Type != "" {
				typeTag = "  " + mcpTagStyle().Render("<"+p.Type+">")
			}

			// required / optional badge
			var reqBadge string
			if p.Required {
				reqBadge = "  " + mcpRequiredStyle().Render("required")
			} else {
				reqBadge = "  " + mcpOptionalStyle().Render("optional")
			}

			b.WriteString("    " + pName + typeTag + reqBadge + "\n")

			if p.Description != "" {
				desc := truncateDescription(p.Description, 120)
				b.WriteString("      " + mutedStyle().Render(desc) + "\n")
			}
		}
	}

	fmt.Fprintln(w, box.Render(b.String()))
}

// ── mcp list ────────────────────────────────────────────────────────────────

// MCPListRow is one row in the mcp list output.
type MCPListRow struct {
	Name              string
	Type              string
	Target            string
	Status            string // "enabled", "disabled", "ok (N tools)", "error"
	EffectiveDeferred bool   // resolved value (per-server override or global default)
	DeferredExplicit  bool   // true = per-server override set, false = inherited from global
}

// PrintMCPList renders the mcp list output (plain or fancy).
func PrintMCPList(w io.Writer, rows []MCPListRow) {
	if !UseFancyLayout() {
		printMCPListPlain(w, rows)
		return
	}
	printMCPListFancy(w, rows)
}

func printMCPListPlain(w io.Writer, rows []MCPListRow) {
	headers := []string{"Name", "Type", "Command", "Status", "Deferred"}
	tableRows := make([][]string, len(rows))
	for i, r := range rows {
		deferred := boolWord(r.EffectiveDeferred)
		if !r.DeferredExplicit {
			deferred += " (default)"
		}
		tableRows[i] = []string{r.Name, r.Type, r.Target, r.Status, deferred}
	}
	// reuse the ASCII table renderer already in helpers.go via the caller
	// (list.go still uses renderTable for the plain path)
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range tableRows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	border := func() {
		fmt.Fprint(w, "+")
		for _, width := range widths {
			fmt.Fprint(w, strings.Repeat("-", width+2)+"+")
		}
		fmt.Fprintln(w)
	}
	writeRow := func(row []string) {
		fmt.Fprint(w, "|")
		for i, cell := range row {
			fmt.Fprintf(w, " %s%s |", cell, strings.Repeat(" ", widths[i]-len(cell)))
		}
		fmt.Fprintln(w)
	}
	border()
	writeRow(headers)
	border()
	for _, row := range tableRows {
		writeRow(row)
	}
	border()
}

func printMCPListFancy(w io.Writer, rows []MCPListRow) {
	inner := InnerWidth()
	box := borderStyle().Width(inner)

	var b strings.Builder

	title := fmt.Sprintf("MCP Servers (%d)", len(rows))
	b.WriteString(titleBarStyle().Render(title))
	b.WriteString("\n")

	contentW := inner - 4
	for i, row := range rows {
		if i > 0 {
			b.WriteString(strings.Repeat("─", contentW) + "\n")
		}
		b.WriteString("\n")

		statusBadge := mcpListStatusStyle(row.Status).Render(row.Status)
		var deferredBadge string
		if row.EffectiveDeferred {
			if row.DeferredExplicit {
				deferredBadge = "  " + mcpTagStyle().Render("deferred")
			} else {
				deferredBadge = "  " + mcpOptionalStyle().Render("deferred (default)")
			}
		}
		b.WriteString("  " + mcpToolNameStyle().Render(row.Name) + "  " + statusBadge + deferredBadge + "\n")
		b.WriteString("  " + mcpTagStyle().Render(row.Type+"  "+row.Target) + "\n")
	}

	fmt.Fprintln(w, box.Render(b.String()))
}

func mcpListStatusStyle(status string) lipgloss.Style {
	switch {
	case status == "enabled":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#2E7D32")).Bold(true)
	case status == "disabled":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B"))
	case strings.HasPrefix(status, "ok"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#2E7D32")).Bold(true)
	case status == "error":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#D54646")).Bold(true)
	default:
		return lipgloss.NewStyle()
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func boolWord(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func coloredBool(v bool) string {
	if v {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#2E7D32")).Bold(true).Render("yes")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#D54646")).Render("no")
}

// truncateDescription strips newlines, collapses whitespace, and caps length.
func truncateDescription(s string, maxLen int) string {
	// collapse newlines and repeated spaces into a single space
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= maxLen {
		return s
	}
	// cut at last space before maxLen
	cut := s[:maxLen]
	if idx := strings.LastIndex(cut, " "); idx > maxLen/2 {
		cut = cut[:idx]
	}
	return cut + "…"
}
