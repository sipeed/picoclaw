// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"encoding/json"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// NormalizeToolCall normalizes a ToolCall to ensure all fields are properly populated.
// It handles cases where Name/Arguments might be in different locations (top-level vs Function)
// and ensures both are populated consistently.
func NormalizeToolCall(tc ToolCall) ToolCall {
	normalized := tc

	// Ensure Name is populated from Function if not set
	if normalized.Name == "" && normalized.Function != nil {
		normalized.Name = normalized.Function.Name
	}

	// Ensure Arguments is not nil
	if normalized.Arguments == nil {
		normalized.Arguments = map[string]any{}
	}

	// Parse Arguments from Function.Arguments if not already set
	if len(normalized.Arguments) == 0 && normalized.Function != nil && normalized.Function.Arguments != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(normalized.Function.Arguments), &parsed); err == nil && parsed != nil {
			normalized.Arguments = parsed
		}
	}

	// Extract thought_signature from ExtraContent if present
	if normalized.ThoughtSignature == "" && normalized.ExtraContent != nil && normalized.ExtraContent.Google != nil {
		normalized.ThoughtSignature = normalized.ExtraContent.Google.ThoughtSignature
	}

	// Ensure Function is populated with consistent values
	argsJSON, _ := json.Marshal(normalized.Arguments)
	if normalized.Function == nil {
		normalized.Function = &protocoltypes.FunctionCall{
			Name:             normalized.Name,
			Arguments:        string(argsJSON),
			ThoughtSignature: normalized.ThoughtSignature,
		}
	} else {
		if normalized.Function.Name == "" {
			normalized.Function.Name = normalized.Name
		}
		if normalized.Name == "" {
			normalized.Name = normalized.Function.Name
		}
		if normalized.Function.Arguments == "" {
			normalized.Function.Arguments = string(argsJSON)
		}
		if normalized.Function.ThoughtSignature == "" {
			normalized.Function.ThoughtSignature = normalized.ThoughtSignature
		}
	}

	// Ensure ExtraContent reflects the thought_signature for Gemini 3 round-trips
	if normalized.ThoughtSignature != "" && (normalized.ExtraContent == nil || normalized.ExtraContent.Google == nil) {
		normalized.ExtraContent = &protocoltypes.ExtraContent{
			Google: &protocoltypes.GoogleExtra{
				ThoughtSignature: normalized.ThoughtSignature,
			},
		}
	}

	return normalized
}
