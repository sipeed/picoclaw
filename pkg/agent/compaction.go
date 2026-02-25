package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func (al *AgentLoop) maybeFlushMemoryBeforeCompaction(
	ctx context.Context,
	agent *AgentInstance,
	sessionKey string,
	tokenEstimate int,
) (bool, error) {
	if !isCompactionModeEnabled(agent.CompactionMode) {
		return false, nil
	}
	if !agent.MemoryFlushEnabled || sessionKey == "" {
		return false, nil
	}

	triggerPoint := agent.ContextWindow - agent.CompactionReserveTokens - agent.MemoryFlushSoftThreshold
	if triggerPoint < agent.ContextWindow/3 {
		triggerPoint = agent.ContextWindow / 3
	}
	if tokenEstimate < triggerPoint {
		return false, nil
	}

	compactionCount, flushedAtCount, _ := agent.Sessions.GetCompactionState(sessionKey)
	if flushedAtCount == compactionCount {
		return false, nil
	}

	if err := al.flushMemorySnapshot(ctx, agent, sessionKey); err != nil {
		return false, err
	}

	agent.Sessions.MarkMemoryFlush(sessionKey, compactionCount)
	_ = agent.Sessions.Save(sessionKey)
	return true, nil
}

func (al *AgentLoop) flushMemorySnapshot(ctx context.Context, agent *AgentInstance, sessionKey string) error {
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) == 0 {
		return nil
	}

	recent := make([]providers.Message, 0, 12)
	for i := len(history) - 1; i >= 0 && len(recent) < 12; i-- {
		msg := history[i]
		if (msg.Role != "user" && msg.Role != "assistant") || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		recent = append([]providers.Message{msg}, recent...)
	}
	if len(recent) == 0 {
		return nil
	}

	var prompt strings.Builder
	prompt.WriteString("Extract durable memory from this chat.\n")
	prompt.WriteString("Return concise markdown bullets under these headings only:\n")
	prompt.WriteString("## Profile\n## Long-term Facts\n## Active Goals\n## Constraints\n## Open Threads\n## Deprecated/Resolved\n")
	prompt.WriteString("\nCHAT:\n")
	for _, m := range recent {
		prompt.WriteString(m.Role)
		prompt.WriteString(": ")
		prompt.WriteString(m.Content)
		prompt.WriteString("\n")
	}

	resp, err := agent.Provider.Chat(
		ctx,
		[]providers.Message{{Role: "user", Content: prompt.String()}},
		nil,
		agent.Model,
		map[string]any{
			"max_tokens":  700,
			"temperature": 0.2,
		},
	)
	if err != nil {
		return err
	}
	if strings.TrimSpace(resp.Content) == "" {
		return fmt.Errorf("empty memory flush response")
	}

	memory := NewMemoryStore(agent.Workspace)
	return memory.OrganizeWriteback(resp.Content)
}

func (al *AgentLoop) compactWithSafeguard(
	ctx context.Context,
	agent *AgentInstance,
	sessionKey string,
) (bool, error) {
	switch normalizeCompactionMode(agent.CompactionMode) {
	case "off":
		return false, nil
	case "legacy":
		beforeHistory := len(agent.Sessions.GetHistory(sessionKey))
		beforeSummary := strings.TrimSpace(agent.Sessions.GetSummary(sessionKey))
		al.summarizeSession(agent, sessionKey)
		afterHistory := len(agent.Sessions.GetHistory(sessionKey))
		afterSummary := strings.TrimSpace(agent.Sessions.GetSummary(sessionKey))
		if afterHistory < beforeHistory || afterSummary != beforeSummary {
			agent.Sessions.IncrementCompactionCount(sessionKey)
			_ = agent.Sessions.Save(sessionKey)
			return true, nil
		}
		return false, nil
	}

	history := sanitizeHistoryForProvider(agent.Sessions.GetHistory(sessionKey))
	if len(history) <= 6 {
		return false, nil
	}

	historyTokens := al.estimateTokens(history)
	historyBudget := int(float64(agent.ContextWindow) * agent.CompactionMaxHistoryShare)
	if historyBudget <= 0 {
		historyBudget = agent.ContextWindow / 2
	}
	if historyTokens <= historyBudget && len(history) < 24 {
		return false, nil
	}

	keepRecentTokens := agent.CompactionKeepRecentTokens
	if keepRecentTokens <= 0 {
		keepRecentTokens = maxInt(1024, agent.ContextWindow/4)
	}

	keepStart := len(history) - 4
	if keepStart < 1 {
		keepStart = 1
	}
	acc := 0
	for i := len(history) - 1; i >= 1; i-- {
		acc += al.estimateMessageTokens(history[i])
		if acc >= keepRecentTokens {
			keepStart = i
			break
		}
	}
	if keepStart <= 0 || keepStart >= len(history) {
		return false, nil
	}

	toSummarize := history[:keepStart]
	kept := history[keepStart:]
	if len(toSummarize) == 0 || len(kept) == 0 {
		return false, nil
	}

	existingSummary := agent.Sessions.GetSummary(sessionKey)
	summary, err := al.generateCompactionSummary(ctx, agent, toSummarize, existingSummary)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(summary) == "" {
		return false, fmt.Errorf("compaction summary unavailable")
	}

	summary = strings.TrimSpace(summary) + "\n\n[Post-compaction refresh: Re-check AGENTS.md and MEMORY.md before continuing.]"
	agent.Sessions.SetSummary(sessionKey, summary)
	agent.Sessions.SetHistory(sessionKey, kept)
	agent.Sessions.IncrementCompactionCount(sessionKey)
	_ = agent.Sessions.Save(sessionKey)

	logger.InfoCF("agent", "Compaction safeguard completed", map[string]any{
		"session_key":           sessionKey,
		"history_tokens_before": historyTokens,
		"kept_messages":         len(kept),
		"summarized_messages":   len(toSummarize),
	})
	return true, nil
}

