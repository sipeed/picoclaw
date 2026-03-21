package config

import (
	"testing"
)

func TestExpandMultiKeyModels_SingleKey(t *testing.T) {
	models := []ModelConfig{
		{
			ModelName: "gpt-4",
			Model:     "openai/gpt-4o",
			APIKey:    "single-key",
		},
	}

	result := ExpandMultiKeyModels(models)

	if len(result) != 1 {
		t.Fatalf("expected 1 model, got %d", len(result))
	}

	if result[0].ModelName != "gpt-4" {
		t.Errorf("expected model_name 'gpt-4', got %q", result[0].ModelName)
	}

	if result[0].APIKey != "single-key" {
		t.Errorf("expected api_key 'single-key', got %q", result[0].APIKey)
	}

	if len(result[0].Fallbacks) != 0 {
		t.Errorf("expected no fallbacks, got %v", result[0].Fallbacks)
	}
}

func TestExpandMultiKeyModels_APIKeysOnly(t *testing.T) {
	models := []ModelConfig{
		{
			ModelName: "glm-4.7",
			Model:     "zhipu/glm-4.7",
			APIBase:   "https://api.example.com",
			APIKeys:   []string{"key1", "key2", "key3"},
		},
	}

	result := ExpandMultiKeyModels(models)

	// Should expand to 3 models
	if len(result) != 3 {
		t.Fatalf("expected 3 models, got %d", len(result))
	}

	// First entry should be the primary with key1 and fallbacks
	primary := result[2] // Primary is added last
	if primary.ModelName != "glm-4.7" {
		t.Errorf("expected primary model_name 'glm-4.7', got %q", primary.ModelName)
	}
	if primary.APIKey != "key1" {
		t.Errorf("expected primary api_key 'key1', got %q", primary.APIKey)
	}
	if len(primary.Fallbacks) != 2 {
		t.Errorf("expected 2 fallbacks, got %d", len(primary.Fallbacks))
	}
	if primary.Fallbacks[0] != "glm-4.7__key_1" {
		t.Errorf("expected first fallback 'glm-4.7__key_1', got %q", primary.Fallbacks[0])
	}
	if primary.Fallbacks[1] != "glm-4.7__key_2" {
		t.Errorf("expected second fallback 'glm-4.7__key_2', got %q", primary.Fallbacks[1])
	}

	// Second entry should be key2
	second := result[0]
	if second.ModelName != "glm-4.7__key_1" {
		t.Errorf("expected second model_name 'glm-4.7__key_1', got %q", second.ModelName)
	}
	if second.APIKey != "key2" {
		t.Errorf("expected second api_key 'key2', got %q", second.APIKey)
	}

	// Third entry should be key3
	third := result[1]
	if third.ModelName != "glm-4.7__key_2" {
		t.Errorf("expected third model_name 'glm-4.7__key_2', got %q", third.ModelName)
	}
	if third.APIKey != "key3" {
		t.Errorf("expected third api_key 'key3', got %q", third.APIKey)
	}
}

func TestExpandMultiKeyModels_APIKeyAndAPIKeys(t *testing.T) {
	models := []ModelConfig{
		{
			ModelName: "gpt-4",
			Model:     "openai/gpt-4o",
			APIKey:    "key0",
			APIKeys:   []string{"key1", "key2"},
		},
	}

	result := ExpandMultiKeyModels(models)

	// Should expand to 3 models (key0 from APIKey + key1, key2 from APIKeys)
	if len(result) != 3 {
		t.Fatalf("expected 3 models, got %d", len(result))
	}

	// Primary should use key0
	primary := result[2]
	if primary.APIKey != "key0" {
		t.Errorf("expected primary api_key 'key0', got %q", primary.APIKey)
	}
	if len(primary.Fallbacks) != 2 {
		t.Errorf("expected 2 fallbacks, got %d", len(primary.Fallbacks))
	}
}

func TestExpandMultiKeyModels_WithExistingFallbacks(t *testing.T) {
	models := []ModelConfig{
		{
			ModelName: "gpt-4",
			Model:     "openai/gpt-4o",
			APIKeys:   []string{"key1", "key2"},
			Fallbacks: []string{"claude-3"},
		},
	}

	result := ExpandMultiKeyModels(models)

	primary := result[1]
	// With 2 keys, we get 1 key fallback + 1 existing fallback = 2 total
	if len(primary.Fallbacks) != 2 {
		t.Fatalf("expected 2 fallbacks, got %d: %v", len(primary.Fallbacks), primary.Fallbacks)
	}

	// Key fallbacks should come first, then existing fallbacks
	if primary.Fallbacks[0] != "gpt-4__key_1" {
		t.Errorf("expected first fallback 'gpt-4__key_1', got %q", primary.Fallbacks[0])
	}
	if primary.Fallbacks[1] != "claude-3" {
		t.Errorf("expected second fallback 'claude-3', got %q", primary.Fallbacks[1])
	}
}

