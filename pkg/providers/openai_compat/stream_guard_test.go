package openai_compat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// A truncated or malformed upstream stream that ends a tool call without
// ever sending its name or id must not produce a ToolCall — such entries
// poison the session store and later Anthropic replays (400s).
func TestChatStream_DropsToolCallsWithoutNameOrID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_a","function":{"name":"exec","arguments":"{\"cmd\""}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":":\"ls\"}"}}]}}]}`,
			// dead payload: no id ever, no name, only arguments
			`{"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"garbage"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		}
		for _, c := range chunks {
			_, _ = w.Write([]byte("data: " + c + "\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.ChatStream(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil, nil)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1 (dead call must be dropped): %+v", len(out.ToolCalls), out.ToolCalls)
	}
	if out.ToolCalls[0].Name != "exec" || out.ToolCalls[0].ID != "call_a" {
		t.Fatalf("ToolCalls[0] = %+v, want exec/call_a", out.ToolCalls[0])
	}
	if out.ToolCalls[0].Arguments["cmd"] != "ls" {
		t.Fatalf("Arguments[cmd] = %v, want ls", out.ToolCalls[0].Arguments["cmd"])
	}
}

// Tool-call delta indexes that do not start at 0 (observed from some
// OpenAI-compatible servers) must not silently drop the last tool call.
func TestChatStream_PreservesToolCallsWithOneBasedIndexes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"c1","function":{"name":"alpha","arguments":"{}"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":2,"id":"c2","function":{"name":"beta","arguments":"{}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		}
		for _, c := range chunks {
			_, _ = w.Write([]byte("data: " + c + "\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.ChatStream(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil, nil)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	if len(out.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2: %+v", len(out.ToolCalls), out.ToolCalls)
	}
	if out.ToolCalls[0].Name != "alpha" || out.ToolCalls[1].Name != "beta" {
		t.Fatalf("names = %q, %q; want alpha, beta", out.ToolCalls[0].Name, out.ToolCalls[1].Name)
	}
}