func (al *AgentLoop) generateCompactionSummary(
	ctx context.Context,
	agent *AgentInstance,
	history []providers.Message,
	existingSummary string,
) (string, error) {
	safeMessages := make([]providers.Message, 0, len(history))
	for _, msg := range history {
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "tool" {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		if msg.Role == "tool" && len(content) > 1200 {
			head := 700
			tail := 300
			content = content[:head] + "\n...\n[tool result condensed]\n...\n" + content[len(content)-tail:]
		}
		msg.Content = content
		safeMessages = append(safeMessages, msg)
	}
	if len(safeMessages) == 0 {
		return strings.TrimSpace(existingSummary), nil
	}

	maxChunkTokens := int(float64(agent.ContextWindow)*0.35) - 1024
	if maxChunkTokens < 512 {
		maxChunkTokens = 512
	}

	chunks := al.splitMessagesByTokenBudget(safeMessages, maxChunkTokens)
	summary := strings.TrimSpace(existingSummary)
	for _, chunk := range chunks {
		next, err := al.summarizeBatchStructured(ctx, agent, chunk, summary)
		if err != nil {
			return "", err
		}
		summary = strings.TrimSpace(next)
	}
	return summary, nil
}

func (al *AgentLoop) summarizeBatchStructured(
	ctx context.Context,
	agent *AgentInstance,
	batch []providers.Message,
	existingSummary string,
) (string, error) {
	var sb strings.Builder
	sb.WriteString("Summarize this conversation segment for future continuity.\n")
	sb.WriteString("Use concise markdown with sections: Intent, Decisions, Tool Results, Pending Actions, Constraints.\n")
	if existingSummary != "" {
		sb.WriteString("\nExisting summary:\n")
		sb.WriteString(existingSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("\nConversation:\n")
	for _, m := range batch {
		sb.WriteString(m.Role)
		sb.WriteString(": ")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}

	resp, err := agent.Provider.Chat(
		ctx,
		[]providers.Message{{Role: "user", Content: sb.String()}},
		nil,
		agent.Model,
		map[string]any{
			"max_tokens":  800,
			"temperature": 0.2,
		},
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func (al *AgentLoop) splitMessagesByTokenBudget(
	messages []providers.Message,
	maxTokens int,
) [][]providers.Message {
	if len(messages) == 0 || maxTokens <= 0 {
		return nil
	}
	chunks := make([][]providers.Message, 0, 4)
	current := make([]providers.Message, 0, 8)
	currentTokens := 0
	for _, msg := range messages {
		msgTokens := al.estimateMessageTokens(msg)
		if len(current) > 0 && currentTokens+msgTokens > maxTokens {
			chunks = append(chunks, current)
			current = make([]providers.Message, 0, 8)
			currentTokens = 0
		}
		current = append(current, msg)
		currentTokens += msgTokens
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (al *AgentLoop) safeCompactionContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 90*time.Second)
}

func normalizeCompactionMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "safeguard":
		return "safeguard"
	case "off", "none", "disabled":
		return "off"
	case "legacy":
		return "legacy"
	default:
		return "safeguard"
	}
}

func isCompactionModeEnabled(mode string) bool {
	return normalizeCompactionMode(mode) != "off"
}
