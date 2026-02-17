package config

import (
	"path/filepath"
	"testing"
)

func TestResolveRuntimePaths_Default(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(EnvPicoClawConfig, "")
	t.Setenv(EnvPicoClawHome, "")

	paths := ResolveRuntimePaths()
	wantHome := filepath.Join(home, ".picoclaw")

	if paths.HomeDir != wantHome {
		t.Errorf("HomeDir = %q, want %q", paths.HomeDir, wantHome)
	}
	if paths.ConfigPath != filepath.Join(wantHome, "config.json") {
		t.Errorf("ConfigPath = %q, want %q", paths.ConfigPath, filepath.Join(wantHome, "config.json"))
	}
	if paths.AuthPath != filepath.Join(wantHome, "auth.json") {
		t.Errorf("AuthPath = %q, want %q", paths.AuthPath, filepath.Join(wantHome, "auth.json"))
	}
	if paths.GlobalSkillsDir != filepath.Join(wantHome, "skills") {
		t.Errorf("GlobalSkillsDir = %q, want %q", paths.GlobalSkillsDir, filepath.Join(wantHome, "skills"))
	}
}

func TestResolveRuntimePaths_UsesPicoClawHomeOverride(t *testing.T) {
	homeOverride := filepath.Join(t.TempDir(), "pico-home")
	t.Setenv(EnvPicoClawConfig, "")
	t.Setenv(EnvPicoClawHome, homeOverride)

	paths := ResolveRuntimePaths()

	if paths.HomeDir != homeOverride {
		t.Errorf("HomeDir = %q, want %q", paths.HomeDir, homeOverride)
	}
	if paths.ConfigPath != filepath.Join(homeOverride, "config.json") {
		t.Errorf("ConfigPath = %q, want %q", paths.ConfigPath, filepath.Join(homeOverride, "config.json"))
	}
	if paths.AuthPath != filepath.Join(homeOverride, "auth.json") {
		t.Errorf("AuthPath = %q, want %q", paths.AuthPath, filepath.Join(homeOverride, "auth.json"))
	}
	if paths.GlobalSkillsDir != filepath.Join(homeOverride, "skills") {
		t.Errorf("GlobalSkillsDir = %q, want %q", paths.GlobalSkillsDir, filepath.Join(homeOverride, "skills"))
	}
}

func TestResolveRuntimePaths_ConfigOverrideTakesPrecedence(t *testing.T) {
	homeOverride := filepath.Join(t.TempDir(), "pico-home")
	configDir := filepath.Join(t.TempDir(), "custom-config-dir")
	configPath := filepath.Join(configDir, "config.json")

	t.Setenv(EnvPicoClawHome, homeOverride)
	t.Setenv(EnvPicoClawConfig, configPath)

	paths := ResolveRuntimePaths()

	if paths.ConfigPath != configPath {
		t.Errorf("ConfigPath = %q, want %q", paths.ConfigPath, configPath)
	}
	if paths.HomeDir != configDir {
		t.Errorf("HomeDir = %q, want %q", paths.HomeDir, configDir)
	}
	if paths.AuthPath != filepath.Join(configDir, "auth.json") {
		t.Errorf("AuthPath = %q, want %q", paths.AuthPath, filepath.Join(configDir, "auth.json"))
	}
	if paths.GlobalSkillsDir != filepath.Join(configDir, "skills") {
		t.Errorf("GlobalSkillsDir = %q, want %q", paths.GlobalSkillsDir, filepath.Join(configDir, "skills"))
	}
}
