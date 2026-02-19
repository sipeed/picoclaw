package providers

import (
	"reflect"
	"testing"
)

func TestExtractToolCallsFromText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		want     []ToolCall
	}{
		{
			name: "Basic tool call",
			text: `Here is the tool call: {"tool_calls":[{"id":"call_1","type":"function","function":{"name":"search","arguments":"{\"query\":\"hello\"}"}}]} and some more text.`,
			want: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Name: "search",
					Arguments: map[string]interface{}{
						"query": "hello",
					},
					Function: &FunctionCall{
						Name:      "search",
						Arguments: `{"query":"hello"}`,
					},
				},
			},
		},
		{
			name: "Brace in string",
			text: `Tool call with brace in string: {"tool_calls":[{"id":"call_2","type":"function","function":{"name":"msg","arguments":"{\"text\":\"Hello { world }\"}"}}]} post-text.`,
			want: []ToolCall{
				{
					ID:   "call_2",
					Type: "function",
					Name: "msg",
					Arguments: map[string]interface{}{
						"text": "Hello { world }",
					},
					Function: &FunctionCall{
						Name:      "msg",
						Arguments: `{"text":"Hello { world }"}`,
					},
				},
			},
		},
		{
			name: "Escaped quote and brace in arguments",
			text: `Complex: {"tool_calls":[{"id":"call_3","type":"function","function":{"name":"exec","arguments":"{\"cmd\":\"echo \\\"}\\\"\"}"}}]}`,
			want: []ToolCall{
				{
					ID:   "call_3",
					Type: "function",
					Name: "exec",
					Arguments: map[string]interface{}{
						"cmd": `echo "}"`,
					},
					Function: &FunctionCall{
						Name:      "exec",
						Arguments: `{"cmd":"echo \"}\""}`,
					},
				},
			},
		},
		{
			name: "No tool calls",
			text: "Just some normal text here.",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolCallsFromText(tt.text)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractToolCallsFromText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripToolCallsFromText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "Basic strip",
			text: "Prefix text. {\"tool_calls\":[]} Suffix text.",
			want: "Prefix text.  Suffix text.",
		},
		{
			name: "No tool calls to strip",
			text: "Normal text.",
			want: "Normal text.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripToolCallsFromText(tt.text); got != tt.want {
				t.Errorf("stripToolCallsFromText() = %v, want %v", got, tt.want)
			}
		})
	}
}
