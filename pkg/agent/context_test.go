package agent

import (
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestGetGlobalConfigDir_UsesPicoClawHomeOverride(t *testing.T) {
	homeOverride := filepath.Join(t.TempDir(), "pico-home")
	t.Setenv(config.EnvPicoClawConfig, "")
	t.Setenv(config.EnvPicoClawHome, homeOverride)

	if got := getGlobalConfigDir(); got != homeOverride {
		t.Errorf("getGlobalConfigDir() = %q, want %q", got, homeOverride)
	}
}

func TestGetGlobalConfigDir_ConfigOverrideTakesPrecedence(t *testing.T) {
	homeOverride := filepath.Join(t.TempDir(), "pico-home")
	configDir := filepath.Join(t.TempDir(), "custom-config-dir")
	configPath := filepath.Join(configDir, "config.json")

	t.Setenv(config.EnvPicoClawHome, homeOverride)
	t.Setenv(config.EnvPicoClawConfig, configPath)

	if got := getGlobalConfigDir(); got != configDir {
		t.Errorf("getGlobalConfigDir() = %q, want %q", got, configDir)
	}
}
