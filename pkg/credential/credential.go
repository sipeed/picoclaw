// Package credential resolves API credential values for model_list entries.
//
// An API key is a form of authorization credential. This package centralizes
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
// An SSH private key is required for both encryption and decryption.
// Key derivation:
//
//	HKDF-SHA256(ikm=HMAC-SHA256(SHA256(sshKeyBytes), passphrase), salt, info)
//
// SSH key path resolution priority:
//
//  1. sshKeyPath argument to Encrypt (explicit)
//  2. PICOCLAW_SSH_KEY_PATH env var
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
// no passphrase is available from PassphraseProvider. Callers can detect this
// with errors.Is to distinguish a missing-passphrase condition from other errors.
var ErrPassphraseRequired = errors.New("credential: enc:// passphrase required")

// ErrDecryptionFailed is returned when an enc:// credential cannot be decrypted,
// indicating a wrong passphrase or SSH key. Callers can detect this with errors.Is.
var ErrDecryptionFailed = errors.New("credential: enc:// decryption failed (wrong passphrase or SSH key?)")

// SSHKeyPathEnvVar is the environment variable that specifies the path to the
// SSH private key used for enc:// credential encryption and decryption.
const SSHKeyPathEnvVar = "PICOCLAW_SSH_KEY_PATH"

// picoclawHome is a package-local copy of config.EnvHome. It is kept here to
// avoid a circular import between pkg/credential and pkg/config.
const picoclawHome = "PICOCLAW_HOME"

const (
	FileScheme = "file://"
	EncScheme  = "enc://"

	hkdfInfo = "picoclaw-credential-v1"
	saltLen  = 16
	nonceLen = 12
	keyLen   = 32
)

// Resolver resolves raw credential strings for model_list api_key fields.
// File references are resolved relative to the directory of the config file.
type Resolver struct {
	configDir         string
	resolvedConfigDir string // symlink-resolved form of configDir
}

// NewResolver returns a Resolver that resolves file:// references relative to
// configDir (typically filepath.Dir of the config file path).
func NewResolver(configDir string) *Resolver {
	resolved := configDir
	if configDir != "" {
		if linkedPath, err := filepath.EvalSymlinks(configDir); err == nil {
			resolved = linkedPath
		}
	}
	return &Resolver{configDir: configDir, resolvedConfigDir: resolved}
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

	if strings.HasPrefix(raw, FileScheme) {
		fileName := strings.TrimSpace(strings.TrimPrefix(raw, FileScheme))
		if fileName == "" {
			return "", fmt.Errorf("credential: file:// reference has no filename")
		}

		baseDir := r.resolvedConfigDir
		if baseDir == "" {
			baseDir = r.configDir
		}
		keyPath := filepath.Join(baseDir, fileName)
		// Resolve symlinks before enforcing containment to prevent escaping via symlinks.
		realKeyPath, err := filepath.EvalSymlinks(keyPath)
		if err != nil {
			return "", fmt.Errorf("credential: failed to resolve credential file path %q: %w", keyPath, err)
		}
		if !isWithinDir(realKeyPath, baseDir) {
			return "", fmt.Errorf("credential: file:// path escapes config directory")
		}
		data, err := os.ReadFile(realKeyPath)
		if err != nil {
			return "", fmt.Errorf("credential: failed to read credential file %q: %w", realKeyPath, err)
		}

		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("credential: credential file %q is empty", realKeyPath)
		}

		return value, nil
	}

	if strings.HasPrefix(raw, EncScheme) {
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

	b64 := strings.TrimPrefix(raw, EncScheme)
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
// sshKeyPath is the SSH private key file to use; pass "" to auto-detect via
// PICOCLAW_SSH_KEY_PATH env var or ~/.ssh/picoclaw_ed25519.key.
// An SSH private key must be resolvable or Encrypt returns an error.
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
	return EncScheme + base64.StdEncoding.EncodeToString(blob), nil
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
	if envPath, ok := os.LookupEnv(SSHKeyPathEnvVar); ok && envPath != "" {
		if clean == filepath.Clean(envPath) {
			return true
		}
	}

	// Within PICOCLAW_HOME.
	if picoHome := os.Getenv(picoclawHome); picoHome != "" {
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

// deriveKey derives a 32-byte AES-256 key from passphrase and SSH private key.
//
// ikm = HMAC-SHA256(key=SHA256(sshKeyBytes), msg=passphrase)
// Final key: HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
// sshKeyPath must be non-empty; returns an error otherwise.
func deriveKey(passphrase, sshKeyPath string, salt []byte) ([]byte, error) {
	if sshKeyPath == "" {
		return nil, fmt.Errorf(
			"credential: SSH private key is required but not found" +
				" (set PICOCLAW_SSH_KEY_PATH or place key at ~/.ssh/picoclaw_ed25519.key)")
	}
	if !allowedSSHKeyPath(sshKeyPath) {
		return nil, fmt.Errorf(
			"credential: SSH key path %q is not in an allowed location (PICOCLAW_SSH_KEY_PATH, PICOCLAW_HOME, or ~/.ssh/)",
			sshKeyPath,
		)
	}
	sshBytes, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("credential: cannot read SSH key %q: %w", sshKeyPath, err)
	}
	sshHash := sha256.Sum256(sshBytes)
	mac := hmac.New(sha256.New, sshHash[:])
	mac.Write([]byte(passphrase))
	ikm := mac.Sum(nil)

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
//  2. PICOCLAW_SSH_KEY_PATH env var
//  3. ~/.ssh/picoclaw_ed25519.key (auto-detection)
//
// Returns "" when no key is found; deriveKey will return an error in that case.
func pickSSHKeyPath(override string) string {
	if override != "" {
		return override
	}
	if p, ok := os.LookupEnv(SSHKeyPathEnvVar); ok {
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
