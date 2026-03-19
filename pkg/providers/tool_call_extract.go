package providers

import (
	"encoding/json"
	"strings"
)

// extractToolCallsFromText parses tool call JSON from response text.
// Both ClaudeCliProvider and CodexCliProvider use this to extract
// tool calls that the model outputs in its response text.
//
// The algorithm is robust to pretty-printed JSON and markdown code fences:
//  1. Strip any surrounding code fences (```json ... ``` or ``` ... ```)
//  2. Find the first '{' and last '}' to extract the JSON candidate
//  3. Unmarshal the candidate and check for a non-empty "tool_calls" array
//  4. Parse each tool call, handling arguments as either a JSON string or object
func extractToolCallsFromText(text string) []ToolCall {
	text = stripCodeFences(text)

	firstBrace := strings.Index(text, "{")
	if firstBrace == -1 {
		return nil
	}
	lastBrace := strings.LastIndex(text, "}")
	if lastBrace == -1 || lastBrace <= firstBrace {
		return nil
	}

	candidate := text[firstBrace : lastBrace+1]

	var wrapper struct {
		ToolCalls []json.RawMessage `json:"tool_calls"`
	}
	if err := json.Unmarshal([]byte(candidate), &wrapper); err != nil {
		return nil
	}
	if len(wrapper.ToolCalls) == 0 {
		return nil
	}

	var result []ToolCall
	for _, raw := range wrapper.ToolCalls {
		var tc struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function"`
		}
		if err := json.Unmarshal(raw, &tc); err != nil {
			continue
		}

		args, rawArgsStr := parseToolCallArguments(tc.Function.Arguments)

		result = append(result, ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Name: tc.Function.Name,
			Arguments: args,
			Function: &FunctionCall{
				Name:      tc.Function.Name,
				Arguments: rawArgsStr,
			},
		})
	}

	return result
}

// stripToolCallsFromText removes tool call JSON from response text,
// preserving any text before the first '{' or after the last '}'.
func stripToolCallsFromText(text string) string {
	stripped := stripCodeFences(text)

	firstBrace := strings.Index(stripped, "{")
	if firstBrace == -1 {
		return text
	}
	lastBrace := strings.LastIndex(stripped, "}")
	if lastBrace == -1 || lastBrace <= firstBrace {
		return text
	}

	// Validate that the candidate is actually a tool_calls JSON blob
	candidate := stripped[firstBrace : lastBrace+1]
	var wrapper struct {
		ToolCalls []json.RawMessage `json:"tool_calls"`
	}
	if err := json.Unmarshal([]byte(candidate), &wrapper); err != nil || len(wrapper.ToolCalls) == 0 {
		return text
	}

	// Use positions in the original (pre-fence-stripped) text to preserve
	// any content outside code fences. If the text was fence-wrapped we strip
	// the whole thing; otherwise cut out just the JSON range.
	originalFirst := strings.Index(text, "{")
	originalLast := strings.LastIndex(text, "}")
	if originalFirst == -1 || originalLast == -1 || originalLast <= originalFirst {
		return text
	}

	return strings.TrimSpace(text[:originalFirst] + text[originalLast+1:])
}

// parseToolCallArguments handles arguments that may be either a JSON string
// (which itself encodes a JSON object) or a JSON object directly.
// Returns the parsed map and a canonical string representation.
func parseToolCallArguments(raw json.RawMessage) (map[string]any, string) {
	if len(raw) == 0 {
		return nil, ""
	}

	trimmed := strings.TrimSpace(string(raw))

	if strings.HasPrefix(trimmed, `"`) {
		// Arguments is a JSON-encoded string; decode the string then parse it as JSON.
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return nil, trimmed
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(encoded), &args); err != nil {
			return nil, encoded
		}
		return args, encoded
	}

	if strings.HasPrefix(trimmed, "{") {
		// Arguments is already a JSON object.
		var args map[string]any
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, trimmed
		}
		return args, trimmed
	}

	return nil, trimmed
}

// stripCodeFences removes markdown code fences (```json...``` or ```...```)
// from text, returning the inner content trimmed of surrounding whitespace.
// If no code fence is present the original text is returned unchanged.
func stripCodeFences(text string) string {
	s := strings.TrimSpace(text)

	for _, fence := range []string{"```json", "```"} {
		if strings.HasPrefix(s, fence) {
			rest := s[len(fence):]
			// The fence opener may be immediately followed by content or a newline.
			if idx := strings.Index(rest, "```"); idx != -1 {
				return strings.TrimSpace(rest[:idx])
			}
		}
	}

	return text
}

