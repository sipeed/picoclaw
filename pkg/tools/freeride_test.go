package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestFreeRideTool_List(t *testing.T) {
	// Mock OpenRouter API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":             "google/gemini-pro-1.5",
					"name":           "Gemini Pro 1.5",
					"context_length": 128000,
					"pricing": map[string]string{
						"prompt":     "0",
						"completion": "0",
					},
					"created": 1700000000,
				},
				{
					"id":             "meta-llama/llama-3-8b",
					"name":           "Llama 3 8B",
					"context_length": 8000,
					"pricing": map[string]string{
						"prompt":     "0.0001",
						"completion": "0.0001",
					},
					"created": 1700000000,
				},
			},
		})
	}))
	defer server.Close()

	// Override default transport to use mock server
	oldTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = &mockTransport{server.URL}
	defer func() { http.DefaultClient.Transport = oldTransport }()

	tool := NewFreeRideTool("config.json", nil)
	result := tool.Execute(context.Background(), map[string]any{
		"command": "list",
	})

	if result.IsError {
		t.Fatalf("Expected no error, got %s", result.ForLLM)
	}

	if !result.Silent {
		t.Errorf("Expected silent result")
	}

	output := result.ForLLM
	if !contains(output, "Gemini Pro 1.5") {
		t.Errorf("Expected Gemini Pro 1.5 in output, got %s", output)
	}
	if contains(output, "Llama 3 8B") {
		t.Errorf("Did not expect paid model Llama 3 8B in output, got %s", output)
	}
}

func TestFreeRideTool_Auto(t *testing.T) {
	os.Setenv("OPENROUTER_API_KEY", "sk-test-key")
	defer os.Unsetenv("OPENROUTER_API_KEY")

	tempDir, err := os.MkdirTemp("", "freeride-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	initialCfg := &config.Config{
		ModelList: []*config.ModelConfig{},
	}
	initialCfg.Agents.Defaults.ModelName = "existing-model"

	if err := config.SaveConfig(configPath, initialCfg); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// Mock OpenRouter API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":             "google/gemini-pro-1.5",
					"name":           "Gemini Pro 1.5",
					"context_length": 128000,
					"pricing": map[string]string{
						"prompt":     "0",
						"completion": "0",
					},
					"created": 1700000000,
				},
			},
		})
	}))
	defer server.Close()

	oldTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = &mockTransport{server.URL}
	defer func() { http.DefaultClient.Transport = oldTransport }()

	var reloadCalled bool
	reloadFunc := func() error {
		reloadCalled = true
		return nil
	}

	tool := NewFreeRideTool(configPath, reloadFunc)
	result := tool.Execute(context.Background(), map[string]any{
		"command": "auto",
	})

	if result.IsError {
		t.Fatalf("Expected no error, got %s", result.ForLLM)
	}

	if !reloadCalled {
		t.Errorf("Expected reloadFunc to be called")
	}

	// Verify config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load updated config: %v", err)
	}

	if len(cfg.ModelList) != 1 {
		t.Errorf("Expected 1 model in ModelList, got %d", len(cfg.ModelList))
	}

	if cfg.ModelList[0].ModelName != "google-gemini-pro-1.5" {
		t.Errorf("Expected model name google-gemini-pro-1.5, got %s", cfg.ModelList[0].ModelName)
	}

	if len(cfg.Agents.Defaults.ModelFallbacks) != 1 {
		t.Errorf("Expected 1 fallback, got %d", len(cfg.Agents.Defaults.ModelFallbacks))
	}
}

type mockTransport struct {
	url string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq, _ := http.NewRequest(req.Method, m.url, req.Body)
	return http.DefaultTransport.RoundTrip(newReq)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
