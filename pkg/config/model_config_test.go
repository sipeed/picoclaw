// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestGetModelConfig_Found(t *testing.T) {
	cfg := &Config{
		Version: CurrentVersion,
		ModelList: []*ModelConfig{
			{ModelName: "test-model", Model: "openai/gpt-4o", APIKeys: SimpleSecureStrings("key1")},
			{ModelName: "other-model", Model: "anthropic/claude", APIKeys: SimpleSecureStrings("key2")},
		},
	}

	result, err := cfg.GetModelConfig("test-model")
	if err != nil {
		t.Fatalf("GetModelConfig() error = %v", err)
	}
	if result.Model != "openai/gpt-4o" {
		t.Errorf("Model = %q, want %q", result.Model, "openai/gpt-4o")
	}
}

func TestGetModelConfig_ByModelID(t *testing.T) {
	cfg := &Config{
		Version: CurrentVersion,
		ModelList: []*ModelConfig{
			{ModelName: "my-alias", Model: "openai/gpt-4o", APIKeys: SimpleSecureStrings("key1"), ThinkingLevel: "high"},
		},
	}

	// Lookup by full model ID should work when model_name doesn't match.
	result, err := cfg.GetModelConfig("openai/gpt-4o")
	if err != nil {
		t.Fatalf("GetModelConfig(full model) error = %v", err)
	}
	if result.ThinkingLevel != "high" {
		t.Errorf("ThinkingLevel = %q, want %q", result.ThinkingLevel, "high")
	}

	// Lookup by bare model ID (without protocol prefix).
	result, err = cfg.GetModelConfig("gpt-4o")
	if err != nil {
		t.Fatalf("GetModelConfig(bare model ID) error = %v", err)
	}
	if result.ModelName != "my-alias" {
		t.Errorf("ModelName = %q, want %q", result.ModelName, "my-alias")
	}
}

func TestGetModelConfig_NotFound(t *testing.T) {
	cfg := &Config{
		ModelList: []*ModelConfig{
			{ModelName: "test-model", Model: "openai/gpt-4o", APIKeys: SimpleSecureStrings("key1")},
		},
	}

	_, err := cfg.GetModelConfig("nonexistent")
	if err == nil {
		t.Fatal("GetModelConfig() expected error for nonexistent model")
	}
}

func TestGetModelConfig_EmptyList(t *testing.T) {
	cfg := &Config{
		ModelList: []*ModelConfig{},
	}

	_, err := cfg.GetModelConfig("any-model")
	if err == nil {
		t.Fatal("GetModelConfig() expected error for empty model list")
	}
}

func TestGetModelConfig_RoundRobin(t *testing.T) {
	cfg := &Config{
		ModelList: []*ModelConfig{
			{ModelName: "lb-model", Model: "openai/gpt-4o-1", APIKeys: SimpleSecureStrings("key1")},
			{ModelName: "lb-model", Model: "openai/gpt-4o-2", APIKeys: SimpleSecureStrings("key2")},
			{ModelName: "lb-model", Model: "openai/gpt-4o-3", APIKeys: SimpleSecureStrings("key3")},
		},
	}

	// Test round-robin distribution
	results := make(map[string]int)
	for range 30 {
		result, err := cfg.GetModelConfig("lb-model")
		if err != nil {
			t.Fatalf("GetModelConfig() error = %v", err)
		}
		results[result.Model]++
	}

	// Each model should appear roughly 10 times (30 calls / 3 models)
	for model, count := range results {
		if count < 5 || count > 15 {
			t.Errorf("Model %s appeared %d times, expected ~10", model, count)
		}
	}
}

func TestGetModelConfig_RoundRobinStartsFromFirstMatch(t *testing.T) {
	rrCounter.Store(0)

	cfg := &Config{
		ModelList: []*ModelConfig{
			{ModelName: "lb-model", Model: "openai/gpt-4o-1", APIKeys: SimpleSecureStrings("key1")},
			{ModelName: "lb-model", Model: "openai/gpt-4o-2", APIKeys: SimpleSecureStrings("key2")},
			{ModelName: "lb-model", Model: "openai/gpt-4o-3", APIKeys: SimpleSecureStrings("key3")},
		},
	}

	wantOrder := []string{
		"openai/gpt-4o-1",
		"openai/gpt-4o-2",
		"openai/gpt-4o-3",
		"openai/gpt-4o-1",
		"openai/gpt-4o-2",
	}

	for i, want := range wantOrder {
		result, err := cfg.GetModelConfig("lb-model")
		if err != nil {
			t.Fatalf("GetModelConfig() call %d error = %v", i, err)
		}
		if result.Model != want {
			t.Fatalf("GetModelConfig() call %d model = %q, want %q", i, result.Model, want)
		}
	}
}

func TestGetModelConfig_Concurrent(t *testing.T) {
	cfg := &Config{
		ModelList: []*ModelConfig{
			{ModelName: "concurrent-model", Model: "openai/gpt-4o-1", APIKeys: SimpleSecureStrings("key1")},
			{ModelName: "concurrent-model", Model: "openai/gpt-4o-2", APIKeys: SimpleSecureStrings("key2")},
		},
	}

	const goroutines = 100
	const iterations = 10

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for range goroutines {
		wg.Go(func() {
			for range iterations {
				_, err := cfg.GetModelConfig("concurrent-model")
				if err != nil {
					errors <- err
				}
			}
		})
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent GetModelConfig() error: %v", err)
	}
}

func TestModelConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ModelConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ModelConfig{
				ModelName: "test",
				Model:     "openai/gpt-4o",
			},
			wantErr: false,
		},
		{
			name: "missing model_name",
			config: ModelConfig{
				Model: "openai/gpt-4o",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: ModelConfig{
				ModelName: "test",
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  ModelConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateModelList(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string // partial error message to check
	}{
		{
			name: "valid list",
			config: &Config{
				ModelList: []*ModelConfig{
					{ModelName: "test1", Model: "openai/gpt-4o"},
					{ModelName: "test2", Model: "anthropic/claude"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid entry",
			config: &Config{
				ModelList: []*ModelConfig{
					{ModelName: "test1", Model: "openai/gpt-4o"},
					{ModelName: "", Model: "anthropic/claude"}, // missing model_name
				},
			},
			wantErr: true,
			errMsg:  "model_name is required",
		},
		{
			name: "empty list",
			config: &Config{
				ModelList: []*ModelConfig{},
			},
			wantErr: false,
		},
		{
			// Load balancing: multiple entries with same model_name are allowed
			name: "duplicate model_name for load balancing",
			config: &Config{
				ModelList: []*ModelConfig{},
			},
			wantErr: false, // Changed: duplicates are allowed for load balancing
		},
		{
			// Load balancing: non-adjacent entries with same model_name are also allowed
			name: "duplicate model_name non-adjacent for load balancing",
			config: &Config{
				ModelList: []*ModelConfig{
					{ModelName: "model-a", Model: "openai/gpt-4o"},
					{ModelName: "model-b", Model: "anthropic/claude"},
					{ModelName: "model-a", Model: "openai/gpt-4-turbo"},
				},
			},
			wantErr: false, // Changed: duplicates are allowed for load balancing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateModelList()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelList() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateModelList() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestModelConfig_RequestTimeoutParsing(t *testing.T) {
	jsonData := `{
		"model_name": "slow-local",
		"model": "openai/local-model",
		"api_base": "http://localhost:11434/v1",
		"request_timeout": 300
	}`

	var cfg ModelConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.RequestTimeout != 300 {
		t.Fatalf("RequestTimeout = %d, want 300", cfg.RequestTimeout)
	}
}

func TestModelConfig_RequestTimeoutDefaultZeroValue(t *testing.T) {
	jsonData := `{
		"model_name": "default-timeout",
		"model": "openai/gpt-4o",
		"api_key": "test-key"
	}`

	var cfg ModelConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.RequestTimeout != 0 {
		t.Fatalf("RequestTimeout = %d, want 0", cfg.RequestTimeout)
	}
}

func TestModelConfig_APIKeySingular(t *testing.T) {
	// Regression test: "api_key" (singular string) must be accepted
	// and populate APIKeys, so the Authorization header is set.
	jsonData := `{
		"model_name": "openrouter-test",
		"model": "openrouter/qwen/qwen3.6-plus:free",
		"api_base": "https://openrouter.ai/api/v1",
		"api_key": "sk-or-v1-test"
	}`

	var cfg ModelConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.APIKey() != "sk-or-v1-test" {
		t.Fatalf("APIKey() = %q, want %q", cfg.APIKey(), "sk-or-v1-test")
	}
}

func TestModelConfig_APIKeysPlural(t *testing.T) {
	// "api_keys" (plural array) must still work.
	jsonData := `{
		"model_name": "multi-key",
		"model": "openai/gpt-4o",
		"api_keys": ["key-a", "key-b"]
	}`

	var cfg ModelConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.APIKey() != "key-a" {
		t.Fatalf("APIKey() = %q, want %q", cfg.APIKey(), "key-a")
	}
	if len(cfg.APIKeys) != 2 {
		t.Fatalf("len(APIKeys) = %d, want 2", len(cfg.APIKeys))
	}
}

func TestModelConfig_APIKeyBothForms(t *testing.T) {
	// When both "api_key" and "api_keys" are provided, the singular key
	// should be prepended (if not already present in the array).
	jsonData := `{
		"model_name": "both",
		"model": "openai/gpt-4o",
		"api_key": "key-singular",
		"api_keys": ["key-plural"]
	}`

	var cfg ModelConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.APIKey() != "key-singular" {
		t.Fatalf("APIKey() = %q, want %q", cfg.APIKey(), "key-singular")
	}
	if len(cfg.APIKeys) != 2 {
		t.Fatalf("len(APIKeys) = %d, want 2", len(cfg.APIKeys))
	}
}

func TestModelConfig_APIKeyDuplicate(t *testing.T) {
	// If api_key is already in api_keys, don't duplicate it.
	jsonData := `{
		"model_name": "dup",
		"model": "openai/gpt-4o",
		"api_key": "same-key",
		"api_keys": ["same-key"]
	}`

	var cfg ModelConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(cfg.APIKeys) != 1 {
		t.Fatalf("len(APIKeys) = %d, want 1 (deduped)", len(cfg.APIKeys))
	}
}
