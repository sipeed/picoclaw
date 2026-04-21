package integrationtools

import (
	"context"
	"fmt"
)

type MessageTool struct {
	messageDispatchTool
}

func NewMessageTool() *MessageTool {
	return &MessageTool{
		messageDispatchTool: newMessageDispatchTool(),
	}
}

func (t *MessageTool) Name() string {
	return "message"
}

func (t *MessageTool) Description() string {
	return "Send a structured or plain-text message to the user. Use for interactive UI elements (options, cards, forms, progress, todos, alerts) that need immediate rendering. Always include content as plain-text fallback."
}

func (t *MessageTool) Parameters() map[string]any {
	optionItemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"label":       map[string]any{"type": "string"},
			"value":       map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
		},
		"required": []string{"label", "value"},
	}
	actionItemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"label": map[string]any{"type": "string"},
			"value": map[string]any{"type": "string"},
		},
		"required": []string{"label"},
	}
	structuredPartSchema := map[string]any{
		"type": "object",
		"oneOf": []any{
			map[string]any{
				"description": "Interactive options list for the user to choose from.",
				"properties": map[string]any{
					"type":    map[string]any{"type": "string", "const": "options"},
					"options": map[string]any{"type": "array", "items": optionItemSchema, "minItems": 1},
					"mode":    map[string]any{"type": "string", "enum": []string{"single", "multiple"}, "default": "single"},
				},
				"required": []string{"type", "options"},
			},
			map[string]any{
				"description": "Rich card. Built-in semantic kinds: 'form', 'progress', 'todo', 'alert'. Custom kinds are also accepted and passed through to the frontend renderer.",
				"properties": map[string]any{
					"type":    map[string]any{"type": "string", "const": "card"},
					"title":   map[string]any{"type": "string"},
					"kind":    map[string]any{"type": "string", "description": "Semantic subtype. Built-in: 'form'|'progress'|'todo'|'alert'. Custom values are forwarded to the frontend as-is."},
					"blocks":  map[string]any{"type": "array"},
					"actions": map[string]any{"type": "array", "items": actionItemSchema},
				},
				"required": []string{"type"},
			},
			map[string]any{
				"description": "Custom part type. Any object with a 'type' string field is accepted and forwarded to the frontend renderer as-is.",
				"properties": map[string]any{
					"type": map[string]any{"type": "string"},
				},
				"required": []string{"type"},
			},
		},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The visible text summary to send. Keep this even when using structured UI payloads so non-structured clients still have a readable fallback.",
			},
			"channel": map[string]any{
				"type":        "string",
				"description": "Optional: target channel (telegram, whatsapp, etc.)",
			},
			"chat_id": map[string]any{
				"type":        "string",
				"description": "Optional: target chat/user ID",
			},
			"reply_to_message_id": map[string]any{
				"type":        "string",
				"description": "Optional: reply target message ID for channels that support threaded replies",
			},
			"structured": map[string]any{
				"description": "Optional structured payload for rich UI rendering. Can be a single part or an array of parts.",
				"oneOf": []any{
					structuredPartSchema,
					map[string]any{
						"type":     "array",
						"items":    structuredPartSchema,
						"minItems": 1,
					},
				},
			},
		},
		"required": []string{"content"},
	}
}

func (t *MessageTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	content, ok := args["content"].(string)
	if !ok {
		return &ToolResult{ForLLM: "content is required", IsError: true}
	}

	structured, err := parseMessageStructuredArgs(args)
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true}
	}
	return t.executeSend(ctx, args, content, structured)
}

func parseMessageStructuredArgs(args map[string]any) (any, error) {
	if _, exists := args["options"]; exists {
		return nil, fmt.Errorf("message does not accept top-level options; use structured.type='options'")
	}

	if rawStructured, ok := args["structured"]; ok && rawStructured != nil {
		return normalizeStructuredPayload(rawStructured)
	}

	return nil, nil
}

func normalizeStructuredPayload(rawStructured any) (any, error) {
	switch structured := rawStructured.(type) {
	case map[string]any:
		return normalizeStructuredEntry(structured, "structured")
	case []any:
		if len(structured) == 0 {
			return nil, fmt.Errorf("structured must not be empty")
		}
		result := make([]any, 0, len(structured))
		for index, item := range structured {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("structured[%d] must be an object", index)
			}
			normalized, err := normalizeStructuredEntry(entry, fmt.Sprintf("structured[%d]", index))
			if err != nil {
				return nil, err
			}
			result = append(result, normalized)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("structured must be an object or array")
	}
}

// StructuredPart is implemented by every canonical and alias structured part type.
// Parse validates and ingests raw LLM input; ToMap serializes back to the wire format.
// This mirrors VS Code's ChatResponsePart design: each kind owns its own schema.
