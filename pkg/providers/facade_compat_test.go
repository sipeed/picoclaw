package providers

import (
	"testing"

	cliprovider "github.com/sipeed/picoclaw/pkg/providers/cli"
	oauthprovider "github.com/sipeed/picoclaw/pkg/providers/oauth"
)

func TestNormalizeToolCallFacadeMatchesCLIProvider(t *testing.T) {
	input := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: &FunctionCall{
			Name:      "read_file",
			Arguments: `{"path":"README.md"}`,
		},
	}

	got := NormalizeToolCall(input)
	want := cliprovider.NormalizeToolCall(input)

	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Function == nil || want.Function == nil {
		t.Fatalf("Function should not be nil: got=%v want=%v", got.Function, want.Function)
	}
	if got.Function.Name != want.Function.Name {
		t.Fatalf("Function.Name = %q, want %q", got.Function.Name, want.Function.Name)
	}
	if got.Function.Arguments != want.Function.Arguments {
		t.Fatalf("Function.Arguments = %q, want %q", got.Function.Arguments, want.Function.Arguments)
	}
	if got.Arguments["path"] != want.Arguments["path"] {
		t.Fatalf("Arguments[path] = %v, want %v", got.Arguments["path"], want.Arguments["path"])
	}
}

func TestAntigravityFacadeSignaturesRemainAvailable(t *testing.T) {
	var _ func(string) (string, error) = FetchAntigravityProjectID
	var _ func(string, string) ([]AntigravityModelInfo, error) = FetchAntigravityModels
	var _ AntigravityModelInfo = oauthprovider.AntigravityModelInfo{}
}
