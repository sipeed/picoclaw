package agent

import (
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// repairOrphanedToolPairs scans messages and:
// 1. Injects synthetic tool_result messages for any tool_use that lacks a matching result
// 2. Drops tool_result messages that lack a preceding tool_use
func repairOrphanedToolPairs(msgs []providers.Message) []providers.Message {
	if len(msgs) == 0 {
		return msgs
	}

	// 1. Collect all tool_call IDs from assistant messages
	toolCallIDs := map[string]bool{}
	for _, m := range msgs {
		if m.Role == "assistant" {
			for _, tc := range m.ToolCalls {
				if tc.ID != "" {
					toolCallIDs[tc.ID] = true
				}
			}
		}
	}

	// 2. Drop orphaned tool_results (no matching tool_call)
	filtered := make([]providers.Message, 0, len(msgs))
	for i, m := range msgs {
		if m.Role == "tool" && m.ToolCallID != "" && !toolCallIDs[m.ToolCallID] {
			logger.DebugCF("agent", "Dropping orphaned tool_result", map[string]any{
				"tool_call_id": m.ToolCallID,
				"index":        i,
			})
			continue
		}
		filtered = append(filtered, m)
	}

	// 3. Collect existing tool_result IDs
	resultIDs := map[string]bool{}
	for _, m := range filtered {
		if m.Role == "tool" && m.ToolCallID != "" {
			resultIDs[m.ToolCallID] = true
		}
	}

	// 4. Build repaired slice, injecting synthetic results for orphaned tool_calls.
	//    Track pending tool_calls from each assistant message and flush missing
	//    results after the last consecutive tool result (or at end of input).
	repaired := make([]providers.Message, 0, len(filtered))
	var pendingCalls []providers.ToolCall

	flushPending := func() {
		for _, tc := range pendingCalls {
			if tc.ID == "" || resultIDs[tc.ID] {
				continue
			}
			name := tc.Name
			if tc.Function != nil {
				name = tc.Function.Name
			}
			logger.DebugCF("agent", "Injecting synthetic tool_result for orphaned tool_call", map[string]any{
				"tool_call_id": tc.ID,
				"tool_name":    name,
			})
			repaired = append(repaired, providers.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    "[tool result unavailable — session history was compressed]",
			})
			resultIDs[tc.ID] = true
		}
		pendingCalls = nil
	}

	for _, m := range filtered {
		// When we hit a non-tool message and have pending calls, flush synthetics
		if m.Role != "tool" && pendingCalls != nil {
			flushPending()
		}

		repaired = append(repaired, m)

		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			pendingCalls = m.ToolCalls
		}
	}

	// Flush any remaining pending calls at the end of input
	flushPending()

	return repaired
}
