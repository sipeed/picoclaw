// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func (al *AgentLoop) runTurn(
	ctx context.Context,
	ts *turnState,
	pipeline *Pipeline,
) (turnResult, error) {
	turnCtx, turnCancel := context.WithCancel(ctx)
	defer turnCancel()
	ts.setTurnCancel(turnCancel)

	// Inject turnState and AgentLoop into context so tools (e.g. spawn) can retrieve them.
	turnCtx = withTurnState(turnCtx, ts)
	turnCtx = WithAgentLoop(turnCtx, al)

	al.registerActiveTurn(ts)
	defer al.clearActiveTurn(ts)

	if al.takePendingStop(ts.sessionKey) {
		_ = ts.requestHardAbort()
	}

	turnStatus := TurnEndStatusCompleted
	defer func() {
		al.emitEvent(
			runtimeevents.KindAgentTurnEnd,
			ts.eventMeta("runTurn", "turn.end"),
			TurnEndPayload{
				Status:          turnStatus,
				Iterations:      ts.currentIteration(),
				Duration:        time.Since(ts.startedAt),
				FinalContentLen: ts.finalContentLen(),
			},
		)
	}()

	if ts.hardAbortRequested() {
		turnStatus = TurnEndStatusAborted
		return al.abortTurn(ts)
	}

	al.emitEvent(
		runtimeevents.KindAgentTurnStart,
		ts.eventMeta("runTurn", "turn.start"),
		TurnStartPayload{
			UserMessage: ts.userMessage,
			MediaCount:  len(ts.media),
		},
	)

	// SetupTurn extracts the one-time initialization phase.
	exec, err := pipeline.SetupTurn(turnCtx, ts)
	if err != nil {
		return turnResult{}, err
	}

	// Convenience references to exec fields used throughout the turn loop.
	messages := exec.messages
	pendingMessages := exec.pendingMessages
	maxMediaSize := pipeline.Cfg.Agents.Defaults.GetMaxMediaSize()
	finalContent := exec.finalContent

	for ts.currentIteration() < ts.agent.MaxIterations || len(exec.pendingMessages) > 0 || func() bool {
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
			// For subsequent iterations, read from exec.pendingMessages which
			// is where ExecuteTools (or initial poll) deposits steering.
			// We do NOT call dequeueSteeringMessagesForScope here because
			// steering was already consumed from al.steering by ExecuteTools.
			if len(exec.pendingMessages) > 0 {
				exec.markSteeringObserved()
				pendingMessages = append(pendingMessages, exec.pendingMessages...)
				exec.pendingMessages = nil
			}
		} else if !ts.opts.SkipInitialSteeringPoll {
			if steerMsgs := al.dequeueSteeringMessagesForScopeWithFallback(ts.sessionKey); len(steerMsgs) > 0 {
				exec.markSteeringObserved()
				pendingMessages = append(pendingMessages, steerMsgs...)
			}
		}

		// Check if parent turn has ended (SubTurn support from HEAD)
		if ts.parentTurnState != nil && ts.IsParentEnded() {
			if !ts.critical {
				logger.InfoCF(
					"agent",
					"Parent turn ended, non-critical SubTurn exiting gracefully",
					map[string]any{
						"agent_id":  ts.agentID,
						"iteration": iteration,
						"turn_id":   ts.turnID,
					},
				)
				break
			}
			logger.InfoCF(
				"agent",
				"Parent turn ended, critical SubTurn continues running",
				map[string]any{
					"agent_id":  ts.agentID,
					"iteration": iteration,
					"turn_id":   ts.turnID,
				},
			)
		}

		// Poll for pending SubTurn results (from HEAD)
		if ts.pendingResults != nil {
			select {
			case result, ok := <-ts.pendingResults:
				if ok && result != nil && result.ForLLM != "" {
					content := al.cfg.FilterSensitiveData(result.ForLLM)
					msg := subTurnResultPromptMessage(content)
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
				runtimeevents.KindAgentSteeringInjected,
				ts.eventMeta("runTurn", "turn.steering.injected"),
				SteeringInjectedPayload{
					Count:           len(pendingMessages),
					TotalContentLen: totalContentLen,
				},
			)
			// Clear exec.pendingMessages after injection so InitialSteeringMessages
			// are not re-injected on subsequent iterations (Issue 2 fix).
			exec.pendingMessages = nil
		}
		// Always sync messages into exec.messages so CallLLM sees the updated state
		exec.messages = messages

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  ts.agent.ID,
				"iteration": iteration,
				"max":       ts.agent.MaxIterations,
			})

		// Execute LLM call via Pipeline
		ts.setPhase(TurnPhaseRunning)
		ctrl, callErr := pipeline.CallLLM(ctx, turnCtx, ts, exec, iteration)
		if callErr != nil {
			turnStatus = TurnEndStatusError
			return turnResult{}, callErr
		}
		messages = exec.messages
		pendingMessages = exec.pendingMessages
		finalContent = exec.finalContent

		switch ctrl {
		case ControlContinue:
			continue
		case ControlBreak:
			// Hard abort: delegate to abortTurn (sets TurnEndStatusAborted)
			if exec.abortedByHardAbort {
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}
			// Hook abort (HookActionAbortTurn): sets TurnEndStatusError, returns error
			if exec.abortedByHook {
				turnStatus = TurnEndStatusError
				return turnResult{}, fmt.Errorf("hook requested turn abort")
			}
			// Ensure empty response falls back to DefaultResponse
			if finalContent == "" {
				finalContent = ts.opts.DefaultResponse
			}
			finalContent = renderFinalTurnReply(turnCtx, al, ts, exec, finalContent)
			return pipeline.Finalize(ctx, turnCtx, ts, exec, turnStatus, finalContent)
		case ControlToolLoop:
			// Execute tools via Pipeline
			toolCtrl := pipeline.ExecuteTools(ctx, turnCtx, ts, exec, iteration)
			switch toolCtrl {
			case ToolControlContinue:
				// Re-read exec.messages since ExecuteTools may have updated it
				// (added tool results/skipped messages) before returning ControlContinue
				messages = exec.messages
				continue
			case ToolControlFinalize:
				renderedContent, rendered := tryRenderFinalTurnReply(
					turnCtx,
					al,
					ts,
					exec,
					finalContent,
				)
				if !rendered {
					messages = exec.messages
					continue
				}
				if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(
					steerMsgs,
				) > 0 {
					exec.markSteeringObserved()
					logger.InfoCF(
						"agent",
						"Steering arrived during terminal render; continuing turn",
						map[string]any{
							"agent_id":       ts.agent.ID,
							"iteration":      iteration,
							"steering_count": len(steerMsgs),
						},
					)
					exec.pendingMessages = append(exec.pendingMessages, steerMsgs...)
					messages = exec.messages
					continue
				}
				return pipeline.Finalize(ctx, turnCtx, ts, exec, turnStatus, renderedContent)
			case ToolControlBreak:
				// Hard abort: delegate to abortTurn (sets TurnEndStatusAborted)
				if exec.abortedByHardAbort {
					turnStatus = TurnEndStatusAborted
					return al.abortTurn(ts)
				}
				// Hook abort (HookActionAbortTurn): sets TurnEndStatusError, returns error
				if exec.abortedByHook {
					turnStatus = TurnEndStatusError
					return turnResult{}, fmt.Errorf("hook requested turn abort")
				}
				// ExecuteTools returned ControlBreak:
				// - allResponsesHandled=true: finalize without DefaultResponse (exec.finalContent empty)
				// - allResponsesHandled=false: coordinator applies DefaultResponse before finalize
				if exec.allResponsesHandled {
					finalContent = ""
				}
				finalContent = renderFinalTurnReply(turnCtx, al, ts, exec, finalContent)
				return pipeline.Finalize(ctx, turnCtx, ts, exec, turnStatus, finalContent)
			}
		}
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
	finalContent = renderFinalTurnReply(turnCtx, al, ts, exec, finalContent)

	// Check hard abort before finalizing (may have been set during tool execution)
	if ts.hardAbortRequested() {
		turnStatus = TurnEndStatusAborted
		return al.abortTurn(ts)
	}

	return pipeline.Finalize(ctx, turnCtx, ts, exec, turnStatus, finalContent)
}

