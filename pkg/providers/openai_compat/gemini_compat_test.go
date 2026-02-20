package openai_compat

import (
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestToolCall_Gemini3RoundTrip(t *testing.T) {
	// Simulate a response from Gemini 3.x OpenAI-compat endpoint
	responseJSON := `{
		"id": "call_gemini_123",
		"type": "function",
		"function": {
			"name": "read_file",
			"arguments": "{\"path\":\"README.md\"}"
		},
		"extra_content": {
			"google": {
				"thought_signature": "gemini_thought_sig_abc"
			}
		}
	}`

	type APIResponse struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function *struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
		ExtraContent *struct {
			Google *struct {
				ThoughtSignature string `json:"thought_signature"`
			} `json:"google"`
		} `json:"extra_content"`
	}

	var apiResponse APIResponse
	if err := json.Unmarshal([]byte(responseJSON), &apiResponse); err != nil {
		t.Fatalf("failed to unmarshal mock Gemini response: %v", err)
	}


	// 1. Verify parseResponse logic (manually simulated here based on provider.go)
	thoughtSignature := ""
	if apiResponse.ExtraContent != nil && apiResponse.ExtraContent.Google != nil {
		thoughtSignature = apiResponse.ExtraContent.Google.ThoughtSignature
	}

	toolCall := protocoltypes.ToolCall{
		ID:               apiResponse.ID,
		Type:             apiResponse.Type,
		Name:             apiResponse.Function.Name,
		Arguments:        map[string]any{"path": "README.md"},
		ThoughtSignature: thoughtSignature,
		Function: &protocoltypes.FunctionCall{
			Name:             apiResponse.Function.Name,
			Arguments:        apiResponse.Function.Arguments,
			ThoughtSignature: thoughtSignature,
		},
	}

	if thoughtSignature != "" {
		toolCall.ExtraContent = &protocoltypes.ExtraContent{
			Google: &protocoltypes.GoogleExtra{
				ThoughtSignature: thoughtSignature,
			},
		}
	}

	// 2. Verify Serialization Round-trip
	// In subsequent turns, the agent sends back the assistant message with tool calls.
	assistantMessage := protocoltypes.Message{
		Role:      "assistant",
		ToolCalls: []protocoltypes.ToolCall{toolCall},
	}

	marshaled, err := json.Marshal(assistantMessage)
	if err != nil {
		t.Fatalf("failed to marshal assistant message: %v", err)
	}

	// The marshaled JSON must contain:
	// - "function": {"name": "...", "arguments": "...", "thought_signature": "..."}
	// - "extra_content": {"google": {"thought_signature": "..."}}
	
	var result map[string]any
	if err := json.Unmarshal(marshaled, &result); err != nil {
		t.Fatalf("failed to unmarshal marshaled assistant message: %v", err)
	}

	toolCalls := result["tool_calls"].([]any)
	if len(toolCalls) == 0 {
		t.Fatal("no tool calls found in marshaled message")
	}

	tc0 := toolCalls[0].(map[string]any)

	// Check extra_content (Used by some versions of Gemini 3.x)
	extra, ok := tc0["extra_content"].(map[string]any)
	if !ok {
		t.Error("extra_content missing from serialized tool call")
	} else {
		google := extra["google"].(map[string]any)
		if google["thought_signature"] != "gemini_thought_sig_abc" {
			t.Errorf("extra_content.google.thought_signature = %v, want gemini_thought_sig_abc", google["thought_signature"])
		}
	}

	// Check function.thought_signature (Used by other versions of Gemini 3.x)
	function, ok := tc0["function"].(map[string]any)
	if !ok {
		t.Error("function field missing from serialized tool call")
	} else {
		if function["thought_signature"] != "gemini_thought_sig_abc" {
			t.Errorf("function.thought_signature = %v, want gemini_thought_sig_abc", function["thought_signature"])
		}
	}
}
