package cliui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

// RenderCommandHelp builds Ruff-style sectioned, two-column help when
// UseFancyLayout(); otherwise plain Cobra-style text.
func RenderCommandHelp(c *cobra.Command) string {
	if !UseFancyLayout() {
		return plainCommandHelp(c)
	}
	syncFlags(c)

	var b strings.Builder
	head, sub := helpIntro(c)
	if head != "" {
		b.WriteString(helpIntroStyle().Render(head))
		b.WriteString("\n")
	}
	if sub != "" {
		b.WriteString(mutedStyle().Render(sub))
		b.WriteString("\n")
	}
	if head != "" || sub != "" {
		b.WriteString("\n")
	}

	inner := InnerWidth()
	contentW := inner - 6
	if contentW < 36 {
		contentW = 36
	}

	// Usage
	usageBody := bodyStyle().MaxWidth(contentW).Render(styleUsageTokens(c.UseLine()))
	b.WriteString(sectionPanel("Usage", usageBody, inner))
	b.WriteString("\n")

	// Examples
	if ex := strings.TrimSpace(c.Example); ex != "" {
		exBody := bodyStyle().Width(contentW).Render(ex)
		b.WriteString(sectionPanel("Examples", exBody, inner))
		b.WriteString("\n")
	}

	// Subcommands
	subs := visibleSubcommands(c)
	if len(subs) > 0 {
		rows := make([][2]string, 0, len(subs))
		for _, sub := range subs {
			left := sub.Name()
			if a := sub.Aliases; len(a) > 0 {
				left += " (" + strings.Join(a, ", ") + ")"
			}
			rows = append(rows, [2]string{left, sub.Short})
		}
		b.WriteString(sectionPanel("Commands", renderTwoColPairs(rows, contentW), inner))
		b.WriteString("\n")
	}

	// Local options
	local := c.LocalFlags()
	opts := collectFlagRows(local)
	if len(opts) > 0 {
		title := "Options"
		if !c.HasParent() {
			title = "Flags"
		}
		b.WriteString(sectionPanel(title, renderTwoColPairs(opts, contentW), inner))
		b.WriteString("\n")
	}

	// Global (inherited) options
	if c.HasAvailableInheritedFlags() {
		inh := collectFlagRows(c.InheritedFlags())
		if len(inh) > 0 {
			b.WriteString(sectionPanel("Global options", renderTwoColPairs(inh, contentW), inner))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// RenderCommandQuickRef prints the same Usage / Flags / Global sections as help,
// for embedding after errors (stderr). outerW is typically InnerStderrWidth().
func RenderCommandQuickRef(c *cobra.Command, outerW int) string {
	if c == nil || outerW < 40 {
		return ""
	}
	syncFlags(c)
	contentW := outerW - 6
	if contentW < 36 {
		contentW = 36
	}
	var b strings.Builder
	usageBody := bodyStyle().MaxWidth(contentW).Render(styleUsageTokens(c.UseLine()))
	b.WriteString(sectionPanel("Usage", usageBody, outerW))
	b.WriteString("\n")
	if len(c.Aliases) > 0 {
		al := "Aliases: " + strings.Join(c.Aliases, ", ")
		alBody := mutedStyle().MaxWidth(contentW).Render(al)
		b.WriteString(sectionPanel("Aliases", alBody, outerW))
		b.WriteString("\n")
	}
	opts := collectFlagRows(c.LocalFlags())
	if len(opts) > 0 {
		title := "Options"
		if !c.HasParent() {
			title = "Flags"
		}
		b.WriteString(sectionPanel(title, renderTwoColPairs(opts, contentW), outerW))
		b.WriteString("\n")
	}
	if c.HasAvailableInheritedFlags() {
		inh := collectFlagRows(c.InheritedFlags())
		if len(inh) > 0 {
			b.WriteString(sectionPanel("Global options", renderTwoColPairs(inh, contentW), outerW))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func syncFlags(c *cobra.Command) {
	_ = c.LocalFlags()
	if c.HasAvailableInheritedFlags() {
		_ = c.InheritedFlags()
	}
}

func plainCommandHelp(c *cobra.Command) string {
	desc := c.Long
	if desc == "" {
		desc = c.Short
	}
	desc = strings.TrimRight(desc, " \t\n\r")
	var b strings.Builder
	if desc != "" {
		fmt.Fprintln(&b, desc)
		fmt.Fprintln(&b)
	}
	if c.Runnable() || c.HasSubCommands() {
		b.WriteString(c.UsageString())
	}
	return b.String()
}

func helpIntro(c *cobra.Command) (head, sub string) {
	head = strings.TrimSpace(c.Short)
	long := strings.TrimSpace(c.Long)
	if long == "" || long == head {
		return head, ""
	}
	lines := strings.Split(long, "\n")
	var rest []string
	for i, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if i == 0 && ln == head {
			continue
		}
		rest = append(rest, ln)
	}
	sub = strings.Join(rest, "\n")
	return head, sub
}

func visibleSubcommands(c *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, sub := range c.Commands() {
		if sub.Hidden {
			continue
		}
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

func sectionPanel(title, body string, width int) string {
	head := titleBarStyle().Render(title) + "\n\n"
	return borderStyle().Width(width).Render(head + body)
}

// styleUsageTokens highlights PicoClaw-blue command tokens and red <placeholders>/[groups].
func styleUsageTokens(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		ia := strings.Index(s, "<")
		ib := strings.Index(s, "[")
		next, kind := -1, 0 // 1 = angle, 2 = bracket
		switch {
		case ia >= 0 && (ib < 0 || ia < ib):
			next, kind = ia, 1
		case ib >= 0:
			next, kind = ib, 2
		}
		if next < 0 {
			b.WriteString(helpIdentStyle().Render(s))
			break
		}
		if next > 0 {
			b.WriteString(helpIdentStyle().Render(s[:next]))
		}
		s = s[next:]
		if kind == 1 {
			j := strings.Index(s, ">")
			if j < 0 {
				b.WriteString(helpIdentStyle().Render(s))
				break
			}
			b.WriteString(helpPlaceholderStyle().Render(s[:j+1]))
			s = s[j+1:]
			continue
		}
		j := strings.Index(s, "]")
		if j < 0 {
			b.WriteString(helpIdentStyle().Render(s))
			break
		}
		b.WriteString(helpPlaceholderStyle().Render(s[:j+1]))
		s = s[j+1:]
	}
	return b.String()
}

func collectFlagRows(fs *flag.FlagSet) [][2]string {
	var names []string
	seen := map[string][2]string{}
	fs.VisitAll(func(f *flag.Flag) {
		if f.Hidden {
			return
		}
		left := formatFlagLeft(f)
		right := f.Usage
		if f.Deprecated != "" {
			right += " (deprecated: " + f.Deprecated + ")"
		}
		names = append(names, f.Name)
		seen[f.Name] = [2]string{left, right}
	})
	sort.Strings(names)
	rows := make([][2]string, 0, len(names))
	for _, n := range names {
		rows = append(rows, seen[n])
	}
	return rows
}

func formatFlagLeft(f *flag.Flag) string {
	if len(f.Shorthand) > 0 {
		return "-" + f.Shorthand + ", --" + f.Name
	}
	return "--" + f.Name
}

func renderTwoColPairs(rows [][2]string, contentW int) string {
	if len(rows) == 0 {
		return ""
	}
	leftW := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > leftW {
			leftW = w
		}
	}
	const minLeft, maxLeft = 16, 34
	if leftW < minLeft {
		leftW = minLeft
	}
	if leftW > maxLeft {
		leftW = maxLeft
	}
	gap := "  "
	rightW := contentW - leftW - lipgloss.Width(gap)
	if rightW < 24 {
		rightW = 24
	}

	var b strings.Builder
	for _, r := range rows {
		left := helpIdentStyle().Width(leftW).Align(lipgloss.Left).Render(r[0])
		right := bodyStyle().Width(rightW).Render(strings.TrimSpace(r[1]))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
