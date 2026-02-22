package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockDevTargetSetter implements miniapp.DevTargetSetter for testing.
type mockDevTargetSetter struct {
	target string
	err    error
}

func (m *mockDevTargetSetter) SetDevTarget(target string) error {
	if m.err != nil {
		return m.err
	}
	m.target = target
	return nil
}

func (m *mockDevTargetSetter) GetDevTarget() string {
	return m.target
}

func TestDevPreviewTool_Start(t *testing.T) {
	setter := &mockDevTargetSetter{}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://localhost:3000",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if setter.target != "http://localhost:3000" {
		t.Errorf("expected target http://localhost:3000, got %q", setter.target)
	}
	if !strings.Contains(result.ForLLM, "started") {
		t.Errorf("expected result to contain 'started', got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_StartMissingTarget(t *testing.T) {
	setter := &mockDevTargetSetter{}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
	})

	if !result.IsError {
		t.Error("expected error for missing target")
	}
}

func TestDevPreviewTool_StartError(t *testing.T) {
	setter := &mockDevTargetSetter{err: fmt.Errorf("only localhost")}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "start",
		"target": "http://example.com:3000",
	})

	if !result.IsError {
		t.Error("expected error for setter failure")
	}
}

func TestDevPreviewTool_Stop(t *testing.T) {
	setter := &mockDevTargetSetter{target: "http://localhost:3000"}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "stop",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if setter.target != "" {
		t.Errorf("expected empty target after stop, got %q", setter.target)
	}
}

func TestDevPreviewTool_Status(t *testing.T) {
	setter := &mockDevTargetSetter{target: "http://localhost:8080"}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "active") {
		t.Errorf("expected 'active' in result, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "http://localhost:8080") {
		t.Errorf("expected target URL in result, got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_StatusInactive(t *testing.T) {
	setter := &mockDevTargetSetter{}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "status",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "not active") {
		t.Errorf("expected 'not active' in result, got %q", result.ForLLM)
	}
}

func TestDevPreviewTool_UnknownAction(t *testing.T) {
	setter := &mockDevTargetSetter{}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "restart",
	})

	if !result.IsError {
		t.Error("expected error for unknown action")
	}
}

func TestDevPreviewTool_MissingAction(t *testing.T) {
	setter := &mockDevTargetSetter{}
	tool := NewDevPreviewTool(setter)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("expected error for missing action")
	}
}

func TestDevPreviewTool_NameAndSchema(t *testing.T) {
	setter := &mockDevTargetSetter{}
	tool := NewDevPreviewTool(setter)

	if tool.Name() != "dev_preview" {
		t.Errorf("expected name dev_preview, got %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
	params := tool.Parameters()
	if params == nil {
		t.Fatal("expected non-nil parameters")
	}
}
