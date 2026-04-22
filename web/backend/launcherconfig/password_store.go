package launcherconfig

import (
	"context"
	"errors"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const passwordBcryptCost = 12

// PasswordStore keeps the dashboard bcrypt hash in launcher-config.json.
// It is used on platforms where the SQLite-backed dashboard auth store is not
// available.
type PasswordStore struct {
	path     string
	fallback Config
	mu       sync.Mutex
}

// NewPasswordStore returns a config-backed password store.
func NewPasswordStore(path string, fallback Config) *PasswordStore {
	return &PasswordStore{
		path:     path,
		fallback: fallback,
	}
}

// IsInitialized reports whether a dashboard password hash exists in config.
func (s *PasswordStore) IsInitialized(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	cfg, err := s.load()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(cfg.DashboardPasswordHash) != "", nil
}

// SetPassword hashes plain with bcrypt and writes it to launcher-config.json.
func (s *PasswordStore) SetPassword(ctx context.Context, plain string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len([]rune(plain)) == 0 {
		return errors.New("password must not be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), passwordBcryptCost)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := Load(s.path, s.fallback)
	if err != nil {
		return err
	}
	cfg.DashboardPasswordHash = string(hash)
	cfg.LegacyLauncherToken = ""
	return Save(s.path, cfg)
}

// VerifyPassword returns true iff plain matches the stored bcrypt hash.
func (s *PasswordStore) VerifyPassword(ctx context.Context, plain string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	cfg, err := s.load()
	if err != nil {
		return false, err
	}
	hash := strings.TrimSpace(cfg.DashboardPasswordHash)
	if hash == "" {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return err == nil, err
}

func (s *PasswordStore) load() (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Load(s.path, s.fallback)
}
