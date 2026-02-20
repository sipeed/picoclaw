package tools

import (
	"context"
	"fmt"
)

type PushoverTool struct {
	pushoverCallback func(message string) error
}

func NewPushoverTool() *PushoverTool {
	return &PushoverTool{}
}

func (t *PushoverTool) Name() string {
	return "pushover"
}

func (t *PushoverTool) Description() string {
	return "Send a push notification to your phone via Pushover. Use this when you need to notify yourself of something important."
}

func (t *PushoverTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "The notification message to send to your phone",
			},
		},
		"required": []string{"message"},
	}
}

func (t *PushoverTool) SetPushoverCallback(callback func(message string) error) {
	t.pushoverCallback = callback
}

func (t *PushoverTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	message, ok := args["message"].(string)
	if !ok {
		return &ToolResult{ForLLM: "message is required", IsError: true}
	}

	if t.pushoverCallback == nil {
		return &ToolResult{ForLLM: "Pushover not configured", IsError: true}
	}

	if err := t.pushoverCallback(message); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("failed to send pushover notification: %v", err),
			IsError: true,
			Err:     err,
		}
	}

	return &ToolResult{
		ForLLM: fmt.Sprintf("Push notification sent: %s", message),
	}
}
