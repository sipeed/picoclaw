// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func (al *AgentLoop) runTurn(ctx context.Context, ts *turnState) (turnResult, error) {
	turnCtx, turnCancel := context.WithCancel(ctx)
	defer turnCancel()
	ts.setTurnCancel(turnCancel)

	// Inject turnState and AgentLoop into context so tools (e.g. spawn) can retrieve them.
	turnCtx = withTurnState(turnCtx, ts)
	turnCtx = WithAgentLoop(turnCtx, al)

	al.registerActiveTurn(ts)
	defer al.clearActiveTurn(ts)

	turnStatus := TurnEndStatusCompleted
	defer func() {
		al.emitEvent(
			EventKindTurnEnd,
			ts.eventMeta("runTurn", "turn.end"),
			TurnEndPayload{
				Status:          turnStatus,
				Iterations:      ts.currentIteration(),
				Duration:        time.Since(ts.startedAt),
				FinalContentLen: ts.finalContentLen(),
			},
		)
	}()

	al.emitEvent(
		EventKindTurnStart,
		ts.eventMeta("runTurn", "turn.start"),
		TurnStartPayload{
			UserMessage: ts.userMessage,
			MediaCount:  len(ts.media),
		},
	)

	var history []providers.Message
	var summary string
	if !ts.opts.NoHistory {
		// ContextManager assembles budget-aware history and summary.
		if resp, err := al.contextManager.Assemble(turnCtx, &AssembleRequest{
			SessionKey: ts.sessionKey,
			Budget:     ts.agent.ContextWindow,
			MaxTokens:  ts.agent.MaxTokens,
		}); err == nil && resp != nil {
			history = resp.History
			summary = resp.Summary
		}
	}
	ts.captureRestorePoint(history, summary)

	messages := ts.agent.ContextBuilder.BuildMessages(
		history,
		summary,
		ts.userMessage,
		ts.media,
		ts.channel,
		ts.chatID,
		ts.opts.Dispatch.SenderID(),
		ts.opts.SenderDisplayName,
		activeSkillNames(ts.agent, ts.opts)...,
	)

	cfg := al.GetConfig()
	maxMediaSize := cfg.Agents.Defaults.GetMaxMediaSize()
	messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)

	if !ts.opts.NoHistory {
		toolDefs := ts.agent.Tools.ToProviderDefs()
		if isOverContextBudget(ts.agent.ContextWindow, messages, toolDefs, ts.agent.MaxTokens) {
			logger.WarnCF("agent", "Proactive compression: context budget exceeded before LLM call",
				map[string]any{"session_key": ts.sessionKey})
			if err := al.contextManager.Compact(turnCtx, &CompactRequest{
				SessionKey: ts.sessionKey,
				Reason:     ContextCompressReasonProactive,
				Budget:     ts.agent.ContextWindow,
			}); err != nil {
				logger.WarnCF("agent", "Proactive compact failed", map[string]any{
					"session_key": ts.sessionKey,
					"error":       err.Error(),
				})
			}
			ts.refreshRestorePointFromSession(ts.agent)
			// Re-assemble from CM after compact.
			if resp, err := al.contextManager.Assemble(turnCtx, &AssembleRequest{
				SessionKey: ts.sessionKey,
				Budget:     ts.agent.ContextWindow,
				MaxTokens:  ts.agent.MaxTokens,
			}); err == nil && resp != nil {
				history = resp.History
				summary = resp.Summary
			}
			messages = ts.agent.ContextBuilder.BuildMessages(
				history, summary, ts.userMessage,
				ts.media, ts.channel, ts.chatID,
				ts.opts.Dispatch.SenderID(), ts.opts.SenderDisplayName,
				activeSkillNames(ts.agent, ts.opts)...,
			)
			messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)
		}
	}

	// Save user message to session (from Incoming)
	if !ts.opts.NoHistory && (strings.TrimSpace(ts.userMessage) != "" || len(ts.media) > 0) {
		rootMsg := providers.Message{
			Role:    "user",
			Content: ts.userMessage,
			Media:   append([]string(nil), ts.media...),
		}
		if len(rootMsg.Media) > 0 {
			ts.agent.Sessions.AddFullMessage(ts.sessionKey, rootMsg)
		} else {
			ts.agent.Sessions.AddMessage(ts.sessionKey, rootMsg.Role, rootMsg.Content)
		}
		ts.recordPersistedMessage(rootMsg)
		ts.ingestMessage(turnCtx, al, rootMsg)
	}

	activeCandidates, activeModel, usedLight := al.selectCandidates(ts.agent, ts.userMessage, messages)
	activeProvider := ts.agent.Provider
	if usedLight && ts.agent.LightProvider != nil {
		activeProvider = ts.agent.LightProvider
	}
	pendingMessages := append([]providers.Message(nil), ts.opts.InitialSteeringMessages...)
	var finalContent string

