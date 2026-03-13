package tools

import (
	"context"
	"strings"
	"testing"
)

type mockCtxTool struct {
	mockRegistryTool

	channel string

	chatID string
}

func (m *mockCtxTool) SetContext(channel, chatID string) {
	m.channel = channel

	m.chatID = chatID
}

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

func TestToolRegistry_Get_FuzzyMatch(t *testing.T) {
	r := NewToolRegistry()

	r.Register(newMockTool("read_file", "reads a file"))

	r.Register(newMockTool("edit_file", "edits a file"))

	r.Register(newMockTool("web_search", "searches the web"))

	tests := []struct {
		query string

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

func TestToolRegistry_ExecuteWithContext_ContextualTool(t *testing.T) {
	r := NewToolRegistry()

	ct := &mockCtxTool{
		mockRegistryTool: *newMockTool("ctx_tool", "needs context"),
	}

	r.Register(ct)

	r.ExecuteWithContext(context.Background(), "ctx_tool", nil, "telegram", "chat-42", nil)

	if ct.channel != "telegram" {
		t.Errorf("expected channel 'telegram', got %q", ct.channel)
	}

	if ct.chatID != "chat-42" {
		t.Errorf("expected chatID 'chat-42', got %q", ct.chatID)
	}
}

func TestToolRegistry_ExecuteWithContext_SkipsEmptyContext(t *testing.T) {
	r := NewToolRegistry()

	ct := &mockCtxTool{
		mockRegistryTool: *newMockTool("ctx_tool", "needs context"),
	}

	r.Register(ct)

	r.ExecuteWithContext(context.Background(), "ctx_tool", nil, "", "", nil)

	if ct.channel != "" || ct.chatID != "" {
		t.Error("SetContext should not be called with empty channel/chatID")
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

func TestToolRegistry_GetSummaries_WithParamHint(t *testing.T) {
	r := NewToolRegistry()

	r.Register(&mockRegistryTool{
		name: "spawn",

		desc: "Spawn a subagent",

		params: map[string]any{
			"type": "object",

			"properties": map[string]any{
				"task": map[string]any{"type": "string"},

				"preset": map[string]any{"type": "string"},
			},

			"required": []string{"task"},
		},

		result: SilentResult("ok"),
	})

	summaries := r.GetSummaries()

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	if !strings.Contains(summaries[0], "(task, preset?)") {
		t.Errorf("expected param hint in summary, got %q", summaries[0])
	}
}
