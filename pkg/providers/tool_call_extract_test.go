package providers

import (
	"testing"
)

// --- extractToolCallsFromText tests ---

func TestExtractToolCallsFromText_NoJSON(t *testing.T) {
	got := extractToolCallsFromText("Just plain text with no JSON.")
	if len(got) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(got))
	}
}

func TestExtractToolCallsFromText_CompactJSON(t *testing.T) {
	text := `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"NYC\"}"}}]}`
	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got))
	}
	if got[0].Name != "get_weather" {
		t.Errorf("Name = %q, want %q", got[0].Name, "get_weather")
	}
	if got[0].Arguments["location"] != "NYC" {
		t.Errorf("Arguments[location] = %v, want NYC", got[0].Arguments["location"])
	}
}

func TestExtractToolCallsFromText_PrettyPrintedJSON(t *testing.T) {
	// Bug case: LLM returns pretty-printed JSON with newlines after {
	text := `{
  "tool_calls": [
    {
      "id": "call_1",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"location\": \"NYC\"}"
      }
    }
  ]
}`
	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool call from pretty-printed JSON, got %d", len(got))
	}
	if got[0].Name != "get_weather" {
		t.Errorf("Name = %q, want %q", got[0].Name, "get_weather")
	}
	if got[0].Arguments["location"] != "NYC" {
		t.Errorf("Arguments[location] = %v, want NYC", got[0].Arguments["location"])
	}
}

func TestExtractToolCallsFromText_JSONInCodeFence(t *testing.T) {
	// LLM wraps output in markdown code fences
	text := "```json\n" + `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_files","arguments":"{\"path\":\"/tmp\"}"}}]}` + "\n```"
	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool call from code-fenced JSON, got %d", len(got))
	}
	if got[0].Name != "list_files" {
		t.Errorf("Name = %q, want %q", got[0].Name, "list_files")
	}
	if got[0].Arguments["path"] != "/tmp" {
		t.Errorf("Arguments[path] = %v, want /tmp", got[0].Arguments["path"])
	}
}

func TestExtractToolCallsFromText_JSONInPlainCodeFence(t *testing.T) {
	// Code fence without language tag
	text := "```\n" + `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"do_thing","arguments":"{}"}}]}` + "\n```"
	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got))
	}
	if got[0].Name != "do_thing" {
		t.Errorf("Name = %q, want %q", got[0].Name, "do_thing")
	}
}

func TestExtractToolCallsFromText_ArgumentsAsObject(t *testing.T) {
	// Arguments field is a JSON object, not a JSON string
	text := `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"create_file","arguments":{"path":"/tmp/out.txt","content":"hello world"}}}]}`
	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got))
	}
	if got[0].Name != "create_file" {
		t.Errorf("Name = %q, want %q", got[0].Name, "create_file")
	}
	if got[0].Arguments["path"] != "/tmp/out.txt" {
		t.Errorf("Arguments[path] = %v, want /tmp/out.txt", got[0].Arguments["path"])
	}
	if got[0].Arguments["content"] != "hello world" {
		t.Errorf("Arguments[content] = %v, want 'hello world'", got[0].Arguments["content"])
	}
}

func TestExtractToolCallsFromText_MixedTextAroundJSON(t *testing.T) {
	// Text both before and after the JSON block
	text := "Let me search for that.\n" + `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{\"query\":\"golang\"}"}}]}` + "\nI'll get back to you."
	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool call with surrounding text, got %d", len(got))
	}
	if got[0].Name != "search" {
		t.Errorf("Name = %q, want %q", got[0].Name, "search")
	}
	if got[0].Arguments["query"] != "golang" {
		t.Errorf("Arguments[query] = %v, want golang", got[0].Arguments["query"])
	}
}

func TestExtractToolCallsFromText_InvalidJSON(t *testing.T) {
	got := extractToolCallsFromText(`{"tool_calls":invalid}`)
	if len(got) != 0 {
		t.Errorf("expected 0 tool calls for invalid JSON, got %d", len(got))
	}
}

func TestExtractToolCallsFromText_EmptyToolCalls(t *testing.T) {
	got := extractToolCallsFromText(`{"tool_calls":[]}`)
	if len(got) != 0 {
		t.Errorf("expected 0 tool calls for empty array, got %d", len(got))
	}
}

func TestExtractToolCallsFromText_NoToolCallsKey(t *testing.T) {
	got := extractToolCallsFromText(`{"other_key":"value"}`)
	if len(got) != 0 {
		t.Errorf("expected 0 tool calls for unrelated JSON, got %d", len(got))
	}
}

