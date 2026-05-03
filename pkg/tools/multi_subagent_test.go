package tools

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

type recordingMultiSubagentSpawner struct {
	mu      sync.Mutex
	cfgs    []SubTurnConfig
	byTask  map[string]*ToolResult
	errTask map[string]error
}

func (s *recordingMultiSubagentSpawner) SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfgs = append(s.cfgs, cfg)
	res := (*ToolResult)(nil)
	var err error
	if s.byTask != nil {
		res = s.byTask[cfg.SystemPrompt]
	}
	if s.errTask != nil {
		err = s.errTask[cfg.SystemPrompt]
	}
	return res, err
}

func TestMultiSubagentTool_RequiresSpawner(t *testing.T) {
	tool := NewMultiSubagentTool(nil)
	res := tool.Execute(context.Background(), map[string]any{
		"calls": []any{map[string]any{"task": "hello"}},
	})
	if !res.IsError {
		t.Fatal("expected error when spawner is missing")
	}
	if !strings.Contains(res.ForLLM, "Subagent manager not configured") {
		t.Fatalf("unexpected error: %s", res.ForLLM)
	}
}

func TestMultiSubagentTool_ValidatesCalls(t *testing.T) {
	tool := NewMultiSubagentTool(nil)
	tool.SetSpawner(&recordingMultiSubagentSpawner{})

	cases := []map[string]any{
		{},
		{"calls": []any{}},
		{"calls": []any{"bad"}},
		{"calls": []any{map[string]any{"task": "   "}}},
	}
	for _, args := range cases {
		res := tool.Execute(context.Background(), args)
		if !res.IsError {
			t.Fatalf("expected validation error for args: %#v", args)
		}
	}
}

func TestMultiSubagentTool_AllowlistRejectsTargetedCall(t *testing.T) {
	tool := NewMultiSubagentTool(nil)
	tool.SetSpawner(&recordingMultiSubagentSpawner{})
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID == "mini"
	})

	res := tool.Execute(context.Background(), map[string]any{
		"calls": []any{
			map[string]any{"task": "one", "agent_id": "code"},
		},
	})
	if !res.IsError {
		t.Fatal("expected allowlist rejection")
	}
	if !strings.Contains(res.ForLLM, "not allowed to target agent 'code'") {
		t.Fatalf("unexpected allowlist error: %s", res.ForLLM)
	}
}

func TestMultiSubagentTool_UsesDefaultModelAndGroupsResults(t *testing.T) {
	promptAlpha := "You are a subagent labeled \"alpha\". Complete the given task independently and provide a clear, concise result.\n\nTask: task a"
	promptBeta := "You are a subagent. Complete the given task independently and provide a clear, concise result.\n\nTask: task b"
	spawner := &recordingMultiSubagentSpawner{
		byTask: map[string]*ToolResult{
			promptAlpha: {ForLLM: "alpha llm", ForUser: "alpha user"},
			promptBeta:  {ForLLM: "beta llm", ForUser: ""},
		},
	}
	tool := &MultiSubagentTool{
		spawner:      spawner,
		defaultModel: "main-model",
		maxTokens:    123,
		temperature:  0.4,
	}

	res := tool.Execute(context.Background(), map[string]any{
		"calls": []any{
			map[string]any{"task": "task a", "label": "alpha", "agent_id": "mini"},
			map[string]any{"task": "task b"},
		},
	})
	if res.IsError {
		t.Fatalf("unexpected error result: %s", res.ForLLM)
	}
	if len(spawner.cfgs) != 2 {
		t.Fatalf("expected 2 subturns, got %d", len(spawner.cfgs))
	}
	for i, cfg := range spawner.cfgs {
		if cfg.Model != "main-model" {
			t.Fatalf("cfg[%d].Model = %q, want %q", i, cfg.Model, "main-model")
		}
		if cfg.Async {
			t.Fatalf("cfg[%d].Async = true, want false", i)
		}
		if cfg.MaxTokens != 123 {
			t.Fatalf("cfg[%d].MaxTokens = %d, want 123", i, cfg.MaxTokens)
		}
		if cfg.Temperature != 0.4 {
			t.Fatalf("cfg[%d].Temperature = %v, want 0.4", i, cfg.Temperature)
		}
	}
	if !strings.Contains(res.ForLLM, "[alpha | agent=mini]") {
		t.Fatalf("expected labeled grouped LLM output, got: %s", res.ForLLM)
	}
	if !strings.Contains(res.ForLLM, "[call-2 | agent=default]") {
		t.Fatalf("expected default grouped LLM output, got: %s", res.ForLLM)
	}
	if !strings.Contains(res.ForUser, "alpha: alpha user") {
		t.Fatalf("expected ForUser to prefer subresult user content, got: %s", res.ForUser)
	}
	if !strings.Contains(res.ForUser, "call-2: beta llm") {
		t.Fatalf("expected ForUser fallback to llm content, got: %s", res.ForUser)
	}
}

func TestMultiSubagentTool_PropagatesPerCallErrors(t *testing.T) {
	promptOne := "You are a subagent labeled \"one\". Complete the given task independently and provide a clear, concise result.\n\nTask: task 1"
	promptTwo := "You are a subagent labeled \"two\". Complete the given task independently and provide a clear, concise result.\n\nTask: task 2"
	promptThree := "You are a subagent labeled \"three\". Complete the given task independently and provide a clear, concise result.\n\nTask: task 3"
	spawner := &recordingMultiSubagentSpawner{
		byTask: map[string]*ToolResult{
			promptOne:   {ForLLM: "ok-one", ForUser: "ok-one"},
			promptThree: {ForLLM: "suberror", ForUser: "suberror", IsError: true},
		},
		errTask: map[string]error{
			promptTwo: errors.New("boom"),
		},
	}
	tool := &MultiSubagentTool{
		spawner:      spawner,
		defaultModel: "main-model",
	}

	res := tool.Execute(context.Background(), map[string]any{
		"calls": []any{
			map[string]any{"task": "task 1", "label": "one"},
			map[string]any{"task": "task 2", "label": "two"},
			map[string]any{"task": "task 3", "label": "three"},
		},
	})
	if !res.IsError {
		t.Fatal("expected aggregated error state")
	}
	if !strings.Contains(res.ForLLM, "[two | agent=default] ERROR: boom") {
		t.Fatalf("expected explicit per-call error in llm output, got: %s", res.ForLLM)
	}
	if !strings.Contains(res.ForLLM, "[three | agent=default]\nsuberror") {
		t.Fatalf("expected subresult error content in llm output, got: %s", res.ForLLM)
	}
}
