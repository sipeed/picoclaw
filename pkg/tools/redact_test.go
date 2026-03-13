package tools

import (
	"fmt"
	"testing"
)

func TestRedactSecrets_OpenAIKey(t *testing.T) {
	input := `Config loaded: api_key is sk-proj-abc123def456ghi789jkl012mno345pqr678`
	result := RedactSecrets(input)
	if result == input {
		t.Error("expected OpenAI key to be redacted")
	}
	if !containsRedacted(result) {
		t.Errorf("expected [REDACTED] in output, got: %s", result)
	}
}

func TestRedactSecrets_AnthropicKey(t *testing.T) {
	input := `Using key: sk-ant-api03-abcdefghijklmnopqrstuvwxyz123456`
	result := RedactSecrets(input)
	if !containsRedacted(result) {
		t.Errorf("expected [REDACTED] in output, got: %s", result)
	}
}

func TestRedactSecrets_JSONFields(t *testing.T) {
	input := `{"api_key": "my-super-secret-key-12345", "model": "gpt-4"}`
	result := RedactSecrets(input)
	if result == input {
		t.Error("expected JSON api_key value to be redacted")
	}
	// Key name should be preserved for context
	if !contains(result, `"api_key"`) {
		t.Error("expected key name to be preserved")
	}
	if !containsRedacted(result) {
		t.Errorf("expected [REDACTED] in output, got: %s", result)
	}
	// Model value should NOT be redacted
	if !contains(result, "gpt-4") {
		t.Error("expected non-secret fields to be preserved")
	}
}

func TestRedactSecrets_MultipleSecretFields(t *testing.T) {
	input := `{
		"api_key": "sk-abcdefghijklmnopqrstuvwxyz",
		"token": "test-fake-token-value-placeholder-0000",
		"channel_secret": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4"
	}`
	result := RedactSecrets(input)
	if contains(result, "sk-abcdefghijklmnopqrstuvwxyz") {
		t.Error("api_key value should be redacted")
	}
	if contains(result, "test-fake-token-value") {
		t.Error("token value should be redacted")
	}
}

func TestRedactSecrets_SlackToken(t *testing.T) {
	// Use a clearly fake token that still matches the xoxb- pattern
	fakeToken := "xoxb-fake" + "-placeholder-abcdefghij"
	input := "Bot token: " + fakeToken
	result := RedactSecrets(input)
	if !containsRedacted(result) {
		t.Errorf("expected Slack token to be redacted, got: %s", result)
	}
}

func TestRedactSecrets_AWSAccessKey(t *testing.T) {
	input := `AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE`
	result := RedactSecrets(input)
	if contains(result, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("expected AWS key to be redacted, got: %s", result)
	}
}

func TestRedactSecrets_GitHubToken(t *testing.T) {
	input := `GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef01234`
	result := RedactSecrets(input)
	if contains(result, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Errorf("expected GitHub token to be redacted, got: %s", result)
	}
}

func TestRedactSecrets_NoSecrets(t *testing.T) {
	input := `{"model": "gpt-4", "temperature": 0.7, "message": "Hello world"}`
	result := RedactSecrets(input)
	if result != input {
		t.Errorf("expected no changes for non-secret content, got: %s", result)
	}
}

func TestRedactSecrets_EmptyString(t *testing.T) {
	result := RedactSecrets("")
	if result != "" {
		t.Error("expected empty string to pass through unchanged")
	}
}

func TestRedactSecrets_PreservesPrefix(t *testing.T) {
	input := `key: sk-proj-abc123def456ghi789jkl012mno345pqr678`
	result := RedactSecrets(input)
	// Should show first 4 chars for identification
	if !contains(result, "sk-p") {
		t.Errorf("expected prefix to be preserved for identification, got: %s", result)
	}
}

func TestSanitizeResult_ForLLMAndForUser(t *testing.T) {
	r := &ToolResult{
		ForLLM:  `Read config.json: {"api_key": "sk-ant-secret12345678901234"}`,
		ForUser: `Config: {"api_key": "sk-ant-secret12345678901234"}`,
	}
	SanitizeResult(r)
	if contains(r.ForLLM, "sk-ant-secret") {
		t.Error("ForLLM should have secret redacted")
	}
	if contains(r.ForUser, "sk-ant-secret") {
		t.Error("ForUser should have secret redacted")
	}
}

func TestSanitizeResult_NilResult(t *testing.T) {
	result := SanitizeResult(nil)
	if result != nil {
		t.Error("expected nil to pass through")
	}
}

func TestSanitizeResult_ErrorField(t *testing.T) {
	r := &ToolResult{
		ForLLM: "error occurred",
		Err:    fmt.Errorf("auth failed with key sk-proj-abc123def456ghi789jkl012mno345pqr678"),
	}
	SanitizeResult(r)
	if contains(r.Err.Error(), "sk-proj-abc123") {
		t.Error("Err field should have secret redacted")
	}
}

func TestContainsSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`normal text`, false},
		{`key: sk-proj-abc123def456ghi789jkl012mno`, true},
		{`{"api_key": "super-secret-value-here"}`, true},
		{`{"model": "gpt-4"}`, false},
	}
	for _, tc := range tests {
		got := ContainsSecret(tc.input)
		if got != tc.expected {
			t.Errorf("ContainsSecret(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestIsSecretEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"OPENAI_API_KEY", true},
		{"ARK_API_KEY", true},
		{"CHANNEL_SECRET", true},
		{"BOT_TOKEN", true},
		{"DATABASE_PASSWORD", true},
		{"HOME", false},
		{"PATH", false},
		{"GOPATH", false},
		{"MODEL_NAME", false},
	}
	for _, tc := range tests {
		got := IsSecretEnvVar(tc.name)
		if got != tc.expected {
			t.Errorf("IsSecretEnvVar(%q) = %v, want %v", tc.name, got, tc.expected)
		}
	}
}

func containsRedacted(s string) bool {
	return contains(s, "[REDACTED]")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
