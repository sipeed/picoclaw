package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_FlexibleStringSlice_MixedArray(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "x",
      "allow_from": ["u1", 123, true]
    }
  }
}`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	got := cfg.Channels.Telegram.AllowFrom
	want := []string{"u1", "123", "true"}
	if len(got) != len(want) {
		t.Fatalf("allow_from len=%d, want=%d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("allow_from[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadConfig_FlexibleStringSlice_SingleString(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "x",
      "allow_from": "solo-user"
    }
  }
}`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if len(cfg.Channels.Telegram.AllowFrom) != 1 || cfg.Channels.Telegram.AllowFrom[0] != "solo-user" {
		t.Fatalf("allow_from=%v, want [solo-user]", cfg.Channels.Telegram.AllowFrom)
	}
}

func TestLoadConfig_InvalidConfigSyntax(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	invalid := "agents:\n  defaults:\n    model: [unclosed"

	if err := os.WriteFile(configPath, []byte(invalid), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() expected error for invalid syntax")
	}
}

func TestLoadConfig_AutoMigrateProvidersToModelList(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "model_list": [],
  "agents": {
    "defaults": {
      "provider": "openai",
      "model": "gpt-5.2"
    }
  },
  "providers": {
    "openai": {
      "api_key": "sk-test"
    }
  }
}`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if len(cfg.ModelList) == 0 {
		t.Fatal("model_list should be auto-migrated from providers")
	}
	if cfg.ModelList[0].APIKey != "sk-test" {
		t.Fatalf("migrated api_key=%q, want sk-test", cfg.ModelList[0].APIKey)
	}
}

func TestLoadConfig_ValidateModelListError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
  "model_list": [
    {
      "model_name": "broken",
      "model": ""
    }
  ]
}`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() expected validation error")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Fatalf("error=%v, want contains 'model is required'", err)
	}
}
