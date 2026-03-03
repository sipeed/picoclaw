// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// MemoryDigestWorker runs as a background goroutine and periodically extracts
// long-term memories from pending TurnRecords.
//
// Design:
//   - Fixed interval trigger (default 5 minutes).
//   - No llmActive yield mechanism (personal agent, low QPS, API rate-limits handle it).
//   - Processes up to 50 pending turns per cycle, grouped by channel_key.
//   - On completion, marks turns as "processed" and archives old processed turns.
type MemoryDigestWorker struct {
	store    *TurnStore
	memory   *MemoryStore
	provider providers.LLMProvider
	model    string
	interval time.Duration
}

// MemoryDigestConfig holds tunable parameters.
type MemoryDigestConfig struct {
	Interval       time.Duration // Polling period (default: 5 minutes)
	BatchLimit     int           // Max pending turns per cycle (default: 50)
	ArchiveAfterDays int         // Archive processed turns older than N days (default: 7)
}

func defaultDigestConfig() MemoryDigestConfig {
	return MemoryDigestConfig{
		Interval:         5 * time.Minute,
		BatchLimit:       50,
		ArchiveAfterDays: 7,
	}
}

// NewMemoryDigestWorker creates a worker. provider/model may be nil/empty
// if only archival (no LLM extraction) is desired.
func NewMemoryDigestWorker(
	store *TurnStore,
	memory *MemoryStore,
	provider providers.LLMProvider,
	model string,
) *MemoryDigestWorker {
	return &MemoryDigestWorker{
		store:    store,
		memory:   memory,
		provider: provider,
		model:    model,
		interval: defaultDigestConfig().Interval,
	}
}

// SetInterval overrides the polling interval (e.g. for testing).
func (w *MemoryDigestWorker) SetInterval(d time.Duration) {
	w.interval = d
}

// Start launches the background goroutine. It respects ctx cancellation.
func (w *MemoryDigestWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.runOnce(ctx); err != nil {
					logger.WarnCF("memory_digest", "runOnce error", map[string]any{"error": err.Error()})
				}
			}
		}
	}()
	logger.DebugCF("memory_digest", "Worker started", map[string]any{"interval": w.interval.String()})
}

// RunOnceNow triggers an immediate digest cycle (useful for testing).
func (w *MemoryDigestWorker) RunOnceNow(ctx context.Context) error {
	return w.runOnce(ctx)
}

// runOnce executes one full digest cycle.
func (w *MemoryDigestWorker) runOnce(ctx context.Context) error {
	if w.store == nil {
		return nil
	}
	cfg := defaultDigestConfig()

	// Step 1: Load pending turns.
	pending, err := w.store.QueryPending(cfg.BatchLimit)
	if err != nil {
		return fmt.Errorf("query pending: %w", err)
	}
	if len(pending) == 0 {
		logger.DebugCF("memory_digest", "No pending turns", nil)
		// Still run archival.
		return w.archive(cfg)
	}

	logger.DebugCF("memory_digest", "Processing pending turns",
		map[string]any{"count": len(pending)})

	// Step 2: Group by channel_key to avoid mixing user memories.
	groups := make(map[string][]TurnRecord)
	for _, t := range pending {
		groups[t.ChannelKey] = append(groups[t.ChannelKey], t)
	}

	// Step 3: For each group, call LLM to extract memories.
	for channelKey, turns := range groups {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := w.processGroup(ctx, channelKey, turns); err != nil {
			logger.WarnCF("memory_digest", "Group processing error",
				map[string]any{"channel": channelKey, "error": err.Error()})
			// Continue with other groups.
		}
	}

	// Step 6: Archive old processed turns.
	return w.archive(cfg)
}

