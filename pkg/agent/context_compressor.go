package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ContextCompressor handles context window management: force compression,
// summarization triggers, and session history optimization.
// Extracted from AgentLoop to improve separation of concerns.
type ContextCompressor struct {
	bus         *bus.MessageBus
	summarizing *sync.Map
	wg          *sync.WaitGroup
}

// NewContextCompressor creates a new ContextCompressor.
// The wg parameter tracks in-flight goroutines for graceful shutdown.
func NewContextCompressor(msgBus *bus.MessageBus, summarizing *sync.Map, wg *sync.WaitGroup) *ContextCompressor {
	return &ContextCompressor{
		bus:         msgBus,
		summarizing: summarizing,
		wg:          wg,
	}
}

// MaybeSummarize triggers summarization if the session history exceeds thresholds.
func (cc *ContextCompressor) MaybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
	newHistory := agent.Sessions.GetHistory(sessionKey)
	tokenEstimate := cc.EstimateTokens(newHistory)
	threshold := agent.ContextWindow * ContextWindowUsagePercent / 100

	if len(newHistory) > SummarizeMessageThreshold || tokenEstimate > threshold {
		summarizeKey := agent.ID + ":" + sessionKey
		if _, loading := cc.summarizing.LoadOrStore(summarizeKey, true); !loading {
			cc.wg.Add(1)
			go func() {
				defer cc.wg.Done()
				defer cc.summarizing.Delete(summarizeKey)
				if !constants.IsInternalChannel(channel) {
					pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer pubCancel()
					cc.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
						Channel: channel,
						ChatID:  chatID,
						Content: "Memory threshold reached. Optimizing conversation history...",
					})
				}
				cc.SummarizeSession(agent, sessionKey)
			}()
		}
	}
}

// ForceCompression aggressively reduces context when the limit is hit.
// It drops the oldest 50% of messages (keeping system prompt and last user message).
func (cc *ContextCompressor) ForceCompression(agent *AgentInstance, sessionKey string) {
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) <= MinHistoryForCompression {
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

	newHistory := make([]providers.Message, 0)

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

// SummarizeSession summarizes the conversation history for a session.
func (cc *ContextCompressor) SummarizeSession(agent *AgentInstance, sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), SummarizationTimeout)
	defer cancel()

	history := agent.Sessions.GetHistory(sessionKey)
	summary := agent.Sessions.GetSummary(sessionKey)

	// Keep last N messages for continuity
	if len(history) <= MessagesKeptAfterSummary {
		return
	}

	toSummarize := history[:len(history)-MessagesKeptAfterSummary]

	// Oversized Message Guard
	maxMessageTokens := agent.ContextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		// Use byte-based estimation (~2 bytes per token), matching original behavior.
		// For ASCII text this gives 0.5 chars/token; for CJK (3 bytes/rune) it
		// over-estimates slightly, which is the safer direction for the guard.
		msgTokens := len(m.Content) / 2
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	// Multi-Part Summarization
	var finalSummary string
	if len(validMessages) > MultiPartSummarizationThreshold {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := cc.SummarizeBatch(ctx, agent, part1, "")
		s2, _ := cc.SummarizeBatch(ctx, agent, part2, "")

		mergePrompt := fmt.Sprintf(
			"Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s",
			s1,
			s2,
		)
		resp, err := agent.Provider.Chat(
			ctx,
			[]providers.Message{{Role: "user", Content: mergePrompt}},
			nil,
			agent.Model,
			map[string]any{
				"max_tokens":  SummarizeMaxTokens,
				"temperature": SummarizeTemperature,
			},
		)
		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = cc.SummarizeBatch(ctx, agent, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		agent.Sessions.SetSummary(sessionKey, finalSummary)
		agent.Sessions.TruncateHistory(sessionKey, MessagesKeptAfterSummary)
		agent.Sessions.Save(sessionKey)
	}
}

// SummarizeBatch summarizes a batch of messages.
func (cc *ContextCompressor) SummarizeBatch(
	ctx context.Context,
	agent *AgentInstance,
	batch []providers.Message,
	existingSummary string,
) (string, error) {
	var sb strings.Builder
	sb.WriteString("Provide a concise summary of this conversation segment, preserving core context and key points.\n")
	if existingSummary != "" {
		sb.WriteString("Existing context: ")
		sb.WriteString(existingSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("\nCONVERSATION:\n")
	for _, m := range batch {
		fmt.Fprintf(&sb, "%s: %s\n", m.Role, m.Content)
	}
	prompt := sb.String()

	response, err := agent.Provider.Chat(
		ctx,
		[]providers.Message{{Role: "user", Content: prompt}},
		nil,
		agent.Model,
		map[string]any{
			"max_tokens":  SummarizeMaxTokens,
			"temperature": SummarizeTemperature,
		},
	)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

// EstimateTokens estimates the number of tokens in a message list.
// Uses byte length / 2 (~2 bytes per token) to match the original behavior.
func (cc *ContextCompressor) EstimateTokens(messages []providers.Message) int {
	totalBytes := 0
	for _, m := range messages {
		totalBytes += len(m.Content)
	}
	return totalBytes / 2
}
