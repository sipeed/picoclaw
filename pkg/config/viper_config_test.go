package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
agents:
  defaults:
    workspace: ~/.picoclaw/workspace
    model: gpt-4
    max_tokens: 8192
    temperature: 0.7
    max_tool_iterations: 20
channels: {}
providers: {}
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	cfg, err := LoadConfig(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	if cfg.Agents.Defaults.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", cfg.Agents.Defaults.Model)
	}

	if cfg.Agents.Defaults.MaxTokens != 8192 {
		t.Errorf("Expected max_tokens 8192, got %d", cfg.Agents.Defaults.MaxTokens)
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "claude-3",
      "max_tokens": 4096
    }
  },
  "channels": {},
  "providers": {}
}`

	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	cfg, err := LoadConfig(jsonFile)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	if cfg.Agents.Defaults.Model != "claude-3" {
		t.Errorf("Expected model 'claude-3', got '%s'", cfg.Agents.Defaults.Model)
	}

	if cfg.Agents.Defaults.MaxTokens != 4096 {
		t.Errorf("Expected max_tokens 4096, got %d", cfg.Agents.Defaults.MaxTokens)
	}
}

func TestLoadConfig_NonExistent(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.yaml")
	if err != nil {
		t.Errorf("Should not return error for non-existent file, got: %v", err)
	}

	if cfg == nil {
		t.Error("Should return default config for non-existent file")
	}
}

func TestLoadConfig_YAMLProviderAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
agents:
  defaults:
    model: glm-4.7
providers:
  zhipu:
    api_key: test-zhipu-key
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	cfg, err := LoadConfig(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	if cfg.Providers.Zhipu.APIKey != "test-zhipu-key" {
		t.Fatalf("Expected zhipu api_key to be loaded, got %q", cfg.Providers.Zhipu.APIKey)
	}
}
