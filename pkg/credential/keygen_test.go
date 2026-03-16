package credential

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateSSHKey_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test_ed25519.key")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	// Private key must exist.
	privInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("private key file missing: %v", err)
	}

	// Check permissions on non-Windows (Windows does not support Unix permission bits).
	if runtime.GOOS != "windows" {
		if got := privInfo.Mode().Perm(); got != 0o600 {
			t.Errorf("private key permissions = %04o, want 0600", got)
		}
	}

	// Public key must exist.
	pubPath := keyPath + ".pub"
	pubInfo, err := os.Stat(pubPath)
	if err != nil {
		t.Fatalf("public key file missing: %v", err)
	}
	if runtime.GOOS != "windows" {
		if got := pubInfo.Mode().Perm(); got != 0o644 {
			t.Errorf("public key permissions = %04o, want 0644", got)
		}
	}

	// Private key must be parseable as an OpenSSH ed25519 key.
	privPEM, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	privKey, err := ssh.ParseRawPrivateKey(privPEM)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}
	if _, ok := privKey.(*ed25519.PrivateKey); !ok {
		t.Errorf("private key type = %T, want *ed25519.PrivateKey", privKey)
	}

	// Public key must be parseable as authorized_keys line.
	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatalf("read public key: %v", err)
	}
	pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(pubBytes)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	if pubKey == nil {
		t.Fatal("expected non-nil public key")
	}
	if len(rest) > 0 {
		t.Errorf("unexpected trailing bytes after public key: %d bytes", len(rest))
	}
}

func TestGenerateSSHKey_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test_ed25519.key")

	// Generate twice; second call must not error and must produce a different key.
	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("first GenerateSSHKey() error = %v", err)
	}
	first, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read first key: %v", err)
	}

	if err = GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("second GenerateSSHKey() error = %v", err)
	}
	second, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read second key: %v", err)
	}

	// Two independently generated Ed25519 keys must differ.
	if string(first) == string(second) {
		t.Error("expected overwritten key to differ from original")
	}
}

func TestGenerateSSHKey_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Nested directory that does not yet exist.
	keyPath := filepath.Join(dir, "subdir", ".ssh", "picoclaw_ed25519.key")

	if err := GenerateSSHKey(keyPath); err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("private key not created: %v", err)
	}
}
