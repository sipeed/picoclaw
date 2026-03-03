// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

// executor.go - Phase 2 (ExecuteLLM) logic extracted from loop.go.
// Contains the LLM iteration loop, tool handling, reasoning output,
// context compression, and logging helpers.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func (al *AgentLoop) targetReasoningChannelID(channelName string) (chatID string) {
	if al.channelManager == nil {
		return ""
	}
	if ch, ok := al.channelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

func (al *AgentLoop) handleReasoning(ctx context.Context, reasoningContent, channelName, channelID string) {
	if reasoningContent == "" || channelName == "" || channelID == "" {
		return
	}

	// Check context cancellation before attempting to publish,
	// since PublishOutbound's select may race between send and ctx.Done().
	if ctx.Err() != nil {
		return
	}

	// Use a short timeout so the goroutine does not block indefinitely when
	// the outbound bus is full.  Reasoning output is best-effort; dropping it
	// is acceptable to avoid goroutine accumulation.
	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()

	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: channelName,
		ChatID:  channelID,
		Content: reasoningContent,
	}); err != nil {
		// Treat context.DeadlineExceeded / context.Canceled as expected
		// (bus full under load, or parent canceled).  Check the error
		// itself rather than ctx.Err(), because pubCtx may time out
		// (5 s) while the parent ctx is still active.
		// Also treat ErrBusClosed as expected — it occurs during normal
		// shutdown when the bus is closed before all goroutines finish.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			errors.Is(err, bus.ErrBusClosed) {
			logger.DebugCF("agent", "Reasoning publish skipped (timeout/cancel)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish reasoning (best-effort)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		}
	}
}

