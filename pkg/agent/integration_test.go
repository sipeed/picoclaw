// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// M5 Integration — TurnStore → BuildInstantMemory → BuildPhase2Messages
// ---------------------------------------------------------------------------

// TestInstantMemoryIntegration_EndToEnd inserts realistic turns into a real
// TurnStore, runs BuildInstantMemory with tag filtering, then assembles Phase 2
// messages and validates:
//   - correct message ordering (system → memory → always_keep → rest → user)
//   - strict user/assistant role alternation after the system message
//   - always_keep turns appear before lower-score turns
//   - the current user message is always last
func TestInstantMemoryIntegration_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer store.Close()

	now := time.Now().Unix()

	// Seed realistic turns.
	turns := []TurnRecord{
		{ID: "turn-1", Ts: now - 3600, Score: 10, ChannelKey: "cli:main",
			Intent: "task", Tags: []string{"deploy", "ci"},
			UserMsg: "Deploy to staging", Reply: "Deployed successfully to staging environment.",
			Tokens: 50},
		{ID: "turn-2", Ts: now - 3000, Score: 2, ChannelKey: "cli:main",
			Intent: "chat", Tags: []string{"chat"},
			UserMsg: "hi", Reply: "Hello!",
			Tokens: 10},
		{ID: "turn-3", Ts: now - 2000, Score: 6, ChannelKey: "cli:main",
			Intent: "code", Tags: []string{"golang", "refactor"},
			UserMsg: "Refactor the handler", Reply: "Done, split into 3 functions.",
			Tokens: 40},
		{ID: "turn-4", Ts: now - 500, Score: 4, ChannelKey: "cli:main",
			Intent: "question", Tags: []string{"api"},
			UserMsg: "What's the endpoint for users?", Reply: "GET /api/v1/users",
			Tokens: 20},
		{ID: "turn-5", Ts: now - 100, Score: 3, ChannelKey: "cli:main",
			Intent: "task", Tags: []string{"test"},
			UserMsg: "Run all tests", Reply: "All 42 tests passed.",
			Tokens: 15},
	}
	for _, tr := range turns {
		if err := store.Insert(tr); err != nil {
			t.Fatalf("Insert(%s): %v", tr.ID, err)
		}
	}

	// Query with tags=["deploy"] — should get turn-1 (always_keep + tag match),
	// turn-3/4/5 (recent 3). turn-2 is low score, no tag match, not recent.
	cfg := InstantMemoryCfg{
		HighScoreThreshold: 7,
		RecentCount:        3,
		MaxTokenRatio:      0.6,
		ContextWindow:      100000,
	}
	selected := BuildInstantMemory(store, []string{"deploy"}, "cli:main", cfg)

	// Verify turn-1 is selected (always_keep).
	hasT1 := false
	for _, s := range selected {
		if s.ID == "turn-1" {
			hasT1 = true
		}
	}
	if !hasT1 {
		t.Error("expected always_keep turn-1 to be selected")
	}

	// Verify turn-2 is NOT selected.
	for _, s := range selected {
		if s.ID == "turn-2" {
			t.Error("expected low-score turn-2 to be excluded")
		}
	}

	// Assemble Phase 2 messages.
	systemPrompt := "You are a helpful assistant.\n\n## Runtime\nlinux amd64"
	longTermMemory := "User prefers Go. User's name is Alice."
	currentMsg := "Deploy to production now"

	msgs := BuildPhase2Messages(systemPrompt, longTermMemory, selected, currentMsg, cfg.HighScoreThreshold)

	// --- Validate message structure ---

	// 1. First message is system.
	if msgs[0].Role != "system" {
		t.Fatalf("msgs[0].Role = %s, want system", msgs[0].Role)
	}
	if !strings.Contains(msgs[0].Content, "helpful assistant") {
		t.Error("system message should contain prompt text")
	}

	// 2. Last message is current user message.
	last := msgs[len(msgs)-1]
	if last.Role != "user" || last.Content != currentMsg {
		t.Errorf("last message = role=%s content=%q, want user %q", last.Role, last.Content, currentMsg)
	}

	// 3. Role alternation: after system, messages must alternate user/assistant.
	for i := 1; i < len(msgs); i++ {
		expectedRole := "user"
		if i%2 == 0 {
			expectedRole = "assistant"
		}
		if msgs[i].Role != expectedRole {
			t.Errorf("msgs[%d].Role = %s, want %s (content: %.50s...)",
				i, msgs[i].Role, expectedRole, msgs[i].Content)
		}
	}

	// 4. Long-term memory should be in msgs[1] (user role).
	if !strings.Contains(msgs[1].Content, "Long-term Memory") {
		t.Error("msgs[1] should contain long-term memory")
	}

	// 5. Always_keep turns (score >= 7) should appear before lower-score turns.
	alwaysKeepEnd := -1
	restStart := len(msgs)
	for i := 3; i < len(msgs)-1; i += 2 { // user messages from turns, skip system+memory+ack
		content := msgs[i].Content
		// Check if this is an always_keep turn by looking for turn-1 content.
		if strings.Contains(content, "Deploy to staging") {
			alwaysKeepEnd = i
		}
	}
	for i := 3; i < len(msgs)-1; i += 2 {
		content := msgs[i].Content
		// First non-always-keep turn.
		if !strings.Contains(content, "Deploy to staging") && !strings.Contains(content, "Long-term Memory") {
			restStart = i
			break
		}
	}
	if alwaysKeepEnd >= 0 && restStart < len(msgs) && alwaysKeepEnd > restStart {
		t.Errorf("always_keep turns should come before rest: alwaysKeepEnd=%d, restStart=%d",
			alwaysKeepEnd, restStart)
	}

	t.Logf("Phase 2 assembled %d messages from %d selected turns", len(msgs), len(selected))
	for i, m := range msgs {
		preview := m.Content
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		t.Logf("  [%d] role=%-10s content=%q", i, m.Role, preview)
	}
}

