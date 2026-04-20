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
					"created":              1700000000,
					"supported_parameters": []string{"tools"},
				},
				{
					"id":             "meta-llama/llama-3-8b",
					"name":           "Llama 3 8B",
					"context_length": 8000,
					"pricing": map[string]string{
						"prompt":     "0.0001",
						"completion": "0.0001",
					},
					"created":              1700000000,
					"supported_parameters": []string{"tools"},
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

	if result.Silent {
		t.Errorf("Expected non-silent result")
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
					"created":              1700000000,
					"supported_parameters": []string{"tools"},
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

func TestFreeRideTool_SetTimeout(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "freeride-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	initialCfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{
				ModelName: "google-gemini-pro-1.5",
				Model:     "google/gemini-pro-1.5",
				Protocol:  "openrouter",
			},
			{
				ModelName: "meta-llama-3-8b",
				Model:     "meta/llama-3-8b",
				Protocol:  "openrouter",
			},
			{
				ModelName: "gpt-4o",
				Model:     "openai/gpt-4o",
				Protocol:  "openai",
			},
		},
	}
	initialCfg.Agents.Defaults.ModelName = "gpt-4o"

	if err := config.SaveConfig(configPath, initialCfg); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	var reloadCalled bool
	reloadFunc := func() error {
		reloadCalled = true
		return nil
	}

	tool := NewFreeRideTool(configPath, reloadFunc)
	result := tool.Execute(context.Background(), map[string]any{
		"command": "settimeout",
		"timeout": 180,
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

	// Should have updated 2 openrouter models
	if cfg.ModelList[0].RequestTimeout != 180 {
		t.Errorf("Expected timeout 180 for google-gemini-pro-1.5, got %d", cfg.ModelList[0].RequestTimeout)
	}
	if cfg.ModelList[1].RequestTimeout != 180 {
		t.Errorf("Expected timeout 180 for meta-llama-3-8b, got %d", cfg.ModelList[1].RequestTimeout)
	}
	// openai model should NOT be updated
	if cfg.ModelList[2].RequestTimeout != 0 {
		t.Errorf("Expected timeout 0 for gpt-4o (non-openrouter), got %d", cfg.ModelList[2].RequestTimeout)
	}
}

func TestFreeRideTool_SetTimeout_NoOpenRouterModels(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "freeride-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	initialCfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{
				ModelName: "gpt-4o",
				Model:     "openai/gpt-4o",
				Protocol:  "openai",
			},
		},
	}

	if err := config.SaveConfig(configPath, initialCfg); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	tool := NewFreeRideTool(configPath, nil)
	result := tool.Execute(context.Background(), map[string]any{
		"command": "settimeout",
		"timeout": 180,
	})

	if !result.IsError {
		t.Fatalf("Expected error when no OpenRouter models, got success")
	}

	if !contains(result.ForLLM, "no OpenRouter models") {
		t.Errorf("Expected error message about no OpenRouter models, got %s", result.ForLLM)
	}
}

func TestFreeRideTool_SetTimeout_MinimumTooLow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "freeride-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")
	initialCfg := &config.Config{
		ModelList: []*config.ModelConfig{
			{
				ModelName: "google-gemini-pro-1.5",
				Model:     "google/gemini-pro-1.5",
				Protocol:  "openrouter",
			},
		},
	}

	if err := config.SaveConfig(configPath, initialCfg); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	tool := NewFreeRideTool(configPath, nil)
	result := tool.Execute(context.Background(), map[string]any{
		"command": "settimeout",
		"timeout": 20, // too low
	})

	if !result.IsError {
		t.Fatalf("Expected error when timeout < 30, got success")
	}

	if !contains(result.ForLLM, "at least 30") {
		t.Errorf("Expected error message about minimum 30 seconds, got %s", result.ForLLM)
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