// runLLMIteration executes the LLM call loop with tool handling.
func (al *AgentLoop) runLLMIteration(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts processOptions,
) (string, int, []ToolCallRecord, error) {
	iteration := 0
	var finalContent string
	var toolRecords []ToolCallRecord

	for iteration < agent.MaxIterations {
		iteration++

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"seq":       opts.MsgSeqId,
				"agent_id":  agent.ID,
				"iteration": iteration,
				"max":       agent.MaxIterations,
			})

		// Build tool definitions
		providerToolDefs := agent.Tools.ToProviderDefs()

		// Log LLM request details
		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"seq":               opts.MsgSeqId,
				"agent_id":          agent.ID,
				"iteration":         iteration,
				"model":             agent.Model,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        agent.MaxTokens,
				"temperature":       agent.Temperature,
				"system_prompt_len": len(messages[0].Content),
			})

		// Log full messages (detailed)
		logger.DebugCF("agent", "Full LLM request",
			map[string]any{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(messages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		// Call LLM with fallback chain if candidates are configured.
		var response *providers.LLMResponse
		var err error

		callLLM := func() (*providers.LLMResponse, error) {
			if len(agent.Candidates) > 1 && al.fallback != nil {
				fbResult, fbErr := al.fallback.Execute(ctx, agent.Candidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						return agent.Provider.Chat(ctx, messages, providerToolDefs, model, map[string]any{
							"max_tokens":       agent.MaxTokens,
							"temperature":      agent.Temperature,
							"prompt_cache_key": agent.ID,
						})
					},
				)
				if fbErr != nil {
					return nil, fbErr
				}
				if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
					logger.InfoCF("agent", fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
						fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
						map[string]any{"agent_id": agent.ID, "iteration": iteration})
				}
				return fbResult.Response, nil
			}
			return agent.Provider.Chat(ctx, messages, providerToolDefs, agent.Model, map[string]any{
				"max_tokens":       agent.MaxTokens,
				"temperature":      agent.Temperature,
				"prompt_cache_key": agent.ID,
			})
		}

		// Retry loop for context/token errors
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = callLLM()
			if err == nil {
				break
			}

			errMsg := strings.ToLower(err.Error())

			// Check if this is a network/HTTP timeout — not a context window error.
			isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(errMsg, "deadline exceeded") ||
				strings.Contains(errMsg, "client.timeout") ||
				strings.Contains(errMsg, "timed out") ||
				strings.Contains(errMsg, "timeout exceeded")

			// Detect real context window / token limit errors, excluding network timeouts.
			isContextError := !isTimeoutError && (strings.Contains(errMsg, "context_length_exceeded") ||
				strings.Contains(errMsg, "context window") ||
				strings.Contains(errMsg, "maximum context length") ||
				strings.Contains(errMsg, "token limit") ||
				strings.Contains(errMsg, "too many tokens") ||
				strings.Contains(errMsg, "max_tokens") ||
				strings.Contains(errMsg, "invalidparameter") ||
				strings.Contains(errMsg, "prompt is too long") ||
				strings.Contains(errMsg, "request too large"))

			if isTimeoutError && retry < maxRetries {
				backoff := time.Duration(retry+1) * 5 * time.Second
				logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
					"error":   err.Error(),
					"retry":   retry,
					"backoff": backoff.String(),
				})
				time.Sleep(backoff)
				continue
			}

			if isContextError && retry < maxRetries {
				logger.WarnCF("agent", "Context window error detected, attempting compression", map[string]any{
					"error": err.Error(),
					"retry": retry,
				})

				if retry == 0 && !constants.IsInternalChannel(opts.Channel) {
					al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: "Context window exceeded. Compressing history and retrying...",
					})
				}

				al.forceCompression(agent, opts.SessionKey)
				newHistory := agent.Sessions.GetHistory(opts.SessionKey)
				newSummary := agent.Sessions.GetSummary(opts.SessionKey)
				messages = agent.ContextBuilder.BuildMessages(
					newHistory, newSummary, "",
					nil, opts.Channel, opts.ChatID,
				)
				continue
			}
			break
		}

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]any{
					"seq":       opts.MsgSeqId,
					"agent_id":  agent.ID,
					"iteration": iteration,
					"error":     err.Error(),
				})
			return "", iteration, toolRecords, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		go al.handleReasoning(ctx, response.Reasoning, opts.Channel, al.targetReasoningChannelID(opts.Channel))

		logger.DebugCF("agent", "LLM response",
			map[string]any{
				"seq":            opts.MsgSeqId,
				"agent_id":       agent.ID,
				"iteration":      iteration,
				"content_chars":  len(response.Content),
				"tool_calls":     len(response.ToolCalls),
				"reasoning":      response.Reasoning,
				"target_channel": al.targetReasoningChannelID(opts.Channel),
				"channel":        opts.Channel,
			})
		// Check if no tool calls - we're done
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"seq":           opts.MsgSeqId,
					"agent_id":      agent.ID,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		normalizedToolCalls := make([]providers.ToolCall, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			normalizedToolCalls = append(normalizedToolCalls, providers.NormalizeToolCall(tc))
		}

		// Log tool calls
		toolNames := make([]string, 0, len(normalizedToolCalls))
		for _, tc := range normalizedToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]any{
				"agent_id":  agent.ID,
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		// Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:             "assistant",
			Content:          response.Content,
			ReasoningContent: response.ReasoningContent,
		}
		for _, tc := range normalizedToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			// Copy ExtraContent to ensure thought_signature is persisted for Gemini 3
			extraContent := tc.ExtraContent
			thoughtSignature := ""
			if tc.Function != nil {
				thoughtSignature = tc.Function.ThoughtSignature
			}

			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Name: tc.Name,
				Function: &providers.FunctionCall{
					Name:             tc.Name,
					Arguments:        string(argumentsJSON),
					ThoughtSignature: thoughtSignature,
				},
				ExtraContent:     extraContent,
				ThoughtSignature: thoughtSignature,
			})
		}
		messages = append(messages, assistantMsg)

		// Save assistant message with tool calls to session
		agent.Sessions.AddFullMessage(opts.SessionKey, assistantMsg)

		// Execute tool calls
		for _, tc := range normalizedToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]any{
					"agent_id":  agent.ID,
					"tool":      tc.Name,
					"iteration": iteration,
				})

			// Create async callback for tools that implement AsyncTool
			// NOTE: Following openclaw's design, async tools do NOT send results directly to users.
			// Instead, they notify the agent via PublishInbound, and the agent decides
			// whether to forward the result to the user (in processSystemMessage).
			asyncCallback := func(callbackCtx context.Context, result *tools.ToolResult) {
				// Log the async completion but don't send directly to user
				// The agent will handle user notification via processSystemMessage
				if !result.Silent && result.ForUser != "" {
					logger.InfoCF("agent", "Async tool completed, agent will handle notification",
						map[string]any{
							"tool":        tc.Name,
							"content_len": len(result.ForUser),
						})
				}
			}

			toolStart := time.Now()
			toolResult := agent.Tools.ExecuteWithContext(
				ctx,
				tc.Name,
				tc.Arguments,
				opts.Channel,
				opts.ChatID,
				asyncCallback,
			)
			toolDuration := time.Since(toolStart)

			// Record tool call for post-LLM processors.
			record := ToolCallRecord{Name: tc.Name, Duration: toolDuration}
			if toolResult.Err != nil {
				record.Error = toolResult.Err.Error()
			}
			toolRecords = append(toolRecords, record)

			// Send ForUser content to user immediately if not Silent
			if !toolResult.Silent && toolResult.ForUser != "" && opts.SendResponse {
				al.bus.PublishOutbound(ctx, bus.OutboundMessage{
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

			// If tool returned media refs, publish them as outbound media
			if len(toolResult.Media) > 0 && opts.SendResponse {
				parts := make([]bus.MediaPart, 0, len(toolResult.Media))
				for _, ref := range toolResult.Media {
					part := bus.MediaPart{Ref: ref}
					// Populate metadata from MediaStore when available
					if al.mediaStore != nil {
						if _, meta, err := al.mediaStore.ResolveWithMeta(ref); err == nil {
							part.Filename = meta.Filename
							part.ContentType = meta.ContentType
							part.Type = inferMediaType(meta.Filename, meta.ContentType)
						}
					}
					parts = append(parts, part)
				}
				al.bus.PublishOutboundMedia(ctx, bus.OutboundMediaMessage{
					Channel: opts.Channel,
					ChatID:  opts.ChatID,
					Parts:   parts,
				})
			}

			// Determine content for LLM based on tool result
			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)

			// Save tool result message to session
			agent.Sessions.AddFullMessage(opts.SessionKey, toolResultMsg)
		}
	}

	return finalContent, iteration, toolRecords, nil
}

