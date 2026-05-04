package tools

import (
	"context"
	"testing"
)

type capturingSubTurnSpawner struct {
	lastCfg SubTurnConfig
}

func (s *capturingSubTurnSpawner) SpawnSubTurn(_ context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	s.lastCfg = cfg
	return &ToolResult{ForLLM: "ok", ForUser: "ok"}, nil
}

func TestSubagentToolRejectsDisallowedAgentID(t *testing.T) {
	tool := &SubagentTool{defaultModel: "main-model"}
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID != "code"
	})

	result := tool.Execute(context.Background(), map[string]any{
		"task":     "do something",
		"agent_id": "code",
	})

	if result == nil || !result.IsError {
		t.Fatalf("expected error result, got %#v", result)
	}
}

func TestSubagentToolUsesResolvedTargetModelForAgentID(t *testing.T) {
	spawner := &capturingSubTurnSpawner{}
	tool := &SubagentTool{defaultModel: "main-model"}
	tool.SetSpawner(spawner)
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID == "code"
	})
	tool.SetTargetModelResolver(func(targetAgentID string) string {
		if targetAgentID == "code" {
			return "code-model"
		}
		return ""
	})

	result := tool.Execute(context.Background(), map[string]any{
		"task":     "do something",
		"agent_id": "code",
	})

	if result == nil || result.IsError {
		t.Fatalf("expected success result, got %#v", result)
	}
	if spawner.lastCfg.Model != "code-model" {
		t.Fatalf("expected model code-model, got %q", spawner.lastCfg.Model)
	}
}

func TestSubagentToolFallsBackToDefaultModelWithoutAgentID(t *testing.T) {
	spawner := &capturingSubTurnSpawner{}
	tool := &SubagentTool{defaultModel: "main-model"}
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{
		"task": "do something",
	})

	if result == nil || result.IsError {
		t.Fatalf("expected success result, got %#v", result)
	}
	if spawner.lastCfg.Model != "main-model" {
		t.Fatalf("expected model main-model, got %q", spawner.lastCfg.Model)
	}
}
