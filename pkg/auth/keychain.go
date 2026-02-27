package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringServiceName = "picoclaw"
	keyringUser        = "credentials"
)

var (
	ErrKeychainNotAvailable = errors.New("keychain not available")
	ErrKeychainAccessDenied = errors.New("keychain access denied")
)

// KeychainBackend provides an interface for OS keychain operations.
type KeychainBackend interface {
	// Store stores the credential in the OS keychain.
	Store(provider string, cred *AuthCredential) error
	// Retrieve retrieves the credential from the OS keychain.
	Retrieve(provider string) (*AuthCredential, error)
	// Delete removes the credential from the OS keychain.
	Delete(provider string) error
	// IsAvailable checks if the keychain is available on this system.
	IsAvailable() bool
}

// OSKeychain implements KeychainBackend using the OS-native keychain.
type OSKeychain struct{}

// NewOSKeychain creates a new OS keychain backend.
func NewOSKeychain() *OSKeychain {
	return &OSKeychain{}
}

// Store stores a credential in the OS keychain.
func (k *OSKeychain) Store(provider string, cred *AuthCredential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("marshaling credential: %w", err)
	}

	key := k.keyForProvider(provider)
	if err := keyring.Set(keyringServiceName, key, string(data)); err != nil {
		return fmt.Errorf("storing in keychain: %w", k.mapKeychainError(err))
	}

	return nil
}

// Retrieve retrieves a credential from the OS keychain.
func (k *OSKeychain) Retrieve(provider string) (*AuthCredential, error) {
	key := k.keyForProvider(provider)
	data, err := keyring.Get(keyringServiceName, key)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieving from keychain: %w", k.mapKeychainError(err))
	}

	var cred AuthCredential
	if err := json.Unmarshal([]byte(data), &cred); err != nil {
		return nil, fmt.Errorf("unmarshaling credential: %w", err)
	}

	return &cred, nil
}

// Delete removes a credential from the OS keychain.
func (k *OSKeychain) Delete(provider string) error {
	key := k.keyForProvider(provider)
	if err := keyring.Delete(keyringServiceName, key); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("deleting from keychain: %w", k.mapKeychainError(err))
	}
	return nil
}

// IsAvailable checks if the OS keychain is available.
func (k *OSKeychain) IsAvailable() bool {
	// Try a test operation to verify keychain availability
	testKey := "__picoclaw_test__"
	testValue := "test"

	// On Windows, macOS, and Linux with a secret service, this should work
	err := keyring.Set(keyringServiceName, testKey, testValue)
	if err != nil {
		return false
	}

	// Clean up test entry
	_ = keyring.Delete(keyringServiceName, testKey)
	return true
}

func (k *OSKeychain) keyForProvider(provider string) string {
	return fmt.Sprintf("provider_%s", provider)
}

func (k *OSKeychain) mapKeychainError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Platform-specific error mapping
	if strings.Contains(errStr, "access denied") ||
		strings.Contains(errStr, "user canceled") ||
		strings.Contains(errStr, "authorization failed") ||
		strings.Contains(errStr, "locked collection") {
		return ErrKeychainAccessDenied
	}

	return err
}

// MockKeychain is a mock implementation for testing.
type MockKeychain struct {
	data map[string]*AuthCredential
}

// NewMockKeychain creates a new mock keychain for testing.
func NewMockKeychain() *MockKeychain {
	return &MockKeychain{
		data: make(map[string]*AuthCredential),
	}
}

func (m *MockKeychain) Store(provider string, cred *AuthCredential) error {
	m.data[provider] = cred
	return nil
}

func (m *MockKeychain) Retrieve(provider string) (*AuthCredential, error) {
	cred, ok := m.data[provider]
	if !ok {
		return nil, nil
	}
	return cred, nil
}

func (m *MockKeychain) Delete(provider string) error {
	delete(m.data, provider)
	return nil
}

func (m *MockKeychain) IsAvailable() bool {
	return true
}

// FallbackKeychain is a keychain that falls back to file-based encryption.
type FallbackKeychain struct {
	primary   KeychainBackend
	encryptor *Encryptor
}

// NewFallbackKeychain creates a keychain that tries the primary backend first,
// then falls back to file-based encryption if unavailable.
func NewFallbackKeychain(primary KeychainBackend, encryptor *Encryptor) *FallbackKeychain {
	return &FallbackKeychain{
		primary:   primary,
		encryptor: encryptor,
	}
}

func (f *FallbackKeychain) Store(provider string, cred *AuthCredential) error {
	if f.primary.IsAvailable() {
		if err := f.primary.Store(provider, cred); err == nil {
			return nil
		}
		// Fall through to encrypted file storage
	}

	// Use encrypted file storage as fallback
	encData, err := f.encryptor.EncryptCredential(cred)
	if err != nil {
		return fmt.Errorf("encrypting credential: %w", err)
	}

	return storeEncryptedCredential(provider, encData)
}

func (f *FallbackKeychain) Retrieve(provider string) (*AuthCredential, error) {
	if f.primary.IsAvailable() {
		cred, err := f.primary.Retrieve(provider)
		if err == nil && cred != nil {
			return cred, nil
		}
		// Fall through to encrypted file storage
	}

	// Try encrypted file storage
	encData, err := loadEncryptedCredential(provider)
	if err != nil {
		return nil, err
	}
	if encData == nil {
		return nil, nil
	}

	return f.encryptor.DecryptCredential(encData)
}

func (f *FallbackKeychain) Delete(provider string) error {
	// Delete from both backends
	if f.primary.IsAvailable() {
		_ = f.primary.Delete(provider)
	}
	return deleteEncryptedCredential(provider)
}

func (f *FallbackKeychain) IsAvailable() bool {
	return true // Always available due to fallback
}
