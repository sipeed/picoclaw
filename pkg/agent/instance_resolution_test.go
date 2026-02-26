package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveModelString tests that model names are correctly resolved to full model strings
func TestResolveModelString(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		modelListEntry *config.ModelConfig
		expectedResult string
	}{
		{
			name:           "model name without slash - looks up in model_list",
			modelName:      "gemini-flash",
			modelListEntry: &config.ModelConfig{Model: "antigravity/gemini-3-flash"},
			expectedResult: "antigravity/gemini-3-flash",
		},
		{
			name:           "model name with slash - returned as-is",
			modelName:      "openrouter/free",
			modelListEntry: nil,
			expectedResult: "openrouter/free",
		},
		{
			name:           "model name not in model_list - returned as-is",
			modelName:      "unknown-model",
			modelListEntry: nil,
			expectedResult: "unknown-model",
		},
		{
			name:           "nested protocol model",
			modelName:      "openrouter-nested",
			modelListEntry: &config.ModelConfig{Model: "openrouter/openrouter/free"},
			expectedResult: "openrouter/openrouter/free",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			if tt.modelListEntry != nil {
				tt.modelListEntry.ModelName = tt.modelName
				cfg.ModelList = []config.ModelConfig{*tt.modelListEntry}
			}

			result := resolveModelString(cfg, tt.modelName)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestResolveFallbackModelStrings tests that multiple model names are resolved correctly
func TestResolveFallbackModelStrings(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{ModelName: "gemini-flash", Model: "antigravity/gemini-3-flash"},
			{ModelName: "openrouter-free", Model: "openrouter/free"},
			{ModelName: "claude-sonnet", Model: "anthropic/claude-sonnet-4-20250514"},
		},
	}

	modelNames := []string{"gemini-flash", "openrouter-free", "claude-sonnet"}
	result := resolveFallbackModelStrings(cfg, modelNames)

	expected := []string{
		"antigravity/gemini-3-flash",
		"openrouter/free",
		"anthropic/claude-sonnet-4-20250514",
	}

	assert.Equal(t, expected, result)
}

// TestResolveFallbackModelStrings_MixedInput tests resolution with mixed input
func TestResolveFallbackModelStrings_MixedInput(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{ModelName: "gemini-flash", Model: "antigravity/gemini-3-flash"},
		},
	}

	// Mix of model names and full model strings
	modelNames := []string{
		"gemini-flash",         // Should resolve to "antigravity/gemini-3-flash"
		"openrouter/free",      // Already a full string, kept as-is
		"anthropic/claude-3-5",  // Already a full string, kept as-is
	}
	result := resolveFallbackModelStrings(cfg, modelNames)

	expected := []string{
		"antigravity/gemini-3-flash",
		"openrouter/free",
		"anthropic/claude-3-5",
	}

	assert.Equal(t, expected, result)
}

// TestNewAgentInstance_ModelResolution is an integration test that verifies
// the full model resolution flow when creating an agent instance
func TestNewAgentInstance_ModelResolution(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelConfig{
			{
				ModelName: "gemini-flash",
				Model:     "antigravity/gemini-3-flash",
				APIKey:    "test-key",
			},
			{
				ModelName: "openrouter-free",
				Model:     "openrouter/free",
				APIKey:    "sk-or-test",
				APIBase:   "https://openrouter.ai/api/v1",
			},
		},
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:      t.TempDir(),
				ModelName:      "gemini-flash",
				ModelFallbacks: []string{"openrouter-free"},
			},
		},
	}

	// We don't need an actual provider for this test since we're not calling Chat
	// Use the existing mockProvider from mock_provider_test.go
	provider := &mockProvider{}

	instance := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, provider)

	require.NotNil(t, instance)

	// Verify the agent's Model field is still the model_name (not overwritten)
	assert.Equal(t, "gemini-flash", instance.Model,
		"Agent's Model field should remain as the model_list entry name")

	// Verify the Candidates have been resolved to full model strings
	require.Len(t, instance.Candidates, 2, "Should have 2 candidates (primary + fallback)")

	// First candidate should be the resolved primary model
	assert.Equal(t, "antigravity", instance.Candidates[0].Provider,
		"Primary candidate provider should be 'antigravity'")
	assert.Equal(t, "gemini-3-flash", instance.Candidates[0].Model,
		"Primary candidate model should be 'gemini-3-flash' (not 'antigravity/gemini-3-flash')")

	// Second candidate should be the resolved fallback model
	assert.Equal(t, "openrouter", instance.Candidates[1].Provider,
		"Fallback candidate provider should be 'openrouter'")
	assert.Equal(t, "free", instance.Candidates[1].Model,
		"Fallback candidate model should be 'free' (not 'openrouter/free')")
}