turnLoop:
	for ts.currentIteration() < ts.agent.MaxIterations || len(pendingMessages) > 0 || func() bool {
		graceful, _ := ts.gracefulInterruptRequested()
		return graceful
	}() {
		if ts.hardAbortRequested() {
			turnStatus = TurnEndStatusAborted
			return al.abortTurn(ts)
		}

		iteration := ts.currentIteration() + 1
		ts.setIteration(iteration)
		ts.setPhase(TurnPhaseRunning)

		if iteration > 1 {
			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				pendingMessages = append(pendingMessages, steerMsgs...)
			}
		} else if !ts.opts.SkipInitialSteeringPoll {
			if steerMsgs := al.dequeueSteeringMessagesForScopeWithFallback(ts.sessionKey); len(steerMsgs) > 0 {
				pendingMessages = append(pendingMessages, steerMsgs...)
			}
		}

		// Check if parent turn has ended (SubTurn support from HEAD)
		if ts.parentTurnState != nil && ts.IsParentEnded() {
			if !ts.critical {
				logger.InfoCF("agent", "Parent turn ended, non-critical SubTurn exiting gracefully", map[string]any{
					"agent_id":  ts.agentID,
					"iteration": iteration,
					"turn_id":   ts.turnID,
				})
				break
			}
			logger.InfoCF("agent", "Parent turn ended, critical SubTurn continues running", map[string]any{
				"agent_id":  ts.agentID,
				"iteration": iteration,
				"turn_id":   ts.turnID,
			})
		}

		// Poll for pending SubTurn results (from HEAD)
		if ts.pendingResults != nil {
			select {
			case result, ok := <-ts.pendingResults:
				if ok && result != nil && result.ForLLM != "" {
					content := al.cfg.FilterSensitiveData(result.ForLLM)
					msg := providers.Message{Role: "user", Content: fmt.Sprintf("[SubTurn Result] %s", content)}
					pendingMessages = append(pendingMessages, msg)
				}
			default:
				// No results available
			}
		}

		// Inject pending steering messages
		if len(pendingMessages) > 0 {
			resolvedPending := resolveMediaRefs(pendingMessages, al.mediaStore, maxMediaSize)
			totalContentLen := 0
			for i, pm := range pendingMessages {
				messages = append(messages, resolvedPending[i])
				totalContentLen += len(pm.Content)
				if !ts.opts.NoHistory {
					ts.agent.Sessions.AddFullMessage(ts.sessionKey, pm)
					ts.recordPersistedMessage(pm)
					ts.ingestMessage(turnCtx, al, pm)
				}
				logger.InfoCF("agent", "Injected steering message into context",
					map[string]any{
						"agent_id":    ts.agent.ID,
						"iteration":   iteration,
						"content_len": len(pm.Content),
						"media_count": len(pm.Media),
					})
			}
			al.emitEvent(
				EventKindSteeringInjected,
				ts.eventMeta("runTurn", "turn.steering.injected"),
				SteeringInjectedPayload{
					Count:           len(pendingMessages),
					TotalContentLen: totalContentLen,
				},
			)
			pendingMessages = nil
		}

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  ts.agent.ID,
				"iteration": iteration,
				"max":       ts.agent.MaxIterations,
			})

		gracefulTerminal, _ := ts.gracefulInterruptRequested()
		providerToolDefs := ts.agent.Tools.ToProviderDefs()

		// Native web search support (from HEAD)
		_, hasWebSearch := ts.agent.Tools.Get("web_search")
		useNativeSearch := al.cfg.Tools.Web.PreferNative &&
			hasWebSearch &&
			func() bool {
				// Check if provider supports native search
				if ns, ok := ts.agent.Provider.(interface{ SupportsNativeSearch() bool }); ok {
					return ns.SupportsNativeSearch()
				}
				return false
			}()

		if useNativeSearch {
			// Filter out client-side web_search tool
			filtered := make([]providers.ToolDefinition, 0, len(providerToolDefs))
			for _, td := range providerToolDefs {
				if td.Function.Name != "web_search" {
					filtered = append(filtered, td)
				}
			}
			providerToolDefs = filtered
		}

		// Resolve media:// refs produced by tool results (e.g. load_image).
		// Skipped on iteration 1 because inbound user media is already resolved
		// before entering the loop; only subsequent iterations can contain new
		// tool-generated media refs that need base64 encoding.
		if iteration > 1 {
			messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)
		}

		callMessages := messages
		if gracefulTerminal {
			callMessages = append(append([]providers.Message(nil), messages...), ts.interruptHintMessage())
			providerToolDefs = nil
			ts.markGracefulTerminalUsed()
		}

		llmOpts := map[string]any{
			"max_tokens":       ts.agent.MaxTokens,
			"temperature":      ts.agent.Temperature,
			"prompt_cache_key": ts.agent.ID,
		}
		if useNativeSearch {
			llmOpts["native_search"] = true
		}
		if ts.agent.ThinkingLevel != ThinkingOff {
			if tc, ok := ts.agent.Provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
				llmOpts["thinking_level"] = string(ts.agent.ThinkingLevel)
			} else {
				logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
					map[string]any{"agent_id": ts.agent.ID, "thinking_level": string(ts.agent.ThinkingLevel)})
			}
		}

		llmModel := activeModel
		if al.hooks != nil {
			llmReq, decision := al.hooks.BeforeLLM(turnCtx, &LLMHookRequest{
				Meta:             ts.eventMeta("runTurn", "turn.llm.request"),
				Context:          cloneTurnContext(ts.turnCtx),
				Model:            llmModel,
				Messages:         callMessages,
				Tools:            providerToolDefs,
				Options:          llmOpts,
				GracefulTerminal: gracefulTerminal,
			})
			switch decision.normalizedAction() {
			case HookActionContinue, HookActionModify:
				if llmReq != nil {
					llmModel = llmReq.Model
					callMessages = llmReq.Messages
					providerToolDefs = llmReq.Tools
					llmOpts = llmReq.Options
				}
			case HookActionAbortTurn:
				turnStatus = TurnEndStatusError
				return turnResult{}, al.hookAbortError(ts, "before_llm", decision)
			case HookActionHardAbort:
				_ = ts.requestHardAbort()
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}
		}

		al.emitEvent(
			EventKindLLMRequest,
			ts.eventMeta("runTurn", "turn.llm.request"),
			LLMRequestPayload{
				Model:         llmModel,
				MessagesCount: len(callMessages),
				ToolsCount:    len(providerToolDefs),
				MaxTokens:     ts.agent.MaxTokens,
				Temperature:   ts.agent.Temperature,
			},
		)

		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"agent_id":          ts.agent.ID,
				"iteration":         iteration,
				"model":             llmModel,
				"messages_count":    len(callMessages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        ts.agent.MaxTokens,
				"temperature":       ts.agent.Temperature,
				"system_prompt_len": len(callMessages[0].Content),
			})
		logger.DebugCF("agent", "Full LLM request",
			map[string]any{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(callMessages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		callLLM := func(messagesForCall []providers.Message, toolDefsForCall []providers.ToolDefinition) (*providers.LLMResponse, error) {
			providerCtx, providerCancel := context.WithCancel(turnCtx)
			ts.setProviderCancel(providerCancel)
			defer func() {
				providerCancel()
				ts.clearProviderCancel(providerCancel)
			}()

			al.activeRequests.Add(1)
			defer al.activeRequests.Done()

			if len(activeCandidates) > 1 && al.fallback != nil {
				fbResult, fbErr := al.fallback.Execute(
					providerCtx,
					activeCandidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						candidateProvider := activeProvider
						if cp, ok := ts.agent.CandidateProviders[providers.ModelKey(provider, model)]; ok {
							candidateProvider = cp
						}
						return candidateProvider.Chat(ctx, messagesForCall, toolDefsForCall, model, llmOpts)
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
						map[string]any{"agent_id": ts.agent.ID, "iteration": iteration},
					)
				}
				return fbResult.Response, nil
			}
			return activeProvider.Chat(providerCtx, messagesForCall, toolDefsForCall, llmModel, llmOpts)
		}

		var response *providers.LLMResponse
		var err error
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = callLLM(callMessages, providerToolDefs)
			if err == nil {
				break
			}
			if ts.hardAbortRequested() && errors.Is(err, context.Canceled) {
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}

			// Retry without media if vision is unsupported
			if hasMediaRefs(callMessages) && isVisionUnsupportedError(err) && retry < maxRetries {
				al.emitEvent(
					EventKindLLMRetry,
					ts.eventMeta("runTurn", "turn.llm.retry"),
					LLMRetryPayload{
						Attempt:    retry + 1,
						MaxRetries: maxRetries,
						Reason:     "vision_unsupported",
						Error:      err.Error(),
						Backoff:    0,
					},
				)
				logger.WarnCF("agent", "Vision unsupported, retrying without media", map[string]any{
					"error": err.Error(),
					"retry": retry,
				})
				callMessages = stripMessageMedia(callMessages)
				// Also strip media from session history to prevent future errors
				if !ts.opts.NoHistory {
					history = stripMessageMedia(history)
					ts.agent.Sessions.SetHistory(ts.sessionKey, history)
					for i := range ts.persistedMessages {
						ts.persistedMessages[i].Media = nil
					}
					ts.refreshRestorePointFromSession(ts.agent)
				}
				continue
			}

			errMsg := strings.ToLower(err.Error())
			isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(errMsg, "deadline exceeded") ||
				strings.Contains(errMsg, "client.timeout") ||
				strings.Contains(errMsg, "timed out") ||
				strings.Contains(errMsg, "timeout exceeded")

			isContextError := !isTimeoutError && (strings.Contains(errMsg, "context_length_exceeded") ||
				strings.Contains(errMsg, "context window") ||
				strings.Contains(errMsg, "context_window") ||
				strings.Contains(errMsg, "maximum context length") ||
				strings.Contains(errMsg, "token limit") ||
				strings.Contains(errMsg, "too many tokens") ||
				strings.Contains(errMsg, "max_tokens") ||
				strings.Contains(errMsg, "invalidparameter") ||
				strings.Contains(errMsg, "prompt is too long") ||
				strings.Contains(errMsg, "request too large"))

			if isTimeoutError && retry < maxRetries {
				backoff := time.Duration(retry+1) * 5 * time.Second
				al.emitEvent(
					EventKindLLMRetry,
					ts.eventMeta("runTurn", "turn.llm.retry"),
					LLMRetryPayload{
						Attempt:    retry + 1,
						MaxRetries: maxRetries,
						Reason:     "timeout",
						Error:      err.Error(),
						Backoff:    backoff,
					},
				)
				logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
					"error":   err.Error(),
					"retry":   retry,
					"backoff": backoff.String(),
				})
				if sleepErr := sleepWithContext(turnCtx, backoff); sleepErr != nil {
					if ts.hardAbortRequested() {
						turnStatus = TurnEndStatusAborted
						return al.abortTurn(ts)
					}
					err = sleepErr
					break
				}
				continue
			}

			if isContextError && retry < maxRetries && !ts.opts.NoHistory {
				al.emitEvent(
					EventKindLLMRetry,
					ts.eventMeta("runTurn", "turn.llm.retry"),
					LLMRetryPayload{
						Attempt:    retry + 1,
						MaxRetries: maxRetries,
						Reason:     "context_limit",
						Error:      err.Error(),
					},
				)
				logger.WarnCF(
					"agent",
					"Context window error detected, attempting compression",
					map[string]any{
						"error": err.Error(),
						"retry": retry,
					},
				)

				if retry == 0 && !constants.IsInternalChannel(ts.channel) {
					al.bus.PublishOutbound(ctx, outboundMessageForTurn(
						ts,
						"Context window exceeded. Compressing history and retrying...",
					))
				}

				if compactErr := al.contextManager.Compact(turnCtx, &CompactRequest{
					SessionKey: ts.sessionKey,
					Reason:     ContextCompressReasonRetry,
					Budget:     ts.agent.ContextWindow,
				}); compactErr != nil {
					logger.WarnCF("agent", "Context overflow compact failed", map[string]any{
						"session_key": ts.sessionKey,
						"error":       compactErr.Error(),
					})
				}
				ts.refreshRestorePointFromSession(ts.agent)
				// Re-assemble from CM after compact.
				if asmResp, asmErr := al.contextManager.Assemble(turnCtx, &AssembleRequest{
					SessionKey: ts.sessionKey,
					Budget:     ts.agent.ContextWindow,
					MaxTokens:  ts.agent.MaxTokens,
				}); asmErr == nil && asmResp != nil {
					history = asmResp.History
					summary = asmResp.Summary
				}
				messages = ts.agent.ContextBuilder.BuildMessages(
					history, summary, "",
					nil, ts.channel, ts.chatID, ts.opts.Dispatch.SenderID(), ts.opts.SenderDisplayName,
					activeSkillNames(ts.agent, ts.opts)...,
				)
				callMessages = messages
				if gracefulTerminal {
					callMessages = append(append([]providers.Message(nil), messages...), ts.interruptHintMessage())
				}
				continue
			}
			break
		}

		if err != nil {
			turnStatus = TurnEndStatusError
			al.emitEvent(
				EventKindError,
				ts.eventMeta("runTurn", "turn.error"),
				ErrorPayload{
					Stage:   "llm",
					Message: err.Error(),
				},
			)
			logger.ErrorCF("agent", "LLM call failed",
				map[string]any{
					"agent_id":  ts.agent.ID,
					"iteration": iteration,
					"model":     llmModel,
					"error":     err.Error(),
				})
			return turnResult{}, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		if al.hooks != nil {
			llmResp, decision := al.hooks.AfterLLM(turnCtx, &LLMHookResponse{
				Meta:     ts.eventMeta("runTurn", "turn.llm.response"),
				Context:  cloneTurnContext(ts.turnCtx),
				Model:    llmModel,
				Response: response,
			})
			switch decision.normalizedAction() {
			case HookActionContinue, HookActionModify:
				if llmResp != nil && llmResp.Response != nil {
					response = llmResp.Response
				}
			case HookActionAbortTurn:
				turnStatus = TurnEndStatusError
				return turnResult{}, al.hookAbortError(ts, "after_llm", decision)
			case HookActionHardAbort:
				_ = ts.requestHardAbort()
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}
		}

		// Save finishReason to turnState for SubTurn truncation detection
		if innerTS := turnStateFromContext(ctx); innerTS != nil {
			innerTS.SetLastFinishReason(response.FinishReason)
			// Save usage for token budget tracking
			if response.Usage != nil {
				innerTS.SetLastUsage(response.Usage)
			}
		}

		reasoningContent := response.Reasoning
		if reasoningContent == "" {
			reasoningContent = response.ReasoningContent
		}
		if ts.channel == "pico" {
			go al.publishPicoReasoning(turnCtx, reasoningContent, ts.chatID)
		} else {
			go al.handleReasoning(
				turnCtx,
				reasoningContent,
				ts.channel,
				al.targetReasoningChannelID(ts.channel),
			)
		}
		al.emitEvent(
			EventKindLLMResponse,
			ts.eventMeta("runTurn", "turn.llm.response"),
			LLMResponsePayload{
				ContentLen:   len(response.Content),
				ToolCalls:    len(response.ToolCalls),
				HasReasoning: response.Reasoning != "" || response.ReasoningContent != "",
			},
		)

		llmResponseFields := map[string]any{
			"agent_id":       ts.agent.ID,
			"iteration":      iteration,
			"content_chars":  len(response.Content),
			"tool_calls":     len(response.ToolCalls),
			"reasoning":      response.Reasoning,
			"target_channel": al.targetReasoningChannelID(ts.channel),
			"channel":        ts.channel,
		}
		if response.Usage != nil {
			llmResponseFields["prompt_tokens"] = response.Usage.PromptTokens
			llmResponseFields["completion_tokens"] = response.Usage.CompletionTokens
			llmResponseFields["total_tokens"] = response.Usage.TotalTokens
		}
		logger.DebugCF("agent", "LLM response", llmResponseFields)

		if al.bus != nil && ts.channel == "pico" && len(response.ToolCalls) > 0 && ts.opts.AllowInterimPicoPublish {
			if strings.TrimSpace(response.Content) != "" {
				outCtx, outCancel := context.WithTimeout(turnCtx, 3*time.Second)
				err := al.bus.PublishOutbound(outCtx, bus.OutboundMessage{
					Channel: ts.channel,
					ChatID:  ts.chatID,
					Content: response.Content,
				})
				outCancel()
				if err != nil {
					logger.WarnCF("agent", "Failed to publish pico interim tool-call content", map[string]any{
						"error":     err.Error(),
						"channel":   ts.channel,
						"chat_id":   ts.chatID,
						"iteration": iteration,
					})
				}
			}
		}

		if len(response.ToolCalls) == 0 || gracefulTerminal {
			responseContent := response.Content
			if responseContent == "" && response.ReasoningContent != "" && ts.channel != "pico" {
				responseContent = response.ReasoningContent
			}
			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				logger.InfoCF("agent", "Steering arrived after direct LLM response; continuing turn",
					map[string]any{
						"agent_id":       ts.agent.ID,
						"iteration":      iteration,
						"steering_count": len(steerMsgs),
					})
				pendingMessages = append(pendingMessages, steerMsgs...)
				continue
			}
			finalContent = responseContent
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"agent_id":      ts.agent.ID,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		normalizedToolCalls := make([]providers.ToolCall, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			normalizedToolCalls = append(normalizedToolCalls, providers.NormalizeToolCall(tc))
		}

		toolNames := make([]string, 0, len(normalizedToolCalls))
		for _, tc := range normalizedToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]any{
				"agent_id":  ts.agent.ID,
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		allResponsesHandled := len(normalizedToolCalls) > 0
		assistantMsg := providers.Message{
			Role:             "assistant",
			Content:          response.Content,
			ReasoningContent: response.ReasoningContent,
		}
		for _, tc := range normalizedToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
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
		if !ts.opts.NoHistory {
			ts.agent.Sessions.AddFullMessage(ts.sessionKey, assistantMsg)
			ts.recordPersistedMessage(assistantMsg)
			ts.ingestMessage(turnCtx, al, assistantMsg)
		}

		ts.setPhase(TurnPhaseTools)
		for i, tc := range normalizedToolCalls {
			if ts.hardAbortRequested() {
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}

			toolName := tc.Name
			toolArgs := cloneStringAnyMap(tc.Arguments)

			if al.hooks != nil {
				toolReq, decision := al.hooks.BeforeTool(turnCtx, &ToolCallHookRequest{
					Meta:      ts.eventMeta("runTurn", "turn.tool.before"),
					Context:   cloneTurnContext(ts.turnCtx),
					Tool:      toolName,
					Arguments: toolArgs,
				})
				switch decision.normalizedAction() {
				case HookActionContinue, HookActionModify:
					if toolReq != nil {
						toolName = toolReq.Tool
						toolArgs = toolReq.Arguments
					}
				case HookActionRespond:
					// Hook returns result directly, skip tool execution.
					// SECURITY: This bypasses ApproveTool, allowing hooks to respond
					// for any tool name without approval. This is intentional for
					// plugin tools but means a before_tool hook can override even
					// sensitive tools like bash. Hook configuration should be
					// carefully reviewed to prevent unauthorized tool execution.
					if toolReq != nil && toolReq.HookResult != nil {
						hookResult := toolReq.HookResult

						argsJSON, _ := json.Marshal(toolArgs)
						argsPreview := utils.Truncate(string(argsJSON), 200)
						logger.InfoCF("agent", fmt.Sprintf("Tool call (hook respond): %s(%s)", toolName, argsPreview),
							map[string]any{
								"agent_id":  ts.agent.ID,
								"tool":      toolName,
								"iteration": iteration,
							})

						// Emit ToolExecStart event (same as normal tool execution)
						al.emitEvent(
							EventKindToolExecStart,
							ts.eventMeta("runTurn", "turn.tool.start"),
							ToolExecStartPayload{
								Tool:      toolName,
								Arguments: cloneEventArguments(toolArgs),
							},
						)

						// Send tool feedback to chat channel if enabled (same as normal tool execution)
						if al.cfg.Agents.Defaults.IsToolFeedbackEnabled() &&
							ts.channel != "" &&
							!ts.opts.SuppressToolFeedback {
							argsJSON, _ := json.Marshal(toolArgs)
							feedbackPreview := utils.Truncate(
								string(argsJSON),
								al.cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength(),
							)
							feedbackMsg := utils.FormatToolFeedbackMessage(toolName, feedbackPreview)
							fbCtx, fbCancel := context.WithTimeout(turnCtx, 3*time.Second)
							_ = al.bus.PublishOutbound(fbCtx, bus.OutboundMessage{
								Channel: ts.channel,
								ChatID:  ts.chatID,
								Content: feedbackMsg,
							})
							fbCancel()
						}

						toolDuration := time.Duration(0) // Hook execution time unknown

						// Send ForUser content to user
						// For ResponseHandled results, send regardless of SendResponse setting,
						// same as normal tool execution path.
						shouldSendForUser := !hookResult.Silent && hookResult.ForUser != "" &&
							(ts.opts.SendResponse || hookResult.ResponseHandled)
						if shouldSendForUser {
							al.bus.PublishOutbound(ctx, bus.OutboundMessage{
								Context: bus.InboundContext{
									Channel: ts.channel,
									ChatID:  ts.chatID,
									Raw: map[string]string{
										"is_tool_call": "true",
									},
								},
								Content: hookResult.ForUser,
							})
						}

						// Handle media from hook result (same as normal tool execution)
						if len(hookResult.Media) > 0 && hookResult.ResponseHandled {
							parts := make([]bus.MediaPart, 0, len(hookResult.Media))
							for _, ref := range hookResult.Media {
								part := bus.MediaPart{Ref: ref}
								if al.mediaStore != nil {
									if _, meta, err := al.mediaStore.ResolveWithMeta(ref); err == nil {
										part.Filename = meta.Filename
										part.ContentType = meta.ContentType
										part.Type = inferMediaType(meta.Filename, meta.ContentType)
									}
								}
								parts = append(parts, part)
							}
							outboundMedia := bus.OutboundMediaMessage{
								Channel: ts.channel,
								ChatID:  ts.chatID,
								Parts:   parts,
							}
							if al.channelManager != nil && ts.channel != "" && !constants.IsInternalChannel(ts.channel) {
								if err := al.channelManager.SendMedia(ctx, outboundMedia); err != nil {
									logger.WarnCF("agent", "Failed to deliver hook media",
										map[string]any{
											"agent_id": ts.agent.ID,
											"tool":     toolName,
											"channel":  ts.channel,
											"chat_id":  ts.chatID,
											"error":    err.Error(),
										})
									// Same as normal tool execution: notify LLM about delivery failure
									hookResult.IsError = true
									hookResult.ForLLM = fmt.Sprintf("failed to deliver attachment: %v", err)
								}
							} else if al.bus != nil {
								al.bus.PublishOutboundMedia(ctx, outboundMedia)
								// Same as normal tool execution: bus only queues, media not yet delivered
								hookResult.ResponseHandled = false
							}
						}

						// Track response handling status (same as normal tool execution)
						if !hookResult.ResponseHandled {
							allResponsesHandled = false
						}

						// Build tool message
						contentForLLM := hookResult.ContentForLLM()
						if al.cfg.Tools.IsFilterSensitiveDataEnabled() {
							contentForLLM = al.cfg.FilterSensitiveData(contentForLLM)
						}

						toolResultMsg := providers.Message{
							Role:       "tool",
							Content:    contentForLLM,
							ToolCallID: tc.ID,
						}

						// Handle media for LLM vision (same as normal tool execution)
						if len(hookResult.Media) > 0 && !hookResult.ResponseHandled {
							hookResult.ArtifactTags = buildArtifactTags(al.mediaStore, hookResult.Media)
							// Recalculate contentForLLM after adding ArtifactTags
							contentForLLM = hookResult.ContentForLLM()
							if al.cfg.Tools.IsFilterSensitiveDataEnabled() {
								contentForLLM = al.cfg.FilterSensitiveData(contentForLLM)
							}
							toolResultMsg.Content = contentForLLM
							toolResultMsg.Media = append(toolResultMsg.Media, hookResult.Media...)
						}

						// Emit ToolExecEnd event (after filtering, same as normal tool execution)
						al.emitEvent(
							EventKindToolExecEnd,
							ts.eventMeta("runTurn", "turn.tool.end"),
							ToolExecEndPayload{
								Tool:       toolName,
								Duration:   toolDuration,
								ForLLMLen:  len(contentForLLM),
								ForUserLen: len(hookResult.ForUser),
								IsError:    hookResult.IsError,
								Async:      hookResult.Async,
							},
						)

						messages = append(messages, toolResultMsg)
						if !ts.opts.NoHistory {
							ts.agent.Sessions.AddFullMessage(ts.sessionKey, toolResultMsg)
							ts.recordPersistedMessage(toolResultMsg)
							ts.ingestMessage(turnCtx, al, toolResultMsg)
						}

						// Same as normal tool execution: check for steering/interrupt/SubTurn after each tool
						if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
							pendingMessages = append(pendingMessages, steerMsgs...)
						}

						skipReason := ""
						skipMessage := ""
						if len(pendingMessages) > 0 {
							skipReason = "queued user steering message"
							skipMessage = "Skipped due to queued user message."
						} else if gracefulPending, _ := ts.gracefulInterruptRequested(); gracefulPending {
							skipReason = "graceful interrupt requested"
							skipMessage = "Skipped due to graceful interrupt."
						}

						if skipReason != "" {
							remaining := len(normalizedToolCalls) - i - 1
							if remaining > 0 {
								logger.InfoCF("agent", "Turn checkpoint: skipping remaining tools after hook respond",
									map[string]any{
										"agent_id":  ts.agent.ID,
										"completed": i + 1,
										"skipped":   remaining,
										"reason":    skipReason,
									})
								for j := i + 1; j < len(normalizedToolCalls); j++ {
									skippedTC := normalizedToolCalls[j]
									al.emitEvent(
										EventKindToolExecSkipped,
										ts.eventMeta("runTurn", "turn.tool.skipped"),
										ToolExecSkippedPayload{
											Tool:   skippedTC.Name,
											Reason: skipReason,
										},
									)
									skippedMsg := providers.Message{
										Role:       "tool",
										Content:    skipMessage,
										ToolCallID: skippedTC.ID,
									}
									messages = append(messages, skippedMsg)
									if !ts.opts.NoHistory {
										ts.agent.Sessions.AddFullMessage(ts.sessionKey, skippedMsg)
										ts.recordPersistedMessage(skippedMsg)
									}
								}
							}
							break
						}

						// Also poll for any SubTurn results that arrived during tool execution.
						if ts.pendingResults != nil {
							select {
							case result, ok := <-ts.pendingResults:
								if ok && result != nil && result.ForLLM != "" {
									content := al.cfg.FilterSensitiveData(result.ForLLM)
									msg := providers.Message{Role: "user", Content: fmt.Sprintf("[SubTurn Result] %s", content)}
									messages = append(messages, msg)
									ts.agent.Sessions.AddFullMessage(ts.sessionKey, msg)
								}
							default:
								// No results available
							}
						}

						continue
					}
					// If no HookResult, fall back to continue with warning
					logger.WarnCF("agent", "Hook returned respond action but no HookResult provided",
						map[string]any{
							"agent_id": ts.agent.ID,
							"tool":     toolName,
							"action":   "respond",
						})
				case HookActionDenyTool:
					allResponsesHandled = false
					denyContent := hookDeniedToolContent("Tool execution denied by hook", decision.Reason)
					al.emitEvent(
						EventKindToolExecSkipped,
						ts.eventMeta("runTurn", "turn.tool.skipped"),
						ToolExecSkippedPayload{
							Tool:   toolName,
							Reason: denyContent,
						},
					)
					deniedMsg := providers.Message{
						Role:       "tool",
						Content:    denyContent,
						ToolCallID: tc.ID,
					}
					messages = append(messages, deniedMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, deniedMsg)
						ts.recordPersistedMessage(deniedMsg)
					}
					continue
				case HookActionAbortTurn:
					turnStatus = TurnEndStatusError
					return turnResult{}, al.hookAbortError(ts, "before_tool", decision)
				case HookActionHardAbort:
					_ = ts.requestHardAbort()
					turnStatus = TurnEndStatusAborted
					return al.abortTurn(ts)
				}
			}

			if al.hooks != nil {
				approval := al.hooks.ApproveTool(turnCtx, &ToolApprovalRequest{
					Meta:      ts.eventMeta("runTurn", "turn.tool.approve"),
					Context:   cloneTurnContext(ts.turnCtx),
					Tool:      toolName,
					Arguments: toolArgs,
				})
				if !approval.Approved {
					allResponsesHandled = false
					denyContent := hookDeniedToolContent("Tool execution denied by approval hook", approval.Reason)
					al.emitEvent(
						EventKindToolExecSkipped,
						ts.eventMeta("runTurn", "turn.tool.skipped"),
						ToolExecSkippedPayload{
							Tool:   toolName,
							Reason: denyContent,
						},
					)
					deniedMsg := providers.Message{
						Role:       "tool",
						Content:    denyContent,
						ToolCallID: tc.ID,
					}
					messages = append(messages, deniedMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, deniedMsg)
						ts.recordPersistedMessage(deniedMsg)
					}
					continue
				}
			}

			argsJSON, _ := json.Marshal(toolArgs)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", toolName, argsPreview),
				map[string]any{
					"agent_id":  ts.agent.ID,
					"tool":      toolName,
					"iteration": iteration,
				})
			al.emitEvent(
				EventKindToolExecStart,
				ts.eventMeta("runTurn", "turn.tool.start"),
				ToolExecStartPayload{
					Tool:      toolName,
					Arguments: cloneEventArguments(toolArgs),
				},
			)

			// Send tool feedback to chat channel if enabled (from HEAD)
			if al.cfg.Agents.Defaults.IsToolFeedbackEnabled() &&
				ts.channel != "" &&
				!ts.opts.SuppressToolFeedback {
				feedbackPreview := utils.Truncate(
					string(argsJSON),
					al.cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength(),
				)
				feedbackMsg := utils.FormatToolFeedbackMessage(tc.Name, feedbackPreview)
				fbCtx, fbCancel := context.WithTimeout(turnCtx, 3*time.Second)
				_ = al.bus.PublishOutbound(fbCtx, outboundMessageForTurn(ts, feedbackMsg))
				fbCancel()
			}

			toolCallID := tc.ID
			toolIteration := iteration
			asyncToolName := toolName
			asyncCallback := func(_ context.Context, result *tools.ToolResult) {
				// Send ForUser content directly to the user (immediate feedback),
				// mirroring the synchronous tool execution path.
				if !result.Silent && result.ForUser != "" {
					outCtx, outCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer outCancel()
					_ = al.bus.PublishOutbound(outCtx, outboundMessageForTurn(ts, result.ForUser))
				}

				// Determine content for the agent loop (ForLLM or error).
				content := result.ContentForLLM()
				if content == "" {
					return
				}

				// Filter sensitive data before publishing
				content = al.cfg.FilterSensitiveData(content)

				logger.InfoCF("agent", "Async tool completed, publishing result",
					map[string]any{
						"tool":        asyncToolName,
						"content_len": len(content),
						"channel":     ts.channel,
					})
				al.emitEvent(
					EventKindFollowUpQueued,
					ts.scope.meta(toolIteration, "runTurn", "turn.follow_up.queued"),
					FollowUpQueuedPayload{
						SourceTool: asyncToolName,
						ContentLen: len(content),
					},
				)

				pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer pubCancel()
				_ = al.bus.PublishInbound(pubCtx, bus.InboundMessage{
					Context: bus.InboundContext{
						Channel:  "system",
						ChatID:   fmt.Sprintf("%s:%s", ts.channel, ts.chatID),
						ChatType: "direct",
						SenderID: fmt.Sprintf("async:%s", asyncToolName),
					},
					Content: content,
				})
			}

			toolStart := time.Now()
			execCtx := tools.WithToolInboundContext(
				turnCtx,
				ts.channel,
				ts.chatID,
				ts.opts.Dispatch.MessageID(),
				ts.opts.Dispatch.ReplyToMessageID(),
			)
			execCtx = tools.WithToolSessionContext(
				execCtx,
				ts.agent.ID,
				ts.sessionKey,
				ts.opts.Dispatch.SessionScope,
			)
			toolResult := ts.agent.Tools.ExecuteWithContext(
				execCtx,
				toolName,
				toolArgs,
				ts.channel,
				ts.chatID,
				asyncCallback,
			)
			toolDuration := time.Since(toolStart)

			if ts.hardAbortRequested() {
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}

			if al.hooks != nil {
				toolResp, decision := al.hooks.AfterTool(turnCtx, &ToolResultHookResponse{
					Meta:      ts.eventMeta("runTurn", "turn.tool.after"),
					Context:   cloneTurnContext(ts.turnCtx),
					Tool:      toolName,
					Arguments: toolArgs,
					Result:    toolResult,
					Duration:  toolDuration,
				})
				switch decision.normalizedAction() {
				case HookActionContinue, HookActionModify:
					if toolResp != nil {
						if toolResp.Tool != "" {
							toolName = toolResp.Tool
						}
						if toolResp.Result != nil {
							toolResult = toolResp.Result
						}
					}
				case HookActionAbortTurn:
					turnStatus = TurnEndStatusError
					return turnResult{}, al.hookAbortError(ts, "after_tool", decision)
				case HookActionHardAbort:
					_ = ts.requestHardAbort()
					turnStatus = TurnEndStatusAborted
					return al.abortTurn(ts)
				}
			}

			if toolResult == nil {
				toolResult = tools.ErrorResult("hook returned nil tool result")
			}

			if len(toolResult.Media) > 0 && toolResult.ResponseHandled {
				parts := make([]bus.MediaPart, 0, len(toolResult.Media))
				for _, ref := range toolResult.Media {
					part := bus.MediaPart{Ref: ref}
					if al.mediaStore != nil {
						if _, meta, err := al.mediaStore.ResolveWithMeta(ref); err == nil {
							part.Filename = meta.Filename
							part.ContentType = meta.ContentType
							part.Type = inferMediaType(meta.Filename, meta.ContentType)
						}
					}
					parts = append(parts, part)
				}
				outboundMedia := bus.OutboundMediaMessage{
					Channel: ts.channel,
					ChatID:  ts.chatID,
					Context: outboundContextFromInbound(
						ts.opts.Dispatch.InboundContext,
						ts.channel,
						ts.chatID,
						ts.opts.Dispatch.ReplyToMessageID(),
					),
					AgentID:    ts.agent.ID,
					SessionKey: ts.sessionKey,
					Scope:      outboundScopeFromSessionScope(ts.opts.Dispatch.SessionScope),
					Parts:      parts,
				}
				if al.channelManager != nil && ts.channel != "" && !constants.IsInternalChannel(ts.channel) {
					if err := al.channelManager.SendMedia(ctx, outboundMedia); err != nil {
						logger.WarnCF("agent", "Failed to deliver handled tool media",
							map[string]any{
								"agent_id": ts.agent.ID,
								"tool":     toolName,
								"channel":  ts.channel,
								"chat_id":  ts.chatID,
								"error":    err.Error(),
							})
						toolResult = tools.ErrorResult(fmt.Sprintf("failed to deliver attachment: %v", err)).WithError(err)
					}
				} else if al.bus != nil {
					al.bus.PublishOutboundMedia(ctx, outboundMedia)
					// Queuing media is only best-effort; it has not been delivered yet.
					toolResult.ResponseHandled = false
				}
			}

			if len(toolResult.Media) > 0 && !toolResult.ResponseHandled {
				// For tools like load_image that produce media refs without sending them
				// to the user channel (ResponseHandled == false), both Media and ArtifactTags
				// coexist on the result:
				//   - Media: carries media:// refs that resolveMediaRefs will base64-encode
				//     into image_url parts in the next LLM iteration (enabling vision).
				//   - ArtifactTags: exposes the local file path as a structured [file:…] tag
				//     in the tool result text, so the LLM knows an artifact was produced.
				toolResult.ArtifactTags = buildArtifactTags(al.mediaStore, toolResult.Media)
			}

			if !toolResult.ResponseHandled {
				allResponsesHandled = false
			}

			shouldSendForUser := !toolResult.Silent &&
				toolResult.ForUser != "" &&
				(ts.opts.SendResponse || toolResult.ResponseHandled)
			if shouldSendForUser {
				al.bus.PublishOutbound(ctx, outboundMessageForTurn(ts, toolResult.ForUser))
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]any{
						"tool":        toolName,
						"content_len": len(toolResult.ForUser),
					})
			}
			contentForLLM := toolResult.ContentForLLM()

			// Filter sensitive data (API keys, tokens, secrets) before sending to LLM
			if al.cfg.Tools.IsFilterSensitiveDataEnabled() {
				contentForLLM = al.cfg.FilterSensitiveData(contentForLLM)
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: toolCallID,
			}
			if len(toolResult.Media) > 0 && !toolResult.ResponseHandled {
				toolResultMsg.Media = append(toolResultMsg.Media, toolResult.Media...)
			}
			al.emitEvent(
				EventKindToolExecEnd,
				ts.eventMeta("runTurn", "turn.tool.end"),
				ToolExecEndPayload{
					Tool:       toolName,
					Duration:   toolDuration,
					ForLLMLen:  len(contentForLLM),
					ForUserLen: len(toolResult.ForUser),
					IsError:    toolResult.IsError,
					Async:      toolResult.Async,
				},
			)
			messages = append(messages, toolResultMsg)
			if !ts.opts.NoHistory {
				ts.agent.Sessions.AddFullMessage(ts.sessionKey, toolResultMsg)
				ts.recordPersistedMessage(toolResultMsg)
				ts.ingestMessage(turnCtx, al, toolResultMsg)
			}

			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				pendingMessages = append(pendingMessages, steerMsgs...)
			}

			skipReason := ""
			skipMessage := ""
			if len(pendingMessages) > 0 {
				skipReason = "queued user steering message"
				skipMessage = "Skipped due to queued user message."
			} else if gracefulPending, _ := ts.gracefulInterruptRequested(); gracefulPending {
				skipReason = "graceful interrupt requested"
				skipMessage = "Skipped due to graceful interrupt."
			}

			if skipReason != "" {
				remaining := len(normalizedToolCalls) - i - 1
				if remaining > 0 {
					logger.InfoCF("agent", "Turn checkpoint: skipping remaining tools",
						map[string]any{
							"agent_id":  ts.agent.ID,
							"completed": i + 1,
							"skipped":   remaining,
							"reason":    skipReason,
						})
					for j := i + 1; j < len(normalizedToolCalls); j++ {
						skippedTC := normalizedToolCalls[j]
						al.emitEvent(
							EventKindToolExecSkipped,
							ts.eventMeta("runTurn", "turn.tool.skipped"),
							ToolExecSkippedPayload{
								Tool:   skippedTC.Name,
								Reason: skipReason,
							},
						)
						skippedMsg := providers.Message{
							Role:       "tool",
							Content:    skipMessage,
							ToolCallID: skippedTC.ID,
						}
						messages = append(messages, skippedMsg)
						if !ts.opts.NoHistory {
							ts.agent.Sessions.AddFullMessage(ts.sessionKey, skippedMsg)
							ts.recordPersistedMessage(skippedMsg)
						}
					}
				}
				break
			}

			// Also poll for any SubTurn results that arrived during tool execution.
			if ts.pendingResults != nil {
				select {
				case result, ok := <-ts.pendingResults:
					if ok && result != nil && result.ForLLM != "" {
						content := al.cfg.FilterSensitiveData(result.ForLLM)
						msg := providers.Message{Role: "user", Content: fmt.Sprintf("[SubTurn Result] %s", content)}
						messages = append(messages, msg)
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, msg)
					}
				default:
					// No results available
				}
			}
		}

		if allResponsesHandled {
			if len(pendingMessages) > 0 {
				logger.InfoCF("agent", "Pending steering exists after handled tool delivery; continuing turn before finalizing",
					map[string]any{
						"agent_id":       ts.agent.ID,
						"steering_count": len(pendingMessages),
						"session_key":    ts.sessionKey,
					})
				finalContent = ""
				goto turnLoop
			}

			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				logger.InfoCF("agent", "Steering arrived after handled tool delivery; continuing turn before finalizing",
					map[string]any{
						"agent_id":       ts.agent.ID,
						"steering_count": len(steerMsgs),
						"session_key":    ts.sessionKey,
					})
				pendingMessages = append(pendingMessages, steerMsgs...)
				finalContent = ""
				goto turnLoop
			}

			summaryMsg := providers.Message{
				Role:    "assistant",
				Content: handledToolResponseSummary,
			}

			if !ts.opts.NoHistory {
				ts.agent.Sessions.AddMessage(ts.sessionKey, summaryMsg.Role, summaryMsg.Content)
				ts.recordPersistedMessage(summaryMsg)
				ts.ingestMessage(turnCtx, al, summaryMsg)
				if err := ts.agent.Sessions.Save(ts.sessionKey); err != nil {
					turnStatus = TurnEndStatusError
					al.emitEvent(
						EventKindError,
						ts.eventMeta("runTurn", "turn.error"),
						ErrorPayload{
							Stage:   "session_save",
							Message: err.Error(),
						},
					)
					return turnResult{}, err
				}
			}
			if ts.opts.EnableSummary {
				al.contextManager.Compact(turnCtx, &CompactRequest{SessionKey: ts.sessionKey, Reason: ContextCompressReasonSummarize, Budget: ts.agent.ContextWindow})
			}

			ts.setPhase(TurnPhaseCompleted)
			ts.setFinalContent("")
			logger.InfoCF("agent", "Tool output satisfied delivery; ending turn without follow-up LLM",
				map[string]any{
					"agent_id":   ts.agent.ID,
					"iteration":  iteration,
					"tool_count": len(normalizedToolCalls),
				})
			return turnResult{
				finalContent: "",
				status:       turnStatus,
				followUps:    append([]bus.InboundMessage(nil), ts.followUps...),
			}, nil
		}

		ts.agent.Tools.TickTTL()
		logger.DebugCF("agent", "TTL tick after tool execution", map[string]any{
			"agent_id": ts.agent.ID, "iteration": iteration,
		})
	}

	if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
		logger.InfoCF("agent", "Steering arrived after turn completion; continuing turn before finalizing",
			map[string]any{
				"agent_id":       ts.agent.ID,
				"steering_count": len(steerMsgs),
				"session_key":    ts.sessionKey,
			})
		pendingMessages = append(pendingMessages, steerMsgs...)
		finalContent = ""
		goto turnLoop
	}

	if ts.hardAbortRequested() {
		turnStatus = TurnEndStatusAborted
		return al.abortTurn(ts)
	}

	if finalContent == "" {
		if ts.currentIteration() >= ts.agent.MaxIterations && ts.agent.MaxIterations > 0 {
			finalContent = toolLimitResponse
		} else {
			finalContent = ts.opts.DefaultResponse
		}
	}

	ts.setPhase(TurnPhaseFinalizing)
	ts.setFinalContent(finalContent)
	if !ts.opts.NoHistory {
		finalMsg := providers.Message{Role: "assistant", Content: finalContent}
		ts.agent.Sessions.AddMessage(ts.sessionKey, finalMsg.Role, finalMsg.Content)
		ts.recordPersistedMessage(finalMsg)
		ts.ingestMessage(turnCtx, al, finalMsg)
		if err := ts.agent.Sessions.Save(ts.sessionKey); err != nil {
			turnStatus = TurnEndStatusError
			al.emitEvent(
				EventKindError,
				ts.eventMeta("runTurn", "turn.error"),
				ErrorPayload{
					Stage:   "session_save",
					Message: err.Error(),
				},
			)
			return turnResult{}, err
		}
	}

	if ts.opts.EnableSummary {
		al.contextManager.Compact(
			turnCtx,
			&CompactRequest{
				SessionKey: ts.sessionKey,
				Reason:     ContextCompressReasonSummarize,
				Budget:     ts.agent.ContextWindow,
			},
		)
	}

	ts.setPhase(TurnPhaseCompleted)
	return turnResult{
		finalContent: finalContent,
		status:       turnStatus,
		followUps:    append([]bus.InboundMessage(nil), ts.followUps...),
	}, nil
}

