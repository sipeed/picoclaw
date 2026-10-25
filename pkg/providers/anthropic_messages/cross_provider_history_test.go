package anthropicmessages

import (
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestBuildRequestBody_CrossProviderHistory(t *testing.T) {
	msgs := []protocoltypes.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", ToolCalls: []protocoltypes.ToolCall{
			{ID: "q1", Function: &protocoltypes.FunctionCall{Name: "", Arguments: `{}`}},
		}},
		{Role: "tool", ToolCallID: "q1", Content: "r"},
		{Role: "assistant", ToolCalls: []protocoltypes.ToolCall{
			{ID: "ok", Function: &protocoltypes.FunctionCall{Name: "exec", Arguments: `{"c":"ls"}`}},
		}},
		{Role: "tool", ToolCallID: "ok", Content: "list"},
	}
	got, err := buildRequestBody(msgs, nil, "claude-haiku-5-5", map[string]any{"max_tokens": 4096})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got["messages"])
	var decoded []map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	// expected: user(hi), assistant(tool_use ok), user(tool_result ok) = 3
	if len(decoded) != 3 {
		t.Fatalf("expected 3 messages, got %d: %s", len(decoded), b)
	}
	for i, m := range decoded {
		if c, ok := m["content"]; !ok || c == nil {
			t.Errorf("message %d: content missing", i)
		}
	}
}
