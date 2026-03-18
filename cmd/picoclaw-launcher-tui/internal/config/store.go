package configstore

import (
	"errors"
	"os"
	"path/filepath"

	picoclawconfig "github.com/sipeed/picoclaw/pkg/config"
)

const (
	configDirName  = ".picoclaw"
	configFileName = "config.json"
)

// ConfigPath returns the path to the config file.
// Priority: $PICOCLAW_CONFIG > $PICOCLAW_HOME/config.json > ~/.picoclaw/config.json
func ConfigPath() (string, error) {
	if configPath := os.Getenv("PICOCLAW_CONFIG"); configPath != "" {
		return configPath, nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// ConfigDir returns the directory for picoclaw configuration.
// Priority: $PICOCLAW_HOME > ~/.picoclaw
func ConfigDir() (string, error) {
	if home := os.Getenv("PICOCLAW_HOME"); home != "" {
		return home, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName), nil
}

func Load() (*picoclawconfig.Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return picoclawconfig.LoadConfig(path)
}

func Save(cfg *picoclawconfig.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return picoclawconfig.SaveConfig(path, cfg)
}
