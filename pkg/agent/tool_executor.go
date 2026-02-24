package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// ToolExecutor handles executing tool calls requested by the LLM,
// including async callback setup, result routing, and context updates.
// Extracted from AgentLoop to improve separation of concerns.
type ToolExecutor struct {
	bus *bus.MessageBus
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor(msgBus *bus.MessageBus) *ToolExecutor {
	return &ToolExecutor{bus: msgBus}
}

// ExecuteToolCalls processes a list of normalized tool calls, executes each one,
// sends user-facing results to the bus, and returns the tool result messages
// to be appended to the conversation.
func (te *ToolExecutor) ExecuteToolCalls(
	ctx context.Context,
	agent *AgentInstance,
	toolCalls []providers.ToolCall,
	opts processOptions,
) []providers.Message {
	var resultMessages []providers.Message

	for _, tc := range toolCalls {
		argsJSON, _ := json.Marshal(tc.Arguments)
		argsPreview := utils.Truncate(string(argsJSON), 200)
		logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
			map[string]any{
				"agent_id":  agent.ID,
				"tool":      tc.Name,
			})

		// Create async callback for tools that implement AsyncTool
		// NOTE: Following openclaw's design, async tools do NOT send results directly to users.
		// Instead, they notify the agent via PublishInbound, and the agent decides
		// whether to forward the result to the user (in processSystemMessage).
		asyncCallback := func(callbackCtx context.Context, result *tools.ToolResult) {
			if !result.Silent && result.ForUser != "" {
				logger.InfoCF("agent", "Async tool completed, agent will handle notification",
					map[string]any{
						"tool":        tc.Name,
						"content_len": len(result.ForUser),
					})
			}
		}

		toolResult := agent.Tools.ExecuteWithContext(
			ctx,
			tc.Name,
			tc.Arguments,
			opts.Channel,
			opts.ChatID,
			asyncCallback,
		)

		// Send ForUser content to user immediately if not Silent
		if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
			te.bus.PublishOutbound(bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: toolResult.ForUser,
			})
			logger.DebugCF("agent", "Sent tool result to user",
				map[string]any{
					"tool":        tc.Name,
					"content_len": len(toolResult.ForUser),
				})
		}

		// Determine content for LLM based on tool result
		contentForLLM := toolResult.ForLLM
		if contentForLLM == "" && toolResult.Err != nil {
			contentForLLM = toolResult.Err.Error()
		}

		resultMessages = append(resultMessages, providers.Message{
			Role:       "tool",
			Content:    contentForLLM,
			ToolCallID: tc.ID,
		})
	}

	return resultMessages
}

// UpdateToolContexts updates the context for tools that need channel/chatID info.
func (te *ToolExecutor) UpdateToolContexts(agent *AgentInstance, channel, chatID string) {
	// Use ContextualTool interface instead of type assertions
	contextualToolNames := []string{"message", "spawn", "subagent"}
	for _, name := range contextualToolNames {
		if tool, ok := agent.Tools.Get(name); ok {
			if ct, ok := tool.(tools.ContextualTool); ok {
				ct.SetContext(channel, chatID)
			}
		}
	}
}
