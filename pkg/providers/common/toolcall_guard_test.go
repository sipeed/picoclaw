package common

import (
	"strings"
	"testing"
)

// ParseResponse must drop tool calls that lack a name or an id (malformed
// upstreams) instead of handing them to history — they cannot be executed
// and later strict-provider replays (Anthropic) 400 on them.
func TestParseResponse_DropsToolCallsWithoutNameOrID(t *testing.T) {
	body := strings.NewReader(`{"choices":[{"message":{"content":"",
		"tool_calls":[
			{"id":"x1","type":"function"},
			{"id":"","type":"function","function":{"name":"y","arguments":"{}"}},
			{"id":"x3","type":"function","function":{"name":"ok","arguments":"{\"a\":1}"}}
		]},"finish_reason":"tool_calls"}]}`)
	out, err := ParseResponse(body)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1 (two malformed must be dropped): %+v", len(out.ToolCalls), out.ToolCalls)
	}
	tc := out.ToolCalls[0]
	if tc.ID != "x3" || tc.Name != "ok" {
		t.Fatalf("ToolCall = %+v, want x3/ok", tc)
	}
	if tc.Arguments["a"] != float64(1) {
		t.Fatalf("Arguments[a] = %v, want 1", tc.Arguments["a"])
	}
}
