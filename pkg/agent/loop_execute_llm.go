package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jane/pkg/bus"
	"jane/pkg/constants"
	"jane/pkg/logger"
	"jane/pkg/providers"
)

// executeLLMWithRetry handles calling the LLM, managing the fallback chain, and retrying
// on timeouts or context window limits.
func (al *AgentLoop) executeLLMWithRetry(
	ctx context.Context,
	agent *AgentInstance,
	opts processOptions,
	messages *[]providers.Message, // pointer to allow replacing messages slice on compression
	providerToolDefs []providers.ToolDefinition,
	activeCandidates []providers.FallbackCandidate,
	activeModel string,
	iteration int,
) (*providers.LLMResponse, error) {

	llmOpts := map[string]any{
		"max_tokens":       agent.MaxTokens,
		"temperature":      agent.Temperature,
		"prompt_cache_key": agent.ID,
	}

	if agent.ThinkingLevel != ThinkingOff {
		if tc, ok := agent.Provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
			llmOpts["thinking_level"] = string(agent.ThinkingLevel)
		} else {
			logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
				map[string]any{"agent_id": agent.ID, "thinking_level": string(agent.ThinkingLevel)})
		}
	}

	callLLM := func() (*providers.LLMResponse, error) {
		if len(activeCandidates) > 1 && al.fallback != nil {
			fbResult, fbErr := al.fallback.Execute(
				ctx,
				activeCandidates,
				func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
					return agent.Provider.Chat(ctx, *messages, providerToolDefs, model, llmOpts)
				},
			)
			if fbErr != nil {
				return nil, fbErr
			}
			if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
				logger.InfoCF(
					"agent",
					fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
						fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
					map[string]any{"agent_id": agent.ID, "iteration": iteration},
				)
			}
			return fbResult.Response, nil
		}
		return agent.Provider.Chat(ctx, *messages, providerToolDefs, activeModel, llmOpts)
	}

	var response *providers.LLMResponse
	var err error

	// Retry loop for context/token errors
	maxRetries := 2
	for retry := 0; retry <= maxRetries; retry++ {
		response, err = callLLM()
		if err == nil {
			break
		}

		isTimeoutError := false
		isContextError := false

		var failErr *providers.FailoverError
		if errors.As(err, &failErr) {
			if failErr.Reason == providers.FailoverTimeout {
				isTimeoutError = true
			} else if failErr.Reason == providers.FailoverContextLength {
				isContextError = true
			}
		} else {
			// If not a fallback error, check directly using ClassifyError
			// The provider might not be wrapped if no fallback chain is active
			if directFailErr := providers.ClassifyError(err, "", ""); directFailErr != nil {
				if directFailErr.Reason == providers.FailoverTimeout {
					isTimeoutError = true
				} else if directFailErr.Reason == providers.FailoverContextLength {
					isContextError = true
				}
			}
		}

		if isTimeoutError && retry < maxRetries {
			// Exponential backoff: 2s, 4s, 8s
			backoff := time.Duration(1<<(retry+1)) * time.Second
			logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
				"error":   err.Error(),
				"retry":   retry,
				"backoff": backoff.String(),
			})
			time.Sleep(backoff)
			continue
		}

		if isContextError && retry < maxRetries {
			logger.WarnCF(
				"agent",
				"Context window error detected, attempting compression",
				map[string]any{
					"error": err.Error(),
					"retry": retry,
				},
			)

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

			*messages = agent.ContextBuilder.BuildMessages(
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
				"agent_id":  agent.ID,
				"iteration": iteration,
				"error":     err.Error(),
			})
		return nil, fmt.Errorf("LLM call failed after retries: %w", err)
	}

	return response, nil
}
