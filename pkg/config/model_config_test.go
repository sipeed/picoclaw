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
		ModelList: []ModelConfig{
			{ModelName: "test-model", Model: "openai/gpt-4o", APIKey: "key1"},
			{ModelName: "other-model", Model: "anthropic/claude", APIKey: "key2"},
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

func TestGetModelConfig_NotFound(t *testing.T) {
	cfg := &Config{
		ModelList: []ModelConfig{
			{ModelName: "test-model", Model: "openai/gpt-4o", APIKey: "key1"},
		},
	}

	_, err := cfg.GetModelConfig("nonexistent")
	if err == nil {
		t.Fatal("GetModelConfig() expected error for nonexistent model")
	}
}

func TestGetModelConfig_EmptyList(t *testing.T) {
	cfg := &Config{
		ModelList: []ModelConfig{},
	}

	_, err := cfg.GetModelConfig("any-model")
	if err == nil {
		t.Fatal("GetModelConfig() expected error for empty model list")
	}
}

func TestGetModelConfig_RoundRobin(t *testing.T) {
	cfg := &Config{
		ModelList: []ModelConfig{
			{ModelName: "lb-model", Model: "openai/gpt-4o-1", APIKey: "key1"},
			{ModelName: "lb-model", Model: "openai/gpt-4o-2", APIKey: "key2"},
			{ModelName: "lb-model", Model: "openai/gpt-4o-3", APIKey: "key3"},
		},
	}

	// Test round-robin distribution
	results := make(map[string]int)
	for i := 0; i < 30; i++ {
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

func TestGetModelConfig_Concurrent(t *testing.T) {
	cfg := &Config{
		ModelList: []ModelConfig{
			{ModelName: "concurrent-model", Model: "openai/gpt-4o-1", APIKey: "key1"},
			{ModelName: "concurrent-model", Model: "openai/gpt-4o-2", APIKey: "key2"},
		},
	}

	const goroutines = 100
	const iterations = 10

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := cfg.GetModelConfig("concurrent-model")
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent GetModelConfig() error: %v", err)
	}
}

func TestAgentDefaults_GetModelName_BackwardCompat(t *testing.T) {
	tests := []struct {
		name     string
		defaults AgentDefaults
		wantName string
	}{
		{
			name:     "new model_name field only",
			defaults: AgentDefaults{ModelName: "new-model"},
			wantName: "new-model",
		},
		{
			name:     "old model field only",
			defaults: AgentDefaults{Model: "legacy-model"},
			wantName: "legacy-model",
		},
		{
			name:     "both fields - model_name takes precedence",
			defaults: AgentDefaults{ModelName: "new-model", Model: "old-model"},
			wantName: "new-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.defaults.GetModelName(); got != tt.wantName {
				t.Errorf("GetModelName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestAgentDefaults_JSON_BackwardCompat(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantName string
	}{
		{
			name:     "new model_name field",
			json:     `{"model_name": "gpt4"}`,
			wantName: "gpt4",
		},
		{
			name:     "old model field",
			json:     `{"model": "gpt4"}`,
			wantName: "gpt4",
		},
		{
			name:     "both fields - model_name wins",
			json:     `{"model_name": "new", "model": "old"}`,
			wantName: "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var defaults AgentDefaults
			if err := json.Unmarshal([]byte(tt.json), &defaults); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if got := defaults.GetModelName(); got != tt.wantName {
				t.Errorf("GetModelName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestFullConfig_JSON_BackwardCompat(t *testing.T) {
	// Test complete config with both old and new formats
	oldFormat := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "gpt4",
				"max_tokens": 4096
			}
		},
		"model_list": [
			{
				"model_name": "gpt4",
				"model": "openai/gpt-4o",
				"api_key": "test-key"
			}
		]
	}`

	newFormat := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model_name": "gpt4",
				"max_tokens": 4096
			}
		},
		"model_list": [
			{
				"model_name": "gpt4",
				"model": "openai/gpt-4o",
				"api_key": "test-key"
			}
		]
	}`

	for name, jsonStr := range map[string]string{
		"old format (model)":    oldFormat,
		"new format (model_name)": newFormat,
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{}
			if err := json.Unmarshal([]byte(jsonStr), cfg); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			// Check that GetModelName returns correct value
			if got := cfg.Agents.Defaults.GetModelName(); got != "gpt4" {
				t.Errorf("GetModelName() = %q, want %q", got, "gpt4")
			}

			// Check that GetModelConfig works
			modelCfg, err := cfg.GetModelConfig("gpt4")
			if err != nil {
				t.Fatalf("GetModelConfig error: %v", err)
			}
			if modelCfg.Model != "openai/gpt-4o" {
				t.Errorf("Model = %q, want %q", modelCfg.Model, "openai/gpt-4o")
			}
		})
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
				ModelList: []ModelConfig{
					{ModelName: "test1", Model: "openai/gpt-4o"},
					{ModelName: "test2", Model: "anthropic/claude"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid entry",
			config: &Config{
				ModelList: []ModelConfig{
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
				ModelList: []ModelConfig{},
			},
			wantErr: false,
		},
		{
			// Load balancing: multiple entries with same model_name are allowed
			name: "duplicate model_name for load balancing",
			config: &Config{
				ModelList: []ModelConfig{
					{ModelName: "gpt-4", Model: "openai/gpt-4o", APIKey: "key1"},
					{ModelName: "gpt-4", Model: "openai/gpt-4-turbo", APIKey: "key2"},
				},
			},
			wantErr: false, // Changed: duplicates are allowed for load balancing
		},
		{
			// Load balancing: non-adjacent entries with same model_name are also allowed
			name: "duplicate model_name non-adjacent for load balancing",
			config: &Config{
				ModelList: []ModelConfig{
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
