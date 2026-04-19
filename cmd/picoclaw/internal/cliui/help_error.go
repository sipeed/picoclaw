package cliui

import (
	"strings"

	"github.com/spf13/cobra"
)

// FormatCLIError formats errors with the same boxed sections as help. When ctx
// is the command that was running when the error occurred, Usage / Flags panels
// are appended so styling matches picoclaw -h.
func FormatCLIError(msg string, ctx *cobra.Command) string {
	msg = strings.TrimRight(msg, "\n")
	if !UseFancyStderr() {
		s := "Error: " + msg + "\n"
		if ctx != nil && showErrHint(msg) {
			s += "\n" + plainCommandHelp(ctx)
		}
		return s
	}
	w := InnerStderrWidth()
	contentW := w - 6
	if contentW < 36 {
		contentW = 36
	}

	title := titleBarStyle().Render("Error") + "\n\n"

	paras := strings.Split(msg, "\n")
	var body strings.Builder
	for i, p := range paras {
		p = strings.TrimRight(p, " ")
		if p == "" {
			continue
		}
		st := bodyStyle().Width(contentW)
		if i > 0 {
			body.WriteString("\n")
		}
		if i == 0 {
			body.WriteString(st.Render(p))
		} else {
			body.WriteString(mutedStyle().Width(contentW).Render(p))
		}
	}

	foot := ""
	if showErrHint(msg) {
		if ctx != nil {
			foot = "\n\n" + mutedStyle().Width(contentW).
				Render("Full command help: "+ctx.CommandPath()+" --help")
		} else {
			foot = "\n\n" + mutedStyle().Width(contentW).
				Render("Tip: picoclaw --help   ·   picoclaw <command> --help")
		}
	}

	out := borderStyle().Width(w).Render(title+body.String()+foot) + "\n"
	if ctx != nil && showErrHint(msg) {
		if ref := RenderCommandQuickRef(ctx, w); ref != "" {
			out += "\n" + ref
		}
	}
	return out
}

func showErrHint(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "unknown flag") ||
		strings.Contains(m, "unknown shorthand flag") ||
		strings.Contains(m, "flag needs an argument") ||
		strings.Contains(m, "invalid argument") ||
		strings.Contains(m, "required flag") ||
		strings.Contains(m, "usage:")
}
