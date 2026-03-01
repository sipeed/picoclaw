package infra

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveHomeDir returns the effective home directory for PicoClaw.
// It checks the PICOCLAW_HOME environment variable first,
// falls back to ~/.picoclaw if not set or empty.
func ResolveHomeDir() string {
	if envHome := strings.TrimSpace(os.Getenv("PICOCLAW_HOME")); envHome != "" {
		return envHome
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		// Extreme fallback
		return filepath.Join(os.TempDir(), ".picoclaw")
	}
	return filepath.Join(home, ".picoclaw")
}
