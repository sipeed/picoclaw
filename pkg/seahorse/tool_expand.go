package seahorse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// ExpandTool recovers full message content by ID.
type ExpandTool struct {
	engine *RetrievalEngine
}

func NewExpandTool(engine *RetrievalEngine) *ExpandTool {
	return &ExpandTool{engine: engine}
}

func (t *ExpandTool) Name() string {
	return "short_expand"
}

func (t *ExpandTool) Description() string {
	return `Get full message content by ID.

Use when short_grep returns messages and you need complete content (not just snippet).

Parameters:
- message_ids (required): Array of message ID strings (from short_grep results)
- all_conversations: Expand IDs from any conversation (default: current conversation only)

Returns message with:
- content: Full text content
- parts: Structured content
  - text: Full text
  - tool_use: name, arguments, toolCallId
  - tool_result: toolCallId only (content omitted - re-run tool if needed)
  - media: mediaUri (file path), mimeType

Notes:
- tool_result content is not returned (can be large). Re-run the tool if you need the result.
- Media files are stored on disk at mediaUri path, use bash to access.
- By default, IDs outside the current conversation are rejected and reported in rejectedMessageIds.
- If short_grep used all_conversations: true, pass all_conversations: true to expand those IDs.

Example:
  {"message_ids": ["10", "25"]}
  {"message_ids": ["10", "25"], "all_conversations": true}`
}

func (t *ExpandTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message_ids": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Message IDs to expand (from short_grep results, e.g., [\"10\", \"25\"])",
			},
			"all_conversations": map[string]any{
				"type":        "boolean",
				"description": "Expand IDs across all conversations (default: current conversation only)",
			},
		},
		"required": []string{"message_ids"},
	}
}

func (t *ExpandTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	idsRaw, ok := args["message_ids"].([]any)
	if !ok || len(idsRaw) == 0 {
		return tools.ErrorResult(
			"Missing required 'message_ids' argument. " +
				"Example: {\"message_ids\": [\"10\", \"25\"]}")
	}

	// Parse message IDs
	messageIDs := make([]int64, 0, len(idsRaw))
	for _, id := range idsRaw {
		switch v := id.(type) {
		case string:
			var n int64
			if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
				return tools.ErrorResult(fmt.Sprintf("Invalid message_id %q: %v", v, err))
			}
			messageIDs = append(messageIDs, n)
		case float64:
			messageIDs = append(messageIDs, int64(v))
		}
	}

	allConversations, _ := args["all_conversations"].(bool)
	var conversationID int64
	if !allConversations {
		var found bool
		var err error
		conversationID, found, err = t.engine.ConversationIDForSession(ctx, tools.ToolSessionKey(ctx))
		if err != nil {
			return tools.ErrorResult("Expand failed: resolve current conversation: " + err.Error())
		}
		if !found {
			return tools.ErrorResult("Expand failed: no current conversation found for this session. Use all_conversations: true to expand across conversations.")
		}
	}

	result, err := t.engine.ExpandMessagesScoped(ctx, messageIDs, conversationID, allConversations)
	if err != nil {
		return tools.ErrorResult("Expand failed: " + err.Error())
	}

	// Build response with filtered parts
	messages := make([]map[string]any, 0, len(result.Messages))
	for _, msg := range result.Messages {
		parts := make([]map[string]any, 0, len(msg.Parts))
		for _, p := range msg.Parts {
			part := map[string]any{"type": p.Type}
			switch p.Type {
			case "text":
				part["text"] = p.Text
			case "tool_use":
				part["name"] = p.Name
				part["arguments"] = p.Arguments
				part["toolCallId"] = p.ToolCallID
			case "tool_result":
				// Omit content - can be large, re-run tool if needed
				part["toolCallId"] = p.ToolCallID
			case "media":
				part["mediaUri"] = p.MediaURI
				part["mimeType"] = p.MimeType
			}
			parts = append(parts, part)
		}

		messages = append(messages, map[string]any{
			"id":             fmt.Sprintf("%d", msg.ID),
			"role":           msg.Role,
			"content":        msg.Content,
			"parts":          parts,
			"conversationId": msg.ConversationID,
		})
	}

	output := map[string]any{
		"success":            true,
		"tokenCount":         result.TokenCount,
		"messages":           messages,
		"rejectedMessageIds": result.RejectedMessageIDs,
	}
	data, _ := json.Marshal(output)
	return tools.NewToolResult(string(data))
}
