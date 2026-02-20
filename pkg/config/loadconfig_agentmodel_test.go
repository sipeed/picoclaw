package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_AgentModelConfigStringFormat(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "agents": {
    "list": [
      {
        "id": "test",
        "model": "gpt-4"
      }
    ]
  },
  "channels": {},
  "providers": {}
}`

	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	cfg, err := LoadConfig(jsonFile)
	if err != nil {
		t.Fatalf("Failed to load config with string model format: %v", err)
	}

	if len(cfg.Agents.List) == 0 {
		t.Fatal("Expected at least one agent")
	}

	agent := cfg.Agents.List[0]
	if agent.Model == nil {
		t.Fatal("Agent model is nil")
	}

	if agent.Model.Primary != "gpt-4" {
		t.Errorf("Expected model primary 'gpt-4', got '%s'", agent.Model.Primary)
	}

	if agent.Model.Fallbacks != nil && len(agent.Model.Fallbacks) > 0 {
		t.Errorf("Expected no fallbacks, got %v", agent.Model.Fallbacks)
	}
}

func TestLoadConfig_AgentModelConfigObjectFormat(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "agents": {
    "list": [
      {
        "id": "test",
        "model": {
          "primary": "claude-opus",
          "fallbacks": ["gpt-4", "haiku"]
        }
      }
    ]
  },
  "channels": {},
  "providers": {}
}`

	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	cfg, err := LoadConfig(jsonFile)
	if err != nil {
		t.Fatalf("Failed to load config with object model format: %v", err)
	}

	if len(cfg.Agents.List) == 0 {
		t.Fatal("Expected at least one agent")
	}

	agent := cfg.Agents.List[0]
	if agent.Model == nil {
		t.Fatal("Agent model is nil")
	}

	if agent.Model.Primary != "claude-opus" {
		t.Errorf("Expected model primary 'claude-opus', got '%s'", agent.Model.Primary)
	}

	if len(agent.Model.Fallbacks) != 2 {
		t.Errorf("Expected 2 fallbacks, got %d", len(agent.Model.Fallbacks))
	}
}
