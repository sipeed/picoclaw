package commands

import (
	"context"
	"fmt"
)

// statsCommand returns the definition for the /stats command
func statsCommand() Definition {
	return Definition{
		Name:        "stats",
		Description: "Show token usage statistics for current session",
		Handler:     handleStats,
		Strict:      true,
	}
}

func handleStats(ctx context.Context, req Request, rt *Runtime) error {
	if rt.GetTokenStats == nil {
		return req.Reply("Token stats not available")
	}

	stats := rt.GetTokenStats()
	if stats == nil {
		return req.Reply("No token usage data for current session")
	}

	return req.Reply(fmt.Sprintf(`Token Stats (Current Session)
Model: %s
Prompt: %d
Completion: %d
Total: %d
Calls: %d`,
		stats.Model,
		stats.PromptTokens,
		stats.CompletionTokens,
		stats.TotalTokens,
		stats.CallCount))
}