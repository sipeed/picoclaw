// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import "encoding/json"

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

	// Ensure Function is populated with consistent values
	argsJSON, _ := json.Marshal(normalized.Arguments)
	if normalized.Function == nil {
		normalized.Function = &FunctionCall{
			Name:      normalized.Name,
			Arguments: string(argsJSON),
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
	}

	return normalized
}
