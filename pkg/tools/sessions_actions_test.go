package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type stubSessionsSendExecutor struct {
	reply   string
	err     error
	called  bool
	content string
	key     string
	channel string
	chatID  string
}

func (s *stubSessionsSendExecutor) ProcessSessionMessage(
	ctx context.Context,
	content, sessionKey, channel, chatID string,
) (string, error) {
	s.called = true
	s.content = content
	s.key = sessionKey
	s.channel = channel
	s.chatID = chatID
	if s.err != nil {
		return "", s.err
	}
	return s.reply, nil
}

func TestSessionsSendTool_Success(t *testing.T) {
	exec := &stubSessionsSendExecutor{reply: "target reply"}
	tool := NewSessionsSendTool(exec)
	tool.SetContext("cli", "direct")

	result := tool.Execute(context.Background(), map[string]any{
		"session_key": "agent:main:main",
		"message":     "hello",
	})
	if result.IsError {
		t.Fatalf("sessions_send returned error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Fatalf("sessions_send should be silent")
	}
	if !exec.called {
		t.Fatalf("expected executor to be called")
	}

	var payload struct {
		Status     string `json:"status"`
		SessionKey string `json:"session_key"`
		Reply      string `json:"reply"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("expected status ok, got %q", payload.Status)
	}
	if payload.Reply != "target reply" {
		t.Fatalf("expected reply %q, got %q", "target reply", payload.Reply)
	}
}

func TestSessionsSendTool_TimeoutFromContext(t *testing.T) {
	exec := &stubSessionsSendExecutor{
		err: context.DeadlineExceeded,
	}
	tool := NewSessionsSendTool(exec)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	result := tool.Execute(ctx, map[string]any{
		"session_key": "agent:main:main",
		"message":     "hello",
	})
	if result.IsError {
		t.Fatalf("timeout should be reported as structured silent result, got error: %s", result.ForLLM)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if payload.Status != "timeout" {
		t.Fatalf("expected timeout status, got %q", payload.Status)
	}
}

func TestSessionsSpawnTool_AcceptedAndAllowlist(t *testing.T) {
	manager := NewSubagentManager(&MockLLMProvider{}, "test-model", t.TempDir(), nil)
	tool := NewSessionsSpawnTool(manager)
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID != "blocked"
	})

	result := tool.Execute(context.Background(), map[string]any{
		"task":     "do something",
		"label":    "unit",
		"agent_id": "worker",
	})
	if result.IsError {
		t.Fatalf("sessions_spawn returned error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Fatalf("sessions_spawn should be silent")
	}

	var payload struct {
		Status string `json:"status"`
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if payload.Status != "accepted" {
		t.Fatalf("expected status accepted, got %q", payload.Status)
	}
	if payload.TaskID == "" {
		t.Fatalf("expected non-empty task_id")
	}

	denied := tool.Execute(context.Background(), map[string]any{
		"task":     "do something else",
		"agent_id": "blocked",
	})
	if !denied.IsError {
		t.Fatalf("expected allowlist failure")
	}
}