// ---------------------------------------------------------------------------
// M4 Integration — MemoryDigest runOnce
// ---------------------------------------------------------------------------

// TestMemoryDigestIntegration_RunOnce inserts pending TurnRecords, runs
// MemoryDigest.runOnce with a mock LLM, and verifies:
//   - TurnRecords are transitioned from "pending" to "processed"
//   - MemoryStore receives new entries from the LLM extraction
func TestMemoryDigestIntegration_RunOnce(t *testing.T) {
	dir := t.TempDir()

	turnStore, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer turnStore.Close()

	memStore := NewMemoryStore(dir)
	defer memStore.Close()

	now := time.Now().Unix()

	// Insert pending turns.
	for i := 0; i < 3; i++ {
		tr := TurnRecord{
			ID:         "digest-" + string(rune('a'+i)),
			Ts:         now - int64(300*(3-i)),
			Score:      5,
			ChannelKey: "cli:main",
			Intent:     "task",
			Tags:       []string{"golang"},
			UserMsg:    "Do task " + string(rune('A'+i)),
			Reply:      "Done with task " + string(rune('A'+i)),
			Tokens:     30,
			Status:     "pending",
		}
		if err := turnStore.Insert(tr); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	// Verify pending.
	pending, _ := turnStore.QueryPending(50)
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending, got %d", len(pending))
	}

	// Create a mock provider that returns a memory extraction response.
	mp := &mockLLMProvider{
		response: `{"memories": [{"content": "User worked on Go tasks A, B, C", "tags": ["golang", "task"]}]}`,
	}

	// Create and run MemoryDigest.
	worker := NewMemoryDigestWorker(turnStore, memStore, mp, "test-model")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	worker.runOnce(ctx)

	// Verify turns are now processed.
	pendingAfter, _ := turnStore.QueryPending(50)
	if len(pendingAfter) != 0 {
		t.Errorf("expected 0 pending after runOnce, got %d", len(pendingAfter))
	}

	// Verify memory store has entries.
	memCtx := memStore.GetMemoryContext()
	if memCtx == "" {
		t.Error("expected MemoryStore to have entries after digest, got empty")
	} else {
		t.Logf("MemoryStore context after digest:\n%s", memCtx)
	}
}
