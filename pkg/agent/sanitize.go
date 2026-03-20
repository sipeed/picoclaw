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
// Uses a two-pass approach:
//  1. Forward pass: decide which assistant tool_call messages to retain,
//     tracking which tool_call IDs survive. Also collect all tool_result IDs.
//  2. Forward pass: emit retained messages, dropping tool results whose
//     tool_call was not retained.
//
// This is applied after history compression to fix pairs that were split
// when forceCompression() or summarizeSession() truncated the history.
func sanitizeToolPairs(messages []providers.Message) []providers.Message {
	// Collect all tool_result IDs (needed to decide if assistant msgs survive)
	toolResultIDs := make(map[string]bool)
	for _, m := range messages {
		if m.Role == "tool" && m.ToolCallID != "" {
			toolResultIDs[m.ToolCallID] = true
		}
	}

	// Forward pass: decide which assistant tool_call messages to keep,
	// tracking the set of retained tool_call IDs.
	retainedCallIDs := make(map[string]bool)
	type decision struct {
		keep     bool
		modified bool // true if we strip tool_calls but keep text
	}
	assistantDecisions := make(map[int]decision) // index -> decision

	for i, m := range messages {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			allHaveResults := true
			for _, tc := range m.ToolCalls {
				if tc.ID != "" && !toolResultIDs[tc.ID] {
					allHaveResults = false
					break
				}
			}
			if allHaveResults {
				assistantDecisions[i] = decision{keep: true}
				for _, tc := range m.ToolCalls {
					if tc.ID != "" {
						retainedCallIDs[tc.ID] = true
					}
				}
			} else if m.Content != "" {
				assistantDecisions[i] = decision{keep: true, modified: true}
			} else {
				assistantDecisions[i] = decision{keep: false}
			}
		}
	}

	// Emit pass: build result using retained IDs
	result := make([]providers.Message, 0, len(messages))
	removed := 0
	modified := 0

	for i, m := range messages {
		switch {
		case m.Role == "tool" && m.ToolCallID != "":
			if retainedCallIDs[m.ToolCallID] {
				result = append(result, m)
			} else {
				removed++
				logger.DebugCF("agent", "sanitizeToolPairs: removing orphaned tool result",
					map[string]interface{}{"tool_call_id": m.ToolCallID})
			}

		case m.Role == "assistant" && len(m.ToolCalls) > 0:
			d := assistantDecisions[i]
			if !d.keep {
				removed++
				logger.DebugCF("agent", "sanitizeToolPairs: removing orphaned assistant tool_call message",
					map[string]interface{}{"tool_call_count": len(m.ToolCalls)})
			} else if d.modified {
				modified++
				logger.DebugCF("agent", "sanitizeToolPairs: stripping orphaned tool_calls, keeping text",
					map[string]interface{}{"tool_call_count": len(m.ToolCalls)})
				result = append(result, providers.Message{
					Role:    "assistant",
					Content: m.Content,
				})
			} else {
				result = append(result, m)
			}

		default:
			result = append(result, m)
		}
	}

	if removed > 0 || modified > 0 {
		logger.WarnCF("agent", "sanitizeToolPairs: cleaned orphaned tool pair messages",
			map[string]interface{}{"removed": removed, "modified": modified})
	}

	return result
}
