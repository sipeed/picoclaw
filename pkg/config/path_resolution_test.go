package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPath_DefaultJSONWhenMissing(t *testing.T) {
	home := t.TempDir()
	got := ResolveConfigPath(home)
	want := filepath.Join(home, ".picoclaw", "config.json")
	if got != want {
		t.Fatalf("ResolveConfigPath() = %q, want %q", got, want)
	}
}

func TestResolveConfigPath_PicksExistingNonJSON(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".picoclaw")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	yamlPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(yamlPath, []byte("agents: {}"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	got := ResolveConfigPath(home)
	if got != yamlPath {
		t.Fatalf("ResolveConfigPath() = %q, want %q", got, yamlPath)
	}
}

func TestResolveConfigPath_PrefersJSONWhenBothExist(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".picoclaw")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	yamlPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(yamlPath, []byte("agents: {}"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	jsonPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(jsonPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	got := ResolveConfigPath(home)
	if got != jsonPath {
		t.Fatalf("ResolveConfigPath() = %q, want %q", got, jsonPath)
	}
}
