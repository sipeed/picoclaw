// Package credential resolves API credential values for model_list entries.
//
// An API key is a form of authorization credential. This package centralises
// how raw credential strings—plaintext or file references—are resolved into
// their actual values, keeping that logic out of the config loader.
//
// Supported formats for the api_key field:
//
//   - Plaintext:   "sk-abc123"          → returned as-is
//   - File ref:    "file://filename.key" → content read from configDir/filename.key
//   - Encrypted:   "enc://<base64>"     → AES-256-GCM decrypt via PICOCLAW_KEY_PASSPHRASE
//   - Empty:       ""                   → returned as-is (auth_method=oauth etc.)
//
// Encryption uses AES-256-GCM with HKDF-SHA256 key derivation (< 1ms, safe for embedded Linux).
// Key derivation:
//
//	With SSH key:  HKDF-SHA256(ikm=HMAC-SHA256(SHA256(sshKeyBytes), passphrase), salt, info)
//	Without:       HKDF-SHA256(ikm=SHA256(passphrase), salt, info)
//
// SSH key path resolution priority:
//
//  1. sshKeyPath argument to Encrypt (explicit)
//  2. PICOCLAW_SSH_KEY_PATH env var (set to "" to disable auto-detection)
//  3. ~/.ssh/picoclaw_ed25519.key (os.UserHomeDir is cross-platform)
package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PassphraseEnvVar is the environment variable that holds the encryption passphrase.
// Other packages (e.g. config) reference this constant to avoid duplicating the string.
const PassphraseEnvVar = "PICOCLAW_KEY_PASSPHRASE"

// PassphraseProvider is the function used to retrieve the passphrase for enc://
// credential decryption. It defaults to reading PICOCLAW_KEY_PASSPHRASE from the
// process environment. Replace it at startup to use a different source, such as
// an in-memory SecureStore, so that all LoadConfig() calls everywhere share the
// same passphrase source without needing os.Environ.
//
// Example (launcher main.go):
//
//	credential.PassphraseProvider = apiHandler.passphraseStore.Get
var PassphraseProvider func() string = func() string {
	return os.Getenv(PassphraseEnvVar)
}

// ErrPassphraseRequired is returned when an enc:// credential is encountered but
// PICOCLAW_KEY_PASSPHRASE is not set. Callers can detect this with errors.Is to
// distinguish a missing-passphrase condition from other credential errors.
var ErrPassphraseRequired = errors.New("credential: enc:// key requires " + PassphraseEnvVar + " env var")

// ErrDecryptionFailed is returned when an enc:// credential cannot be decrypted,
// indicating a wrong passphrase or SSH key. Callers can detect this with errors.Is.
var ErrDecryptionFailed = errors.New("credential: enc:// decryption failed (wrong passphrase or SSH key?)")

const (
	fileScheme = "file://"
	encScheme  = "enc://"
	hkdfInfo   = "picoclaw-credential-v1"
	saltLen    = 16
	nonceLen   = 12
	keyLen     = 32
	sshKeyEnv  = "PICOCLAW_SSH_KEY_PATH"
)

// Resolver resolves raw credential strings for model_list api_key fields.
// File references are resolved relative to the directory of the config file.
type Resolver struct {
	configDir string
}

// NewResolver returns a Resolver that resolves file:// references relative to
// configDir (typically filepath.Dir of the config file path).
func NewResolver(configDir string) *Resolver {
	return &Resolver{configDir: configDir}
}

// Resolve returns the actual credential value for raw:
//
//   - ""                → "" (no error; auth_method=oauth needs no key)
//   - "file://name.key" → trimmed content of configDir/name.key
//   - anything else     → raw unchanged (plaintext credential)
func (r *Resolver) Resolve(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	if strings.HasPrefix(raw, fileScheme) {
		fileName := strings.TrimSpace(strings.TrimPrefix(raw, fileScheme))
		if fileName == "" {
			return "", fmt.Errorf("credential: file:// reference has no filename")
		}

		keyPath := filepath.Join(r.configDir, fileName)
		// Prevent path traversal: "../../etc/passwd" or "/abs/path" must not escape configDir.
		if !isWithinDir(keyPath, r.configDir) {
			return "", fmt.Errorf("credential: file:// path escapes config directory")
		}
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return "", fmt.Errorf("credential: failed to read credential file %q: %w", keyPath, err)
		}

		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("credential: credential file %q is empty", keyPath)
		}

		return value, nil
	}

	if strings.HasPrefix(raw, encScheme) {
		return resolveEncrypted(raw)
	}

	// Plaintext credential — return unchanged.
	return raw, nil
}

// resolveEncrypted decrypts an enc:// credential using PassphraseProvider.
func resolveEncrypted(raw string) (string, error) {
	passphrase := PassphraseProvider()
	if passphrase == "" {
		return "", ErrPassphraseRequired
	}

	sshKeyPath := pickSSHKeyPath("") // override="": consult env then auto-detect

	b64 := strings.TrimPrefix(raw, encScheme)
	blob, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("credential: enc:// invalid base64: %w", err)
	}
	if len(blob) < saltLen+nonceLen+1 {
		return "", fmt.Errorf("credential: enc:// payload too short")
	}

	salt := blob[:saltLen]
	nonce := blob[saltLen : saltLen+nonceLen]
	ciphertext := blob[saltLen+nonceLen:]

	key, err := deriveKey(passphrase, sshKeyPath, salt)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("credential: enc:// cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("credential: enc:// gcm init: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}
	return string(plaintext), nil
}

