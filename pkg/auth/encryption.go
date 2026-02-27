package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
)

var (
	ErrEncryptionFailed     = errors.New("encryption failed")
	ErrDecryptionFailed     = errors.New("decryption failed")
	ErrInvalidCiphertext    = errors.New("invalid ciphertext")
	ErrKeyNotFound          = errors.New("encryption key not found")
	ErrUnsupportedAlgorithm = errors.New("unsupported encryption algorithm")
)

// EncryptionAlgorithm defines the supported encryption algorithms.
type EncryptionAlgorithm string

const (
	AlgorithmChaCha20Poly1305 EncryptionAlgorithm = "chacha20-poly1305"
	AlgorithmAES256GCM        EncryptionAlgorithm = "aes-256-gcm"
)

// EncryptedData represents encrypted credential data with metadata.
type EncryptedData struct {
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// Encryptor provides encryption/decryption functionality for credentials.
type Encryptor struct {
	algorithm EncryptionAlgorithm
	key       []byte
}

// NewEncryptor creates a new encryptor with the specified algorithm.
func NewEncryptor(algorithm string) (*Encryptor, error) {
	alg := EncryptionAlgorithm(algorithm)
	if alg != AlgorithmChaCha20Poly1305 && alg != AlgorithmAES256GCM {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algorithm)
	}

	key, err := getOrCreateEncryptionKey(alg)
	if err != nil {
		return nil, fmt.Errorf("getting encryption key: %w", err)
	}

	return &Encryptor{
		algorithm: alg,
		key:       key,
	}, nil
}

// Encrypt encrypts the given data and returns base64-encoded ciphertext.
func (e *Encryptor) Encrypt(plaintext []byte) (*EncryptedData, error) {
	switch e.algorithm {
	case AlgorithmChaCha20Poly1305:
		return e.encryptChaCha20Poly1305(plaintext)
	case AlgorithmAES256GCM:
		return e.encryptAES256GCM(plaintext)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, e.algorithm)
	}
}

// Decrypt decrypts the given encrypted data.
func (e *Encryptor) Decrypt(data *EncryptedData) ([]byte, error) {
	switch data.Algorithm {
	case string(AlgorithmChaCha20Poly1305):
		return e.decryptChaCha20Poly1305(data)
	case string(AlgorithmAES256GCM):
		return e.decryptAES256GCM(data)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, data.Algorithm)
	}
}

// EncryptCredential encrypts an AuthCredential struct.
func (e *Encryptor) EncryptCredential(cred *AuthCredential) (*EncryptedData, error) {
	data, err := json.Marshal(cred)
	if err != nil {
		return nil, fmt.Errorf("marshaling credential: %w", err)
	}
	return e.Encrypt(data)
}

// DecryptCredential decrypts encrypted data into an AuthCredential.
func (e *Encryptor) DecryptCredential(encData *EncryptedData) (*AuthCredential, error) {
	plaintext, err := e.Decrypt(encData)
	if err != nil {
		return nil, err
	}

	var cred AuthCredential
	if err := json.Unmarshal(plaintext, &cred); err != nil {
		return nil, fmt.Errorf("unmarshaling credential: %w", err)
	}

	return &cred, nil
}

func (e *Encryptor) encryptChaCha20Poly1305(plaintext []byte) (*EncryptedData, error) {
	aead, err := chacha20poly1305.NewX(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: creating cipher: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: generating nonce: %v", ErrEncryptionFailed, err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	return &EncryptedData{
		Algorithm:  string(AlgorithmChaCha20Poly1305),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func (e *Encryptor) decryptChaCha20Poly1305(data *EncryptedData) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: creating cipher: %v", ErrDecryptionFailed, err)
	}

	nonce, err := base64.StdEncoding.DecodeString(data.Nonce)
	if err != nil {
		return nil, fmt.Errorf("%w: decoding nonce: %v", ErrInvalidCiphertext, err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: decoding ciphertext: %v", ErrInvalidCiphertext, err)
	}

	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("%w: invalid nonce size", ErrInvalidCiphertext)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: decrypting: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func (e *Encryptor) encryptAES256GCM(plaintext []byte) (*EncryptedData, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: creating cipher: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: creating GCM: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: generating nonce: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedData{
		Algorithm:  string(AlgorithmAES256GCM),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func (e *Encryptor) decryptAES256GCM(data *EncryptedData) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: creating cipher: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: creating GCM: %v", ErrDecryptionFailed, err)
	}

	nonce, err := base64.StdEncoding.DecodeString(data.Nonce)
	if err != nil {
		return nil, fmt.Errorf("%w: decoding nonce: %v", ErrInvalidCiphertext, err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: decoding ciphertext: %v", ErrInvalidCiphertext, err)
	}

	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("%w: invalid nonce size", ErrInvalidCiphertext)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: decrypting: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// getOrCreateEncryptionKey retrieves or creates an encryption key for file-based encryption.
// This is used as a fallback when OS keychain is not available.
func getOrCreateEncryptionKey(algorithm EncryptionAlgorithm) ([]byte, error) {
	keySize := 32 // Both ChaCha20-Poly1305 and AES-256 use 32-byte keys

	keyPath, err := encryptionKeyPath()
	if err != nil {
		return nil, err
	}

	// Try to read existing key
	key, err := os.ReadFile(keyPath)
	if err == nil && len(key) == keySize {
		return key, nil
	}

	// Generate new key
	key = make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating key directory: %w", err)
	}

	// Write key with restricted permissions
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("writing key: %w", err)
	}

	return key, nil
}

func encryptionKeyPath() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home dir: %w", err)
		}
	}
	return filepath.Join(home, ".picoclaw", ".key"), nil
}

// DeleteEncryptionKey removes the encryption key file.
// This should be called when all credentials are deleted.
func DeleteEncryptionKey() error {
	keyPath, err := encryptionKeyPath()
	if err != nil {
		return err
	}

	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
