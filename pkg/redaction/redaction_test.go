package redaction

import (
	"testing"
)

func TestRedactor_Redact_APIKeys(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	tests := []struct {
		name       string
		input      string
		wantRedact bool
	}{
		{
			name:       "OpenAI key",
			input:      "api_key=sk-proj-1234567890abcdefghijklmnop",
			wantRedact: true,
		},
		{
			name:       "Anthropic key",
			input:      "api_key: sk-ant-api03-1234567890abcdefghijklmnop",
			wantRedact: true,
		},
		{
			name:       "Bearer token",
			input:      "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantRedact: true,
		},
		{
			name:       "JWT token",
			input:      "token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			wantRedact: true,
		},
		{
			name:       "AWS access key",
			input:      "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			wantRedact: true,
		},
		{
			name:       "plain text not redacted",
			input:      "This is a normal message without sensitive data",
			wantRedact: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			if tt.wantRedact {
				if result == tt.input {
					t.Errorf("Expected redaction for %q, got unchanged", tt.name)
				}
				if !contains(result, "[REDACTED]") {
					t.Errorf("Expected [REDACTED] in result, got: %s", result)
				}
			} else {
				if result != tt.input {
					t.Errorf("Unexpected redaction for %q: %s", tt.name, result)
				}
			}
		})
	}
}

func TestRedactor_Redact_Emails(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple email",
			input:    "Contact: test@example.com",
			expected: "Contact: t***@example.com",
		},
		{
			name:     "email in JSON",
			input:    `{"email": "user.name@company.org"}`,
			expected: `{"email": "u***@company.org"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			if result == tt.input {
				t.Errorf("Expected email to be masked, got: %s", result)
			}
		})
	}
}

func TestRedactor_Redact_Passwords(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	tests := []struct {
		name       string
		input      string
		wantRedact bool
	}{
		{
			name:       "password field",
			input:      "password=mysecretpassword123",
			wantRedact: true,
		},
		{
			name:       "passwd field",
			input:      "passwd: secret123",
			wantRedact: true,
		},
		{
			name:       "JSON password",
			input:      `{"password": "mysecret", "user": "john"}`,
			wantRedact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			if tt.wantRedact && result == tt.input {
				t.Errorf("Expected password redaction for %q, got unchanged", tt.name)
			}
		})
	}
}

func TestRedactor_Redact_PhoneNumbers(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	tests := []struct {
		name       string
		input      string
		wantRedact bool
	}{
		{
			name:       "US phone format",
			input:      "Phone: (555) 123-4567",
			wantRedact: true,
		},
		{
			name:       "International format",
			input:      "Phone: +1 555 123 4567",
			wantRedact: true,
		},
		{
			name:       "Simple format",
			input:      "Call 555-123-4567",
			wantRedact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			if tt.wantRedact && result == tt.input {
				t.Errorf("Expected phone redaction for %q, got unchanged", tt.name)
			}
		})
	}
}

func TestRedactor_Redact_IPAddresses(t *testing.T) {
	config := DefaultConfig()
	config.RedactIPAddresses = true
	r := NewRedactor(config)

	tests := []struct {
		name       string
		input      string
		wantRedact bool
	}{
		{
			name:       "IPv4 address",
			input:      "Server IP: 192.168.1.100",
			wantRedact: true,
		},
		{
			name:       "Localhost",
			input:      "Connect to 127.0.0.1:8080",
			wantRedact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			if tt.wantRedact && result == tt.input {
				t.Errorf("Expected IP redaction for %q, got unchanged", tt.name)
			}
		})
	}
}

func TestRedactor_RedactFields(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	tests := []struct {
		name       string
		input      map[string]any
		wantRedact []string // keys that should be redacted
	}{
		{
			name: "password field",
			input: map[string]any{
				"username": "john",
				"password": "secret123",
			},
			wantRedact: []string{"password"},
		},
		{
			name: "api_key field",
			input: map[string]any{
				"api_key": "sk-1234567890",
				"user":    "john",
			},
			wantRedact: []string{"api_key"},
		},
		{
			name: "nested fields",
			input: map[string]any{
				"config": map[string]any{
					"token": "abc123",
				},
			},
			wantRedact: []string{"token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactFields(tt.input)
			for _, key := range tt.wantRedact {
				// Check nested
				if nested, ok := result["config"].(map[string]any); ok {
					if val, exists := nested[key]; exists {
						if val == tt.input["config"].(map[string]any)[key] {
							t.Errorf("Expected %q to be redacted", key)
						}
					}
				} else if val, exists := result[key]; exists {
					if val == "[REDACTED]" {
						// Good
					} else if val == tt.input[key] {
						t.Errorf("Expected %q to be redacted, got: %v", key, val)
					}
				}
			}
		})
	}
}

func TestRedactor_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false
	r := NewRedactor(config)

	input := "password=mysecret123 api_key=sk-1234567890"
	result := r.Redact(input)

	if result != input {
		t.Errorf("Expected no redaction when disabled, got: %s", result)
	}
}

func TestRedactor_CustomPatterns(t *testing.T) {
	config := DefaultConfig()
	config.CustomPatterns = []string{`CUSTOM-[A-Z0-9]+`}
	r := NewRedactor(config)

	input := "Token: CUSTOM-ABC123XYZ"
	result := r.Redact(input)

	if !contains(result, "[REDACTED]") {
		t.Errorf("Expected custom pattern to be redacted, got: %s", result)
	}
}

func TestRedactor_AddCustomPattern(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	err := r.AddCustomPattern(`MYSECRET-[a-z]+`)
	if err != nil {
		t.Fatalf("Failed to add custom pattern: %v", err)
	}

	input := "Code: MYSECRET-hiddenvalue"
	result := r.Redact(input)

	if !contains(result, "[REDACTED]") {
		t.Errorf("Expected custom pattern to be redacted, got: %s", result)
	}
}

func TestMaskEmail(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	tests := []struct {
		email    string
		expected string
	}{
		{"test@example.com", "t***@example.com"},
		{"ab@domain.org", "a***@domain.org"},
		{"longemail@company.net", "l***@company.net"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := r.maskEmail(tt.email)
			if result != tt.expected {
				t.Errorf("maskEmail(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestIsSensitiveKey(t *testing.T) {
	r := NewRedactor(DefaultConfig())

	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"api_key", true},
		{"secret", true},
		{"token", true},
		{"access_token", true},
		{"credential", true},
		{"username", false},
		{"email", false},
		{"name", false},
		{"id", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := r.isSensitiveKey(tt.key)
			if result != tt.expected {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestGlobalRedactor(t *testing.T) {
	// Reset to default
	SetGlobalConfig(DefaultConfig())

	input := "password=secret123"
	result := Redact(input)

	if result == input {
		t.Error("Expected global Redact to redact sensitive data")
	}

	fields := map[string]any{
		"api_key": "sk-12345",
	}
	resultFields := RedactFields(fields)

	if resultFields["api_key"] != "[REDACTED]" {
		t.Error("Expected global RedactFields to redact sensitive fields")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
