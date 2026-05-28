package providers

import "testing"

func TestNormalizeToolCall_PreservesExtraContentGoogleThoughtSignature(t *testing.T) {
	tc := NormalizeToolCall(ToolCall{
		ID:        "call_1",
		Name:      "search",
		Arguments: map[string]any{"q": "pico"},
		ExtraContent: &ExtraContent{
			Google: &GoogleExtra{ThoughtSignature: "sig-1"},
		},
	})

	if tc.ThoughtSignature != "sig-1" {
		t.Fatalf("ThoughtSignature = %q, want sig-1", tc.ThoughtSignature)
	}
	if tc.Function == nil {
		t.Fatal("Function is nil")
	}
	if tc.Function.ThoughtSignature != "sig-1" {
		t.Fatalf("Function.ThoughtSignature = %q, want sig-1", tc.Function.ThoughtSignature)
	}
}
