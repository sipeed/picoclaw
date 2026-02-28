package server

import (
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

// ── Model identification helpers ─────────────────────────────────

func TestIsOpenAIModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"openai", true},
		{"openai/gpt-4o", true},
		{"openai/gpt-5.2", true},
		{"anthropic", false},
		{"anthropic/claude-sonnet-4.6", false},
		{"openai-compatible", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isOpenAIModel(tt.model); got != tt.want {
			t.Errorf("isOpenAIModel(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

func TestIsAnthropicModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"anthropic", true},
		{"anthropic/claude-sonnet-4.6", true},
		{"openai", false},
		{"openai/gpt-4o", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isAnthropicModel(tt.model); got != tt.want {
			t.Errorf("isAnthropicModel(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

func TestIsAntigravityModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"antigravity", true},
		{"google-antigravity", true},
		{"antigravity/gemini-3-flash", true},
		{"google-antigravity/gemini-3-flash", true},
		{"openai", false},
		{"antigravity-custom", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isAntigravityModel(tt.model); got != tt.want {
			t.Errorf("isAntigravityModel(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

// ── Config update helpers ────────────────────────────────────────

func writeTempConfigViaSave(t *testing.T, cfg *config.Config) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return path
}

func loadTempConfig(t *testing.T, path string) *config.Config {
	t.Helper()
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func TestUpdateConfigAfterLogin_OpenAI_ExistingModel(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{ModelName: "gpt-4o", Model: "openai/gpt-4o"},
		},
	}
	path := writeTempConfigViaSave(t, cfg)

	cred := &auth.AuthCredential{AuthMethod: "oauth"}
	updateConfigAfterLogin(path, "openai", cred)

	result := loadTempConfig(t, path)

	// Model-level auth_method persists through serialization
	if len(result.ModelList) != 1 {
		t.Fatalf("expected 1 model, got %d", len(result.ModelList))
	}
	if result.ModelList[0].AuthMethod != "oauth" {
		t.Errorf("expected model auth_method=oauth, got %q", result.ModelList[0].AuthMethod)
	}
}

func TestUpdateConfigAfterLogin_OpenAI_NoExistingModel(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{ModelName: "claude", Model: "anthropic/claude-sonnet-4.6"},
		},
	}
	path := writeTempConfigViaSave(t, cfg)

	cred := &auth.AuthCredential{AuthMethod: "oauth"}
	updateConfigAfterLogin(path, "openai", cred)

	result := loadTempConfig(t, path)

	if len(result.ModelList) != 2 {
		t.Fatalf("expected 2 models (original + added), got %d", len(result.ModelList))
	}
	if result.ModelList[1].Model != "openai/gpt-5.2" {
		t.Errorf("expected added model openai/gpt-5.2, got %q", result.ModelList[1].Model)
	}
	if result.Agents.Defaults.ModelName != "gpt-5.2" {
		t.Errorf("expected default model_name=gpt-5.2, got %q", result.Agents.Defaults.ModelName)
	}
}

func TestUpdateConfigAfterLogin_Anthropic(t *testing.T) {
	cfg := &config.Config{}
	path := writeTempConfigViaSave(t, cfg)

	cred := &auth.AuthCredential{AuthMethod: "token"}
	updateConfigAfterLogin(path, "anthropic", cred)

	result := loadTempConfig(t, path)

	// Model should be added with correct auth_method
	if len(result.ModelList) != 1 {
		t.Fatalf("expected 1 model added, got %d", len(result.ModelList))
	}
	if result.ModelList[0].Model != "anthropic/claude-sonnet-4.6" {
		t.Errorf("expected model anthropic/claude-sonnet-4.6, got %q", result.ModelList[0].Model)
	}
	if result.ModelList[0].AuthMethod != "token" {
		t.Errorf("expected model auth_method=token, got %q", result.ModelList[0].AuthMethod)
	}
}

func TestUpdateConfigAfterLogin_GoogleAntigravity(t *testing.T) {
	cfg := &config.Config{}
	path := writeTempConfigViaSave(t, cfg)

	cred := &auth.AuthCredential{AuthMethod: "oauth"}
	updateConfigAfterLogin(path, "google-antigravity", cred)

	result := loadTempConfig(t, path)

	// Model should be added with correct auth_method
	if len(result.ModelList) != 1 {
		t.Fatalf("expected 1 model added, got %d", len(result.ModelList))
	}
	if result.ModelList[0].Model != "antigravity/gemini-3-flash" {
		t.Errorf("expected model antigravity/gemini-3-flash, got %q", result.ModelList[0].Model)
	}
	if result.ModelList[0].AuthMethod != "oauth" {
		t.Errorf("expected model auth_method=oauth, got %q", result.ModelList[0].AuthMethod)
	}
}

func TestClearAuthMethodInConfig(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{ModelName: "gpt-4o", Model: "openai/gpt-4o", AuthMethod: "oauth"},
			{ModelName: "claude", Model: "anthropic/claude-sonnet-4.6", AuthMethod: "token"},
		},
	}
	path := writeTempConfigViaSave(t, cfg)

	clearAuthMethodInConfig(path, "openai")

	result := loadTempConfig(t, path)

	// Openai model auth_method should be cleared
	if result.ModelList[0].AuthMethod != "" {
		t.Errorf("expected openai model auth_method cleared, got %q", result.ModelList[0].AuthMethod)
	}
	// Anthropic model should be unchanged
	if result.ModelList[1].AuthMethod != "token" {
		t.Errorf("expected anthropic model auth_method unchanged, got %q", result.ModelList[1].AuthMethod)
	}
}

func TestClearAllAuthMethodsInConfig(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{ModelName: "gpt-4o", Model: "openai/gpt-4o", AuthMethod: "oauth"},
			{ModelName: "claude", Model: "anthropic/claude-sonnet-4.6", AuthMethod: "token"},
			{ModelName: "gemini", Model: "antigravity/gemini-3-flash", AuthMethod: "oauth"},
		},
	}
	path := writeTempConfigViaSave(t, cfg)

	clearAllAuthMethodsInConfig(path)

	result := loadTempConfig(t, path)

	for i, m := range result.ModelList {
		if m.AuthMethod != "" {
			t.Errorf("model[%d] auth_method not cleared, got %q", i, m.AuthMethod)
		}
	}
}
