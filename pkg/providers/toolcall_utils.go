// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

// NormalizeToolCall normalizes a ToolCall to ensure all fields are properly populated.
// It handles cases where Name/Arguments might be in different locations (top-level vs Function)
// and ensures both are populated consistently.
func NormalizeToolCall(tc ToolCall) ToolCall {
	normalized := tc

	// Ensure Name is populated from Function if not set.
	if normalized.Name == "" && normalized.Function != nil {
		normalized.Name = normalized.Function.Name
	}

	// Ensure Arguments is not nil.
	if normalized.Arguments == nil {
		normalized.Arguments = map[string]any{}
	}

	// Populate top-level arguments from Function arguments when needed.
	if len(normalized.Arguments) == 0 && normalized.Function != nil && len(normalized.Function.Arguments) > 0 {
		normalized.Arguments = cloneToolArgs(normalized.Function.Arguments)
	}

	// Ensure Function is populated with consistent values.
	if normalized.Function == nil {
		normalized.Function = &FunctionCall{
			Name:      normalized.Name,
			Arguments: cloneToolArgs(normalized.Arguments),
		}
	} else {
		if normalized.Function.Name == "" {
			normalized.Function.Name = normalized.Name
		}
		if normalized.Name == "" {
			normalized.Name = normalized.Function.Name
		}
		if len(normalized.Function.Arguments) == 0 {
			normalized.Function.Arguments = cloneToolArgs(normalized.Arguments)
		}
	}

	return normalized
}

func cloneToolArgs(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
