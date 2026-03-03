package auth

import (
	"strings"
	"testing"
)

func TestLoginSetupToken(t *testing.T) {
	// A valid token: correct prefix + at least 80 chars
	validToken := "sk-ant-oat01-" + strings.Repeat("a", 80)

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"valid token", validToken, ""},
		{"empty input", "", "expected prefix sk-ant-oat01-"},
		{"wrong prefix", "sk-ant-api-" + strings.Repeat("a", 80), "expected prefix sk-ant-oat01-"},
		{"too short", "sk-ant-oat01-short", "too short"},
		{"whitespace only", "   ", "expected prefix sk-ant-oat01-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input + "\n")
			cred, err := LoginSetupToken(r)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cred.AccessToken != validToken {
				t.Errorf("AccessToken = %q, want %q", cred.AccessToken, validToken)
			}
			if cred.Provider != "anthropic" {
				t.Errorf("Provider = %q, want %q", cred.Provider, "anthropic")
			}
			if cred.AuthMethod != "oauth" {
				t.Errorf("AuthMethod = %q, want %q", cred.AuthMethod, "oauth")
			}
		})
	}
}

func TestLoginSetupToken_EmptyReader(t *testing.T) {
	r := strings.NewReader("")
	_, err := LoginSetupToken(r)
	if err == nil {
		t.Fatal("expected error for empty reader, got nil")
	}
}
