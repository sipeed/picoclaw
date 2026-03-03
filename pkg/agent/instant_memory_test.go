package agent

import (
	"strings"
	"testing"
	"time"
)

func TestBuildInstantMemory_BasicAssembly(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTurnStore(dir)
	if err != nil {
		t.Fatalf("NewTurnStore: %v", err)
	}
	defer store.Close()

	now := time.Now().Unix()

	// High-score turn (always_keep).
	store.Insert(TurnRecord{ID: "t1", Ts: now - 100, Score: 9, ChannelKey: "cli:direct",
		Intent: "code", Tags: []string{"refactor"}, UserMsg: "refactor it", Reply: strings.Repeat("x", 300)})

	// Low-score irrelevant turn.
	store.Insert(TurnRecord{ID: "t2", Ts: now - 80, Score: 2, ChannelKey: "cli:direct",
		Intent: "chat", Tags: []string{"chat"}, UserMsg: "hi", Reply: "hello"})

	// Tag-matched turn, moderate score.
	store.Insert(TurnRecord{ID: "t3", Ts: now - 60, Score: 5, ChannelKey: "cli:direct",
		Intent: "task", Tags: []string{"deploy", "ci"}, UserMsg: "deploy staging", Reply: "done"})

	// Recent turns.
	store.Insert(TurnRecord{ID: "t4", Ts: now - 20, Score: 3, ChannelKey: "cli:direct",
		Intent: "question", Tags: []string{"api"}, UserMsg: "what's the api?", Reply: "check docs"})
	store.Insert(TurnRecord{ID: "t5", Ts: now - 10, Score: 4, ChannelKey: "cli:direct",
		Intent: "task", Tags: []string{"test"}, UserMsg: "run tests", Reply: "all passed"})

	cfg := InstantMemoryCfg{
		HighScoreThreshold: 7,
		RecentCount:        3,
		MaxTokenRatio:      0.6,
		ContextWindow:      100000,
	}

	turns := BuildInstantMemory(store, []string{"deploy"}, "cli:direct", cfg)

	// Should include: t1 (high-score), t3 (tag-match "deploy"), t4/t5 (recent 3 → also t3)
	if len(turns) < 3 {
		t.Errorf("expected at least 3 turns, got %d", len(turns))
		for _, tt := range turns {
			t.Logf("  turn: id=%s score=%d tags=%v", tt.ID, tt.Score, tt.Tags)
		}
	}

	// Should be sorted by ts ASC.
	for i := 1; i < len(turns); i++ {
		if turns[i].Ts < turns[i-1].Ts {
			t.Errorf("turns not sorted: turns[%d].Ts=%d < turns[%d].Ts=%d",
				i, turns[i].Ts, i-1, turns[i-1].Ts)
		}
	}

	// t1 (always_keep) must be present.
	found := false
	for _, tt := range turns {
		if tt.ID == "t1" {
			found = true
		}
	}
	if !found {
		t.Error("expected always_keep turn t1 to be included")
	}

	// t2 (low-score, no tag match, not recent enough) should be excluded.
	for _, tt := range turns {
		if tt.ID == "t2" {
			t.Error("expected low-score irrelevant turn t2 to be excluded")
		}
	}
}

func TestBuildInstantMemory_NilStore(t *testing.T) {
	turns := BuildInstantMemory(nil, []string{"deploy"}, "cli:direct", DefaultInstantMemoryCfg(8192))
	if turns != nil {
		t.Errorf("expected nil, got %v", turns)
	}
}

func TestBuildPhase2Messages_Ordering(t *testing.T) {
	turns := []TurnRecord{
		{ID: "t1", Ts: 100, Score: 9, Intent: "code", Tags: []string{"refactor"},
			UserMsg: "refactor it", Reply: "done refactoring", Tokens: 20},
		{ID: "t2", Ts: 200, Score: 3, Intent: "question",
			UserMsg: "what next?", Reply: "do X", Tokens: 10},
		{ID: "t3", Ts: 300, Score: 8, Intent: "debug", Tags: []string{"deploy"},
			UserMsg: "fix deploy", Reply: "fixed", Tokens: 10},
	}

	msgs := BuildPhase2Messages("You are a helpful assistant.", "User prefers Go.", turns, "hello world", 7)

	// Expected order:
	// [0] system
	// [1] user (long_term_memory)
	// [2] assistant (ack)
	// [3,4] always_keep t1 (user/assistant)
	// [5,6] always_keep t3 (user/assistant)
	// [7,8] rest t2 (user/assistant)
	// [9] current user message
	if len(msgs) < 5 {
		t.Fatalf("expected at least 5 messages, got %d", len(msgs))
	}

	if msgs[0].Role != "system" {
		t.Errorf("msgs[0].Role = %s, want system", msgs[0].Role)
	}

	// Last message should be the current user message.
	last := msgs[len(msgs)-1]
	if last.Role != "user" || last.Content != "hello world" {
		t.Errorf("last message = %+v, want user 'hello world'", last)
	}

	// All messages should alternate user/assistant (after system).
	for i := 1; i < len(msgs)-1; i++ {
		expected := "user"
		if i%2 == 0 {
			expected = "assistant"
		}
		if msgs[i].Role != expected {
			t.Errorf("msgs[%d].Role = %s, want %s (content: %s)",
				i, msgs[i].Role, expected, msgs[i].Content[:min(len(msgs[i].Content), 30)])
		}
	}
}

func TestBuildPhase2Messages_NoHistory(t *testing.T) {
	msgs := BuildPhase2Messages("sys prompt", "", nil, "hi", 7)

	// Should have: system + user message = 2
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[1].Role != "user" {
		t.Errorf("unexpected roles: %s, %s", msgs[0].Role, msgs[1].Role)
	}
}

func TestTruncateToTokenBudget(t *testing.T) {
	turns := []TurnRecord{
		{ID: "a", Tokens: 100},
		{ID: "b", Tokens: 200},
		{ID: "c", Tokens: 300},
		{ID: "d", Tokens: 150},
	}
	result := truncateToTokenBudget(turns, 500)
	// Total = 750, budget = 500. Drop oldest first.
	// Drop "a" (100) → 650, still over.
	// Drop "b" (200) → 450, fits.
	if len(result) != 2 {
		t.Errorf("expected 2 turns, got %d", len(result))
	}
	if result[0].ID != "c" || result[1].ID != "d" {
		t.Errorf("expected [c, d], got [%s, %s]", result[0].ID, result[1].ID)
	}
}
