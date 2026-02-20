package providers

import (
	"encoding/json"
	"fmt"
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

// extractXMLToolCalls parses XML tool call blocks (e.g. <ns:toolcall>)
// into structured ToolCall objects. Used as a fallback when the provider returns
// tool calls as XML in Content but not in the structured tool_calls field.
//
// Expected format:
//
//	<ns:toolcall>
//	<invoke name="tool_name">
//	<parameter name="param">value</parameter>
//	</invoke>
//	</ns:toolcall>
func extractXMLToolCalls(text string) []ToolCall {
	var result []ToolCall
	remaining := text
	callIdx := 0

	for {
		// Find next :toolcall> block
		idx := strings.Index(remaining, ":toolcall>")
		if idx == -1 {
			break
		}
		tagStart := strings.LastIndex(remaining[:idx], "<")
		if tagStart == -1 {
			break
		}
		ns := remaining[tagStart+1 : idx]
		closeTag := "</" + ns + ":toolcall>"
		closeIdx := strings.Index(remaining, closeTag)
		if closeIdx == -1 {
			break
		}

		block := remaining[idx+len(":toolcall>") : closeIdx]
		remaining = remaining[closeIdx+len(closeTag):]

		// Parse <invoke> elements within the block
		invokeRemaining := block
		for {
			invokeStart := strings.Index(invokeRemaining, "<invoke")
			if invokeStart == -1 {
				break
			}
			invokeEnd := strings.Index(invokeRemaining[invokeStart:], "</invoke>")
			if invokeEnd == -1 {
				break
			}
			invokeBody := invokeRemaining[invokeStart : invokeStart+invokeEnd+len("</invoke>")]
			invokeRemaining = invokeRemaining[invokeStart+invokeEnd+len("</invoke>"):]

			// Extract tool name from <invoke name="...">
			nameStart := strings.Index(invokeBody, `name="`)
			if nameStart == -1 {
				continue
			}
			nameStart += len(`name="`)
			nameEnd := strings.Index(invokeBody[nameStart:], `"`)
			if nameEnd == -1 {
				continue
			}
			toolName := invokeBody[nameStart : nameStart+nameEnd]

			// Extract parameters
			args := make(map[string]interface{})
			paramRemaining := invokeBody
			for {
				pStart := strings.Index(paramRemaining, "<parameter")
				if pStart == -1 {
					break
				}
				pNameStart := strings.Index(paramRemaining[pStart:], `name="`)
				if pNameStart == -1 {
					break
				}
				pNameStart += pStart + len(`name="`)
				pNameEnd := strings.Index(paramRemaining[pNameStart:], `"`)
				if pNameEnd == -1 {
					break
				}
				paramName := paramRemaining[pNameStart : pNameStart+pNameEnd]

				// Find closing > of the <parameter ...> tag
				tagClose := strings.Index(paramRemaining[pNameStart:], ">")
				if tagClose == -1 {
					break
				}
				valueStart := pNameStart + tagClose + 1
				valueEnd := strings.Index(paramRemaining[valueStart:], "</parameter>")
				if valueEnd == -1 {
					break
				}
				paramValue := paramRemaining[valueStart : valueStart+valueEnd]
				args[paramName] = paramValue
				paramRemaining = paramRemaining[valueStart+valueEnd+len("</parameter>"):]
			}

			// Build Arguments JSON string
			argsJSON, _ := json.Marshal(args)

			callIdx++
			result = append(result, ToolCall{
				ID:        fmt.Sprintf("xmltc_%d", callIdx),
				Type:      "function",
				Name:      toolName,
				Arguments: args,
				Function: &FunctionCall{
					Name:      toolName,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return result
}

// stripXMLToolCalls removes XML tool call blocks (e.g. <ns:toolcall>...</ns:toolcall>)
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
	// Extract namespace (e.g. "ns" from "<ns:toolcall>")
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
