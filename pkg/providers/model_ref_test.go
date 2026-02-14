package providers

import "testing"

func TestParseModelRef_WithSlash(t *testing.T) {
	ref := ParseModelRef("anthropic/claude-opus", "openai")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", ref.Provider)
	}
	if ref.Model != "claude-opus" {
		t.Errorf("model = %q, want claude-opus", ref.Model)
	}
}

func TestParseModelRef_WithoutSlash(t *testing.T) {
	ref := ParseModelRef("gpt-4", "openai")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Provider != "openai" {
		t.Errorf("provider = %q, want openai", ref.Provider)
	}
	if ref.Model != "gpt-4" {
		t.Errorf("model = %q, want gpt-4", ref.Model)
	}
}

func TestParseModelRef_Empty(t *testing.T) {
	ref := ParseModelRef("", "openai")
	if ref != nil {
		t.Errorf("expected nil for empty string, got %+v", ref)
	}
}

func TestParseModelRef_EmptyModelAfterSlash(t *testing.T) {
	ref := ParseModelRef("openai/", "default")
	if ref != nil {
		t.Errorf("expected nil for empty model, got %+v", ref)
	}
}

func TestParseModelRef_WhitespaceHandling(t *testing.T) {
	ref := ParseModelRef("  anthropic / claude-opus  ", "openai")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", ref.Provider)
	}
	if ref.Model != "claude-opus" {
		t.Errorf("model = %q, want claude-opus", ref.Model)
	}
}

func TestNormalizeProvider(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"OpenAI", "openai"},
		{"ANTHROPIC", "anthropic"},
		{"z.ai", "zai"},
		{"z-ai", "zai"},
		{"Z.AI", "zai"},
		{"opencode-zen", "opencode"},
		{"qwen", "qwen-portal"},
		{"kimi-code", "kimi-coding"},
		{"gpt", "openai"},
		{"claude", "anthropic"},
		{"glm", "zhipu"},
		{"google", "gemini"},
		{"groq", "groq"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeProvider(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeProvider(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestModelKey(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		want     string
	}{
		{"openai", "gpt-4", "openai/gpt-4"},
		{"Anthropic", "Claude-Opus", "anthropic/claude-opus"},
		{"claude", "sonnet", "anthropic/sonnet"},
		{"z.ai", "Model-X", "zai/model-x"},
	}

	for _, tt := range tests {
		got := ModelKey(tt.provider, tt.model)
		if got != tt.want {
			t.Errorf("ModelKey(%q, %q) = %q, want %q", tt.provider, tt.model, got, tt.want)
		}
	}
}

func TestParseModelRef_ProviderNormalization(t *testing.T) {
	ref := ParseModelRef("Z.AI/model-x", "default")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Provider != "zai" {
		t.Errorf("provider = %q, want zai", ref.Provider)
	}
}

func TestParseModelRef_DefaultProviderNormalization(t *testing.T) {
	ref := ParseModelRef("gpt-4o", "GPT")
	if ref == nil {
		t.Fatal("expected non-nil ref")
	}
	if ref.Provider != "openai" {
		t.Errorf("provider = %q, want openai (normalized from GPT)", ref.Provider)
	}
}
