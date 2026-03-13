package tools

import (
	"context"
	"strings"
	"testing"
)

func (m *mockAsyncRegistryTool) SetCallback(cb AsyncCallback) {
	m.lastCB = cb
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

func TestToolRegistry_Get_ExactMatch(t *testing.T) {
	r := NewToolRegistry()

	r.Register(newMockTool("read_file", "reads a file"))
	r.Register(newMockTool("edit_file", "edits a file"))
	r.Register(newMockTool("web_search", "searches the web"))

	// Exact matches should work
	for _, name := range []string{"read_file", "edit_file", "web_search"} {
		tool, ok := r.Get(name)
		if !ok {
			t.Errorf("Get(%q) not found", name)
			continue
		}
		if tool.Name() != name {
			t.Errorf("Get(%q).Name() = %q", name, tool.Name())
		}
	}

	// Non-exact names should not match (Get is exact-only)
	for _, name := range []string{"readfile", "ReadFile", "read-file"} {
		if _, ok := r.Get(name); ok {
			t.Errorf("Get(%q) should not match (exact lookup only)", name)
		}
	}
}

func TestToolRegistry_ExecuteWithContext_InjectsContext(t *testing.T) {
	r := NewToolRegistry()

	// Tool that reads context from ctx via ToolChannel/ToolChatID
	contextCapture := newMockTool("ctx_tool", "needs context")
	r.Register(contextCapture)

	result := r.ExecuteWithContext(
		context.Background(), "ctx_tool", nil, "telegram", "chat-42", nil,
	)
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestBuildParamHint(t *testing.T) {
	tests := []struct {
		name string

		schema map[string]any

		want string
	}{
		{
			name: "required and optional",

			schema: map[string]any{
				"type": "object",

				"properties": map[string]any{
					"task": map[string]any{"type": "string"},

					"label": map[string]any{"type": "string"},
				},

				"required": []string{"task"},
			},

			want: "(task, label?)",
		},

		{
			name: "all required",

			schema: map[string]any{
				"type": "object",

				"properties": map[string]any{
					"command": map[string]any{"type": "string"},
				},

				"required": []string{"command"},
			},

			want: "(command)",
		},

		{
			name: "no properties",

			schema: map[string]any{
				"type": "object",
			},

			want: "",
		},

		{
			name: "empty schema",

			schema: map[string]any{},

			want: "",
		},

		{
			name: "nil schema",

			schema: nil,

			want: "",
		},

		{
			name: "multiple optional sorted",

			schema: map[string]any{
				"type": "object",

				"properties": map[string]any{
					"task": map[string]any{"type": "string"},

					"preset": map[string]any{"type": "string"},

					"label": map[string]any{"type": "string"},

					"agent_id": map[string]any{"type": "string"},
				},

				"required": []string{"task"},
			},

			want: "(task, agent_id?, label?, preset?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildParamHint(tt.schema)

			if got != tt.want {
				t.Errorf("buildParamHint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToolRegistry_GetSummaries_Format(t *testing.T) {
	r := NewToolRegistry()

	r.Register(&mockRegistryTool{
		name:   "spawn",
		desc:   "Spawn a subagent",
		result: SilentResult("ok"),
	})

	summaries := r.GetSummaries()

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	if !strings.Contains(summaries[0], "spawn") {
		t.Errorf("expected tool name in summary, got %q", summaries[0])
	}

	if !strings.Contains(summaries[0], "Spawn a subagent") {
		t.Errorf("expected description in summary, got %q", summaries[0])
	}
}