// Encrypt encrypts plaintext and returns an enc:// credential string.
//
// passphrase is required (PICOCLAW_KEY_PASSPHRASE value).
// sshKeyPath is the SSH private key file to incorporate; pass "" to use
// PICOCLAW_SSH_KEY_PATH env var or ~/.ssh/ auto-detection, or set
// PICOCLAW_SSH_KEY_PATH="" before calling to force passphrase-only mode.
func Encrypt(passphrase, sshKeyPath, plaintext string) (string, error) {
	if passphrase == "" {
		return "", fmt.Errorf("credential: passphrase must not be empty")
	}
	sshKeyPath = pickSSHKeyPath(sshKeyPath)

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("credential: failed to generate salt: %w", err)
	}

	key, err := deriveKey(passphrase, sshKeyPath, salt)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("credential: cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("credential: gcm init: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("credential: failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	blob := make([]byte, 0, saltLen+nonceLen+len(ciphertext))
	blob = append(blob, salt...)
	blob = append(blob, nonce...)
	blob = append(blob, ciphertext...)
	return encScheme + base64.StdEncoding.EncodeToString(blob), nil
}

// isWithinDir reports whether path is contained within (or equal to) dir.
// Uses filepath.IsLocal on the relative path for robust cross-platform traversal detection.
func isWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(path))
	return err == nil && filepath.IsLocal(rel)
}

// allowedSSHKeyPath reports whether path is in a permitted location for SSH key files:
//   - exact match with PICOCLAW_SSH_KEY_PATH env var
//   - within the PICOCLAW_HOME env var directory
//   - within ~/.ssh/
func allowedSSHKeyPath(path string) bool {
	if path == "" {
		return true // passphrase-only mode; no file will be read
	}
	clean := filepath.Clean(path)

	// Exact match with PICOCLAW_SSH_KEY_PATH.
	if envPath, ok := os.LookupEnv(sshKeyEnv); ok && envPath != "" {
		if clean == filepath.Clean(envPath) {
			return true
		}
	}

	// Within PICOCLAW_HOME.
	if picoHome := os.Getenv("PICOCLAW_HOME"); picoHome != "" {
		if isWithinDir(clean, picoHome) {
			return true
		}
	}

	// Within ~/.ssh/.
	if userHome, err := os.UserHomeDir(); err == nil {
		if isWithinDir(clean, filepath.Join(userHome, ".ssh")) {
			return true
		}
	}

	return false
}

// deriveKey derives a 32-byte AES-256 key from passphrase and optional SSH key.
//
// With SSH key:  ikm = HMAC-SHA256(key=SHA256(sshKeyBytes), msg=passphrase)
// Without:       ikm = SHA256(passphrase)
// Final key:     HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
func deriveKey(passphrase, sshKeyPath string, salt []byte) ([]byte, error) {
	var ikm []byte
	if sshKeyPath != "" {
		if !allowedSSHKeyPath(sshKeyPath) {
			return nil, fmt.Errorf("credential: SSH key path %q is not in an allowed location (PICOCLAW_SSH_KEY_PATH, PICOCLAW_HOME, or ~/.ssh/)", sshKeyPath)
		}
		sshBytes, err := os.ReadFile(sshKeyPath)
		if err != nil {
			return nil, fmt.Errorf("credential: cannot read SSH key %q: %w", sshKeyPath, err)
		}
		sshHash := sha256.Sum256(sshBytes)
		mac := hmac.New(sha256.New, sshHash[:])
		mac.Write([]byte(passphrase))
		ikm = mac.Sum(nil)
	} else {
		h := sha256.Sum256([]byte(passphrase))
		ikm = h[:]
	}

	key, err := hkdf.Key(sha256.New, ikm, salt, hkdfInfo, keyLen)
	if err != nil {
		return nil, fmt.Errorf("credential: HKDF expand failed: %w", err)
	}
	return key, nil
}

// pickSSHKeyPath returns the SSH private key path to use for encryption/decryption.
//
// Priority:
//  1. override (non-empty explicit argument)
//  2. PICOCLAW_SSH_KEY_PATH env var — if the variable is set (even to ""), auto-detection
//     is skipped; set it to "" to force passphrase-only mode
//  3. ~/.ssh/picoclaw_ed25519.key (auto-detection)
//
// Returns "" when no key is found (passphrase-only mode).
func pickSSHKeyPath(override string) string {
	if override != "" {
		return override
	}
	if p, ok := os.LookupEnv(sshKeyEnv); ok {
		return p // respect explicit setting, even if ""
	}
	return findDefaultSSHKey()
}

// findDefaultSSHKey returns the picoclaw-specific SSH key path if it exists.
func findDefaultSSHKey() string {
	p, err := DefaultSSHKeyPath()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
