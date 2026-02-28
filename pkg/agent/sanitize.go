package agent

import (
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// sanitizeToolPairs ensures every assistant message with ToolCalls has
// matching tool results, and every tool result has its preceding tool_call.
// Orphaned messages are removed to prevent provider API errors (e.g.,
// Anthropic's "tool_use ids were provided that do not have a tool_use block").
//
// This is applied after history compression to fix pairs that were split
// when forceCompression() or summarizeSession() truncated the history.
func sanitizeToolPairs(messages []providers.Message) []providers.Message {
	// Build set of tool_call IDs present in assistant messages
	toolCallIDs := make(map[string]bool)
	for _, m := range messages {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				if tc.ID != "" {
					toolCallIDs[tc.ID] = true
				}
			}
		}
	}

	// Build set of tool_result IDs present
	toolResultIDs := make(map[string]bool)
	for _, m := range messages {
		if m.Role == "tool" && m.ToolCallID != "" {
			toolResultIDs[m.ToolCallID] = true
		}
	}

	// Filter: keep tool results only if their tool_call exists,
	// and keep assistant tool_call messages only if all results exist
	result := make([]providers.Message, 0, len(messages))
	removed := 0

	for _, m := range messages {
		switch {
		case m.Role == "tool" && m.ToolCallID != "":
			// Keep tool result only if its tool_call is present
			if toolCallIDs[m.ToolCallID] {
				result = append(result, m)
			} else {
				removed++
				logger.DebugCF("agent", "sanitizeToolPairs: removing orphaned tool result",
					map[string]interface{}{"tool_call_id": m.ToolCallID})
			}

		case m.Role == "assistant" && len(m.ToolCalls) > 0:
			// Check if ALL tool_calls have matching results
			allHaveResults := true
			for _, tc := range m.ToolCalls {
				if tc.ID != "" && !toolResultIDs[tc.ID] {
					allHaveResults = false
					break
				}
			}
			if allHaveResults {
				result = append(result, m)
			} else if m.Content != "" {
				// Keep the text content but strip the tool calls
				removed++
				logger.DebugCF("agent", "sanitizeToolPairs: stripping orphaned tool_calls from assistant message, keeping text content",
					map[string]interface{}{"tool_call_count": len(m.ToolCalls)})
				result = append(result, providers.Message{
					Role:    "assistant",
					Content: m.Content,
				})
			} else {
				// No text content and missing results - drop entirely
				removed++
				logger.DebugCF("agent", "sanitizeToolPairs: removing orphaned assistant message with tool_calls",
					map[string]interface{}{"tool_call_count": len(m.ToolCalls)})
			}

		default:
			result = append(result, m)
		}
	}

	if removed > 0 {
		logger.WarnCF("agent", "sanitizeToolPairs: removed orphaned tool pair messages",
			map[string]interface{}{"removed_count": removed})
	}

	return result
}
