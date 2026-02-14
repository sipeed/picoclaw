package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CodexCliAuth represents the ~/.codex/auth.json file structure.
type CodexCliAuth struct {
	Tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		AccountID    string `json:"account_id"`
	} `json:"tokens"`
}

// ReadCodexCliCredentials reads OAuth tokens from the Codex CLI's auth.json file.
// Expiry is estimated as file modification time + 1 hour (same approach as moltbot).
func ReadCodexCliCredentials() (accessToken, accountID string, expiresAt time.Time, err error) {
	authPath, err := resolveCodexAuthPath()
	if err != nil {
		return "", "", time.Time{}, err
	}

	data, err := os.ReadFile(authPath)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("reading %s: %w", authPath, err)
	}

	var auth CodexCliAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", "", time.Time{}, fmt.Errorf("parsing %s: %w", authPath, err)
	}

	if auth.Tokens.AccessToken == "" {
		return "", "", time.Time{}, fmt.Errorf("no access_token in %s", authPath)
	}

	stat, err := os.Stat(authPath)
	if err != nil {
		expiresAt = time.Now().Add(time.Hour)
	} else {
		expiresAt = stat.ModTime().Add(time.Hour)
	}

	return auth.Tokens.AccessToken, auth.Tokens.AccountID, expiresAt, nil
}

// CreateCodexCliTokenSource creates a token source that reads from ~/.codex/auth.json.
// This allows the existing CodexProvider to reuse Codex CLI credentials.
func CreateCodexCliTokenSource() func() (string, string, error) {
	return func() (string, string, error) {
		token, accountID, expiresAt, err := ReadCodexCliCredentials()
		if err != nil {
			return "", "", fmt.Errorf("reading codex cli credentials: %w", err)
		}

		if time.Now().After(expiresAt) {
			return "", "", fmt.Errorf("codex cli credentials expired (auth.json last modified > 1h ago). Run: codex login")
		}

		return token, accountID, nil
	}
}

func resolveCodexAuthPath() (string, error) {
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home dir: %w", err)
		}
		codexHome = filepath.Join(home, ".codex")
	}
	return filepath.Join(codexHome, "auth.json"), nil
}
