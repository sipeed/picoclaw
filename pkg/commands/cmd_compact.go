package commands

import (
	"context"
	"fmt"
	"strings"
)

func compactCommand() Definition {
	return Definition{
		Name:        "compact",
		Description: "Compact session context and summarize older messages",
		Usage:       "/compact [instructions]",
		Aliases:     []string{"c"},
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil {
				return req.Reply(unavailableMsg)
			}

			if rt.CompactContext == nil {
				return req.Reply("Compaction is not available in the current context.")
			}

			var instructions string
			if fields := strings.Fields(req.Text); len(fields) > 1 {
				instructions = strings.Join(fields[1:], " ")
			}

			droppedMessages, err := rt.CompactContext(instructions)
			if err != nil {
				return req.Reply("Failed to compact context: " + err.Error())
			}

			if droppedMessages > 0 {
				label := "(default)"
				if instructions != "" {
					label = instructions
				}
				return req.Reply(fmt.Sprintf(
					"Context compacted.\n\n- Messages summarized: %d\n- Instructions: %s\n- Older context was condensed into the session summary.",
					droppedMessages,
					label,
				))
			}

			if instructions != "" {
				return req.Reply("Compaction requested, but the session was already small enough that no history was summarized.")
			}
			return req.Reply("Context is already compact enough. No summarization was needed.")
		},
	}
}
