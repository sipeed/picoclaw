package gateway

import (
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// messageToGateway converts a providers.Message to a WebClaw GatewayMessage (as map for JSON).
// Order of messages is preserved by the caller (session history order).
func messageToGateway(m providers.Message, index int, baseTime int64) map[string]any {
	out := map[string]any{
		"role": m.Role,
	}
	if baseTime > 0 {
		out["createdAt"] = baseTime + int64(index)*1000
	}
	// Generate a stable id for the message.
	id := msgID(m, index)
	out["id"] = id

	switch m.Role {
	case "user":
		out["content"] = []map[string]any{
			{"type": "text", "text": m.Content},
		}
	case "assistant":
		var content []map[string]any
		if m.Content != "" {
			content = append(content, map[string]any{"type": "text", "text": m.Content})
		}
		for _, tc := range m.ToolCalls {
			content = append(content, map[string]any{
				"type":      "toolCall",
				"id":        tc.ID,
				"name":      tc.Name,
				"arguments": tc.Arguments,
			})
		}
		if len(content) == 0 {
			content = []map[string]any{{"type": "text", "text": ""}}
		}
		out["content"] = content
	case "tool":
		// WebClaw expects role "toolResult" for tool results.
		out["role"] = "toolResult"
		out["toolCallId"] = m.ToolCallID
		out["content"] = []map[string]any{
			{"type": "text", "text": m.Content},
		}
	default:
		out["content"] = []map[string]any{
			{"type": "text", "text": m.Content},
		}
	}
	return out
}

func msgID(m providers.Message, index int) string {
	h := sha1.New()
	h.Write([]byte(m.Role + m.Content + m.ToolCallID))
	for _, tc := range m.ToolCalls {
		h.Write([]byte(tc.ID + tc.Name))
	}
	return fmt.Sprintf("msg-%x-%d", h.Sum(nil)[:8], index)
}

// historyToGatewayMessages converts session history to WebClaw messages array.
// baseTime is used for createdAt when not stored (e.g. 2026-01-01 00:00:00 UTC ms).
func historyToGatewayMessages(history []providers.Message, baseTime int64) []map[string]any {
	if baseTime == 0 {
		baseTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	}
	out := make([]map[string]any, 0, len(history))
	for i := range history {
		out = append(out, messageToGateway(history[i], i, baseTime))
	}
	return out
}
