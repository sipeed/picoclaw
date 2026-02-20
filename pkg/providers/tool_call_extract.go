package providers

import (
	"encoding/json"
	"strings"
)

// extractToolCallsFromText parses tool call JSON from response text.
// Both ClaudeCliProvider and CodexCliProvider use this to extract
// tool calls that the model outputs in its response text.
func extractToolCallsFromText(text string) []ToolCall {
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
		var args map[string]interface{}
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

// stripXMLToolCalls removes XML tool call blocks (e.g. <minimax:toolcall>...</minimax:toolcall>)
// from response text. Some providers embed raw XML tool calls in Content alongside
// structured tool_calls; this prevents them from leaking to users.
func stripXMLToolCalls(text string) string {
	// Match <vendor:toolcall>...</vendor:toolcall> blocks (any namespace prefix)
	idx := strings.Index(text, ":toolcall>")
	if idx == -1 {
		return text
	}
	// Find the opening tag start: scan backwards for '<'
	tagStart := strings.LastIndex(text[:idx], "<")
	if tagStart == -1 {
		return text
	}
	// Extract namespace (e.g. "minimax" from "<minimax:toolcall>")
	ns := text[tagStart+1 : idx]
	closeTag := "</" + ns + ":toolcall>"
	closeIdx := strings.Index(text, closeTag)
	if closeIdx == -1 {
		return text
	}
	cleaned := text[:tagStart] + text[closeIdx+len(closeTag):]
	// Recursively strip if there are more blocks
	if strings.Contains(cleaned, ":toolcall>") {
		cleaned = stripXMLToolCalls(cleaned)
	}
	return strings.TrimSpace(cleaned)
}

// stripToolCallsFromText removes tool call JSON from response text.
func stripToolCallsFromText(text string) string {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return text
	}

	end := findMatchingBrace(text, start)
	if end == start {
		return text
	}

	return strings.TrimSpace(text[:start] + text[end:])
}
