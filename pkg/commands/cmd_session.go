package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/session"
)

func sessionCommand() Definition {
	return Definition{
		Name:        "session",
		Description: "Manage chat sessions",
		SubCommands: []SubCommand{
			{
				Name:        "list",
				Description: "List sessions for current chat",
				Handler:     sessionListHandler(),
			},
			{
				Name:        "resume",
				Description: "Resume a previous session",
				ArgsUsage:   "<index>",
				Handler:     sessionResumeHandler(),
			},
		},
	}
}

func sessionListHandler() Handler {
	return func(_ context.Context, req Request, rt *Runtime) error {
		if rt == nil || rt.SessionOps == nil || strings.TrimSpace(req.ScopeKey) == "" {
			return req.Reply(unavailableMsg)
		}

		list, err := rt.SessionOps.List(req.ScopeKey)
		if err != nil {
			return req.Reply(fmt.Sprintf("Failed to list sessions: %v", err))
		}
		if len(list) == 0 {
			return req.Reply("No sessions found for current chat.")
		}
		return req.Reply(formatSessionList(list))
	}
}

func sessionResumeHandler() Handler {
	return func(_ context.Context, req Request, rt *Runtime) error {
		if rt == nil || rt.SessionOps == nil || strings.TrimSpace(req.ScopeKey) == "" {
			return req.Reply(unavailableMsg)
		}

		// tokens: [/session, resume, <index>]
		indexStr := nthToken(req.Text, 2)
		if indexStr == "" {
			return req.Reply("Usage: /session resume <index>")
		}
		index, err := strconv.Atoi(indexStr)
		if err != nil || index < 1 {
			return req.Reply("Usage: /session resume <index>")
		}

		sessionKey, err := rt.SessionOps.Resume(req.ScopeKey, index)
		if err != nil {
			return req.Reply(fmt.Sprintf("Failed to resume session %d: %v", index, err))
		}
		return req.Reply(fmt.Sprintf("Resumed session %d: %s", index, sessionKey))
	}
}

func formatSessionList(list []session.SessionMeta) string {
	lines := make([]string, 0, len(list)+1)
	lines = append(lines, "Sessions:")
	now := time.Now()
	for _, item := range list {
		activeMarker := "   "
		if item.Active {
			activeMarker = "[*]"
		}

		summary := truncateSummary(item.Summary, 40)

		age := "-"
		if !item.UpdatedAt.IsZero() {
			age = relativeTime(now, item.UpdatedAt)
		}

		tag := extractSessionTag(item.SessionKey)

		lines = append(lines, fmt.Sprintf(
			"%d. %s %s | %d msgs | %s (%s)",
			item.Ordinal, activeMarker, summary, item.MessageCnt, age, tag,
		))
	}
	return strings.Join(lines, "\n")
}

func truncateSummary(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(no summary)"
	}
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractSessionTag returns the "#N" suffix from a session key, or "#1" for
// the initial session that has no "#" separator.
func extractSessionTag(sessionKey string) string {
	if i := strings.LastIndex(sessionKey, "#"); i >= 0 {
		return "#" + sessionKey[i+1:]
	}
	return "#1"
}

func relativeTime(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}
