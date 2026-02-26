package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SecureStoreConfig configures the secure credential storage.
type SecureStoreConfig struct {
	Enabled     bool
	UseKeychain bool
	Algorithm   string
}

// SecureStore provides secure credential storage with keychain and encryption support.
type SecureStore struct {
	config    SecureStoreConfig
	keychain  KeychainBackend
	encryptor *Encryptor
	mu        sync.RWMutex
}

// NewSecureStore creates a new secure credential store.
func NewSecureStore(config SecureStoreConfig) (*SecureStore, error) {
	store := &SecureStore{
		config: config,
	}

	if config.Enabled {
		// Create encryptor for fallback encryption
		if config.Algorithm == "" {
			config.Algorithm = string(AlgorithmChaCha20Poly1305)
		}

		encryptor, err := NewEncryptor(config.Algorithm)
		if err != nil {
			return nil, fmt.Errorf("creating encryptor: %w", err)
		}
		store.encryptor = encryptor

		// Set up keychain
		if config.UseKeychain {
			osKeychain := NewOSKeychain()
			store.keychain = NewFallbackKeychain(osKeychain, encryptor)
		} else {
			// Use encrypted file storage only
			store.keychain = &fileKeychain{encryptor: encryptor}
		}
	} else {
		// No encryption - use plain file storage
		store.keychain = &plainFileKeychain{}
	}

	return store, nil
}

// GetCredential retrieves a credential from secure storage.
func (s *SecureStore) GetCredential(provider string) (*AuthCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.keychain.Retrieve(provider)
}

// SetCredential stores a credential in secure storage.
func (s *SecureStore) SetCredential(provider string, cred *AuthCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.keychain.Store(provider, cred)
}

// DeleteCredential removes a credential from secure storage.
func (s *SecureStore) DeleteCredential(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.keychain.Delete(provider)
}

// DeleteAllCredentials removes all credentials from secure storage.
func (s *SecureStore) DeleteAllCredentials() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all providers from the store
	store, err := loadPlainStore()
	if err != nil {
		return err
	}

	for provider := range store.Credentials {
		if err := s.keychain.Delete(provider); err != nil {
			return fmt.Errorf("deleting credential for %s: %w", provider, err)
		}
	}

	// Also remove the encryption key if encryption was enabled
	if s.config.Enabled && !s.config.UseKeychain {
		_ = DeleteEncryptionKey()
	}

	// Remove the auth file
	path := authFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// MigrateFromPlainStorage migrates existing plain-text credentials to secure storage.
func (s *SecureStore) MigrateFromPlainStorage() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	store, err := loadPlainStore()
	if err != nil {
		return err
	}

	if len(store.Credentials) == 0 {
		return nil
	}

	// Migrate each credential
	for provider, cred := range store.Credentials {
		if err := s.keychain.Store(provider, cred); err != nil {
			return fmt.Errorf("migrating credential for %s: %w", provider, err)
		}
	}

	// Remove plain-text file after successful migration
	path := authFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plain-text file: %w", err)
	}

	return nil
}

// ListProviders returns all providers with stored credentials.
func (s *SecureStore) ListProviders() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try to load from plain store to get provider list
	store, err := loadPlainStore()
	if err != nil {
		return nil, err
	}

	providers := make([]string, 0, len(store.Credentials))
	for p := range store.Credentials {
		providers = append(providers, p)
	}

	return providers, nil
}

// fileKeychain implements KeychainBackend using encrypted file storage.
type fileKeychain struct {
	encryptor *Encryptor
}

func (f *fileKeychain) Store(provider string, cred *AuthCredential) error {
	encData, err := f.encryptor.EncryptCredential(cred)
	if err != nil {
		return err
	}
	return storeEncryptedCredential(provider, encData)
}

func (f *fileKeychain) Retrieve(provider string) (*AuthCredential, error) {
	encData, err := loadEncryptedCredential(provider)
	if err != nil {
		return nil, err
	}
	if encData == nil {
		return nil, nil
	}
	return f.encryptor.DecryptCredential(encData)
}

func (f *fileKeychain) Delete(provider string) error {
	return deleteEncryptedCredential(provider)
}

func (f *fileKeychain) IsAvailable() bool {
	return true
}

// plainFileKeychain implements KeychainBackend using plain file storage (no encryption).
type plainFileKeychain struct{}

func (p *plainFileKeychain) Store(provider string, cred *AuthCredential) error {
	// Ensure directory exists
	path := authFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	store, err := loadPlainStore()
	if err != nil {
		return err
	}
	store.Credentials[provider] = cred
	return savePlainStore(store)
}

func (p *plainFileKeychain) Retrieve(provider string) (*AuthCredential, error) {
	store, err := loadPlainStore()
	if err != nil {
		return nil, err
	}
	return store.Credentials[provider], nil
}

func (p *plainFileKeychain) Delete(provider string) error {
	store, err := loadPlainStore()
	if err != nil {
		return err
	}
	delete(store.Credentials, provider)
	return savePlainStore(store)
}

func (p *plainFileKeychain) IsAvailable() bool {
	return true
}

// Encrypted store file operations

type encryptedStore struct {
	Credentials map[string]*EncryptedData `json:"credentials"`
}

func encryptedStorePath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".picoclaw", "auth.enc.json")
}

func loadEncryptedStore() (*encryptedStore, error) {
	path := encryptedStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &encryptedStore{Credentials: make(map[string]*EncryptedData)}, nil
		}
		return nil, err
	}

	var store encryptedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Credentials == nil {
		store.Credentials = make(map[string]*EncryptedData)
	}
	return &store, nil
}

func saveEncryptedStore(store *encryptedStore) error {
	path := encryptedStorePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func storeEncryptedCredential(provider string, encData *EncryptedData) error {
	store, err := loadEncryptedStore()
	if err != nil {
		return err
	}
	store.Credentials[provider] = encData
	return saveEncryptedStore(store)
}

func loadEncryptedCredential(provider string) (*EncryptedData, error) {
	store, err := loadEncryptedStore()
	if err != nil {
		return nil, err
	}
	return store.Credentials[provider], nil
}

func deleteEncryptedCredential(provider string) error {
	store, err := loadEncryptedStore()
	if err != nil {
		return err
	}
	delete(store.Credentials, provider)

	// If no more credentials, delete the file
	if len(store.Credentials) == 0 {
		path := encryptedStorePath()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	return saveEncryptedStore(store)
}

// Plain store file operations (for backward compatibility and migration)

func loadPlainStore() (*AuthStore, error) {
	return LoadStore()
}

func savePlainStore(store *AuthStore) error {
	return SaveStore(store)
}
