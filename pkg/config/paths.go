package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvPicoClawConfig = "PICOCLAW_CONFIG"
	EnvPicoClawHome   = "PICOCLAW_HOME"
)

type RuntimePaths struct {
	HomeDir         string
	ConfigPath      string
	AuthPath        string
	GlobalSkillsDir string
}

func ResolveRuntimePaths() RuntimePaths {
	if configPath := expandHome(strings.TrimSpace(os.Getenv(EnvPicoClawConfig))); configPath != "" {
		return buildRuntimePaths(filepath.Dir(configPath), configPath)
	}

	homeDir := expandHome(strings.TrimSpace(os.Getenv(EnvPicoClawHome)))
	if homeDir == "" {
		homeDir = defaultPicoClawHome()
	}

	return buildRuntimePaths(homeDir, filepath.Join(homeDir, "config.json"))
}

func defaultPicoClawHome() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".picoclaw"
	}
	return filepath.Join(home, ".picoclaw")
}

func buildRuntimePaths(homeDir, configPath string) RuntimePaths {
	return RuntimePaths{
		HomeDir:         homeDir,
		ConfigPath:      configPath,
		AuthPath:        filepath.Join(homeDir, "auth.json"),
		GlobalSkillsDir: filepath.Join(homeDir, "skills"),
	}
}
