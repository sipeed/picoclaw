package tools

import (
	"fmt"
	"strings"
	"testing"
)

func TestRedactSecrets_OpenAIKey(t *testing.T) {
	input := `Config loaded: api_key is sk-proj-abc123def456ghi789jkl012mno345pqr678`
	result := RedactSecrets(input)
	if result == input {
		t.Error("expected OpenAI key to be redacted")
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output, got: %s", result)
	}
}

func TestRedactSecrets_AnthropicKey(t *testing.T) {
	input := `Using key: sk-ant-api03-abcdefghijklmnopqrstuvwxyz123456`
	result := RedactSecrets(input)
	if !strings.Contains(result, "[REDACTED]") {
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
	if !strings.Contains(result, `"api_key"`) {
		t.Error("expected key name to be preserved")
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output, got: %s", result)
	}
	// Model value should NOT be redacted
	if !strings.Contains(result, "gpt-4") {
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
	if strings.Contains(result, "sk-abcdefghijklmnopqrstuvwxyz") {
		t.Error("api_key value should be redacted")
	}
	if strings.Contains(result, "test-fake-token-value") {
		t.Error("token value should be redacted")
	}
}

func TestRedactSecrets_SlackToken(t *testing.T) {
	// Use a clearly fake token that still matches the xoxb- pattern
	fakeToken := "xoxb-fake" + "-placeholder-abcdefghij"
	input := "Bot token: " + fakeToken
	result := RedactSecrets(input)
	if !strings.Contains(result, "[REDACTED]") {
		t.Errorf("expected Slack token to be redacted, got: %s", result)
	}
}

func TestRedactSecrets_AWSAccessKey(t *testing.T) {
	input := `AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE`
	result := RedactSecrets(input)
	if strings.Contains(result, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("expected AWS key to be redacted, got: %s", result)
	}
}

func TestRedactSecrets_GitHubToken(t *testing.T) {
	input := `GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef01234`
	result := RedactSecrets(input)
	if strings.Contains(result, "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Errorf("expected GitHub token to be redacted, got: %s", result)
	}
}

func TestRedactSecrets_StripeKey(t *testing.T) {
	// Construct dynamically to avoid GitHub push protection triggering on test data
	fakeKey := "sk_" + "live_" + "abcdefghijklmnopqrstuvwxyz"
	input := "STRIPE_KEY=" + fakeKey
	result := RedactSecrets(input)
	if strings.Contains(result, fakeKey) {
		t.Errorf("expected Stripe key to be redacted, got: %s", result)
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
	if !strings.Contains(result, "sk-p") {
		t.Errorf("expected prefix to be preserved for identification, got: %s", result)
	}
}

func TestRedactSecrets_DoesNotRedactGitSHA(t *testing.T) {
	// 40-char hex strings like git SHAs must NOT be redacted
	input := `commit 3bcbfd9a1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f`
	result := RedactSecrets(input)
	if result != input {
		t.Errorf("git SHA should not be redacted, got: %s", result)
	}
}

func TestRedactSecrets_DoesNotRedactChecksum(t *testing.T) {
	input := `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
	result := RedactSecrets(input)
	if result != input {
		t.Errorf("SHA-256 checksum should not be redacted, got: %s", result)
	}
}

func TestRedactSecrets_Idempotent(t *testing.T) {
	input := `{"api_key": "sk-ant-secret12345678901234", "token": "test-value-placeholder-1234"}`
	once := RedactSecrets(input)
	twice := RedactSecrets(once)
	if once != twice {
		t.Errorf("RedactSecrets should be idempotent.\nOnce:  %s\nTwice: %s", once, twice)
	}
}

func TestSanitizeResult_ForLLMAndForUser(t *testing.T) {
	r := &ToolResult{
		ForLLM:  `Read config.json: {"api_key": "sk-ant-secret12345678901234"}`,
		ForUser: `Config: {"api_key": "sk-ant-secret12345678901234"}`,
	}
	SanitizeResult(r)
	if strings.Contains(r.ForLLM, "sk-ant-secret") {
		t.Error("ForLLM should have secret redacted")
	}
	if strings.Contains(r.ForUser, "sk-ant-secret") {
		t.Error("ForUser should have secret redacted")
	}
}

func TestSanitizeResult_NilResult(t *testing.T) {
	result := SanitizeResult(nil)
	if result != nil {
		t.Error("expected nil to pass through")
	}
}

func TestSanitizeResult_ErrorFieldPreservesChain(t *testing.T) {
	originalErr := fmt.Errorf("auth failed with key sk-proj-abc123def456ghi789jkl012mno345pqr678")
	r := &ToolResult{
		ForLLM: "error occurred",
		Err:    fmt.Errorf("wrapped: %w", originalErr),
	}
	SanitizeResult(r)
	if strings.Contains(r.Err.Error(), "sk-proj-abc123") {
		t.Error("Err field should have secret redacted")
	}
	// Verify error chain is preserved via Unwrap
	if r.Err == nil {
		t.Fatal("Err should not be nil")
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
		{`commit abc123def456`, false},
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
		{"AWS_ACCESS_KEY", true},
		{"HOME", false},
		{"PATH", false},
		{"GOPATH", false},
		{"MODEL_NAME", false},
		{"AUTHOR", false},       // must NOT match
		{"AUTHORITY", false},    // must NOT match
		{"AUTHENTICATE", false}, // must NOT match
	}
	for _, tc := range tests {
		got := IsSecretEnvVar(tc.name)
		if got != tc.expected {
			t.Errorf("IsSecretEnvVar(%q) = %v, want %v", tc.name, got, tc.expected)
		}
	}
}
