package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgauth "github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func captureAuthStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	fn()

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return buf.String()
}

func setAuthStatusTestHome(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv(config.EnvHome, filepath.Join(tmpDir, ".picoclaw"))
	return tmpDir
}

func TestNewStatusSubcommand(t *testing.T) {
	cmd := newStatusCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "Show current auth status", cmd.Short)

	assert.False(t, cmd.HasFlags())
}

func TestAuthStatusCmdShowsCanonicalGoogleAntigravityAfterLegacyRefresh(t *testing.T) {
	tmpDir := setAuthStatusTestHome(t)

	legacyExpiry := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	legacyStore := map[string]any{
		"credentials": map[string]any{
			"antigravity": map[string]any{
				"access_token": "legacy-token",
				"expires_at":   legacyExpiry.Format(time.RFC3339),
				"provider":     "antigravity",
				"auth_method":  "oauth",
				"project_id":   "legacy-project",
			},
		},
	}
	data, err := json.Marshal(legacyStore)
	require.NoError(t, err)

	authPath := filepath.Join(tmpDir, ".picoclaw", "auth.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(authPath), 0o755))
	require.NoError(t, os.WriteFile(authPath, data, 0o600))

	refreshedExpiry := time.Date(2026, 4, 16, 12, 30, 0, 0, time.UTC)
	err = pkgauth.SetCredential("google-antigravity", &pkgauth.AuthCredential{
		AccessToken: "fresh-token",
		ExpiresAt:   refreshedExpiry,
		Provider:    "google-antigravity",
		AuthMethod:  "oauth",
		ProjectID:   "fresh-project",
	})
	require.NoError(t, err)

	output := captureAuthStdout(t, func() {
		require.NoError(t, authStatusCmd())
	})

	assert.Contains(t, output, "\nAuthenticated Providers:")
	assert.Contains(t, output, "\n  google-antigravity:\n")
	assert.NotContains(t, output, "\n  antigravity:\n")
	assert.Contains(t, output, "    Project: fresh-project")
	assert.Contains(t, output, "    Expires: 2026-04-16 12:30")
	assert.Equal(t, 1, strings.Count(output, ":\n    Method: oauth"))
}
