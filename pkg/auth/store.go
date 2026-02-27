package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

type AuthCredential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	AccountID    string    `json:"account_id,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Provider     string    `json:"provider"`
	AuthMethod   string    `json:"auth_method"`
	Email        string    `json:"email,omitempty"`
	ProjectID    string    `json:"project_id,omitempty"`
}

type AuthStore struct {
	Credentials map[string]*AuthCredential `json:"credentials"`
}

func (c *AuthCredential) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

func (c *AuthCredential) NeedsRefresh() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(c.ExpiresAt)
}

func authFilePath() string {
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			home = "."
		}
	}
	return filepath.Join(home, ".picoclaw", "auth.json")
}

func LoadStore() (*AuthStore, error) {
	path := authFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AuthStore{Credentials: make(map[string]*AuthCredential)}, nil
		}
		return nil, err
	}

	var store AuthStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Credentials == nil {
		store.Credentials = make(map[string]*AuthCredential)
	}
	return &store, nil
}

func SaveStore(store *AuthStore) error {
	path := authFilePath()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

// Global secure store instance with lazy initialization.
var (
	globalSecureStore *SecureStore
	secureStoreOnce   sync.Once
	secureStoreConfig SecureStoreConfig
	storeMu           sync.RWMutex
)

// InitSecureStore initializes the global secure store with the given configuration.
// This should be called once at application startup.
func InitSecureStore(config SecureStoreConfig) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	var initErr error
	secureStoreOnce.Do(func() {
		secureStoreConfig = config
		globalSecureStore, initErr = NewSecureStore(config)
	})
	return initErr
}

// ResetSecureStore resets the global secure store. For testing only.
func ResetSecureStore() {
	storeMu.Lock()
	defer storeMu.Unlock()
	globalSecureStore = nil
	secureStoreOnce = sync.Once{}
	secureStoreConfig = SecureStoreConfig{}
}

// getSecureStore returns the global secure store, initializing with defaults if needed.
func getSecureStore() *SecureStore {
	storeMu.RLock()
	if globalSecureStore != nil {
		storeMu.RUnlock()
		return globalSecureStore
	}
	storeMu.RUnlock()

	storeMu.Lock()
	defer storeMu.Unlock()

	if globalSecureStore == nil {
		// Initialize with default config (no encryption for backward compatibility)
		globalSecureStore, _ = NewSecureStore(SecureStoreConfig{
			Enabled:     false,
			UseKeychain: false,
		})
	}
	return globalSecureStore
}

// GetCredential retrieves a credential from secure storage.
// Falls back to plain file storage if secure storage is not initialized.
func GetCredential(provider string) (*AuthCredential, error) {
	return getSecureStore().GetCredential(provider)
}

// SetCredential stores a credential in secure storage.
// Falls back to plain file storage if secure storage is not initialized.
func SetCredential(provider string, cred *AuthCredential) error {
	return getSecureStore().SetCredential(provider, cred)
}

// DeleteCredential removes a credential from secure storage.
func DeleteCredential(provider string) error {
	return getSecureStore().DeleteCredential(provider)
}

// DeleteAllCredentials removes all credentials from secure storage.
func DeleteAllCredentials() error {
	return getSecureStore().DeleteAllCredentials()
}

// MigrateCredentials migrates existing plain-text credentials to secure storage.
func MigrateCredentials() error {
	return getSecureStore().MigrateFromPlainStorage()
}

// ListProviders returns all providers with stored credentials.
func ListProviders() ([]string, error) {
	return getSecureStore().ListProviders()
}