// updateToolContexts updates the context for tools that need channel/chatID info.
func (al *AgentLoop) updateToolContexts(agent *AgentInstance, channel, chatID string) {
	// Use ContextualTool interface instead of type assertions
	if tool, ok := agent.Tools.Get("message"); ok {
		if mt, ok := tool.(tools.ContextualTool); ok {
			mt.SetContext(channel, chatID)
		}
	}
	if tool, ok := agent.Tools.Get("spawn"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
	if tool, ok := agent.Tools.Get("subagent"); ok {
		if st, ok := tool.(tools.ContextualTool); ok {
			st.SetContext(channel, chatID)
		}
	}
}

// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
	newHistory := agent.Sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := agent.ContextWindow * 75 / 100

	if len(newHistory) > 20 || tokenEstimate > threshold {
		summarizeKey := agent.ID + ":" + sessionKey
		if _, loading := al.summarizing.LoadOrStore(summarizeKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(summarizeKey)
				logger.Debug("Memory threshold reached. Optimizing conversation history...")
				al.summarizeSession(agent, sessionKey)
			}()
		}
	}
}

// forceCompression aggressively reduces context when the limit is hit.
// It drops the oldest 50% of messages (keeping system prompt and last user message).
func (al *AgentLoop) forceCompression(agent *AgentInstance, sessionKey string) {
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) <= 4 {
		return
	}

	// Keep system prompt (usually [0]) and the very last message (user's trigger)
	// We want to drop the oldest half of the *conversation*
	// Assuming [0] is system, [1:] is conversation
	conversation := history[1 : len(history)-1]
	if len(conversation) == 0 {
		return
	}

	// Helper to find the mid-point of the conversation
	mid := len(conversation) / 2

	// New history structure:
	// 1. System Prompt (with compression note appended)
	// 2. Second half of conversation
	// 3. Last message

	droppedCount := mid
	keptConversation := conversation[mid:]

	newHistory := make([]providers.Message, 0, 1+len(keptConversation)+1)

	// Append compression note to the original system prompt instead of adding a new system message
	// This avoids having two consecutive system messages which some APIs (like Zhipu) reject
	compressionNote := fmt.Sprintf(
		"\n\n[System Note: Emergency compression dropped %d oldest messages due to context limit]",
		droppedCount,
	)
	enhancedSystemPrompt := history[0]
	enhancedSystemPrompt.Content = enhancedSystemPrompt.Content + compressionNote
	newHistory = append(newHistory, enhancedSystemPrompt)

	newHistory = append(newHistory, keptConversation...)
	newHistory = append(newHistory, history[len(history)-1]) // Last message

	// Update session
	agent.Sessions.SetHistory(sessionKey, newHistory)
	agent.Sessions.Save(sessionKey)

	logger.WarnCF("agent", "Forced compression executed", map[string]any{
		"session_key":  sessionKey,
		"dropped_msgs": droppedCount,
		"new_count":    len(newHistory),
	})
}