func (al *AgentLoop) abortTurn(ts *turnState) (turnResult, error) {
	ts.setPhase(TurnPhaseAborted)
	if !ts.opts.NoHistory {
		if err := ts.restoreSession(ts.agent); err != nil {
			al.emitEvent(
				EventKindError,
				ts.eventMeta("abortTurn", "turn.error"),
				ErrorPayload{
					Stage:   "session_restore",
					Message: err.Error(),
				},
			)
			return turnResult{}, err
		}
	}
	return turnResult{status: TurnEndStatusAborted}, nil
}

func (al *AgentLoop) selectCandidates(
	agent *AgentInstance,
	userMsg string,
	history []providers.Message,
) (candidates []providers.FallbackCandidate, model string, usedLight bool) {
	if agent.Router == nil || len(agent.LightCandidates) == 0 {
		return agent.Candidates, resolvedCandidateModel(agent.Candidates, agent.Model), false
	}

	_, usedLight, score := agent.Router.SelectModel(userMsg, history, agent.Model)
	if !usedLight {
		logger.DebugCF("agent", "Model routing: primary model selected",
			map[string]any{
				"agent_id":  agent.ID,
				"score":     score,
				"threshold": agent.Router.Threshold(),
			})
		return agent.Candidates, resolvedCandidateModel(agent.Candidates, agent.Model), false
	}

	logger.InfoCF("agent", "Model routing: light model selected",
		map[string]any{
			"agent_id":    agent.ID,
			"light_model": agent.Router.LightModel(),
			"score":       score,
			"threshold":   agent.Router.Threshold(),
		})
	return agent.LightCandidates, resolvedCandidateModel(agent.LightCandidates, agent.Router.LightModel()), true
}

