package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func decodeSessionID(t *testing.T, payload string) string {
	t.Helper()
	var out struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("failed to decode exec payload: %v", err)
	}
	if out.SessionID == "" {
		t.Fatalf("missing session_id in payload: %s", payload)
	}
	return out.SessionID
}

func TestExecBackground_WithProcessPoll(t *testing.T) {
	execTool := NewExecTool(t.TempDir(), false)
	processTool := NewProcessTool(execTool.ProcessManager())

	start := execTool.Execute(context.Background(), map[string]any{
		"command":    "sleep 0.2; echo done",
		"background": true,
	})
	if start.IsError {
		t.Fatalf("exec background failed: %s", start.ForLLM)
	}
	sessionID := decodeSessionID(t, start.ForLLM)

	poll := processTool.Execute(context.Background(), map[string]any{
		"action":     "poll",
		"session_id": sessionID,
		"timeout_ms": 3000,
	})
	if poll.IsError {
		t.Fatalf("process poll failed: %s", poll.ForLLM)
	}

	var payload struct {
		Output  string `json:"output"`
		Session struct {
			Status string `json:"status"`
		} `json:"session"`
	}
	if err := json.Unmarshal([]byte(poll.ForLLM), &payload); err != nil {
		t.Fatalf("failed to decode poll payload: %v", err)
	}
	if payload.Session.Status == "running" {
		poll = processTool.Execute(context.Background(), map[string]any{
			"action":     "poll",
			"session_id": sessionID,
			"timeout_ms": 2000,
		})
		if poll.IsError {
			t.Fatalf("second poll failed: %s", poll.ForLLM)
		}
		if err := json.Unmarshal([]byte(poll.ForLLM), &payload); err != nil {
			t.Fatalf("failed to decode second poll payload: %v", err)
		}
	}
	if !strings.Contains(payload.Output, "done") && payload.Session.Status != "completed" {
		t.Fatalf("expected output to contain done or completed status, got: status=%q output=%q", payload.Session.Status, payload.Output)
	}
	if payload.Session.Status != "completed" {
		t.Fatalf("expected completed status, got %q", payload.Session.Status)
	}
}

func TestProcessTool_KillAndRemove(t *testing.T) {
	execTool := NewExecTool(t.TempDir(), false)
	processTool := NewProcessTool(execTool.ProcessManager())

	start := execTool.Execute(context.Background(), map[string]any{
		"command":    "sleep 60",
		"background": true,
	})
	if start.IsError {
		t.Fatalf("exec background failed: %s", start.ForLLM)
	}
	sessionID := decodeSessionID(t, start.ForLLM)

	kill := processTool.Execute(context.Background(), map[string]any{
		"action":     "kill",
		"session_id": sessionID,
	})
	if kill.IsError {
		t.Fatalf("process kill failed: %s", kill.ForLLM)
	}

	poll := processTool.Execute(context.Background(), map[string]any{
		"action":     "poll",
		"session_id": sessionID,
		"timeout_ms": 3000,
	})
	if poll.IsError {
		t.Fatalf("process poll after kill failed: %s", poll.ForLLM)
	}

	var pollPayload struct {
		Session struct {
			Status string `json:"status"`
		} `json:"session"`
	}
	if err := json.Unmarshal([]byte(poll.ForLLM), &pollPayload); err != nil {
		t.Fatalf("failed to decode poll payload: %v", err)
	}
	if pollPayload.Session.Status == "running" {
		t.Fatalf("expected killed session to stop running")
	}

	remove := processTool.Execute(context.Background(), map[string]any{
		"action":     "remove",
		"session_id": sessionID,
	})
	if remove.IsError {
		t.Fatalf("process remove failed: %s", remove.ForLLM)
	}
}

func TestProcessTool_Write(t *testing.T) {
	execTool := NewExecTool(t.TempDir(), false)
	processTool := NewProcessTool(execTool.ProcessManager())

	start := execTool.Execute(context.Background(), map[string]any{
		"command":    "cat",
		"background": true,
	})
	if start.IsError {
		t.Fatalf("exec background failed: %s", start.ForLLM)
	}
	sessionID := decodeSessionID(t, start.ForLLM)

	write := processTool.Execute(context.Background(), map[string]any{
		"action":     "write",
		"session_id": sessionID,
		"data":       "ping\n",
		"eof":        true,
	})
	if write.IsError {
		t.Fatalf("process write failed: %s", write.ForLLM)
	}

	poll := processTool.Execute(context.Background(), map[string]any{
		"action":     "poll",
		"session_id": sessionID,
		"timeout_ms": 3000,
	})
	if poll.IsError {
		t.Fatalf("poll after write failed: %s", poll.ForLLM)
	}

	var payload struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal([]byte(poll.ForLLM), &payload); err != nil {
		t.Fatalf("failed to decode poll payload: %v", err)
	}
	if !strings.Contains(payload.Output, "ping") {
		t.Fatalf("expected output to contain ping, got %q", payload.Output)
	}
}
