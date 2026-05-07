package common

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestNormalizeStoredToolCall_TopLevelFields(t *testing.T) {
	tc := protocoltypes.ToolCall{
		Name:      "search",
		Arguments: map[string]any{"q": "hello"},
	}
	name, args, sig := NormalizeStoredToolCall(tc)
	if name != "search" {
		t.Errorf("name = %q, want %q", name, "search")
	}
	if args["q"] != "hello" {
		t.Errorf("args[q] = %v, want %q", args["q"], "hello")
	}
	if sig != "" {
		t.Errorf("thoughtSignature = %q, want empty", sig)
	}
}

func TestNormalizeStoredToolCall_FallsBackToFunction(t *testing.T) {
	tc := protocoltypes.ToolCall{
		Function: &protocoltypes.FunctionCall{
			Name:             "read_file",
			Arguments:        `{"path":"/tmp"}`,
			ThoughtSignature: "sig123",
		},
	}
	name, args, sig := NormalizeStoredToolCall(tc)
	if name != "read_file" {
		t.Errorf("name = %q, want %q", name, "read_file")
	}
	if args["path"] != "/tmp" {
		t.Errorf("args[path] = %v, want %q", args["path"], "/tmp")
	}
	if sig != "sig123" {
		t.Errorf("thoughtSignature = %q, want %q", sig, "sig123")
	}
}

func TestNormalizeStoredToolCall_TopLevelNameWithFunctionSig(t *testing.T) {
	tc := protocoltypes.ToolCall{
		Name:      "search",
		Arguments: map[string]any{"q": "hi"},
		Function: &protocoltypes.FunctionCall{
			ThoughtSignature: "thought1",
		},
	}
	name, _, sig := NormalizeStoredToolCall(tc)
	if name != "search" {
		t.Errorf("name = %q, want %q", name, "search")
	}
	if sig != "thought1" {
		t.Errorf("thoughtSignature = %q, want %q", sig, "thought1")
	}
}

func TestNormalizeStoredToolCall_NilArgs(t *testing.T) {
	tc := protocoltypes.ToolCall{Name: "test"}
	_, args, _ := NormalizeStoredToolCall(tc)
	if args == nil {
		t.Fatal("args should not be nil")
	}
	if len(args) != 0 {
		t.Errorf("args should be empty, got %v", args)
	}
}

func TestNormalizeStoredToolCall_EmptyArgsParseFromFunction(t *testing.T) {
	tc := protocoltypes.ToolCall{
		Name:      "tool",
		Arguments: map[string]any{},
		Function: &protocoltypes.FunctionCall{
			Arguments: `{"key":"val"}`,
		},
	}
	_, args, _ := NormalizeStoredToolCall(tc)
	if args["key"] != "val" {
		t.Errorf("args[key] = %v, want %q", args["key"], "val")
	}
}

func TestNormalizeStoredToolCall_InvalidFunctionJSON(t *testing.T) {
	tc := protocoltypes.ToolCall{
		Name: "tool",
		Function: &protocoltypes.FunctionCall{
			Arguments: `not-json`,
		},
	}
	_, args, _ := NormalizeStoredToolCall(tc)
	if len(args) != 0 {
		t.Errorf("args should be empty for invalid JSON, got %v", args)
	}
}

func TestResolveToolResponseName_FromMap(t *testing.T) {
	names := map[string]string{"call_1": "search"}
	got := ResolveToolResponseName("call_1", names)
	if got != "search" {
		t.Errorf("got %q, want %q", got, "search")
	}
}

func TestResolveToolResponseName_EmptyID(t *testing.T) {
	got := ResolveToolResponseName("", map[string]string{"x": "y"})
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestResolveToolResponseName_FallsBackToInfer(t *testing.T) {
	got := ResolveToolResponseName("call_search_docs_999", map[string]string{})
	if got != "search_docs" {
		t.Errorf("got %q, want %q", got, "search_docs")
	}
}

func TestInferToolNameFromCallID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"standard format", "call_search_docs_999", "search_docs"},
		{"single name", "call_read_123", "read"},
		{"no call prefix", "some_id", "some_id"},
		{"call prefix no underscore suffix", "call_onlyname", "call_onlyname"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferToolNameFromCallID(tt.id)
			if got != tt.want {
				t.Errorf(
					"InferToolNameFromCallID(%q) = %q, want %q",
					tt.id, got, tt.want,
				)
			}
		})
	}
}
