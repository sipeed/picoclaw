package commands

import (
	"context"
	"fmt"
	"strings"
)

func statusCommand() Definition {
	return Definition{
		Name:        "status",
		Description: "Show current session status (model, context, compactions)",
		Usage:       "/status",
		Aliases:     []string{"s"},
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil {
				return req.Reply(unavailableMsg)
			}

			modelName, provider := "Unknown", "Unknown"
			if rt.GetModelInfo != nil {
				modelName, provider = rt.GetModelInfo()
			}

			stats := SessionStats{}
			if rt.GetSessionStats != nil {
				stats = rt.GetSessionStats()
			}

			compactions := 0
			if rt.GetCompactionCount != nil {
				compactions = rt.GetCompactionCount()
			}

			if stats.ContextWindow == 0 {
				stats.ContextWindow = 200000
			}

			version := stats.Version
			if version == "" {
				version = "dev"
			}

			think := "off"
			if stats.ThinkEnabled {
				think = "on"
			}

			var sessionInfo strings.Builder
			sessionInfo.WriteString(fmt.Sprintf("🦞 *PicoClaw %s*\n\n", version))
			sessionInfo.WriteString(fmt.Sprintf("🧠 *Model:* %s/%s\n", provider, modelName))
			sessionInfo.WriteString(fmt.Sprintf(
				"📚 *Context:* %s %s/%s (%.1f%%)\n",
				makeContextBar(stats.ContextPercent),
				formatTokens(stats.TokenEstimate),
				formatTokens(stats.ContextWindow),
				stats.ContextPercent,
			))
			sessionInfo.WriteString(fmt.Sprintf("💬 *History:* %d messages\n", stats.MessageCount))
			if stats.HasSummary {
				sessionInfo.WriteString("📝 *Summary:* present\n")
			} else {
				sessionInfo.WriteString("📝 *Summary:* none\n")
			}
			sessionInfo.WriteString(fmt.Sprintf("🧹 *Compactions:* %d\n", compactions))
			sessionInfo.WriteString("\n💡 *Tokens:* estimated from stored session history\n")

			if stats.SessionKey != "" {
				sessionInfo.WriteString(fmt.Sprintf("🧵 *Session:* %s", stats.SessionKey))
				if stats.SessionUpdated != "" {
					sessionInfo.WriteString(fmt.Sprintf(" - %s", stats.SessionUpdated))
				}
				sessionInfo.WriteString("\n")
			}

			sessionInfo.WriteString(fmt.Sprintf("⚙️ *Runtime:* direct · 🤖 *Think:* %s\n", think))
			return req.Reply(sessionInfo.String())
		},
	}
}

func makeContextBar(percent float64) string {
	const total = 20
	filled := int(percent / 100 * float64(total))
	if filled > total {
		filled = total
	}
	if filled < 0 {
		filled = 0
	}
	empty := total - filled

	var bar strings.Builder
	bar.WriteString("[")
	for i := 0; i < filled; i++ {
		bar.WriteString("█")
	}
	for i := 0; i < empty; i++ {
		bar.WriteString("░")
	}
	bar.WriteString("]")
	return bar.String()
}

func formatTokens(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fm", float64(tokens)/1000000)
	} else if tokens >= 1000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}