func TestExpandMultiKeyModels_EmptyAPIKeys(t *testing.T) {
	models := []ModelConfig{
		{
			ModelName: "gpt-4",
			Model:     "openai/gpt-4o",
			APIKey:    "",
			APIKeys:   []string{},
		},
	}

	result := ExpandMultiKeyModels(models)

	// Should keep as-is with no changes
	if len(result) != 1 {
		t.Fatalf("expected 1 model, got %d", len(result))
	}

	if result[0].ModelName != "gpt-4" {
		t.Errorf("expected model_name 'gpt-4', got %q", result[0].ModelName)
	}
}

func TestExpandMultiKeyModels_Deduplication(t *testing.T) {
	models := []ModelConfig{
		{
			ModelName: "gpt-4",
			Model:     "openai/gpt-4o",
			APIKey:    "key1",
			APIKeys:   []string{"key1", "key2", "key1"}, // Duplicate key1
		},
	}

	result := ExpandMultiKeyModels(models)

	// Should only create 2 models (deduplicated keys)
	if len(result) != 2 {
		t.Fatalf("expected 2 models (deduplicated), got %d", len(result))
	}

	primary := result[1]
	if primary.APIKey != "key1" {
		t.Errorf("expected primary api_key 'key1', got %q", primary.APIKey)
	}
	if len(primary.Fallbacks) != 1 {
		t.Errorf("expected 1 fallback, got %d", len(primary.Fallbacks))
	}
}

func TestExpandMultiKeyModels_PreservesOtherFields(t *testing.T) {
	models := []ModelConfig{
		{
			ModelName:      "gpt-4",
			Model:          "openai/gpt-4o",
			APIBase:        "https://api.example.com",
			APIKeys:        []string{"key1", "key2"},
			Proxy:          "http://proxy:8080",
			RPM:            60,
			MaxTokensField: "max_completion_tokens",
			RequestTimeout: 30,
			ThinkingLevel:  "high",
		},
	}

	result := ExpandMultiKeyModels(models)

	// Check primary entry preserves all fields
	primary := result[1]
	if primary.APIBase != "https://api.example.com" {
		t.Errorf("expected api_base preserved, got %q", primary.APIBase)
	}
	if primary.Proxy != "http://proxy:8080" {
		t.Errorf("expected proxy preserved, got %q", primary.Proxy)
	}
	if primary.RPM != 60 {
		t.Errorf("expected rpm preserved, got %d", primary.RPM)
	}
	if primary.MaxTokensField != "max_completion_tokens" {
		t.Errorf("expected max_tokens_field preserved, got %q", primary.MaxTokensField)
	}
	if primary.RequestTimeout != 30 {
		t.Errorf("expected request_timeout preserved, got %d", primary.RequestTimeout)
	}
	if primary.ThinkingLevel != "high" {
		t.Errorf("expected thinking_level preserved, got %q", primary.ThinkingLevel)
	}

	// Check additional entry also preserves fields
	additional := result[0]
	if additional.APIBase != "https://api.example.com" {
		t.Errorf("expected additional api_base preserved, got %q", additional.APIBase)
	}
	if additional.RPM != 60 {
		t.Errorf("expected additional rpm preserved, got %d", additional.RPM)
	}
}

func TestMergeAPIKeys(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		apiKeys  []string
		expected []string
	}{
		{
			name:     "both empty",
			apiKey:   "",
			apiKeys:  nil,
			expected: nil,
		},
		{
			name:     "only apiKey",
			apiKey:   "key1",
			apiKeys:  nil,
			expected: []string{"key1"},
		},
		{
			name:     "only apiKeys",
			apiKey:   "",
			apiKeys:  []string{"key1", "key2"},
			expected: []string{"key1", "key2"},
		},
		{
			name:     "both with overlap",
			apiKey:   "key1",
			apiKeys:  []string{"key1", "key2", "key3"},
			expected: []string{"key1", "key2", "key3"},
		},
		{
			name:     "with whitespace",
			apiKey:   "  key1  ",
			apiKeys:  []string{"  key2  ", "  key1  "},
			expected: []string{"key1", "key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeAPIKeys(tt.apiKey, tt.apiKeys)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d keys, got %d", len(tt.expected), len(result))
			}
			for i, k := range result {
				if k != tt.expected[i] {
					t.Errorf("expected key[%d] = %q, got %q", i, tt.expected[i], k)
				}
			}
		})
	}
}
