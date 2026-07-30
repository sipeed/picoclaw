package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestStripToolUseText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "native tool call duplicate",
			content: `[tool_use: exec, args: {"action":"run","command":"ls -la","timeout":5}]`,
			want:    "",
		},
		{
			name:    "nested JSON and brackets in quoted command",
			content: `[tool_use: exec, args: {"action":"run","options":{"env":{"MODE":"scan"}},"command":"printf '[%s]' \"ok\"","items":[1,{"nested":true}]}}]`,
			want:    "",
		},
		{
			name:    "tool user variant without args label",
			content: `[tool_user: exec, {"command":"id"}]`,
			want:    "",
		},
		{
			name:    "tool result",
			content: `Before [tool_result: {"ok":true,"data":[1,2]}] after`,
			want:    "Before  after",
		},
		{
			name:    "mixed explanation and tool call",
			content: "I will inspect it.\n[tool_use: exec, args: {\"command\":\"find /var -name '*.log'\"}]\nPlease wait.",
			want:    "I will inspect it.\n\nPlease wait.",
		},
		{
			name:    "truncated block only",
			content: `[tool_use: exec, args: {"command":"id"}`,
			want:    "",
		},
		{
			name:    "truncated block preserves next line",
			content: "Checking now.\n[tool_use exec args={\"command\":\"id\"}\nA readable answer follows.",
			want:    "Checking now.\nA readable answer follows.",
		},
		{
			name:    "case insensitive",
			content: `[TOOL_USE: exec, args: {"command":"id"}]`,
			want:    "",
		},
		{
			name:    "ordinary brackets untouched",
			content: "Use [tool_usage] and JSON {\"result\":\"ok\"}.",
			want:    "Use [tool_usage] and JSON {\"result\":\"ok\"}.",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := stripToolUseText(tt.content); got != tt.want {
				t.Fatalf("stripToolUseText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeLLMResponseToolProtocol(t *testing.T) {
	t.Parallel()

	response := &providers.LLMResponse{
		Content:          `[tool_use: exec, args: {"action":"run","command":"ls"}]`,
		Reasoning:        `Inspecting. [tool_user: exec, {"command":"ls"}]`,
		ReasoningContent: `[tool_result: {"files":["a","b"]}] Done.`,
		ToolCalls: []providers.ToolCall{{
			ID:   "call_1",
			Name: "exec",
			ExtraContent: &providers.ExtraContent{
				ToolFeedbackExplanation: `[tool_use: exec, args: {"action":"run","command":"ls"}]`,
			},
		}},
	}

	sanitizeLLMResponseToolProtocol(response)

	if response.Content != "" {
		t.Fatalf("Content = %q, want empty", response.Content)
	}
	if response.Reasoning != "Inspecting." {
		t.Fatalf("Reasoning = %q, want %q", response.Reasoning, "Inspecting.")
	}
	if response.ReasoningContent != "Done." {
		t.Fatalf("ReasoningContent = %q, want %q", response.ReasoningContent, "Done.")
	}
	if got := response.ToolCalls[0].ExtraContent.ToolFeedbackExplanation; got != "" {
		t.Fatalf("ToolFeedbackExplanation = %q, want empty", got)
	}
}
