package agent

import (
	"context"
	"encoding/json"
	"fmt"
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
		if prev.Source == rec.Source && prev.Tool == rec.Tool && prev.Text == rec.Text && prev.Error == rec.Error {
			return records
		}
	}
	return append(records, rec)
}

func actionSummaryEligible(al *AgentLoop, ts *turnState, exec *turnExecution) bool {
	if al == nil || ts == nil || exec == nil {
		return false
	}
	if !al.cfg.Agents.Defaults.UseFinalActionSummary() {
		return false
	}
	if !exec.sawSteering {
		return false
	}
	return len(exec.actionLog) >= 2
}

func joinActionTexts(records []TurnActionRecord) string {
	parts := make([]string, 0, len(records))
	for _, rec := range records {
		text := strings.TrimSpace(rec.Text)
		if text == "" {
			continue
		}
		if len(parts) > 0 && parts[len(parts)-1] == text {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n\n")
}

func synthesizeFinalActionSummary(
	ctx context.Context,
	al *AgentLoop,
	ts *turnState,
	exec *turnExecution,
	fallback string,
) string {
	fallback = strings.TrimSpace(fallback)
	if !actionSummaryEligible(al, ts, exec) {
		return fallback
	}

	records := make([]TurnActionRecord, 0, len(exec.actionLog))
	for _, rec := range exec.actionLog {
		if strings.TrimSpace(rec.Text) == "" {
			continue
		}
		records = append(records, rec)
	}
	if len(records) < 2 {
		return fallback
	}

	if fallback == "" {
		fallback = joinActionTexts(records)
	}

	provider := exec.activeProvider
	model := strings.TrimSpace(exec.activeModel)
	if provider == nil {
		provider = ts.agent.Provider
	}
	if model == "" {
		model = ts.agent.Model
	}
	if provider == nil || model == "" {
		return fallback
	}

	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fallback
	}

	messages := []providers.Message{
		{
			Role: "system",
			Content: "You are formatting the final user-facing reply for a completed assistant turn.\n" +
				"Use the same language as the action records.\n" +
				"Summarize only the factual completed outcomes listed below.\n" +
				"Omit pure progress chatter unless it states an actual completed outcome.\n" +
				"Do not invent actions. Do not omit completed actions unless they are exact duplicates.\n" +
				"Keep the reply concise and natural.",
		},
		{
			Role: "user",
			Content: fmt.Sprintf(
				"Action records for this completed turn:\n\n%s\n\nWrite one concise final reply for the user.",
				string(raw),
			),
		},
	}

	resp, err := provider.Chat(ctx, messages, nil, model, map[string]any{
		"max_tokens":  300,
		"temperature": 0.2,
	})
	if err != nil || resp == nil {
		if err != nil {
			logger.WarnCF("agent", "Final action summary synthesis failed", map[string]any{
				"agent_id": ts.agent.ID,
				"error":    err.Error(),
			})
		}
		return fallback
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		content = strings.TrimSpace(resp.ReasoningContent)
	}
	if content == "" {
		return fallback
	}
	return content
}
