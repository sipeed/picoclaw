package credential_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/credential"
)

func TestResolve_PlainKey(t *testing.T) {
	r := credential.NewResolver(t.TempDir())
	got, err := r.Resolve("sk-plaintext-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-plaintext-key" {
		t.Fatalf("got %q, want %q", got, "sk-plaintext-key")
	}
}

func TestResolve_FileKey_Success(t *testing.T) {
	dir := t.TempDir()
	keyFile := "openai_plain.key"
	if err := os.WriteFile(filepath.Join(dir, keyFile), []byte("sk-from-file\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := credential.NewResolver(dir)
	got, err := r.Resolve("file://" + keyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-from-file" {
		t.Fatalf("got %q, want %q", got, "sk-from-file")
	}
}

func TestResolve_FileKey_NotFound(t *testing.T) {
	r := credential.NewResolver(t.TempDir())
	_, err := r.Resolve("file://missing.key")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestResolve_FileKey_Empty(t *testing.T) {
	dir := t.TempDir()
	keyFile := "empty.key"
	if err := os.WriteFile(filepath.Join(dir, keyFile), []byte("   \n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := credential.NewResolver(dir)
	_, err := r.Resolve("file://" + keyFile)
	if err == nil {
		t.Fatal("expected error for empty credential file, got nil")
	}
}

// TestResolve_EncKey_RoundTrip tests basic encryption/decryption round-trip with an SSH key.
func TestResolve_EncKey_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err := os.WriteFile(sshKeyPath, []byte("fake-ssh-key-material\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	const passphrase = "test-passphrase-32bytes-long-ok!"
	const plaintext = "sk-encrypted-secret"

	t.Setenv("PICOCLAW_SSH_KEY_PATH", sshKeyPath)

	enc, err := credential.Encrypt(passphrase, "", plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", passphrase)

	r := credential.NewResolver(t.TempDir())
	got, err := r.Resolve(enc)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != plaintext {
		t.Fatalf("got %q, want %q", got, plaintext)
	}
}

// TestResolve_EncKey_WithSSHKey tests that the SSH key file is incorporated into key derivation.
func TestResolve_EncKey_WithSSHKey(t *testing.T) {
	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err := os.WriteFile(sshKeyPath, []byte("fake-ssh-private-key-material\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	const passphrase = "test-passphrase"
	const plaintext = "sk-ssh-protected-secret"

	// Set PICOCLAW_SSH_KEY_PATH before Encrypt so the path passes allowedSSHKeyPath validation.
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", passphrase)
	t.Setenv("PICOCLAW_SSH_KEY_PATH", sshKeyPath)

	enc, err := credential.Encrypt(passphrase, sshKeyPath, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	r := credential.NewResolver(t.TempDir())
	got, err := r.Resolve(enc)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != plaintext {
		t.Fatalf("got %q, want %q", got, plaintext)
	}
}

func TestResolve_EncKey_NoPassphrase(t *testing.T) {
	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err := os.WriteFile(sshKeyPath, []byte("fake-ssh-key\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("PICOCLAW_SSH_KEY_PATH", sshKeyPath)

	enc, err := credential.Encrypt("some-passphrase", "", "sk-secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "")

	r := credential.NewResolver(t.TempDir())
	_, err = r.Resolve(enc)
	if err == nil {
		t.Fatal("expected error when PICOCLAW_KEY_PASSPHRASE is unset, got nil")
	}
}

func TestResolve_EncKey_BadCiphertext(t *testing.T) {
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "some-passphrase")
	t.Setenv("PICOCLAW_SSH_KEY_PATH", "")

	r := credential.NewResolver(t.TempDir())
	_, err := r.Resolve("enc://!!not-valid-base64!!")
	if err == nil {
		t.Fatal("expected error for invalid enc:// payload, got nil")
	}
}

func TestResolve_EncKey_PayloadTooShort(t *testing.T) {
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "some-passphrase")
	t.Setenv("PICOCLAW_SSH_KEY_PATH", "")

	// Valid base64 but fewer bytes than salt(16)+nonce(12)+1 minimum.
	import64 := "dG9vc2hvcnQ=" // "tooshort" = 8 bytes
	r := credential.NewResolver(t.TempDir())
	_, err := r.Resolve("enc://" + import64)
	if err == nil {
		t.Fatal("expected error for too-short enc:// payload, got nil")
	}
}

func TestResolve_EncKey_WrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err := os.WriteFile(sshKeyPath, []byte("fake-ssh-key\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("PICOCLAW_SSH_KEY_PATH", sshKeyPath)

	enc, err := credential.Encrypt("correct-passphrase", "", "sk-secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "wrong-passphrase")

	r := credential.NewResolver(t.TempDir())
	_, err = r.Resolve(enc)
	if err == nil {
		t.Fatal("expected decryption error for wrong passphrase, got nil")
	}
}

func TestEncrypt_EmptyPassphrase(t *testing.T) {
	_, err := credential.Encrypt("", "", "sk-secret")
	if err == nil {
		t.Fatal("expected error for empty passphrase, got nil")
	}
}

func TestDeriveKey_SSHKeyNotFound(t *testing.T) {
	// Encrypt with a real SSH key path, then try to decrypt with a missing path.
	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err := os.WriteFile(sshKeyPath, []byte("fake-key\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Register the real key path so allowedSSHKeyPath validation passes for Encrypt.
	t.Setenv("PICOCLAW_SSH_KEY_PATH", sshKeyPath)

	enc, err := credential.Encrypt("passphrase", sshKeyPath, "sk-secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Point to a non-existent SSH key so deriveKey's ReadFile fails.
	// The path is still under the same dir, so allowedSSHKeyPath passes (exact env match).
	t.Setenv("PICOCLAW_KEY_PASSPHRASE", "passphrase")
	t.Setenv("PICOCLAW_SSH_KEY_PATH", filepath.Join(dir, "nonexistent_key"))

	r := credential.NewResolver(t.TempDir())
	_, err = r.Resolve(enc)
	if err == nil {
		t.Fatal("expected error when SSH key file is missing, got nil")
	}
}

// TestResolve_FileRef_PathTraversal verifies that file:// references cannot escape configDir
// via relative traversal ("../../etc/passwd") or absolute paths ("/abs/path").
func TestResolve_FileRef_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	// Create a file outside configDir that the traversal would point to.
	outsideFile := filepath.Join(t.TempDir(), "secret.key")
	if err := os.WriteFile(outsideFile, []byte("stolen"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := credential.NewResolver(filepath.Dir(cfgPath))

	cases := []string{
		"file://../../secret.key",
		"file://../secret.key",
		"file://" + outsideFile, // absolute path
	}
	for _, raw := range cases {
		_, err := r.Resolve(raw)
		if err == nil {
			t.Errorf("Resolve(%q): expected path traversal error, got nil", raw)
		}
	}
}

// TestResolve_FileRef_withinConfigDir verifies that a legitimate relative file:// ref works.
func TestResolve_FileRef_withinConfigDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "my.key"), []byte("sk-valid\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	r := credential.NewResolver(dir)
	got, err := r.Resolve("file://my.key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-valid" {
		t.Fatalf("got %q, want %q", got, "sk-valid")
	}
}

// TestEncrypt_SSHKeyOutsideAllowedDirs verifies that Encrypt rejects SSH key paths
// that are not under PICOCLAW_SSH_KEY_PATH, PICOCLAW_HOME, or ~/.ssh/.
func TestEncrypt_SSHKeyOutsideAllowedDirs(t *testing.T) {
	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err := os.WriteFile(sshKeyPath, []byte("fake-key\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Make sure none of the allowed env vars point here.
	t.Setenv("PICOCLAW_SSH_KEY_PATH", "")
	t.Setenv("PICOCLAW_HOME", "")

	_, err := credential.Encrypt("passphrase", sshKeyPath, "sk-secret")
	if err == nil {
		t.Fatal("expected error for SSH key outside allowed directories, got nil")
	}
}
