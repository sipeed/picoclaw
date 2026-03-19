package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// sessionSemaphore is a per-session mutex using a buffered channel.

type sessionSemaphore struct {
	ch chan struct{}
}

func newSessionSemaphore() *sessionSemaphore {
	s := &sessionSemaphore{ch: make(chan struct{}, 1)}

	s.ch <- struct{}{} // initially unlocked

	return s
}

func (al *AgentLoop) gcLoop() {
	ticker := time.NewTicker(30 * time.Minute)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			al.gcSessionLocks()
			al.pruneMediaCache()

		case <-al.done:

			return
		}
	}
}

// gcSessionLocks removes unlocked (idle) sessionSemaphore entries from the map.

func (al *AgentLoop) gcSessionLocks() {
	al.sessionLocks.Range(func(key, val any) bool {
		sem := val.(*sessionSemaphore)

		select {
		case <-sem.ch:

			// Was unlocked — safe to remove

			al.sessionLocks.Delete(key)

		default:

			// Currently locked — in use, keep
		}

		return true
	})
}

func (al *AgentLoop) acquireSessionLock(ctx context.Context, sessionKey string) bool {
	val, _ := al.sessionLocks.LoadOrStore(sessionKey, newSessionSemaphore())

	sem := val.(*sessionSemaphore)

	select {
	case <-sem.ch:

		return true

	case <-ctx.Done():

		return false
	}
}

// releaseSessionLock releases the per-session semaphore.

func (al *AgentLoop) releaseSessionLock(sessionKey string) {
	if val, ok := al.sessionLocks.Load(sessionKey); ok {
		sem := val.(*sessionSemaphore)

		sem.ch <- struct{}{}
	}
}

func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
	newHistory := agent.Sessions.GetHistory(sessionKey)

	tokenEstimate := al.estimateTokens(newHistory)

	threshold := agent.ContextWindow * 75 / 100

	if len(newHistory) > 20 || tokenEstimate > threshold {
		summarizeKey := agent.ID + ":" + sessionKey

		if _, loading := al.summarizing.LoadOrStore(summarizeKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(summarizeKey)

				logger.InfoCF("agent", "Memory threshold reached, optimizing conversation history",

					map[string]any{
						"session_key": sessionKey,

						"history_len": len(newHistory),

						"token_estimate": tokenEstimate,
					})

				al.summarizeSession(agent, sessionKey)
			}()
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
		"session_key": sessionKey,

		"dropped_msgs": droppedCount,

		"new_count": len(newHistory),
	})
}

func (al *AgentLoop) summarizeSession(agent *AgentInstance, sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	defer cancel()

	history := agent.Sessions.GetHistory(sessionKey)

	summary := agent.Sessions.GetSummary(sessionKey)

	// Keep last 4 messages for continuity

	if len(history) <= 4 {
		return
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

			continue
		}

		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	// Multi-Part Summarization

	var finalSummary string

	if len(validMessages) > 10 {
		mid := len(validMessages) / 2

		part1 := validMessages[:mid]

		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, agent, part1, "")

		s2, _ := al.summarizeBatch(ctx, agent, part2, "")

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
				"max_tokens": 1024,

				"temperature": 0.3,

				"prompt_cache_key": agent.ID,
			},
		)

		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = al.summarizeBatch(ctx, agent, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		if err := agent.Sessions.CompactOldTurns(sessionKey, 4, finalSummary); err != nil {
			logger.ErrorCF("agent", "CompactOldTurns failed, falling back",

				map[string]any{"error": err.Error()})

			agent.Sessions.SetSummary(sessionKey, finalSummary)

			agent.Sessions.TruncateHistory(sessionKey, 4)

			agent.Sessions.Save(sessionKey)
		}
	}
}

// summarizeBatch summarizes a batch of messages.

func (al *AgentLoop) summarizeBatch(
	ctx context.Context,

	agent *AgentInstance,

	batch []providers.Message,

	existingSummary string,
) (string, error) {
	var sb strings.Builder

	sb.WriteString("Provide a concise summary of this conversation segment, preserving core context and key points.\n")

	if agent.ContextBuilder.HasActivePlan() {
		sb.WriteString("Note: Active plan in MEMORY.md. Preserve plan progress references.\n")
	}

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
			"max_tokens": 1024,

			"temperature": 0.3,

			"prompt_cache_key": agent.ID,
		},
	)
	if err != nil {
		return "", err
	}

	return response.Content, nil
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
