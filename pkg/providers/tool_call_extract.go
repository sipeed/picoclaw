package providers

import (
	"encoding/json"
	"strings"
)

// extractToolCallsFromText parses tool call JSON from response text.
// Both ClaudeCliProvider and CodexCliProvider use this to extract
// tool calls that the model outputs in its response text.
func extractToolCallsFromText(text string) []ToolCall {
	// Try direct match first (handles compact JSON)
	if calls := extractToolCallsFromTextImpl(text); len(calls) > 0 {
		return calls
	}
	// Try with trimmed whitespace (handles pretty-printed JSON with leading whitespace)
	trimmed := strings.TrimSpace(text)
	if trimmed != text {
		if calls := extractToolCallsFromTextImpl(trimmed); len(calls) > 0 {
			return calls
		}
	}
	// Try to parse the entire text as JSON and look for tool_calls
	// (handles cases where tool_calls is nested or prefixed with other content)
	return extractToolCallsFromJSON(text)
}

// extractToolCallsFromTextImpl does the actual extraction assuming the input
// starts with the JSON containing tool_calls.
func extractToolCallsFromTextImpl(text string) []ToolCall {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return nil
	}

	end := findMatchingBrace(text, start)
	if end == start {
		return nil
	}

	jsonStr := text[start:end]

	var wrapper struct {
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return nil
	}

	var result []ToolCall
	for _, tc := range wrapper.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		result = append(result, ToolCall{
			ID:        tc.ID,
			Type:      tc.Type,
			Name:      tc.Function.Name,
			Arguments: args,
			Function: &FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return result
}

// extractToolCallsFromJSON attempts to parse the entire text as JSON and
// extract tool_calls from anywhere in the object graph.
func extractToolCallsFromJSON(text string) []ToolCall {
	// Try to find "tool_calls": anywhere in the text using a search for the field pattern.
	// Using "tool_calls": (with colon) avoids matching tool_calls inside string values.
	idx := strings.Index(text, `"tool_calls":`)
	if idx == -1 {
		return nil
	}

	// Verify this is a JSON field name, not "tool_calls" inside a string value.
	// The char before the opening quote must be a JSON delimiter ({, [, ,, or whitespace).
	if idx > 0 {
		prevChar := text[idx-1]
		if prevChar != '{' && prevChar != '[' && prevChar != ',' && prevChar != ' ' && prevChar != '\t' && prevChar != '\n' && prevChar != '\r' {
			// "tool_calls": found inside a string value (e.g., "tool_calls_extra" or "prefix tool_calls")
			// Search for the next occurrence
			if nextCalls := extractToolCallsFromJSON(text[idx+len(`"tool_calls":`):]); len(nextCalls) > 0 {
				return nextCalls
			}
			return nil
		}
	}

	// Find the opening brace before "tool_calls"
	braceStart := idx
	for braceStart >= 0 && text[braceStart] != '{' {
		braceStart--
	}
	if braceStart < 0 {
		return nil
	}

	end := findMatchingBrace(text, braceStart)
	if end == braceStart {
		return nil
	}

	jsonStr := text[braceStart:end]
	var wrapper struct {
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil || len(wrapper.ToolCalls) == 0 {
		return nil
	}

	var result []ToolCall
	for _, tc := range wrapper.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		result = append(result, ToolCall{
			ID:        tc.ID,
			Type:      tc.Type,
			Name:      tc.Function.Name,
			Arguments: args,
			Function: &FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return result
}

// stripToolCallsFromText removes tool call JSON from response text.
// Handles both compact JSON and pretty-printed JSON with whitespace.
func stripToolCallsFromText(text string) string {
	// Try direct match first
	result := stripToolCallsImpl(text)
	if result != text {
		return strings.TrimSpace(result)
	}
	// Try with trimmed whitespace
	trimmed := strings.TrimSpace(text)
	if trimmed != text {
		result = stripToolCallsImpl(trimmed)
		if result != trimmed {
			return strings.TrimSpace(result)
		}
	}
	// Try to find and strip tool_calls JSON from anywhere in the text
	return stripToolCallsFromJSON(text)
}

// stripToolCallsImpl does the actual stripping assuming the input starts with JSON.
func stripToolCallsImpl(text string) string {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return text
	}

	end := findMatchingBrace(text, start)
	if end == start {
		return text
	}

	return text[:start] + text[end:]
}

// stripToolCallsFromJSON finds and removes tool_calls JSON from anywhere in the text.
func stripToolCallsFromJSON(text string) string {
	// Search for "tool_calls": (with colon) to avoid matching tool_calls inside string values.
	// This is more specific than just "tool_calls" and reduces false positives.
	idx := strings.Index(text, `"tool_calls":`)
	if idx == -1 {
		return text
	}

	// Verify this is a JSON field name, not "tool_calls" inside a string value.
	// The char before the opening quote must be a JSON delimiter ({, [, ,, or whitespace).
	if idx > 0 {
		prevChar := text[idx-1]
		if prevChar != '{' && prevChar != '[' && prevChar != ',' && prevChar != ' ' && prevChar != '\t' && prevChar != '\n' && prevChar != '\r' {
			// "tool_calls": found inside a string value (e.g., "tool_calls_extra" or "prefix tool_calls")
			// Search for the next occurrence
			return stripToolCallsFromJSON(text[idx+len(`"tool_calls":`):])
		}
	}

	// Find the opening brace before "tool_calls"
	braceStart := idx
	for braceStart >= 0 && text[braceStart] != '{' {
		braceStart--
	}
	if braceStart < 0 {
		return text
	}

	end := findMatchingBrace(text, braceStart)
	if end == braceStart {
		return text
	}

	return strings.TrimSpace(text[:braceStart] + text[end:])
}
