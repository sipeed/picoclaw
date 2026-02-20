package providers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadCodexCliCredentials_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	authJSON := `{
		"tokens": {
			"access_token": "test-access-token",
			"refresh_token": "test-refresh-token",
			"account_id": "org-test123"
		}
	}`
	if err := os.WriteFile(authPath, []byte(authJSON), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", tmpDir)

	token, accountID, expiresAt, err := ReadCodexCliCredentials()
	if err != nil {
		t.Fatalf("ReadCodexCliCredentials() error: %v", err)
	}
	if token != "test-access-token" {
		t.Errorf("token = %q, want %q", token, "test-access-token")
	}
	if accountID != "org-test123" {
		t.Errorf("accountID = %q, want %q", accountID, "org-test123")
	}
	// Expiry should be within ~1 hour from now (file was just written)
	if expiresAt.Before(time.Now()) {
		t.Errorf("expiresAt = %v, should be in the future", expiresAt)
	}
	if expiresAt.After(time.Now().Add(2 * time.Hour)) {
		t.Errorf("expiresAt = %v, should be within ~1 hour", expiresAt)
	}
}

func TestReadCodexCliCredentials_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CODEX_HOME", tmpDir)

	_, _, _, err := ReadCodexCliCredentials()
	if err == nil {
		t.Fatal("expected error for missing auth.json")
	}
}

func TestReadCodexCliCredentials_EmptyToken(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	authJSON := `{"tokens": {"access_token": "", "refresh_token": "r", "account_id": "a"}}`
	if err := os.WriteFile(authPath, []byte(authJSON), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", tmpDir)

	_, _, _, err := ReadCodexCliCredentials()
	if err == nil {
		t.Fatal("expected error for empty access_token")
	}
}

func TestReadCodexCliCredentials_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	if err := os.WriteFile(authPath, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", tmpDir)

	_, _, _, err := ReadCodexCliCredentials()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadCodexCliCredentials_NoAccountID(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	authJSON := `{"tokens": {"access_token": "tok123", "refresh_token": "ref456"}}`
	if err := os.WriteFile(authPath, []byte(authJSON), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", tmpDir)

	token, accountID, _, err := ReadCodexCliCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok123" {
		t.Errorf("token = %q, want %q", token, "tok123")
	}
	if accountID != "" {
		t.Errorf("accountID = %q, want empty", accountID)
	}
}

func TestReadCodexCliCredentials_CodexHomeEnv(t *testing.T) {
	tmpDir := t.TempDir()
	customDir := filepath.Join(tmpDir, "custom-codex")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}

	authJSON := `{"tokens": {"access_token": "custom-token", "refresh_token": "r"}}`
	if err := os.WriteFile(filepath.Join(customDir, "auth.json"), []byte(authJSON), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", customDir)

	token, _, _, err := ReadCodexCliCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "custom-token" {
		t.Errorf("token = %q, want %q", token, "custom-token")
	}
}

func TestCreateCodexCliTokenSource_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	authJSON := `{"tokens": {"access_token": "fresh-token", "refresh_token": "r", "account_id": "acc"}}`
	if err := os.WriteFile(authPath, []byte(authJSON), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", tmpDir)

	source := CreateCodexCliTokenSource()
	token, accountID, err := source()
	if err != nil {
		t.Fatalf("token source error: %v", err)
	}
	if token != "fresh-token" {
		t.Errorf("token = %q, want %q", token, "fresh-token")
	}
	if accountID != "acc" {
		t.Errorf("accountID = %q, want %q", accountID, "acc")
	}
}

func TestCreateCodexCliTokenSource_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	authJSON := `{"tokens": {"access_token": "old-token", "refresh_token": "r"}}`
	if err := os.WriteFile(authPath, []byte(authJSON), 0600); err != nil {
		t.Fatal(err)
	}

	// Set file modification time to 2 hours ago
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(authPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CODEX_HOME", tmpDir)

	source := CreateCodexCliTokenSource()
	_, _, err := source()
	if err == nil {
		t.Fatal("expected error for expired credentials")
	}
}
