package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigSearchPaths(t *testing.T) {
	home := "/tmp/home-test"
	paths := defaultConfigSearchPaths(home)
	if len(paths) < 4 {
		t.Fatalf("expected multiple config search paths, got %d", len(paths))
	}

	hasHomeConfig := false
	for _, p := range paths {
		if p == filepath.Join(home, ".picoclaw", "config.json") {
			hasHomeConfig = true
			break
		}
	}
	if !hasHomeConfig {
		t.Fatal("expected home config path in defaultConfigSearchPaths")
	}
}

func TestHasConfigInPaths(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	if hasConfigInPaths([]string{missing}) {
		t.Fatal("expected false when config does not exist")
	}

	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	if !hasConfigInPaths([]string{missing, cfgPath}) {
		t.Fatal("expected true when one path exists")
	}
}

func TestParseCSV(t *testing.T) {
	items := parseCSV(" 123, ,abc,  xyz ")
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0] != "123" || items[1] != "abc" || items[2] != "xyz" {
		t.Fatalf("unexpected parse result: %#v", items)
	}
}

func TestVisibleLen(t *testing.T) {
	if got := visibleLen("hello"); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
	// ANSI escape codes should not count
	if got := visibleLen("\033[31mred\033[0m"); got != 3 {
		t.Fatalf("expected 3 for colored string, got %d", got)
	}
	if got := visibleLen(""); got != 0 {
		t.Fatalf("expected 0 for empty string, got %d", got)
	}
}

func TestDetectEnvironment(t *testing.T) {
	env := detectEnvironment()
	// Just verify it doesn't panic and returns a struct
	_ = env.OllamaFound
	_ = env.NetworkOnline
	_ = env.NPUDetected
	_ = env.NPUDevice
}

func TestParseOpenAIModels(t *testing.T) {
	data := []byte(`{"data":[{"id":"gpt-4o"},{"id":"gpt-4o-mini"},{"id":"gpt-3.5-turbo"}]}`)
	models, err := parseOpenAIModels(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}
	// Should be sorted
	if models[0] != "gpt-3.5-turbo" {
		t.Fatalf("expected sorted first model gpt-3.5-turbo, got %s", models[0])
	}
}

func TestParseOpenAIModelsEmpty(t *testing.T) {
	data := []byte(`{"data":[]}`)
	models, err := parseOpenAIModels(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("expected 0 models, got %d", len(models))
	}
}

func TestParseGeminiModels(t *testing.T) {
	data := []byte(`{"models":[{"name":"models/gemini-2.5-flash"},{"name":"models/gemini-pro"}]}`)
	models, err := parseGeminiModels(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	// Should strip "models/" prefix and be sorted
	if models[0] != "gemini-2.5-flash" {
		t.Fatalf("expected gemini-2.5-flash, got %s", models[0])
	}
	if models[1] != "gemini-pro" {
		t.Fatalf("expected gemini-pro, got %s", models[1])
	}
}

func TestParseOpenAIModelsInvalidJSON(t *testing.T) {
	_, err := parseOpenAIModels([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseGeminiModelsInvalidJSON(t *testing.T) {
	_, err := parseGeminiModels([]byte(`{broken`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