// GetStartupInfo returns information about loaded tools and skills for logging.
func (al *AgentLoop) GetStartupInfo() map[string]any {
	info := make(map[string]any)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		return info
	}

	// Tools info
	toolsList := agent.Tools.List()
	info["tools"] = map[string]any{
		"count": len(toolsList),
		"names": toolsList,
	}

	// Skills info
	info["skills"] = agent.ContextBuilder.GetSkillsInfo()

	// Agents info
	info["agents"] = map[string]any{
		"count": len(al.registry.ListAgentIDs()),
		"ids":   al.registry.ListAgentIDs(),
	}

	return info
}

// formatMessagesForLog formats messages for logging
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, msg := range messages {
		fmt.Fprintf(&sb, "  [%d] Role: %s\n", i, msg.Role)
		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  ToolCalls:\n")
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&sb, "    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					fmt.Fprintf(&sb, "      Arguments: %s\n", utils.Truncate(tc.Function.Arguments, 200))
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			fmt.Fprintf(&sb, "  Content: %s\n", content)
		}
		if msg.ToolCallID != "" {
			fmt.Fprintf(&sb, "  ToolCallID: %s\n", msg.ToolCallID)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]")
	return sb.String()
}

// formatToolsForLog formats tool definitions for logging
func formatToolsForLog(toolDefs []providers.ToolDefinition) string {
	if len(toolDefs) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, tool := range toolDefs {
		fmt.Fprintf(&sb, "  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		fmt.Fprintf(&sb, "      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			fmt.Fprintf(&sb, "      Parameters: %s\n", utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200))
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// estimateTokens estimates the number of tokens in a message list.
// Uses a safe heuristic of 2.5 characters per token to account for CJK and other
// overheads better than the previous 3 chars/token.
func (al *AgentLoop) estimateTokens(messages []providers.Message) int {
	totalChars := 0
	for _, m := range messages {
		totalChars += utf8.RuneCountInString(m.Content)
	}
	// 2.5 chars per token = totalChars * 2 / 5
	return totalChars * 2 / 5
}
