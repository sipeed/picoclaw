package providers

import (
	"encoding/json"
	"strings"
)

type textToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// extractToolCallsFromText parses tool call JSON from response text.
// Both ClaudeCliProvider and CodexCliProvider use this to extract
// tool calls that the model outputs in its response text.
func extractToolCallsFromText(text string) []ToolCall {
	start, end, rawToolCalls, ok := locateToolCallObject(text)
	if !ok || end <= start {
		return nil
	}

	var result []ToolCall
	for _, tc := range rawToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

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
func stripToolCallsFromText(text string) string {
	start, end, _, ok := locateToolCallObject(text)
	if !ok || end <= start {
		return text
	}

	return strings.TrimSpace(text[:start] + text[end:])
}

func locateToolCallObject(text string) (start int, end int, toolCalls []textToolCall, ok bool) {
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}

		candidateEnd := findMatchingBrace(text, i)
		if candidateEnd <= i {
			continue
		}

		candidateToolCalls, matched := parseToolCallsObject(text[i:candidateEnd])
		if !matched {
			continue
		}

		return i, candidateEnd, candidateToolCalls, true
	}

	return 0, 0, nil, false
}

func parseToolCallsObject(jsonStr string) ([]textToolCall, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return nil, false
	}

	rawToolCalls, exists := obj["tool_calls"]
	if !exists {
		return nil, false
	}

	var toolCalls []textToolCall
	if err := json.Unmarshal(rawToolCalls, &toolCalls); err != nil {
		return nil, false
	}

	return toolCalls, true
}
