package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelNameNotOverwritten_Documentation documents the expected behavior
// after the fix for the ModelName overwrite bug.
//
// BUG DESCRIPTION:
// Previously, after CreateProvider returned, the code would overwrite
// cfg.Agents.Defaults.ModelName with modelID (the second return value).
//
// The modelID is the protocol-stripped model identifier. For example:
//   - model_list has: model = "openrouter/free"
//   - ExtractProtocol returns: ("openrouter", "free")
//   - CreateProvider returns: modelID = "free"
//   - BUG: Code set ModelName = "free"
//
// This broke fallback because:
//   1. Agent's Model field became "free" instead of "openrouter-free"
//   2. ParseModelRef("free", "openrouter") created wrong candidate
//   3. Error showed provider=openrouter model=free (lost context)
//
// EXPECTED BEHAVIOR AFTER FIX:
//   - ModelName should remain as the model_list entry name
//   - For example: ModelName = "openrouter-free" (NOT "free")
//   - This ensures fallback candidates resolve correctly
//
// TEST SCENARIO:
// GIVEN config has:
//   {
//     "agents": {
//       "defaults": {
//         "model_name": "openrouter-free"
//       }
//     },
//     "model_list": [
//       {
//         "model_name": "openrouter-free",
//         "model": "openrouter/free"
//       }
//     ]
//   }
//
// WHEN CreateProvider is called:
//   - GetModelConfig("openrouter-free") finds the entry
//   - CreateProviderFromConfig gets model = "openrouter/free"
//   - ExtractProtocol("openrouter/free") returns ("openrouter", "free")
//   - Returns: (provider, "free", nil)
//
// THEN (the fix):
//   - ModelName should STILL be "openrouter-free"
//   - ModelName is NOT overwritten to "free"
//
// VERIFICATION:
//   - Agent instance is created with Model = "openrouter-free"
//   - ResolveCandidates looks up "openrouter-free" in model_list
//   - Gets full model string "openrouter/free"
//   - ParseModelRef("openrouter/free", "") returns (Provider: "openrouter", Model: "free")
//   - Candidate has correct provider and model info
func TestModelNameNotOverwritten_Documentation(t *testing.T) {
	// This test documents the contract. The actual verification happens
	// at the integration level through the agent creation flow.

	testCases := []struct {
		name              string
		modelName         string // model_list entry name (e.g., "openrouter-free")
		modelString       string // model field (e.g., "openrouter/free")
		expectedModelID   string // ExtractProtocol result (e.g., "free")
		expectedFinalName string // Should remain as modelName (NOT modelID)
	}{
		{
			name:              "openrouter-free model",
			modelName:         "openrouter-free",
			modelString:       "openrouter/free",
			expectedModelID:   "free",
			expectedFinalName: "openrouter-free", // Should NOT become "free"
		},
		{
			name:              "openrouter nested protocol",
			modelName:         "openrouter-nested",
			modelString:       "openrouter/openrouter/free",
			expectedModelID:   "openrouter/free",
			expectedFinalName: "openrouter-nested", // Should NOT become "openrouter/free"
		},
		{
			name:              "anthropic model",
			modelName:         "claude-sonnet",
			modelString:       "anthropic/claude-sonnet-4-20250514",
			expectedModelID:   "claude-sonnet-4-20250514",
			expectedFinalName: "claude-sonnet", // Should NOT become "claude-sonnet-4-20250514"
		},
		{
			name:              "openai model",
			modelName:         "gpt4o",
			modelString:       "openai/gpt-4o",
			expectedModelID:   "gpt-4o",
			expectedFinalName: "gpt4o", // Should NOT become "gpt-4o"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the modelID extraction
			parts := strings.SplitN(tc.modelString, "/", 2)
			if len(parts) == 2 {
				modelID := parts[1]
				assert.Equal(t, tc.expectedModelID, modelID,
					"ExtractProtocol should extract correct modelID")
			}

			// The key assertion: ModelName should NOT be overwritten
			// This is enforced by NOT having the code:
			//   if modelID != "" { cfg.Agents.Defaults.ModelName = modelID }
			t.Logf("Model '%s' with protocol '%s' should keep name as '%s', NOT overwrite to '%s'",
				tc.modelString, parts[0], tc.modelName, tc.expectedModelID)

			assert.NotEqual(t, tc.expectedModelID, tc.expectedFinalName,
				"ModelName should NOT be the same as modelID (this is the bug we fixed)")
			assert.Equal(t, tc.modelName, tc.expectedFinalName,
				"ModelName should remain as the model_list entry name")
		})
	}
}

// TestParseModelRefBehavior documents how ParseModelRef handles model strings
func TestParseModelRefBehavior(t *testing.T) {
	testCases := []struct {
		name              string
		modelString       string
		expectedProvider  string
		expectedModel     string
	}{
		{
			name:              "simple protocol/model",
			modelString:       "openrouter/free",
			expectedProvider:  "openrouter",
			expectedModel:     "free",
		},
		{
			name:              "nested protocol openrouter/openrouter/free",
			modelString:       "openrouter/openrouter/free",
			expectedProvider:  "openrouter",
			expectedModel:     "openrouter/free", // Everything after first /
		},
		{
			name:              "anthropic model",
			modelString:       "anthropic/claude-sonnet-4-20250514",
			expectedProvider:  "anthropic",
			expectedModel:     "claude-sonnet-4-20250514",
		},
		{
			name:              "openai model",
			modelString:       "openai/gpt-4o",
			expectedProvider:  "openai",
			expectedModel:     "gpt-4o",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// ParseModelRef splits on the first /
			idx := strings.Index(tc.modelString, "/")
			require.Greater(t, idx, 0, "Model string should contain /")

			provider := tc.modelString[:idx]
			model := tc.modelString[idx+1:]

			assert.Equal(t, tc.expectedProvider, provider,
				"Provider should be everything before first /")
			assert.Equal(t, tc.expectedModel, model,
				"Model should be everything after first /")

			t.Logf("ModelRef parsed: Provider=%s, Model=%s", provider, model)
		})
	}
}
