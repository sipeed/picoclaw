package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type TurnActionRecord struct {
	Source string `json:"source"`
	Tool   string `json:"tool,omitempty"`
	Text   string `json:"text"`
	Error  bool   `json:"error,omitempty"`
}

func appendTurnActionRecord(
	records []TurnActionRecord,
	source, tool, text string,
	isError bool,
) []TurnActionRecord {
	text = strings.TrimSpace(text)
	if text == "" {
		return records
	}
	rec := TurnActionRecord{
		Source: source,
		Tool:   strings.TrimSpace(tool),
		Text:   text,
		Error:  isError,
	}
	if n := len(records); n > 0 {
		prev := records[n-1]
		if prev.Source == rec.Source && prev.Tool == rec.Tool && prev.Text == rec.Text &&
			prev.Error == rec.Error {
			return records
		}
	}
	return append(records, rec)
}

func finalTurnRenderEligible(al *AgentLoop, exec *turnExecution) bool {
	if al == nil || exec == nil {
		return false
	}
	if !al.cfg.Agents.Defaults.UseFinalTurnRender() {
		return false
	}
	return exec.sawSteering
}

func finalTurnRenderModel(ts *turnState, exec *turnExecution) (providers.LLMProvider, string) {
	if exec != nil {
		if exec.activeProvider != nil && strings.TrimSpace(exec.activeModel) != "" {
			return exec.activeProvider, strings.TrimSpace(exec.activeModel)
		}
		if exec.activeProvider != nil {
			return exec.activeProvider, strings.TrimSpace(ts.agent.Model)
		}
	}
	if ts == nil || ts.agent == nil {
		return nil, ""
	}
	return ts.agent.Provider, strings.TrimSpace(ts.agent.Model)
}

func buildFinalTurnRenderInstruction(exec *turnExecution) string {
	var b strings.Builder
	b.WriteString("Write the final user-facing reply for this already-completed turn.\n")
	b.WriteString("Use the same language and general style as the conversation.\n")
	b.WriteString("Do not call tools.\n")
	b.WriteString(
		"Answer the full accumulated user request across this turn, not only the latest follow-up.\n",
	)
	b.WriteString(
		"If a later follow-up clearly corrected, narrowed, or replaced an earlier request, follow the latest clarified intent.\n",
	)
	b.WriteString(
		"If later follow-ups added to earlier requests, include the completed additive results together.\n",
	)
	b.WriteString(
		"Use only the facts already present in the conversation and tool results. Do not invent missing results.\n",
	)
	b.WriteString("Keep the reply concise and natural.\n")

	if exec == nil || len(exec.actionLog) == 0 {
		return b.String()
	}

	records := make([]TurnActionRecord, 0, len(exec.actionLog))
	for _, rec := range exec.actionLog {
		if strings.TrimSpace(rec.Text) == "" {
			continue
		}
		records = append(records, rec)
	}
	if len(records) == 0 {
		return b.String()
	}

	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return b.String()
	}
	b.WriteString("\nExplicit user-facing outcomes recorded during the turn:\n")
	_, _ = b.Write(raw)
	return b.String()
}

func tryRenderFinalTurnReply(
	ctx context.Context,
	al *AgentLoop,
	ts *turnState,
	exec *turnExecution,
	fallback string,
) (string, bool) {
	fallback = strings.TrimSpace(fallback)
	if !finalTurnRenderEligible(al, exec) {
		return fallback, false
	}
	if exec == nil || len(exec.messages) == 0 {
		return fallback, false
	}

	provider, model := finalTurnRenderModel(ts, exec)
	if provider == nil || model == "" {
		return fallback, false
	}

	messages := append([]providers.Message(nil), exec.messages...)
	instruction := buildFinalTurnRenderInstruction(exec)
	messages = append(messages, providers.Message{
		Role:    "user",
		Content: instruction,
	})

	opts := map[string]any{
		"max_tokens":       minInt(ts.agent.MaxTokens, 800),
		"temperature":      0.2,
		"prompt_cache_key": ts.agent.ID,
	}

	resp, err := provider.Chat(ctx, messages, nil, model, opts)
	if err != nil || resp == nil {
		if err != nil {
			logger.WarnCF("agent", "Final turn render pass failed", map[string]any{
				"agent_id": ts.agent.ID,
				"error":    err.Error(),
			})
		}
		return fallback, false
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		content = strings.TrimSpace(resp.ReasoningContent)
	}
	if content == "" {
		return fallback, false
	}

	logger.InfoCF("agent", "Rendered final reply from accumulated turn context",
		map[string]any{
			"agent_id":            ts.agent.ID,
			"session_key":         ts.sessionKey,
			"messages_count":      len(messages),
			"action_record_count": len(exec.actionLog),
		})
	return content, true
}

func renderFinalTurnReply(
	ctx context.Context,
	al *AgentLoop,
	ts *turnState,
	exec *turnExecution,
	fallback string,
) string {
	content, ok := tryRenderFinalTurnReply(ctx, al, ts, exec, fallback)
	if ok {
		return content
	}
	return strings.TrimSpace(fallback)
}

func shouldFinalizeAfterToolLoopWithRender(al *AgentLoop, exec *turnExecution) bool {
	if !finalTurnRenderEligible(al, exec) {
		return false
	}
	if exec == nil {
		return false
	}
	return !exec.allResponsesHandled
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
