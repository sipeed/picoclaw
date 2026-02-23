package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
)

func setupTestLogs(t *testing.T) {
	t.Helper()
	prev := logger.GetLevel()
	t.Cleanup(func() { logger.SetLevel(prev) })
	logger.SetLevel(logger.DEBUG)

	logger.DebugC("agent", "debug message")
	logger.InfoC("telegram", "message received")
	logger.WarnC("telegram", "webhook retry")
	logger.ErrorC("discord", "connection timeout")
	logger.WarnCF("wecom", "signature failed", map[string]any{
		"token":   "secret-value",
		"nonce":   "safe-value",
	})
}

func TestLogsTool_DefaultLevel(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	var entries []logger.LogEntry
	if err := json.Unmarshal([]byte(result.ForLLM), &entries); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	for _, e := range entries {
		if e.Level == "DEBUG" || e.Level == "INFO" {
			t.Errorf("default level should be WARN, but got %s entry: %s", e.Level, e.Message)
		}
	}
}

func TestLogsTool_LevelFilter(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{
		"level": "ERROR",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	var entries []logger.LogEntry
	if err := json.Unmarshal([]byte(result.ForLLM), &entries); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	for _, e := range entries {
		if e.Level != "ERROR" && e.Level != "FATAL" {
			t.Errorf("expected only ERROR+, got %s: %s", e.Level, e.Message)
		}
	}
}

func TestLogsTool_ComponentFilter(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{
		"level":     "DEBUG",
		"component": "telegram",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	var entries []logger.LogEntry
	if err := json.Unmarshal([]byte(result.ForLLM), &entries); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	for _, e := range entries {
		if e.Component != "telegram" {
			t.Errorf("expected component=telegram, got %s", e.Component)
		}
	}
}

func TestLogsTool_QueryFilter(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{
		"level": "DEBUG",
		"query": "timeout",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	var entries []logger.LogEntry
	if err := json.Unmarshal([]byte(result.ForLLM), &entries); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one entry matching 'timeout'")
	}
	for _, e := range entries {
		if !strings.Contains(strings.ToLower(e.Message), "timeout") {
			t.Errorf("entry should contain 'timeout': %s", e.Message)
		}
	}
}

func TestLogsTool_QueryCaseInsensitive(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{
		"level": "DEBUG",
		"query": "TIMEOUT",
	})

	var entries []logger.LogEntry
	if err := json.Unmarshal([]byte(result.ForLLM), &entries); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("case-insensitive query should match")
	}
}

func TestLogsTool_Limit(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{
		"level": "DEBUG",
		"limit": float64(2),
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	var entries []logger.LogEntry
	if err := json.Unmarshal([]byte(result.ForLLM), &entries); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(entries) > 2 {
		t.Errorf("expected at most 2 entries, got %d", len(entries))
	}
}

func TestLogsTool_LimitMax(t *testing.T) {
	tool := NewLogsTool()

	// limit > 300 should be capped
	result := tool.Execute(context.Background(), map[string]any{
		"level": "DEBUG",
		"limit": float64(999),
	})

	// Should not error, just cap silently
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestLogsTool_FieldsSanitized(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{
		"level":     "WARN",
		"component": "wecom",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	var entries []logger.LogEntry
	if err := json.Unmarshal([]byte(result.ForLLM), &entries); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	found := false
	for _, e := range entries {
		if e.Fields != nil && e.Fields["token"] != nil {
			found = true
			if e.Fields["token"] != "***" {
				t.Errorf("token field should be sanitized, got %v", e.Fields["token"])
			}
			if e.Fields["nonce"] != "safe-value" {
				t.Errorf("nonce field should be preserved, got %v", e.Fields["nonce"])
			}
		}
	}
	if !found {
		t.Error("expected to find wecom entry with token field")
	}
}

func TestLogsTool_NoResults(t *testing.T) {
	prev := logger.GetLevel()
	defer logger.SetLevel(prev)
	logger.SetLevel(logger.DEBUG)

	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{
		"level":     "ERROR",
		"component": "nonexistent-component-xyz",
	})

	if result.IsError {
		t.Fatalf("should not be an error result: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "No log entries found") {
		t.Errorf("expected 'No log entries found' message, got: %s", result.ForLLM)
	}
}

func TestLogsTool_Silent(t *testing.T) {
	setupTestLogs(t)
	tool := NewLogsTool()

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.Silent {
		t.Error("logs tool result should be Silent")
	}
}

func TestLogsTool_ToolInterface(t *testing.T) {
	tool := NewLogsTool()

	if tool.Name() != "logs" {
		t.Errorf("expected name 'logs', got %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Error("parameters should not be nil")
	}
}
