package providers

import (
	"encoding/json"
	"strings"
)

func extractToolCallsFromText(text string) []ToolCall {
	for i := 0; i < len(text); i++ {
		if text[i] == '{' {
			end := findMatchingBrace(text, i)
			if end > i {
				jsonStr := text[i:end]
				// Quick check to avoid expensive parsing if it doesn't mention tool_calls
				if !strings.Contains(jsonStr, "tool_calls") {
					continue
				}

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

				if err := json.Unmarshal([]byte(jsonStr), &wrapper); err == nil && len(wrapper.ToolCalls) > 0 {
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
			}
		}
	}
	return nil
}

// stripToolCallsFromText removes tool call JSON from response text.
func stripToolCallsFromText(text string) string {
	for i := 0; i < len(text); i++ {
		if text[i] == '{' {
			end := findMatchingBrace(text, i)
			if end > i {
				jsonStr := text[i:end]
				if !strings.Contains(jsonStr, "tool_calls") {
					continue
				}

				var wrapper struct {
					ToolCalls interface{} `json:"tool_calls"`
				}

				if err := json.Unmarshal([]byte(jsonStr), &wrapper); err == nil && wrapper.ToolCalls != nil {
					return strings.TrimSpace(text[:i] + text[end:])
				}
			}
		}
	}
	return text
}

// findMatchingBrace finds the index after the closing brace matching the opening brace at pos.
// It accounts for braces inside strings and escaped characters.
func findMatchingBrace(text string, pos int) int {
	depth := 0
	inString := false
	escaped := false

	for i := pos; i < len(text); i++ {
		char := text[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				depth++
			} else if char == '}' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return pos
}
