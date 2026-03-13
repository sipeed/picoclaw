// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"jane/pkg/logger"
	"jane/pkg/providers"
)

// maybeSummarize triggers summarization if the session history exceeds thresholds.
func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
	newHistory := agent.Sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := agent.ContextWindow * agent.SummarizeTokenPercent / 100

	if len(newHistory) > agent.SummarizeMessageThreshold || tokenEstimate > threshold {
		summarizeKey := agent.ID + ":" + sessionKey
		if _, loading := al.summarizing.LoadOrStore(summarizeKey, true); !loading {
			// Send to background worker queue
			select {
			case al.summaryJobs <- summaryJob{agent: agent, sessionKey: sessionKey}:
				// job accepted
			default:
				// Worker queue is full, delete from map so it can be retried later
				logger.WarnCF("agent", "Summarization worker queue is full, skipping summarization", map[string]any{"session_key": sessionKey})
				al.summarizing.Delete(summarizeKey)
			}
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

	// Create logging directory: /logs/{session_id}/summarizer/
	logDir := filepath.Join("logs", sessionKey, "summarizer")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.WarnCF("agent", "Failed to create summarizer log directory", map[string]any{"error": err.Error()})
	}

	// Dump input context
	if b, err := json.MarshalIndent(history, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(logDir, "input_context.json"), b, 0644)
	}

	traceFile, _ := os.OpenFile(filepath.Join(logDir, "trace.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if traceFile != nil {
		defer traceFile.Close()
		fmt.Fprintf(traceFile, "[%s] Starting summarization for session: %s\n", time.Now().Format(time.RFC3339), sessionKey)
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

// summarizeSession summarizes the conversation history for a session.
func (al *AgentLoop) summarizeSession(agent *AgentInstance, sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	history := agent.Sessions.GetHistory(sessionKey)
	summary := agent.Sessions.GetSummary(sessionKey)

	// Keep last 4 messages for continuity
	if len(history) <= 4 {
		return
	}

	// Create logging directory: /logs/{session_id}/summarizer/
	logDir := filepath.Join("logs", sessionKey, "summarizer")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.WarnCF("agent", "Failed to create summarizer log directory", map[string]any{"error": err.Error()})
	}

	// Dump input context
	if b, err := json.MarshalIndent(history, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(logDir, "input_context.json"), b, 0644)
	}

	traceFile, _ := os.OpenFile(filepath.Join(logDir, "trace.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if traceFile != nil {
		defer traceFile.Close()
		fmt.Fprintf(traceFile, "[%s] Starting summarization for session: %s\n", time.Now().Format(time.RFC3339), sessionKey)
	}

	toSummarize := history[:len(history)-4]

	// Oversized Message Guard
	maxMessageTokens := agent.ContextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		msgTokens := len(m.Content) / 2
		if msgTokens > maxMessageTokens {
			omitted = true
			if traceFile != nil {
				fmt.Fprintf(traceFile, "[%s] Omitting oversized message from %s (length: %d)\n", time.Now().Format(time.RFC3339), m.Role, len(m.Content))
			}
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		if traceFile != nil {
			fmt.Fprintf(traceFile, "[%s] No valid messages to summarize.\n", time.Now().Format(time.RFC3339))
		}
		return
	}

	const (
		maxSummarizationMessages = 10
		llmMaxRetries            = 3
		llmTemperature           = 0.3
		fallbackMaxContentLength = 200
	)

	// Multi-Part Summarization
	var finalSummary string
	if len(validMessages) > maxSummarizationMessages {
		mid := len(validMessages) / 2

		mid = al.findNearestUserMessage(validMessages, mid)

		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, agent, part1, "")
		s2, _ := al.summarizeBatch(ctx, agent, part2, "")

		mergePrompt := fmt.Sprintf(
			"Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s",
			s1,
			s2,
		)

		if traceFile != nil {
			fmt.Fprintf(traceFile, "[%s] Merging multi-part summaries...\n", time.Now().Format(time.RFC3339))
		}

		resp, err := al.retryLLMCall(ctx, agent, mergePrompt, llmMaxRetries)
		if err == nil && resp.Content != "" {
			finalSummary = resp.Content
		} else {
			if traceFile != nil {
				fmt.Fprintf(traceFile, "[%s] LLM merge failed: %v. Falling back to concatenation.\n", time.Now().Format(time.RFC3339), err)
			}
			finalSummary = s1 + " " + s2
		}
	} else {
		if traceFile != nil {
			fmt.Fprintf(traceFile, "[%s] Summarizing single batch...\n", time.Now().Format(time.RFC3339))
		}
		finalSummary, _ = al.summarizeBatch(ctx, agent, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		if traceFile != nil {
			fmt.Fprintf(traceFile, "[%s] Summarization complete. Saving session.\n", time.Now().Format(time.RFC3339))
		}
		_ = os.WriteFile(filepath.Join(logDir, "result_summary.md"), []byte(finalSummary), 0644)

		agent.Sessions.SetSummary(sessionKey, finalSummary)
		agent.Sessions.TruncateHistory(sessionKey, 4)
		agent.Sessions.Save(sessionKey)
	} else {
		if traceFile != nil {
			fmt.Fprintf(traceFile, "[%s] Final summary is empty, skipping save.\n", time.Now().Format(time.RFC3339))
		}
	}
}

// findNearestUserMessage finds the nearest user message to the given index.
// It searches backward first, then forward if no user message is found.
func (al *AgentLoop) findNearestUserMessage(messages []providers.Message, mid int) int {
	originalMid := mid

	for mid > 0 && messages[mid].Role != "user" {
		mid--
	}

	if messages[mid].Role == "user" {
		return mid
	}

	mid = originalMid
	for mid < len(messages) && messages[mid].Role != "user" {
		mid++
	}

	if mid < len(messages) {
		return mid
	}

	return originalMid
}

// retryLLMCall calls the LLM with retry logic.
func (al *AgentLoop) retryLLMCall(
	ctx context.Context,
	agent *AgentInstance,
	prompt string,
	maxRetries int,
) (*providers.LLMResponse, error) {
	const (
		llmTemperature = 0.3
	)

	var resp *providers.LLMResponse
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = agent.Provider.Chat(
			ctx,
			[]providers.Message{{Role: "user", Content: prompt}},
			nil,
			agent.Model,
			map[string]any{
				"max_tokens":       agent.MaxTokens,
				"temperature":      llmTemperature,
				"prompt_cache_key": agent.ID,
			},
		)
		if err == nil && resp != nil && resp.Content != "" {
			return resp, nil
		}
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	return resp, err
}

// summarizeBatch summarizes a batch of messages.
func (al *AgentLoop) summarizeBatch(
	ctx context.Context,
	agent *AgentInstance,
	batch []providers.Message,
	existingSummary string,
) (string, error) {
	const (
		llmMaxRetries             = 3
		llmTemperature            = 0.3
		fallbackMinContentLength  = 200
		fallbackMaxContentPercent = 10
	)

	var sb strings.Builder
	sb.WriteString(
		"Provide a concise summary of this conversation segment, preserving core context and key points.\n",
	)
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

	response, err := al.retryLLMCall(ctx, agent, prompt, llmMaxRetries)
	if err == nil && response.Content != "" {
		return strings.TrimSpace(response.Content), nil
	}

	var fallback strings.Builder
	fallback.WriteString("Conversation summary: ")
	for i, m := range batch {
		if i > 0 {
			fallback.WriteString(" | ")
		}
		content := strings.TrimSpace(m.Content)
		runes := []rune(content)
		if len(runes) == 0 {
			fallback.WriteString(fmt.Sprintf("%s: ", m.Role))
			continue
		}

		keepLength := len(runes) * fallbackMaxContentPercent / 100
		if keepLength < fallbackMinContentLength {
			keepLength = fallbackMinContentLength
		}

		if keepLength > len(runes) {
			keepLength = len(runes)
		}

		content = string(runes[:keepLength])
		if keepLength < len(runes) {
			content += "..."
		}
		fallback.WriteString(fmt.Sprintf("%s: %s", m.Role, content))
	}
	return fallback.String(), nil
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
