package session

import "github.com/sipeed/picoclaw/pkg/providers"

// sanitizeRecoveredHistory drops any dangling tail that starts with an
// assistant tool-call message whose tool results were never fully persisted.
//
// This prevents a restarted agent from restoring an unfinished runtime state
// (assistant tool calls plus later steering/user messages) as if it were valid
// history. We keep completed tool-call sequences intact and only trim the
// incomplete suffix.
func sanitizeRecoveredHistory(history []providers.Message) []providers.Message {
	for i := 0; i < len(history); i++ {
		msg := history[i]
		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			continue
		}

		expected := make(map[string]bool, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			expected[tc.ID] = false
		}

		j := i + 1
		for ; j < len(history); j++ {
			next := history[j]
			if next.ToolCallID == "" {
				break
			}
			if _, ok := expected[next.ToolCallID]; ok {
				expected[next.ToolCallID] = true
			}
		}

		complete := true
		for _, found := range expected {
			if !found {
				complete = false
				break
			}
		}
		if !complete {
			return append([]providers.Message(nil), history[:i]...)
		}

		if j > i+1 {
			i = j - 1
		}
	}

	return append([]providers.Message(nil), history...)
}