func (al *AgentLoop) abortTurn(ts *turnState) (turnResult, error) {
	ts.setPhase(TurnPhaseAborted)
	if !ts.opts.NoHistory {
		if err := ts.restoreSession(ts.agent); err != nil {
			al.emitEvent(
				runtimeevents.KindAgentError,
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
	return agent.LightCandidates, resolvedCandidateModel(
		agent.LightCandidates,
		agent.Router.LightModel(),
	), true
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
		logger.WarnCF(
			"agent",
			"Failed to create context manager, falling back to legacy",
			map[string]any{
				"name":  name,
				"error": err.Error(),
			},
		)
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
		provider, providerModel, cleanup, err := al.isolatedSideQuestionProvider(
			agent,
			selectedModelName,
			candidate,
		)
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
		turnCtx = newTurnContext(
			opts.Dispatch.InboundContext,
			opts.Dispatch.RouteResult,
			opts.Dispatch.SessionScope,
		)
	}
	llmModel := activeModel
	if al.hooks != nil {
		llmReq, decision := al.hooks.BeforeLLM(ctx, &LLMHookRequest{
			Meta: HookMeta{
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
						if activeCandidate.Provider == providerName &&
							activeCandidate.Model == model {
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
			runtimeevents.KindAgentLLMRetry,
			HookMeta{
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
			Meta: HookMeta{
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
		return nil, "", func() {}, fmt.Errorf(
			"isolatedSideQuestionProvider: no agent available for /btw",
		)
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
