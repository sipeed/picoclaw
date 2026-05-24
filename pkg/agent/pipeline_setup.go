// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// SetupTurn extracts the one-time initialization phase, returning a
// turnExecution populated with history, messages, and candidate selection.
// It replaces lines 56-145 of the original runTurn.
func (p *Pipeline) SetupTurn(ctx context.Context, ts *turnState) (*turnExecution, error) {
	cfg := p.Cfg
	maxMediaSize := cfg.Agents.Defaults.GetMaxMediaSize()

	var history []providers.Message
	var summary string
	if !ts.opts.NoHistory {
		if resp, err := p.ContextManager.Assemble(ctx, &AssembleRequest{
			SessionKey: ts.sessionKey,
			Budget:     ts.agent.ContextWindow,
			MaxTokens:  ts.agent.MaxTokens,
		}); err == nil && resp != nil {
			history = resp.History
			summary = resp.Summary
		}
	}
	ts.captureRestorePoint(history, summary)

	contextualSkills := ts.activeSkills
	if ts.agent.ContextBuilder != nil {
		contextualSkills = ts.agent.ContextBuilder.ResolveActiveSkillsForContext(ts.activeSkills)
	}
	ts.recordSkillContextSnapshot(skillContextTriggerInitialBuild, contextualSkills)
	initialPromptReq := promptBuildRequestForTurn(ts, history, summary, ts.userMessage, ts.media, cfg)
	initialPromptReq.ActiveSkills = append([]string(nil), contextualSkills...)
	messages := ts.agent.ContextBuilder.BuildMessagesFromPrompt(initialPromptReq)

	messages = resolveMediaRefs(messages, p.MediaStore, maxMediaSize)

	if !ts.opts.NoHistory {
		toolDefs := filterToolsByTurnProfile(ts.agent.Tools.ToProviderDefs(), ts.profile)
		if isOverContextBudget(ts.agent.ContextWindow, messages, toolDefs, ts.agent.MaxTokens) {
			logger.WarnCF("agent", "Proactive compression: context budget exceeded before LLM call",
				map[string]any{"session_key": ts.sessionKey})
			if err := p.ContextManager.Compact(ctx, &CompactRequest{
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
			if resp, err := p.ContextManager.Assemble(ctx, &AssembleRequest{
				SessionKey: ts.sessionKey,
				Budget:     ts.agent.ContextWindow,
				MaxTokens:  ts.agent.MaxTokens,
			}); err == nil && resp != nil {
				history = resp.History
				summary = resp.Summary
			}
			originalHistoryCount := len(history)
			var fit bool
			history, messages, fit = trimHistoryToFitContextWindow(
				history,
				func(trimmedHistory []providers.Message) []providers.Message {
					rebuildPromptReq := promptBuildRequestForTurn(
						ts,
						trimmedHistory,
						summary,
						ts.userMessage,
						ts.media,
						cfg,
					)
					rebuildPromptReq.ActiveSkills = append([]string(nil), contextualSkills...)
					rebuilt := ts.agent.ContextBuilder.BuildMessagesFromPrompt(rebuildPromptReq)
					return resolveMediaRefs(rebuilt, p.MediaStore, maxMediaSize)
				},
				ts.agent.ContextWindow,
				toolDefs,
				ts.agent.MaxTokens,
			)
			if dropped := originalHistoryCount - len(history); dropped > 0 {
				logger.WarnCF("agent", "Trimmed rebuilt history after proactive compaction", map[string]any{
					"session_key":     ts.sessionKey,
					"dropped_msgs":    dropped,
					"remaining_msgs":  len(history),
					"context_window":  ts.agent.ContextWindow,
					"max_tokens":      ts.agent.MaxTokens,
					"still_overlimit": !fit,
				})
			} else if !fit {
				logger.WarnCF("agent", "Context still exceeds budget "+
					"after proactive compaction rebuild", map[string]any{
					"session_key":    ts.sessionKey,
					"history_msgs":   len(history),
					"context_window": ts.agent.ContextWindow,
					"max_tokens":     ts.agent.MaxTokens,
				})
			}
		}
	}

	if !ts.opts.NoHistory && (strings.TrimSpace(ts.userMessage) != "" || len(ts.media) > 0) {
		rootMsg := userPromptMessage(ts.userMessage, ts.media)
		if len(rootMsg.Media) > 0 {
			ts.agent.Sessions.AddFullMessage(ts.sessionKey, rootMsg)
		} else {
			ts.agent.Sessions.AddMessage(ts.sessionKey, rootMsg.Role, rootMsg.Content)
		}
		ts.recordPersistedMessage(rootMsg)
		ts.ingestMessage(ctx, p.al, rootMsg)
	}

	activeCandidates, activeModel, usedLight := p.al.selectCandidates(ts.agent, ts.userMessage, messages)
	activeProvider := ts.agent.Provider
	if usedLight && ts.agent.LightProvider != nil {
		activeProvider = ts.agent.LightProvider
	}
	activeModelName := strings.TrimSpace(ts.agent.Model)
	if usedLight {
		activeModelName = strings.TrimSpace(sideQuestionModelName(ts.agent, true))
	}
	activeModelName = resolvedCandidateModelName(activeCandidates, activeModelName)

	exec := newTurnExecution(
		ts.agent,
		ts.opts,
		history,
		summary,
		messages,
	)
	exec.activeCandidates = activeCandidates
	exec.activeModel = activeModel
	exec.activeModelConfig = resolveActiveModelConfig(
		p.Cfg,
		ts.agent.Workspace,
		activeCandidates,
		activeModel,
		p.Cfg.Agents.Defaults.Provider,
	)
	exec.llmModelName = activeModelName
	exec.activeProvider = activeProvider
	exec.usedLight = usedLight

	return exec, nil
}
