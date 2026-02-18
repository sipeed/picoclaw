package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultConfigUsesOpenRouterAutoModel(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agents.Defaults.Model != "openrouter/auto" {
		t.Fatalf("default model = %q, want %q", cfg.Agents.Defaults.Model, "openrouter/auto")
	}
}

func TestSaveConfigWritesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not reliable on windows")
	}

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create seed config: %v", err)
	}

	cfg := DefaultConfig()
	cfg.Providers.OpenRouter.APIKey = "secret"

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}

	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config perms = %o, want 600", got)
	}
}

func TestLoadConfigNormalizesLegacyGLMModelForOpenRouter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{
  "agents": { "defaults": { "model": "glm-4.7" } },
  "providers": { "openrouter": { "api_key": "sk-or-test" } }
}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config fixture: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Agents.Defaults.Model != "openrouter/auto" {
		t.Fatalf("model = %q, want %q", cfg.Agents.Defaults.Model, "openrouter/auto")
	}
}

func TestLoadConfigKeepsLegacyGLMModelWhenZhipuConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{
  "agents": { "defaults": { "model": "glm-4.7" } },
  "providers": {
    "openrouter": { "api_key": "sk-or-test" },
    "zhipu": { "api_key": "zhipu-key" }
  }
}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config fixture: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Agents.Defaults.Model != "glm-4.7" {
		t.Fatalf("model = %q, want %q", cfg.Agents.Defaults.Model, "glm-4.7")
	}
}

func TestLoadConfigKeepsLegacyGLMModelWhenNoProviderKeysConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{
  "agents": { "defaults": { "model": "glm-4.7" } }
}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config fixture: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Agents.Defaults.Model != "glm-4.7" {
		t.Fatalf("model = %q, want %q", cfg.Agents.Defaults.Model, "glm-4.7")
	}
}