func (al *AgentLoop) resolveContextManager() ContextManager {
	name := al.cfg.Agents.Defaults.ContextManager
	if name == "" || name == "legacy" {
		return &legacyContextManager{al: al}
	}
	factory, ok := lookupContextManager(name)
	if !ok {
		logger.WarnCF("agent", "Unknown context manager, falling back to legacy", map[string]any{
			"name": name,
		})
		return &legacyContextManager{al: al}
	}
	cm, err := factory(al.cfg.Agents.Defaults.ContextManagerConfig, al)
	if err != nil {
		logger.WarnCF("agent", "Failed to create context manager, falling back to legacy", map[string]any{
			"name":  name,
			"error": err.Error(),
		})
		return &legacyContextManager{al: al}
	}
	return cm
}

func (al *AgentLoop) askSideQuestion(
	ctx context.Context,
	agent *AgentInstance,
	opts *processOptions,
	question string,
) (string, error) {
	if agent == nil {
		return "", fmt.Errorf("askSideQuestion: no agent available for /btw")
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("askSideQuestion: %w", fmt.Errorf("Usage: /btw <question>"))
	}

	if opts != nil {
		normalizeProcessOptionsInPlace(opts)
	}

	var media []string
	var channel, chatID, senderID, senderDisplayName string
	if opts != nil {
		media = opts.Media
		channel = opts.Channel
		chatID = opts.ChatID
		senderID = opts.SenderID
		senderDisplayName = opts.SenderDisplayName
	}

	// Build messages with context but WITHOUT adding to session history
	var history []providers.Message
	var summary string
	if opts != nil && !opts.NoHistory {
		if resp, err := al.contextManager.Assemble(ctx, &AssembleRequest{
			SessionKey: opts.SessionKey,
			Budget:     agent.ContextWindow,
			MaxTokens:  agent.MaxTokens,
		}); err == nil && resp != nil {
			history = resp.History
			summary = resp.Summary
		}
	}

	messages := agent.ContextBuilder.BuildMessages(
		history,
		summary,
		question,
		media,
		channel,
		chatID,
		senderID,
		senderDisplayName,
	)

	maxMediaSize := al.GetConfig().Agents.Defaults.GetMaxMediaSize()
	messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)

	activeCandidates, activeModel, usedLight := al.selectCandidates(agent, question, messages)
	selectedModelName := sideQuestionModelName(agent, usedLight)

	llmOpts := map[string]any{
		"max_tokens":       agent.MaxTokens,
		"temperature":      agent.Temperature,
		"prompt_cache_key": agent.ID + ":btw",
	}

	hookModelChanged := false
	callProvider := func(
		ctx context.Context,
		candidate providers.FallbackCandidate,
		model string,
		forceModel bool,
		callMessages []providers.Message,
	) (*providers.LLMResponse, error) {
		provider, providerModel, cleanup, err := al.isolatedSideQuestionProvider(agent, selectedModelName, candidate)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		if !forceModel || strings.TrimSpace(model) == "" {
			model = providerModel
		}
		callOpts := llmOpts
		if _, exists := callOpts["thinking_level"]; !exists && agent.ThinkingLevel != ThinkingOff {
			if tc, ok := provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
				callOpts = shallowCloneLLMOptions(llmOpts)
				callOpts["thinking_level"] = string(agent.ThinkingLevel)
			}
		}
		return provider.Chat(ctx, callMessages, nil, model, callOpts)
	}

	turnCtx := newTurnContext(nil, nil, nil)
	if opts != nil {
		turnCtx = newTurnContext(opts.Dispatch.InboundContext, opts.Dispatch.RouteResult, opts.Dispatch.SessionScope)
	}
	llmModel := activeModel
	if al.hooks != nil {
		llmReq, decision := al.hooks.BeforeLLM(ctx, &LLMHookRequest{
			Meta: EventMeta{
				Source:      "askSideQuestion",
				TracePath:   "turn.llm.request",
				turnContext: cloneTurnContext(turnCtx),
			},
			Context:          cloneTurnContext(turnCtx),
			Model:            llmModel,
			Messages:         messages,
			Tools:            nil,
			Options:          llmOpts,
			GracefulTerminal: false,
		})
		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if llmReq != nil {
				if strings.TrimSpace(llmReq.Model) != "" && llmReq.Model != llmModel {
					hookModelChanged = true
				}
				llmModel = llmReq.Model
				messages = llmReq.Messages
				llmOpts = llmReq.Options
			}
		case HookActionAbortTurn:
			reason := decision.Reason
			if reason == "" {
				reason = "hook requested turn abort"
			}
			return "", fmt.Errorf("hook aborted turn during before_llm: %s", reason)
		case HookActionHardAbort:
			reason := decision.Reason
			if reason == "" {
				reason = "hook requested turn abort"
			}
			return "", fmt.Errorf("hook aborted turn during before_llm: %s", reason)
		}
	}
	if hookModelChanged {
		// Hook-selected models must not continue through the pre-hook fallback
		// candidate list, otherwise fallback execution would call the original
		// candidate model and silently ignore the hook decision.
		activeCandidates = nil
	}

	callSideLLM := func(callMessages []providers.Message) (*providers.LLMResponse, error) {
		if len(activeCandidates) > 1 && al.fallback != nil {
			fbResult, err := al.fallback.Execute(
				ctx,
				activeCandidates,
				func(ctx context.Context, providerName, model string) (*providers.LLMResponse, error) {
					candidate := providers.FallbackCandidate{Provider: providerName, Model: model}
					for _, activeCandidate := range activeCandidates {
						if activeCandidate.Provider == providerName && activeCandidate.Model == model {
							candidate = activeCandidate
							break
						}
					}
					return callProvider(ctx, candidate, model, false, callMessages)
				},
			)
			if err != nil {
				return nil, err
			}
			return fbResult.Response, nil
		}

		var candidate providers.FallbackCandidate
		if len(activeCandidates) > 0 {
			candidate = activeCandidates[0]
		}
		return callProvider(ctx, candidate, llmModel, hookModelChanged, callMessages)
	}

	// Retry without media if vision is unsupported
	// Note: Vision retry is only applied to the initial call. If fallback chain
	// is used, vision errors from fallback providers will not trigger retry.
	var resp *providers.LLMResponse
	var err error
	resp, err = callSideLLM(messages)
	if err != nil && hasMediaRefs(messages) && isVisionUnsupportedError(err) {
		al.emitEvent(
			EventKindLLMRetry,
			EventMeta{
				Source:      "askSideQuestion",
				TracePath:   "turn.llm.retry",
				turnContext: cloneTurnContext(turnCtx),
			},
			LLMRetryPayload{
				Attempt:    1,
				MaxRetries: 1,
				Reason:     "vision_unsupported",
				Error:      err.Error(),
				Backoff:    0,
			},
		)
		messagesWithoutMedia := stripMessageMedia(messages)
		resp, err = callSideLLM(messagesWithoutMedia)
	}
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}

	// Apply after_llm hooks
	if al.hooks != nil {
		llmResp, decision := al.hooks.AfterLLM(ctx, &LLMHookResponse{
			Meta: EventMeta{
				Source:      "askSideQuestion",
				TracePath:   "turn.llm.response",
				turnContext: cloneTurnContext(turnCtx),
			},
			Context:  cloneTurnContext(turnCtx),
			Model:    llmModel,
			Response: resp,
		})
		switch decision.normalizedAction() {
		case HookActionContinue, HookActionModify:
			if llmResp != nil && llmResp.Response != nil {
				resp = llmResp.Response
			}
		case HookActionAbortTurn, HookActionHardAbort:
			reason := decision.Reason
			if reason == "" {
				reason = "hook requested turn abort"
			}
			return "", fmt.Errorf("hook aborted turn during after_llm: %s", reason)
		}
	}

	return sideQuestionResponseContent(resp), nil
}

