package tools

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type VoiceLeaveTool struct {
	bus *bus.MessageBus
}

func NewVoiceLeaveTool(mb *bus.MessageBus) *VoiceLeaveTool {
	return &VoiceLeaveTool{bus: mb}
}

func (t *VoiceLeaveTool) Name() string {
	return "voice_leave"
}

func (t *VoiceLeaveTool) Description() string {
	return "Disconnects the bot from the current voice channel. Use this tool when the user says goodbye or explicitly asks you to leave the voice chat."
}

func (t *VoiceLeaveTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *VoiceLeaveTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)

	if channel != "discord" {
		return &ToolResult{ForLLM: "Can only leave voice channels on Discord", IsError: true}
	}

	t.bus.PublishVoiceControl(ctx, bus.VoiceControl{
		ChatID: chatID,
		Type:   "command",
		Action: "leave",
	})

	logger.InfoCF("agent", "Voice command triggered via tool: leave", map[string]any{"chat_id": chatID})

	return &ToolResult{
		ForLLM: "Successfully sent disconnect command to voice adapter.",
	}
}
