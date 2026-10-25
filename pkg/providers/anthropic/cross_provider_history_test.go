package anthropicprovider

import (
	"encoding/json"
	"testing"
)

// Regression: dead tool calls (empty name/id — e.g. truncated upstream
// streams, or old stores that persisted them without a function object)
// must not survive into the Anthropic request: the assistant turn ends up
// with zero blocks, which marshals without "content" and 400s with
// "messages.N.content: Field required".
func TestBuildParams_CrossProviderHistory(t *testing.T) {
	opts := map[string]any{"max_tokens": 8192}

	msgs := []Message{
		{Role: "user", Content: "hi"},
		// poison entry: tool calls with empty names (all skipped -> empty
		// assistant content) + its orphaned tool result
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "call_q1", Function: &FunctionCall{Name: "", Arguments: `{}`}},
		}},
		{Role: "tool", ToolCallID: "call_q1", Content: "result"},
		// empty assistant turn (reasoning-only from other provider)
		{Role: "assistant", Content: ""},
		// healthy pair must survive untouched
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "call_ok", Function: &FunctionCall{Name: "exec", Arguments: `{"command":"ls"}`}},
		}},
		{Role: "tool", ToolCallID: "call_ok", Content: "filelist"},
		// tool call with empty ID must be dropped (its result becomes orphan)
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "", Function: &FunctionCall{Name: "exec", Arguments: `{}`}},
		}},
		{Role: "user", Content: "bye"},
	}

	params, err := buildParams(msgs, nil, "claude-haiku-5-5", opts)
	if err != nil {
		t.Fatal(err)
	}

	// expected surviving: user(hi), assistant(tool_use call_ok),
	// user(tool_result call_ok), user(bye) = 4
	if len(params.Messages) != 4 {
		b, _ := json.Marshal(params.Messages)
		t.Fatalf("expected 4 messages, got %d: %s", len(params.Messages), b)
	}

	// no message may marshal without a content field / with empty content
	b, err := json.Marshal(params.Messages)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	for i, m := range decoded {
		c, ok := m["content"]
		if !ok || c == nil {
			t.Errorf("message %d: content missing", i)
			continue
		}
		switch v := c.(type) {
		case string:
			if v == "" {
				t.Errorf("message %d: content empty string", i)
			}
		case []any:
			if len(v) == 0 {
				t.Errorf("message %d: content empty array", i)
			}
		}
	}

	// no orphaned tool_result: every tool_result must reference an emitted tool_use
	toolUseIDs := map[string]bool{}
	toolResultIDs := []string{}
	for _, m := range decoded {
		blocks, _ := m["content"].([]any)
		for _, blkany := range blocks {
			blk, _ := blkany.(map[string]any)
			switch blk["type"] {
			case "tool_use":
				toolUseIDs[blk["id"].(string)] = true
			case "tool_result":
				toolResultIDs = append(toolResultIDs, blk["tool_use_id"].(string))
			}
		}
	}
	for _, id := range toolResultIDs {
		if !toolUseIDs[id] {
			t.Errorf("orphaned tool_result for %q", id)
		}
	}
	if !toolUseIDs["call_ok"] {
		t.Error("healthy tool_use call_ok was dropped")
	}
}
