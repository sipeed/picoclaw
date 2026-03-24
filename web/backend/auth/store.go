package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const AuthConfigFileName = "auth-config.json"

type AuthConfigStore struct {
	mu     sync.RWMutex
	path   string
	config Config
}

func NewAuthConfigStore(path string) *AuthConfigStore {
	return &AuthConfigStore{
		path:   path,
		config: DefaultConfig(),
	}
}

func (s *AuthConfigStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.config = DefaultConfig()
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &s.config); err != nil {
		return err
	}

	return nil
}

func (s *AuthConfigStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveUnlocked()
}

func (s *AuthConfigStore) saveUnlocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(s.path, data, 0o600)
}

func (s *AuthConfigStore) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *AuthConfigStore) Set(config Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
	return s.saveUnlocked()
}

func (s *AuthConfigStore) SetCredentials(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	s.config.Enabled = true
	s.config.Username = username
	s.config.PasswordHash = hash

	return s.saveUnlocked()
}

func (s *AuthConfigStore) Enable() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.Enabled = true
	return s.saveUnlocked()
}

func (s *AuthConfigStore) Disable() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.Enabled = false
	return s.saveUnlocked()
}

func (s *AuthConfigStore) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.IsConfigured()
}

func PathForAuthConfig(appConfigPath string) string {
	dir := filepath.Dir(appConfigPath)
	if dir == "" || dir == "." {
		dir = "."
	}
	return filepath.Join(dir, AuthConfigFileName)
}
