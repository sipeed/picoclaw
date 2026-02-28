// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// ToolLoopConfig configures the tool execution loop.
type ToolLoopConfig struct {
	Provider             providers.LLMProvider
	Model                string
	Tools                *ToolRegistry
	MaxIterations        int
	LLMOptions           map[string]any
	RemainingTokenBudget *atomic.Int64
}

// ToolLoopResult contains the result of running the tool loop.
type ToolLoopResult struct {
	Content    string
	Iterations int
	Messages   []providers.Message // Allows caller to retain stateful context across executions
}

// RunToolLoop executes the LLM + tool call iteration loop.
// This is the core agent logic that can be reused by both main agent and subagents.
func RunToolLoop(
	ctx context.Context,
	config ToolLoopConfig,
	messages []providers.Message,
	channel, chatID string,
) (*ToolLoopResult, error) {
	iteration := 0
	var finalContent string

	for iteration < config.MaxIterations {
		iteration++

		logger.DebugCF("toolloop", "LLM iteration",
			map[string]any{
				"iteration": iteration,
				"max":       config.MaxIterations,
			})

		// 1. Build tool definitions
		var providerToolDefs []providers.ToolDefinition
		if config.Tools != nil {
			providerToolDefs = config.Tools.ToProviderDefs()
		}

		// 2. Set default LLM options
		llmOpts := config.LLMOptions
		if llmOpts == nil {
			llmOpts = map[string]any{}
		}
		// 3. Call LLM
		response, err := config.Provider.Chat(ctx, messages, providerToolDefs, config.Model, llmOpts)
		if err != nil {
			logger.ErrorCF("toolloop", "LLM call failed",
				map[string]any{
					"iteration": iteration,
					"error":     err.Error(),
				})
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// 3.5 Token Budget: Soft enforcement with graceful degradation.
		// Budget exhaustion is NOT a hard error — workers get a chance to wrap up gracefully.
		if response.Usage != nil && config.RemainingTokenBudget != nil {
			newBudget := config.RemainingTokenBudget.Add(-int64(response.Usage.TotalTokens))
			originalBudget := newBudget + int64(response.Usage.TotalTokens)

			if newBudget <= 0 {
				// Budget exhausted: signal the worker to wrap up and return partial result.
				logger.WarnCF("toolloop", "Token budget exhausted, injecting wrap-up signal",
					map[string]any{
						"deficit":   -newBudget,
						"iteration": iteration,
					})
				finalContent = response.Content
				messages = append(messages, providers.Message{
					Role:    "assistant",
					Content: response.Content,
				})
				messages = append(messages, providers.Message{
					Role:    "user",
					Content: "[SYSTEM] Token budget has been exhausted. Stop all tool calls immediately and return the best result you have completed so far. Do not call any more tools.",
				})
				// One final LLM call to get a summary/wrap-up from the model
				if finalResp, err := config.Provider.Chat(ctx, messages, nil, config.Model, config.LLMOptions); err == nil {
					finalContent = finalResp.Content
				}
				break
			} else if originalBudget > 0 && newBudget < originalBudget/2 {
				// Budget below 50%: soft warning injected into next iteration's context.
				logger.WarnCF("toolloop", "Token budget below 50%, injecting advisory",
					map[string]any{"remaining": newBudget, "iteration": iteration})
				messages = append(messages, providers.Message{
					Role:    "user",
					Content: "[SYSTEM] Advisory: token budget is running low. Please prioritize completing the most critical parts of your task and avoid unnecessary tool calls.",
				})
			}
		}

		// 3.6 Truncation Recovery: LLM response was cut off (max_tokens hit or malformed JSON).
		// Inject a recovery message so the LLM knows to retry with a shorter, complete response.
		if response.FinishReason == "truncated" {
			logger.WarnCF("toolloop", "LLM response was truncated (max_tokens hit), injecting recovery message",
				map[string]any{"iteration": iteration})
			messages = append(messages, providers.Message{
				Role:    "assistant",
				Content: response.Content,
			})
			messages = append(messages, providers.Message{
				Role:    "user",
				Content: "[SYSTEM] Your previous response was cut off because it exceeded the token limit. Please retry by producing a shorter, complete response. If you were about to call a tool, make sure the full JSON arguments are included without truncation.",
			})
			continue
		}

		// 4. If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			logger.InfoCF("toolloop", "LLM response without tool calls (direct answer)",
				map[string]any{
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		normalizedToolCalls := make([]providers.ToolCall, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			normalizedToolCalls = append(normalizedToolCalls, providers.NormalizeToolCall(tc))
		}

		// 5. Log tool calls
		toolNames := make([]string, 0, len(normalizedToolCalls))
		for _, tc := range normalizedToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("toolloop", "LLM requested tool calls",
			map[string]any{
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		// 6. Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
		}
		for _, tc := range normalizedToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:        tc.ID,
				Type:      "function",
				Name:      tc.Name,
				Arguments: tc.Arguments,
				Function: &providers.FunctionCall{
					Name:      tc.Name,
					Arguments: string(argumentsJSON),
				},
			})
		}
		messages = append(messages, assistantMsg)

		// 7. Execute tool calls
		for _, tc := range normalizedToolCalls {
			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("toolloop", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]any{
					"tool":      tc.Name,
					"iteration": iteration,
				})

			// Execute tool (no async callback for subagents - they run independently)
			var toolResult *ToolResult
			if config.Tools != nil {
				toolResult = config.Tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, channel, chatID, nil)
			} else {
				toolResult = ErrorResult("No tools available")
			}

			// Determine content for LLM
			contentForLLM := toolResult.ForLLM
			if contentForLLM == "" && toolResult.Err != nil {
				contentForLLM = toolResult.Err.Error()
			}

			// Add tool result message
			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)
		}
	}

	return &ToolLoopResult{
		Content:    finalContent,
		Iterations: iteration,
		Messages:   messages,
	}, nil
}
