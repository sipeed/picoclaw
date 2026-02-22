package tools

import "context"

// ExitTool sends an "exit" message via WebSocket to terminate the assistant.
// Only active in voice/assistant input modes (controlled via ActivatableTool).
type ExitTool struct {
	sendCallback SendCallbackWithType
	channel      string
	chatID       string
	inputMode    string
}

func NewExitTool() *ExitTool {
	return &ExitTool{}
}

func (t *ExitTool) Name() string { return "exit" }

func (t *ExitTool) Description() string {
	return "Exit the assistant service. Call this when the user wants to end the conversation (e.g. \"おやすみ\", \"終わり\", \"閉じて\"). Provide a short farewell message."
}

func (t *ExitTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "A short farewell message to speak before exiting (e.g. \"おやすみなさい\")",
			},
		},
		"required": []string{"message"},
	}
}

func (t *ExitTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

func (t *ExitTool) SetSendCallback(cb SendCallbackWithType) {
	t.sendCallback = cb
}

func (t *ExitTool) SetInputMode(mode string) {
	t.inputMode = mode
}

func (t *ExitTool) IsActive() bool {
	return t.inputMode == "voice" || t.inputMode == "assistant"
}

func (t *ExitTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.sendCallback == nil {
		return ErrorResult("exit tool: send callback not configured")
	}
	if t.channel == "" || t.chatID == "" {
		return ErrorResult("exit tool: no active channel context")
	}

	message, _ := args["message"].(string)

	t.sendCallback(t.channel, t.chatID, message, "exit")
	return SilentResult("Exit signal sent.")
}
