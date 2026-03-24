package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	Enabled      bool          `json:"enabled"`
	Username     string        `json:"username"`
	PasswordHash string        `json:"password_hash"`
	SessionTTL   time.Duration `json:"session_ttl"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:      false,
		Username:     "",
		PasswordHash: "",
		SessionTTL:   24 * time.Hour,
	}
}

func (c *Config) IsConfigured() bool {
	return c.Enabled && c.Username != "" && c.PasswordHash != ""
}

func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *Session) Refresh(ttl time.Duration) {
	s.ExpiresAt = time.Now().Add(ttl)
}