func TestExtractToolCallsFromText_MultipleToolCalls(t *testing.T) {
	text := `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"/a\"}"}},{"id":"c2","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"/b\",\"content\":\"x\"}"}}]}`
	got := extractToolCallsFromText(text)
	if len(got) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(got))
	}
	if got[0].Name != "read_file" {
		t.Errorf("[0].Name = %q, want read_file", got[0].Name)
	}
	if got[1].Name != "write_file" {
		t.Errorf("[1].Name = %q, want write_file", got[1].Name)
	}
}

func TestExtractToolCallsFromText_FunctionFieldPreserved(t *testing.T) {
	text := `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{\"k\":\"v\"}"}}]}`
	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got))
	}
	if got[0].Function == nil {
		t.Fatal("Function field should not be nil")
	}
	if got[0].Function.Name != "fn" {
		t.Errorf("Function.Name = %q, want fn", got[0].Function.Name)
	}
	if got[0].Function.Arguments == "" {
		t.Error("Function.Arguments should contain raw JSON string")
	}
}

// --- stripToolCallsFromText tests ---

func TestStripToolCallsFromText_NoToolCalls(t *testing.T) {
	text := "Just regular text."
	got := stripToolCallsFromText(text)
	if got != text {
		t.Errorf("stripToolCallsFromText() = %q, want %q", got, text)
	}
}

func TestStripToolCallsFromText_OnlyJSON(t *testing.T) {
	text := `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]}`
	got := stripToolCallsFromText(text)
	if got != "" {
		t.Errorf("stripToolCallsFromText() = %q, want empty", got)
	}
}

func TestStripToolCallsFromText_TextBeforeJSON(t *testing.T) {
	text := "Let me check the weather.\n" + `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]}`
	got := stripToolCallsFromText(text)
	if got != "Let me check the weather." {
		t.Errorf("stripToolCallsFromText() = %q, want %q", got, "Let me check the weather.")
	}
}

func TestStripToolCallsFromText_TextAfterJSON(t *testing.T) {
	text := `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]}` + "\nDone."
	got := stripToolCallsFromText(text)
	if got != "Done." {
		t.Errorf("stripToolCallsFromText() = %q, want %q", got, "Done.")
	}
}

func TestStripToolCallsFromText_MixedTextAroundJSON(t *testing.T) {
	text := "Before.\n" + `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]}` + "\nAfter."
	got := stripToolCallsFromText(text)
	// TrimSpace only trims the outer ends; internal whitespace from surrounding text is kept.
	if got != "Before.\n\nAfter." {
		t.Errorf("stripToolCallsFromText() = %q, want %q", got, "Before.\n\nAfter.")
	}
}

func TestStripToolCallsFromText_PrettyPrintedJSON(t *testing.T) {
	text := "Thinking...\n" + "{\n  \"tool_calls\": [{\"id\":\"c1\",\"type\":\"function\",\"function\":{\"name\":\"fn\",\"arguments\":\"{}\"}}]\n}"
	got := stripToolCallsFromText(text)
	if got == text {
		t.Error("stripToolCallsFromText() should remove pretty-printed JSON block")
	}
	if got != "Thinking..." {
		t.Errorf("stripToolCallsFromText() = %q, want %q", got, "Thinking...")
	}
}

func TestStripToolCallsFromText_UnrelatedJSON(t *testing.T) {
	// A JSON object that is not a tool_calls wrapper should be left alone
	text := `{"key":"value"}`
	got := stripToolCallsFromText(text)
	if got != text {
		t.Errorf("stripToolCallsFromText() = %q, want %q (unrelated JSON should be preserved)", got, text)
	}
}

// --- stripCodeFences tests ---

func TestStripCodeFences_JSONFence(t *testing.T) {
	inner := `{"tool_calls":[]}`
	text := "```json\n" + inner + "\n```"
	got := stripCodeFences(text)
	if got != inner {
		t.Errorf("stripCodeFences() = %q, want %q", got, inner)
	}
}

func TestStripCodeFences_PlainFence(t *testing.T) {
	inner := `{"tool_calls":[]}`
	text := "```\n" + inner + "\n```"
	got := stripCodeFences(text)
	if got != inner {
		t.Errorf("stripCodeFences() = %q, want %q", got, inner)
	}
}

func TestStripCodeFences_NoFence(t *testing.T) {
	text := `{"tool_calls":[]}`
	got := stripCodeFences(text)
	if got != text {
		t.Errorf("stripCodeFences() = %q, want %q (no fence should return original)", got, text)
	}
}

func TestStripCodeFences_PlainText(t *testing.T) {
	text := "Just some text."
	got := stripCodeFences(text)
	if got != text {
		t.Errorf("stripCodeFences() = %q, want %q", got, text)
	}
}
