package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

func TestSessionsListTool_BasicAndKindFilter(t *testing.T) {
	sm := session.NewSessionManager(t.TempDir())
	sm.AddMessage("agent:main:main", "user", "hello")
	sm.AddMessage("cron:daily-report", "assistant", "done")

	tool := NewSessionsListTool(sm)

	// Basic list
	result := tool.Execute(context.Background(), map[string]any{"limit": 10})
	if result.IsError {
		t.Fatalf("sessions_list returned error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Fatalf("sessions_list should be silent")
	}

	var payload struct {
		Count    int `json:"count"`
		Sessions []struct {
			Key  string `json:"key"`
			Kind string `json:"kind"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("failed to decode sessions_list payload: %v", err)
	}
	if payload.Count != 2 {
		t.Fatalf("expected 2 sessions, got %d", payload.Count)
	}

	kindByKey := map[string]string{}
	for _, s := range payload.Sessions {
		kindByKey[s.Key] = s.Kind
	}
	if kindByKey["agent:main:main"] != "main" {
		t.Fatalf("expected main kind for agent:main:main, got %q", kindByKey["agent:main:main"])
	}
	if kindByKey["cron:daily-report"] != "cron" {
		t.Fatalf("expected cron kind for cron:daily-report, got %q", kindByKey["cron:daily-report"])
	}

	// Kind filter
	filtered := tool.Execute(context.Background(), map[string]any{
		"kinds": []any{"main"},
		"limit": 10,
	})
	if filtered.IsError {
		t.Fatalf("sessions_list (filtered) returned error: %s", filtered.ForLLM)
	}
	if err := json.Unmarshal([]byte(filtered.ForLLM), &payload); err != nil {
		t.Fatalf("failed to decode filtered payload: %v", err)
	}
	if payload.Count != 1 {
		t.Fatalf("expected 1 filtered session, got %d", payload.Count)
	}
	if payload.Sessions[0].Key != "agent:main:main" {
		t.Fatalf("unexpected filtered session key: %q", payload.Sessions[0].Key)
	}
}

func TestSessionsHistoryTool_IncludeToolsToggle(t *testing.T) {
	sm := session.NewSessionManager(t.TempDir())
	key := "agent:main:main"
	sm.AddMessage(key, "user", "first")
	sm.AddFullMessage(key, providers.Message{Role: "tool", Content: "tool-output", ToolCallID: "tc-1"})
	sm.AddMessage(key, "assistant", "second")

	tool := NewSessionsHistoryTool(sm)

	withoutTools := tool.Execute(context.Background(), map[string]any{
		"session_key": key,
		"limit":       10,
	})
	if withoutTools.IsError {
		t.Fatalf("sessions_history returned error: %s", withoutTools.ForLLM)
	}

	var payload struct {
		MessageCount int                 `json:"message_count"`
		Messages     []providers.Message `json:"messages"`
	}
	if err := json.Unmarshal([]byte(withoutTools.ForLLM), &payload); err != nil {
		t.Fatalf("decode sessions_history payload failed: %v", err)
	}
	if payload.MessageCount != 2 {
		t.Fatalf("expected 2 messages without tools, got %d", payload.MessageCount)
	}
	for _, msg := range payload.Messages {
		if msg.Role == "tool" {
			t.Fatalf("tool message should be excluded by default")
		}
	}

	withTools := tool.Execute(context.Background(), map[string]any{
		"session_key":   key,
		"limit":         10,
		"include_tools": true,
	})
	if withTools.IsError {
		t.Fatalf("sessions_history(include_tools=true) returned error: %s", withTools.ForLLM)
	}
	if err := json.Unmarshal([]byte(withTools.ForLLM), &payload); err != nil {
		t.Fatalf("decode sessions_history payload failed: %v", err)
	}
	if payload.MessageCount != 3 {
		t.Fatalf("expected 3 messages with tools, got %d", payload.MessageCount)
	}
}

func TestSessionsHistoryTool_NotFound(t *testing.T) {
	sm := session.NewSessionManager(t.TempDir())
	tool := NewSessionsHistoryTool(sm)

	result := tool.Execute(context.Background(), map[string]any{
		"session_key": "does-not-exist",
	})
	if !result.IsError {
		t.Fatalf("expected error for missing session")
	}
}
