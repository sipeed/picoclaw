package tools

import (
	"context"
	"testing"
)

type mockMemoryWriter struct {
	longTerm string
	daily    string
}

func (m *mockMemoryWriter) WriteLongTerm(content string) error {
	m.longTerm = content
	return nil
}

func (m *mockMemoryWriter) AppendToday(content string) error {
	m.daily += content
	return nil
}

func (m *mockMemoryWriter) ReadLongTerm() string {
	return m.longTerm
}

func (m *mockMemoryWriter) ReadToday() string {
	return m.daily
}

func TestMemoryTool_WriteLongTerm(t *testing.T) {
	w := &mockMemoryWriter{}
	tool := NewMemoryTool(w)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "write_long_term",
		"content": "User prefers dark mode",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if w.longTerm != "User prefers dark mode" {
		t.Errorf("expected 'User prefers dark mode', got '%s'", w.longTerm)
	}
}

func TestMemoryTool_WriteLongTerm_MissingContent(t *testing.T) {
	w := &mockMemoryWriter{}
	tool := NewMemoryTool(w)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "write_long_term",
	})

	if !result.IsError {
		t.Error("expected error for missing content")
	}
}

func TestMemoryTool_AppendDaily(t *testing.T) {
	w := &mockMemoryWriter{}
	tool := NewMemoryTool(w)

	tool.Execute(context.Background(), map[string]interface{}{
		"action":  "append_daily",
		"content": "Met with team.",
	})
	tool.Execute(context.Background(), map[string]interface{}{
		"action":  "append_daily",
		"content": " Discussed roadmap.",
	})

	if w.daily != "Met with team. Discussed roadmap." {
		t.Errorf("expected appended content, got '%s'", w.daily)
	}
}

func TestMemoryTool_ReadLongTerm(t *testing.T) {
	w := &mockMemoryWriter{longTerm: "stored facts"}
	tool := NewMemoryTool(w)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read_long_term",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if result.ForLLM != "stored facts" {
		t.Errorf("expected 'stored facts', got '%s'", result.ForLLM)
	}
}

func TestMemoryTool_ReadLongTerm_Empty(t *testing.T) {
	w := &mockMemoryWriter{}
	tool := NewMemoryTool(w)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read_long_term",
	})

	if result.IsError {
		t.Error("should not error on empty memory")
	}
	if result.ForLLM != "No long-term memory found" {
		t.Errorf("expected empty message, got '%s'", result.ForLLM)
	}
}

func TestMemoryTool_ReadDaily(t *testing.T) {
	w := &mockMemoryWriter{daily: "today's notes"}
	tool := NewMemoryTool(w)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read_daily",
	})

	if result.ForLLM != "today's notes" {
		t.Errorf("expected 'today's notes', got '%s'", result.ForLLM)
	}
}

func TestMemoryTool_UnknownAction(t *testing.T) {
	w := &mockMemoryWriter{}
	tool := NewMemoryTool(w)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "delete_all",
	})

	if !result.IsError {
		t.Error("expected error for unknown action")
	}
}

func TestMemoryTool_MissingAction(t *testing.T) {
	w := &mockMemoryWriter{}
	tool := NewMemoryTool(w)

	result := tool.Execute(context.Background(), map[string]interface{}{})

	if !result.IsError {
		t.Error("expected error for missing action")
	}
}

func TestMemoryTool_NameAndDescription(t *testing.T) {
	w := &mockMemoryWriter{}
	tool := NewMemoryTool(w)

	if tool.Name() != "memory" {
		t.Errorf("expected name 'memory', got '%s'", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Error("parameters should not be nil")
	}
}
