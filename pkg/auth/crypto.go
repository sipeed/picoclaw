package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

const encryptedPrefix = "enc:v1:"

// DeriveKey derives a 32-byte AES-256 key from a master secret using SHA-256.
func DeriveKey(masterSecret string) []byte {
	h := sha256.Sum256([]byte(masterSecret))
	return h[:]
}

// GetMasterKey reads the master key from the PICOCLAW_MASTER_KEY environment variable.
func GetMasterKey() string {
	return os.Getenv("PICOCLAW_MASTER_KEY")
}

// Encrypt encrypts plaintext using AES-256-GCM and returns "enc:v1:<base64>".
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encryptedPrefix + encoded, nil
}

// Decrypt decrypts a value encrypted by Encrypt. If the value does not have the
// encrypted prefix, it is returned unchanged (plaintext passthrough).
func Decrypt(value string, key []byte) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}

	encoded := strings.TrimPrefix(value, encryptedPrefix)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted returns true if the value has the encrypted prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}
