package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
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
	home, _ := os.UserHomeDir()
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

	// Decrypt tokens if master key is set
	if mk := GetMasterKey(); mk != "" {
		key := DeriveKey(mk)
		for _, cred := range store.Credentials {
			if IsEncrypted(cred.AccessToken) {
				if dec, err := Decrypt(cred.AccessToken, key); err == nil {
					cred.AccessToken = dec
				}
			}
			if IsEncrypted(cred.RefreshToken) {
				if dec, err := Decrypt(cred.RefreshToken, key); err == nil {
					cred.RefreshToken = dec
				}
			}
		}
	}

	return &store, nil
}

func SaveStore(store *AuthStore) error {
	path := authFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	// Clone and encrypt tokens if master key is set
	toSave := &AuthStore{Credentials: make(map[string]*AuthCredential, len(store.Credentials))}
	for k, cred := range store.Credentials {
		clone := *cred
		toSave.Credentials[k] = &clone
	}

	if mk := GetMasterKey(); mk != "" {
		key := DeriveKey(mk)
		for _, cred := range toSave.Credentials {
			if cred.AccessToken != "" && !IsEncrypted(cred.AccessToken) {
				if enc, err := Encrypt(cred.AccessToken, key); err == nil {
					cred.AccessToken = enc
				}
			}
			if cred.RefreshToken != "" && !IsEncrypted(cred.RefreshToken) {
				if enc, err := Encrypt(cred.RefreshToken, key); err == nil {
					cred.RefreshToken = enc
				}
			}
		}
	}

	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func GetCredential(provider string) (*AuthCredential, error) {
	store, err := LoadStore()
	if err != nil {
		return nil, err
	}
	cred, ok := store.Credentials[provider]
	if !ok {
		return nil, nil
	}
	return cred, nil
}

func SetCredential(provider string, cred *AuthCredential) error {
	store, err := LoadStore()
	if err != nil {
		return err
	}
	store.Credentials[provider] = cred
	return SaveStore(store)
}

func DeleteCredential(provider string) error {
	store, err := LoadStore()
	if err != nil {
		return err
	}
	delete(store.Credentials, provider)
	return SaveStore(store)
}

func DeleteAllCredentials() error {
	path := authFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