// processGroup extracts memories from a batch of turns belonging to one channel.
func (w *MemoryDigestWorker) processGroup(ctx context.Context, channelKey string, turns []TurnRecord) error {
	// Build a conversation digest for the LLM.
	memories, err := w.extractMemories(ctx, turns)
	if err != nil {
		// Mark them as processed anyway so we don't loop forever.
		logger.WarnCF("memory_digest", "LLM extraction failed, marking as processed",
			map[string]any{"channel": channelKey, "error": err.Error()})
	}

	// Step 4: Write extracted memories.
	if w.memory != nil {
		for _, m := range memories {
			if _, addErr := w.memory.AddEntry(m.Content, m.Tags); addErr != nil {
				logger.WarnCF("memory_digest", "Failed to save memory",
					map[string]any{"error": addErr.Error()})
			}
		}
	}

	// Step 5: Mark all turns as processed.
	for _, t := range turns {
		if setErr := w.store.SetStatus(t.ID, "processed"); setErr != nil {
			logger.WarnCF("memory_digest", "SetStatus failed",
				map[string]any{"id": t.ID, "error": setErr.Error()})
		}
	}

	logger.DebugCF("memory_digest", "Group processed",
		map[string]any{
			"channel":         channelKey,
			"turns":           len(turns),
			"memories_stored": len(memories),
		})
	return nil
}

// digestMemoryResult holds one extracted memory item.
type digestMemoryResult struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

const digestPrompt = `Extract important, durable facts worth remembering from these conversation turns.

Conversation turns:
%s

Respond with ONLY JSON: {"memories": [{"content": "<fact>", "tags": ["tag1"]}]}
Rules:
- max 5 memories total across all turns
- max 3 tags each, lowercase
- skip trivial small-talk
- prefer facts about user preferences, environment, recurring patterns, important decisions
- if nothing worth remembering: {"memories": []}`

// extractMemories calls the LLM to distil memories from a batch of turns.
// Returns nil memories (not error) when the LLM is unconfigured.
func (w *MemoryDigestWorker) extractMemories(ctx context.Context, turns []TurnRecord) ([]digestMemoryResult, error) {
	if w.provider == nil || w.model == "" {
		return nil, nil
	}

	// Build conversation summary for the prompt.
	var sb strings.Builder
	for i, t := range turns {
		reply := t.Reply
		if len(reply) > 500 {
			reply = reply[:500] + "..."
		}
		fmt.Fprintf(&sb, "=== Turn %d (intent: %s, tags: %v) ===\nUser: %s\nAssistant: %s\n\n",
			i+1, t.Intent, t.Tags, t.UserMsg, reply)
	}
	prompt := fmt.Sprintf(digestPrompt, sb.String())

	resp, err := w.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, nil, w.model, map[string]any{"max_tokens": 512, "temperature": 0.1})
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}

	raw := strings.TrimSpace(resp.Content)
	// Strip markdown fences if present.
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var result struct {
		Memories []digestMemoryResult `json:"memories"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// Parsing failure — skip extraction, don't fail the whole batch.
		logger.WarnCF("memory_digest", "Failed to parse LLM response",
			map[string]any{"raw": raw[:min(len(raw), 200)], "error": err.Error()})
		return nil, nil
	}

	// Normalise.
	out := make([]digestMemoryResult, 0, len(result.Memories))
	for _, m := range result.Memories {
		m.Content = strings.TrimSpace(m.Content)
		if m.Content == "" {
			continue
		}
		normalised := make([]string, 0, len(m.Tags))
		for _, t := range m.Tags {
			t = strings.ToLower(strings.TrimSpace(t))
			if t != "" {
				normalised = append(normalised, t)
			}
		}
		m.Tags = normalised
		out = append(out, m)
	}
	return out, nil
}

// archive runs periodic archival of processed turns.
func (w *MemoryDigestWorker) archive(cfg MemoryDigestConfig) error {
	if w.store == nil {
		return nil
	}
	if err := w.store.ArchiveOldProcessed(cfg.ArchiveAfterDays); err != nil {
		return fmt.Errorf("archive: %w", err)
	}
	return nil
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
