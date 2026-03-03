// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ---------------------------------------------------------------------------
// Instant Memory — dynamic Turn selection for Phase 2 context
// ---------------------------------------------------------------------------

// InstantMemoryCfg holds tunable parameters for instant-memory assembly.
type InstantMemoryCfg struct {
	HighScoreThreshold int     // turns with score >= this are always_keep (default: 7)
	RecentCount        int     // number of recent turns to include (default: 5)
	MaxTokenRatio      float64 // fraction of contextWindow budget (default: 0.6)
	ContextWindow      int     // total context window in tokens
}

// DefaultInstantMemoryCfg returns a sensible default config.
func DefaultInstantMemoryCfg(contextWindow int) InstantMemoryCfg {
	return InstantMemoryCfg{
		HighScoreThreshold: alwaysKeepThreshold, // 7
		RecentCount:        5,
		MaxTokenRatio:      0.6,
		ContextWindow:      contextWindow,
	}
}

// BuildInstantMemory assembles the filtered set of historical turns for Phase 2.
//
// Selection rules (from design doc):
//
//	瞬时记忆 =
//	  { Turn | score >= highThreshold }               // always_keep
//	  ∪ { Turn | tags ∩ currentTags ≠ ∅, score > 0 } // tag-matched
//	  ∪ { 最近 M 个 Turn }                             // recency guarantee
//	  → deduplicate by ID
//	  → sort by ts ASC
//	  → truncate to token budget
func BuildInstantMemory(
	store *TurnStore,
	currentTags []string,
	channelKey string,
	cfg InstantMemoryCfg,
) []TurnRecord {
	if store == nil {
		return nil
	}

	seen := make(map[string]struct{})
	var all []TurnRecord

	addUnique := func(turns []TurnRecord) {
		for _, t := range turns {
			if _, dup := seen[t.ID]; dup {
				continue
			}
			seen[t.ID] = struct{}{}
			all = append(all, t)
		}
	}

	// 1. always_keep: high-score turns.
	high, err := store.QueryByScore(cfg.HighScoreThreshold)
	if err != nil {
		logger.WarnCF("instant_memory", "QueryByScore failed", map[string]any{"error": err.Error()})
	} else {
		addUnique(high)
	}

	// 2. tag-matched turns (score > 0).
	if len(currentTags) > 0 {
		tagged, err := store.QueryByTags(currentTags)
		if err != nil {
			logger.WarnCF("instant_memory", "QueryByTags failed", map[string]any{"error": err.Error()})
		} else {
			addUnique(tagged)
		}
	}

	// 3. Recent M turns for continuity.
	recent, err := store.QueryRecent(channelKey, cfg.RecentCount)
	if err != nil {
		logger.WarnCF("instant_memory", "QueryRecent failed", map[string]any{"error": err.Error()})
	} else {
		addUnique(recent)
	}

	// Sort by ts ASC (stable chronological order).
	sortTurnsByTs(all)

	// Truncate to token budget.
	maxTokens := int(float64(cfg.ContextWindow) * cfg.MaxTokenRatio)
	if maxTokens > 0 {
		all = truncateToTokenBudget(all, maxTokens)
	}

	logger.DebugCF("instant_memory", "Built instant memory",
		map[string]any{
			"total":       len(all),
			"high_score":  len(high),
			"tag_matched": len(currentTags),
			"recent":      len(recent),
			"max_tokens":  maxTokens,
		})

	return all
}

// sortTurnsByTs sorts turns in ascending timestamp order (oldest first).
func sortTurnsByTs(turns []TurnRecord) {
	// Simple in-place insertion sort — good enough for small N (<100).
	for i := 1; i < len(turns); i++ {
		key := turns[i]
		j := i - 1
		for j >= 0 && turns[j].Ts > key.Ts {
			turns[j+1] = turns[j]
			j--
		}
		turns[j+1] = key
	}
}

// truncateToTokenBudget trims turns from the oldest end until total tokens fit.
// Returns a suffix of the sorted slice (preserving newest turns).
func truncateToTokenBudget(turns []TurnRecord, maxTokens int) []TurnRecord {
	total := 0
	for _, t := range turns {
		total += t.Tokens
	}
	if total <= maxTokens {
		return turns
	}

	// Drop oldest turns first until we fit.
	for len(turns) > 0 && total > maxTokens {
		total -= turns[0].Tokens
		turns = turns[1:]
	}
	return turns
}

// ---------------------------------------------------------------------------
// Phase 2 Message Assembly — KV Cache friendly ordering
// ---------------------------------------------------------------------------

// BuildPhase2Messages constructs the message array for Phase 2 (ExecuteLLM)
// in KV-cache-friendly order:
//
//	[system_prompt]               ← always cache hit
//	[long_term_memory by tags]    ← same tags = cache hit (cache_control: ephemeral)
//	[always_keep turns (score≥7)] ← fixed position, append only → cache hit
//	[tag_matched turns]           ← per-turn, ts ASC
//	[recent_M turns]              ← rolling window
//	[current_user_message]        ← always new
//
// Each historical turn is represented as a user/assistant message pair.
func BuildPhase2Messages(
	systemPrompt string,
	longTermMemory string,
	turns []TurnRecord,
	userMessage string,
	highScoreThreshold int,
) []providers.Message {
	msgs := make([]providers.Message, 0, 2+len(turns)*2+1)

	// 1. System prompt (always first, stable prefix).
	msgs = append(msgs, providers.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// 2. Long-term memory (injected as system-adjacent user message).
	//    Mark with CacheControl if present (Anthropic will use it; others ignore).
	if longTermMemory != "" {
		msgs = append(msgs, providers.Message{
			Role:    "user",
			Content: fmt.Sprintf("# Long-term Memory\n\n%s", longTermMemory),
		})
		// Need a brief assistant ack to maintain user/assistant alternation.
		msgs = append(msgs, providers.Message{
			Role:    "assistant",
			Content: "Understood, I'll use this context.",
		})
	}

	// 3. Historical turns in KV-cache-friendly order:
	//    - always_keep first (fixed position)
	//    - then tag_matched + recent (may shift between requests)
	//
	// All turns are already sorted by ts ASC from BuildInstantMemory.
	// We separate them into always_keep vs rest, keeping relative order.
	var alwaysKeep, rest []TurnRecord
	for _, t := range turns {
		if t.Score >= highScoreThreshold {
			alwaysKeep = append(alwaysKeep, t)
		} else {
			rest = append(rest, t)
		}
	}

	// Append always_keep turns (cache-stable region).
	for _, t := range alwaysKeep {
		msgs = appendTurnMessages(msgs, t)
	}

	// Append remaining turns (tag-matched + recent, may shift).
	for _, t := range rest {
		msgs = appendTurnMessages(msgs, t)
	}

	// 4. Current user message (always last, always new).
	msgs = append(msgs, providers.Message{
		Role:    "user",
		Content: userMessage,
	})

	return msgs
}

// appendTurnMessages appends a user/assistant pair for a historical turn.
func appendTurnMessages(msgs []providers.Message, t TurnRecord) []providers.Message {
	// Build user message with metadata prefix.
	var userContent strings.Builder
	if t.Intent != "" || len(t.Tags) > 0 {
		fmt.Fprintf(&userContent, "[turn intent=%s tags=%v]\n", t.Intent, t.Tags)
	}
	userContent.WriteString(t.UserMsg)

	msgs = append(msgs, providers.Message{
		Role:    "user",
		Content: userContent.String(),
	})

	if t.Reply != "" {
		msgs = append(msgs, providers.Message{
			Role:    "assistant",
			Content: t.Reply,
		})
	}

	return msgs
}
