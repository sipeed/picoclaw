package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

const Logo = "🦞"

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

func LoadConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig(GetConfigPath())
	if err != nil {
		return nil, err
	}

	// Initialize secure store with config settings
	if err := initSecureStore(cfg); err != nil {
		return nil, fmt.Errorf("initializing secure store: %w", err)
	}

	return cfg, nil
}

// initSecureStore initializes the secure credential store based on config.
func initSecureStore(cfg *config.Config) error {
	return auth.InitSecureStore(auth.SecureStoreConfig{
		Enabled:     cfg.Security.CredentialEncryption.Enabled,
		UseKeychain: cfg.Security.CredentialEncryption.UseKeychain,
		Algorithm:   cfg.Security.CredentialEncryption.Algorithm,
	})
}

// FormatVersion returns the version string with optional git commit
func FormatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// FormatBuildInfo returns build time and go version info
func FormatBuildInfo() (string, string) {
	build := buildTime
	goVer := goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return build, goVer
}

// GetVersion returns the version string
func GetVersion() string {
	return version
}