func (al *AgentLoop) isolatedSideQuestionProvider(
	agent *AgentInstance,
	baseModelName string,
	candidate providers.FallbackCandidate,
) (providers.LLMProvider, string, func(), error) {
	if agent == nil {
		return nil, "", func() {}, fmt.Errorf("isolatedSideQuestionProvider: no agent available for /btw")
	}

	modelCfg, err := al.sideQuestionModelConfig(agent, baseModelName, candidate)
	if err != nil {
		return nil, "", func() {}, fmt.Errorf("isolatedSideQuestionProvider: %w", err)
	}

	factory := al.providerFactory
	if factory == nil {
		factory = providers.CreateProviderFromConfig
	}
	provider, modelID, err := factory(modelCfg)
	if err != nil {
		return nil, "", func() {}, fmt.Errorf("isolatedSideQuestionProvider: %w", err)
	}

	cleanup := func() {
		closeProviderIfStateful(provider)
	}
	return provider, modelID, cleanup, nil
}

func (al *AgentLoop) sideQuestionModelConfig(
	agent *AgentInstance,
	baseModelName string,
	candidate providers.FallbackCandidate,
) (*config.ModelConfig, error) {
	if agent == nil {
		return nil, fmt.Errorf("sideQuestionModelConfig: no agent available for /btw")
	}

	// If candidate has an identity key, use that
	if name := modelNameFromIdentityKey(candidate.IdentityKey); name != "" {
		modelCfg, err := resolvedModelConfig(al.GetConfig(), name, agent.Workspace)
		if err == nil {
			return modelCfg, nil
		}
		// Fallback: create a minimal config if lookup fails
	}

	// Otherwise, clean up the base model name and use it
	baseModelName = strings.TrimSpace(baseModelName)
	modelCfg, err := resolvedModelConfig(al.GetConfig(), baseModelName, agent.Workspace)
	if err != nil {
		// Fallback: create a minimal config for test scenarios
		model := strings.TrimSpace(baseModelName)
		if candidate.Model != "" {
			model = candidate.Model
		}
		if candidate.Provider != "" && candidate.Model != "" {
			model = providers.NormalizeProvider(candidate.Provider) + "/" + candidate.Model
		} else {
			model = ensureProtocolModel(model)
		}
		return &config.ModelConfig{
			ModelName: baseModelName,
			Model:     model,
			Workspace: agent.Workspace,
		}, nil
	}

	// If candidate specifies a different provider/model, override
	clone := *modelCfg
	if candidate.Provider != "" && candidate.Model != "" {
		clone.Model = providers.NormalizeProvider(candidate.Provider) + "/" + candidate.Model
	}
	return &clone, nil
}
