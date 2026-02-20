package tools

import (
	"context"
	"testing"
)

// stubTool is a minimal Tool implementation for testing.
type stubTool struct {
	name string
}

func (s *stubTool) Name() string        { return s.name }
func (s *stubTool) Description() string { return "stub" }
func (s *stubTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *stubTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	return &ToolResult{ForLLM: "ok"}
}

func TestNormalizeToolName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"read_file", "readfile"},
		{"readfile", "readfile"},
		{"ReadFile", "readfile"},
		{"read-file", "readfile"},
		{"edit_file", "editfile"},
		{"web_search", "websearch"},
		{"EXEC", "exec"},
	}
	for _, tt := range tests {
		got := NormalizeToolName(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeToolName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRegistryGet_ExactMatch(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&stubTool{name: "read_file"})

	tool, ok := r.Get("read_file")
	if !ok || tool.Name() != "read_file" {
		t.Errorf("exact match failed")
	}
}

func TestRegistryGet_FuzzyMatch(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&stubTool{name: "read_file"})
	r.Register(&stubTool{name: "edit_file"})
	r.Register(&stubTool{name: "web_search"})

	tests := []struct {
		query    string
		wantName string
	}{
		{"readfile", "read_file"},
		{"ReadFile", "read_file"},
		{"read-file", "read_file"},
		{"editfile", "edit_file"},
		{"EditFile", "edit_file"},
		{"websearch", "web_search"},
		{"WebSearch", "web_search"},
	}
	for _, tt := range tests {
		tool, ok := r.Get(tt.query)
		if !ok {
			t.Errorf("Get(%q) not found, want %q", tt.query, tt.wantName)
			continue
		}
		if tool.Name() != tt.wantName {
			t.Errorf("Get(%q).Name() = %q, want %q", tt.query, tool.Name(), tt.wantName)
		}
	}
}

func TestRegistryGet_NotFound(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&stubTool{name: "read_file"})

	_, ok := r.Get("totally_unknown")
	if ok {
		t.Errorf("Get(totally_unknown) should return false")
	}
}
