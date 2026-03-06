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

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func ConfigDir() (string, error) {
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
