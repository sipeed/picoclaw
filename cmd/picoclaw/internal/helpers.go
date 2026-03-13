package internal

import (
	"os"
	"path/filepath"

	"jane/pkg/config"
)

const Logo = "🦞"

// GetPicoclawHome returns the picoclaw home directory.
// Priority: $PICOCLAW_HOME > ~/.picoclaw
func GetPicoclawHome() string {
	if home := os.Getenv("PICOCLAW_HOME"); home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw")
}

func GetConfigPath() string {
	if configPath := os.Getenv("PICOCLAW_CONFIG"); configPath != "" {
		return configPath
	}
	return filepath.Join(GetPicoclawHome(), "config.json")
}

func LoadConfig() (*config.Config, error) {
	return config.LoadConfig(GetConfigPath())
}
