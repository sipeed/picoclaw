package providers

import (
	"encoding/json"
	"strings"
)

// extractToolCallsFromText parses tool call JSON from response text.
// Both ClaudeCliProvider and CodexCliProvider use this to extract
// tool calls that the model outputs in its response text.
func extractToolCallsFromText(text string) []ToolCall {
	if calls := extractJSONWrapperToolCalls(text); len(calls) > 0 {
		return calls
	}

	if call, ok := extractWebAPIToolCall(text); ok {
		return []ToolCall{call}
	}

	return nil
}

func extractJSONWrapperToolCalls(text string) []ToolCall {
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

func extractWebAPIToolCall(text string) (ToolCall, bool) {
	start := strings.Index(text, "/WebAPI")
	if start == -1 {
		return ToolCall{}, false
	}

	jsonStart := strings.Index(text[start:], "{")
	if jsonStart == -1 {
		return ToolCall{}, false
	}
	jsonStart += start

	jsonEnd := findMatchingBrace(text, jsonStart)
	if jsonEnd == jsonStart {
		return ToolCall{}, false
	}

	jsonStr := text[jsonStart:jsonEnd]

	var webAPICall struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &webAPICall); err != nil {
		return ToolCall{}, false
	}

	if strings.TrimSpace(webAPICall.Name) == "" {
		return ToolCall{}, false
	}

	argBytes, err := json.Marshal(webAPICall.Arguments)
	if err != nil {
		argBytes = []byte("{}")
	}

	return ToolCall{
		Type:      "function",
		Name:      webAPICall.Name,
		Arguments: webAPICall.Arguments,
		Function: &FunctionCall{
			Name:      webAPICall.Name,
			Arguments: string(argBytes),
		},
	}, true
}

// stripToolCallsFromText removes tool call JSON from response text.
func stripToolCallsFromText(text string) string {
	if stripped, ok := stripJSONWrapperToolCalls(text); ok {
		return stripped
	}

	if stripped, ok := stripWebAPIToolCall(text); ok {
		return stripped
	}

	return text
}

func stripJSONWrapperToolCalls(text string) (string, bool) {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return "", false
	}

	end := findMatchingBrace(text, start)
	if end == start {
		return "", false
	}

	return strings.TrimSpace(text[:start] + text[end:]), true
}

func stripWebAPIToolCall(text string) (string, bool) {
	start := strings.Index(text, "/WebAPI")
	if start == -1 {
		return "", false
	}

	endTag := strings.Index(text[start:], "</tool_call>")
	if endTag == -1 {
		jsonStart := strings.Index(text[start:], "{")
		if jsonStart == -1 {
			return "", false
		}
		jsonStart += start

		jsonEnd := findMatchingBrace(text, jsonStart)
		if jsonEnd == jsonStart {
			return "", false
		}

		return strings.TrimSpace(text[:start] + text[jsonEnd:]), true
	}

	endTag += start + len("</tool_call>")
	return strings.TrimSpace(text[:start] + text[endTag:]), true
}
