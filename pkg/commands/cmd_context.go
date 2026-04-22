package commands

import (
	"context"
	"fmt"
)

func contextCommand() Definition {
	return Definition{
		Name:        "context",
		Description: "Show current session context and token usage",
		Usage:       "/context",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetContextStats == nil {
				return req.Reply(unavailableMsg)
			}
			stats := rt.GetContextStats()
			if stats == nil {
				return req.Reply("No active session context.")
			}
			return req.Reply(formatContextStats(stats))
		},
	}
}

func formatContextStats(s *ContextStats) string {
	remaining := s.CompressAtTokens - s.UsedTokens
	if remaining < 0 {
		remaining = 0
	}
	usedWindowPercent := s.UsedTokens * 100 / max(s.TotalTokens, 1)
	return fmt.Sprintf(
		"Context usage  \nMessages: %d  \nUsed: ~%d / %d tokens (%d%%)  \nCompress at: %d tokens  \nCompression progress: %d%%  \nRemaining: ~%d tokens",
		s.MessageCount,
		s.UsedTokens,
		s.TotalTokens,
		usedWindowPercent,
		s.CompressAtTokens,
		s.UsedPercent,
		remaining,
	)
}
