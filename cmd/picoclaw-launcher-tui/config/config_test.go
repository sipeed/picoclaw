package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncSelectedModelToMainConfig_WritesModelNameAndModelList(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	initial := map[string]any{
		"agents": map[string]any{
			"defaults": map[string]any{},
		},
		"model_list": []any{
			map[string]any{
				"model_name": "existing",
				"model":      "openai/gpt-4o-mini",
			},
		},
	}
	data, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal initial config: %v", err)
	}
	if err = os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	scheme := Scheme{Name: "openai", BaseURL: "https://api.openai.com/v1"}
	user := User{Name: "u1", Key: "sk-test"}
	if err = SyncSelectedModelToMainConfig(configPath, scheme, user, "openai/gpt-5.4"); err != nil {
		t.Fatalf("SyncSelectedModelToMainConfig() error = %v", err)
	}

	updatedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read updated config: %v", err)
	}

	var updated map[string]any
	if err = json.Unmarshal(updatedData, &updated); err != nil {
		t.Fatalf("unmarshal updated config: %v", err)
	}

	agents := updated["agents"].(map[string]any)
	defaults := agents["defaults"].(map[string]any)
	if got, ok := defaults["model_name"].(string); !ok || got != "tui-prefer" {
		t.Fatalf("agents.defaults.model_name = %v, want %q", defaults["model_name"], "tui-prefer")
	}
	if _, exists := defaults["model"]; exists {
		t.Fatalf("unexpected legacy field agents.defaults.model present: %v", defaults["model"])
	}

	modelList := updated["model_list"].([]any)
	var tuiPrefer map[string]any
	for _, item := range modelList {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if name, _ := entry["model_name"].(string); name == "tui-prefer" {
			tuiPrefer = entry
			break
		}
	}
	if tuiPrefer == nil {
		t.Fatalf("tui-prefer model entry not found")
	}
	if got, _ := tuiPrefer["model"].(string); got != "openai/gpt-5.4" {
		t.Fatalf("tui-prefer model = %q, want %q", got, "openai/gpt-5.4")
	}
	if got, _ := tuiPrefer["api_key"].(string); got != "sk-test" {
		t.Fatalf("tui-prefer api_key = %q, want %q", got, "sk-test")
	}
	if got, _ := tuiPrefer["api_base"].(string); got != "https://api.openai.com/v1" {
		t.Fatalf("tui-prefer api_base = %q, want %q", got, "https://api.openai.com/v1")
	}
}

func TestSyncSelectedModelToMainConfig_ReplacesExistingTuiPrefer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	initial := map[string]any{
		"agents": map[string]any{
			"defaults": map[string]any{
				"model_name": "old",
			},
		},
		"model_list": []any{
			map[string]any{
				"model_name": "tui-prefer",
				"model":      "openai/old",
				"api_key":    "sk-old",
				"api_base":   "https://old.example.com/v1",
			},
		},
	}
	data, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal initial config: %v", err)
	}
	if err = os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	scheme := Scheme{Name: "new", BaseURL: "https://new.example.com/v1"}
	user := User{Name: "u2", Key: "sk-new"}
	if err = SyncSelectedModelToMainConfig(configPath, scheme, user, "openai/new"); err != nil {
		t.Fatalf("SyncSelectedModelToMainConfig() error = %v", err)
	}

	updatedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read updated config: %v", err)
	}

	var updated map[string]any
	if err = json.Unmarshal(updatedData, &updated); err != nil {
		t.Fatalf("unmarshal updated config: %v", err)
	}

	modelList := updated["model_list"].([]any)
	count := 0
	for _, item := range modelList {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["model_name"].(string)
		if name != "tui-prefer" {
			continue
		}
		count++
		if got, _ := entry["model"].(string); got != "openai/new" {
			t.Fatalf("tui-prefer model = %q, want %q", got, "openai/new")
		}
		if got, _ := entry["api_key"].(string); got != "sk-new" {
			t.Fatalf("tui-prefer api_key = %q, want %q", got, "sk-new")
		}
		if got, _ := entry["api_base"].(string); got != "https://new.example.com/v1" {
			t.Fatalf("tui-prefer api_base = %q, want %q", got, "https://new.example.com/v1")
		}
	}
	if count != 1 {
		t.Fatalf("tui-prefer entry count = %d, want 1", count)
	}
}

func TestSyncSelectedModelToMainConfig_ReturnsErrorOnInvalidPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	parentFile := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	configPath := filepath.Join(parentFile, "config.json")
	scheme := Scheme{Name: "s1", BaseURL: "https://api.example.com/v1"}
	user := User{Name: "u1", Key: "sk-test"}
	err := SyncSelectedModelToMainConfig(configPath, scheme, user, "openai/gpt-5.4")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
