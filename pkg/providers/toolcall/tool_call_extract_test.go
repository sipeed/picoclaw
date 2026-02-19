package toolcall

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// --- FindMatchingBrace tests ---

func TestFindMatchingBrace(t *testing.T) {
	tests := []struct {
		text string
		pos  int
		want int
	}{
		{`{"a":1}`, 0, 7},
		{`{"a":{"b":2}}`, 0, 13},
		{`text {"a":1} more`, 5, 12},
		{`{unclosed`, 0, 0},      // no match returns pos
		{`{}`, 0, 2},             // empty object
		{`{{{}}}`, 0, 6},         // deeply nested
		{`{"a":"b{c}d"}`, 0, 13}, // braces in strings (simplified matcher)
	}
	for _, tt := range tests {
		got := FindMatchingBrace(tt.text, tt.pos)
		if got != tt.want {
			t.Errorf("FindMatchingBrace(%q, %d) = %d, want %d", tt.text, tt.pos, got, tt.want)
		}
	}
}

// --- StripJSONObject tests ---

func TestStripJSONObject(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		pattern string
		want    string
	}{
		{
			name:    "removes matching object",
			text:    `before {"tool_calls":[]} after`,
			pattern: `{"tool_calls"`,
			want:    `before after`,
		},
		{
			name:    "pattern not found returns original",
			text:    "no match here",
			pattern: `{"tool_calls"`,
			want:    "no match here",
		},
		{
			name:    "unmatched brace returns original",
			text:    `{"tool_calls"`,
			pattern: `{"tool_calls"`,
			want:    `{"tool_calls"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripJSONObject(tt.text, tt.pattern)
			if got != tt.want {
				t.Errorf("StripJSONObject() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- StripToolCallsFromText tests ---

func TestStripToolCallsFromText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "removes tool_calls JSON",
			text: `Let me check.` + "\n" + `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]}`,
			want: "Let me check.",
		},
		{
			name: "no tool_calls returns original",
			text: "Just regular text.",
			want: "Just regular text.",
		},
		{
			name: "only tool_calls returns empty",
			text: `{"tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]}`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripToolCallsFromText(tt.text)
			if got != tt.want {
				t.Errorf("StripToolCallsFromText() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- ParseToolCallArguments tests ---

func TestParseToolCallArguments(t *testing.T) {
	tests := []struct {
		name    string
		argsStr string
		want    map[string]interface{}
	}{
		{
			name:    "empty string returns empty map",
			argsStr: "",
			want:    make(map[string]interface{}),
		},
		{
			name:    "valid JSON object",
			argsStr: `{"location":"Tokyo","unit":"celsius"}`,
			want: map[string]interface{}{
				"location": "Tokyo",
				"unit":     "celsius",
			},
		},
		{
			name:    "valid JSON with nested objects",
			argsStr: `{"query":{"text":"hello","lang":"en"}}`,
			want: map[string]interface{}{
				"query": map[string]interface{}{
					"text": "hello",
					"lang": "en",
				},
			},
		},
		{
			name:    "valid JSON with arrays",
			argsStr: `{"items":["a","b","c"],"count":3}`,
			want: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
				"count": float64(3),
			},
		},
		{
			name:    "valid JSON with numbers",
			argsStr: `{"temperature":72.5,"humidity":60}`,
			want: map[string]interface{}{
				"temperature": 72.5,
				"humidity":    float64(60),
			},
		},
		{
			name:    "valid JSON with booleans",
			argsStr: `{"enabled":true,"active":false}`,
			want: map[string]interface{}{
				"enabled": true,
				"active":  false,
			},
		},
		{
			name:    "invalid JSON returns raw string in map",
			argsStr: `{invalid json}`,
			want: map[string]interface{}{
				"raw": `{invalid json}`,
			},
		},
		{
			name:    "malformed JSON returns raw string",
			argsStr: `{"key":value}`,
			want: map[string]interface{}{
				"raw": `{"key":value}`,
			},
		},
		{
			name:    "empty JSON object",
			argsStr: `{}`,
			want:    make(map[string]interface{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolCallArguments(tt.argsStr)
			if len(got) != len(tt.want) {
				t.Errorf("ParseToolCallArguments() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("ParseToolCallArguments() missing key %q", key)
					continue
				}
				if key == "raw" {
					// For raw error cases, just check it exists
					if _, ok := gotVal.(string); !ok {
						t.Errorf("ParseToolCallArguments() raw value is not string")
					}
				} else {
					// For other cases, do deep comparison
					// Handle nested maps first to avoid panic from direct comparison
					if gotMap, ok := gotVal.(map[string]interface{}); ok {
						if wantMap, ok := wantVal.(map[string]interface{}); ok {
							if len(gotMap) != len(wantMap) {
								t.Errorf("ParseToolCallArguments()[%q] nested map length = %d, want %d", key, len(gotMap), len(wantMap))
							} else {
								// Recursively check nested map values
								for nestedKey, nestedWantVal := range wantMap {
									nestedGotVal, ok := gotMap[nestedKey]
									if !ok {
										t.Errorf("ParseToolCallArguments()[%q][%q] missing key", key, nestedKey)
										continue
									}
									if nestedGotVal != nestedWantVal {
										t.Errorf("ParseToolCallArguments()[%q][%q] = %v, want %v", key, nestedKey, nestedGotVal, nestedWantVal)
									}
								}
							}
							continue
						}
					}
					// Handle arrays/slices to avoid panic from direct comparison
					if gotSlice, ok := gotVal.([]interface{}); ok {
						if wantSlice, ok := wantVal.([]interface{}); ok {
							if len(gotSlice) != len(wantSlice) {
								t.Errorf("ParseToolCallArguments()[%q] array length = %d, want %d", key, len(gotSlice), len(wantSlice))
							} else {
								// Compare array elements
								for i := range gotSlice {
									if gotSlice[i] != wantSlice[i] {
										t.Errorf("ParseToolCallArguments()[%q][%d] = %v, want %v", key, i, gotSlice[i], wantSlice[i])
									}
								}
							}
							continue
						}
					}
					// For other values, do direct comparison
					if gotVal != wantVal {
						t.Errorf("ParseToolCallArguments()[%q] = %v, want %v", key, gotVal, wantVal)
					}
				}
			}
		})
	}
}

// --- ExtractToolCallsFromText tests ---

func TestExtractToolCallsFromText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		want     []protocoltypes.ToolCall
		wantLen  int
		checkIDs bool
	}{
		{
			name:    "no tool_calls returns nil",
			text:    "Just regular text without tool calls",
			want:    nil,
			wantLen: 0,
		},
		{
			name:     "text with single tool call",
			text:     `Let me check.` + "\n" + `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Tokyo\"}"}}]}`,
			wantLen:  1,
			checkIDs: true,
		},
		{
			name:     "text with multiple tool calls",
			text:     `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"/tmp/test.txt\"}"}},{"id":"call_2","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"/tmp/out.txt\",\"content\":\"hello\"}"}}]}`,
			wantLen:  2,
			checkIDs: true,
		},
		{
			name:     "tool_calls with empty arguments",
			text:     `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"no_args","arguments":"{}"}}]}`,
			wantLen:  1,
			checkIDs: true,
		},
		{
			name:     "tool_calls with nested JSON arguments",
			text:     `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"complex","arguments":"{\"query\":{\"text\":\"hello\",\"lang\":\"en\"},\"options\":{\"case\":\"lower\"}}"}}]}`,
			wantLen:  1,
			checkIDs: true,
		},
		{
			name:    "malformed JSON returns nil",
			text:    `{"tool_calls":[invalid json]}`,
			want:    nil,
			wantLen: 0,
		},
		{
			name:    "unclosed brace returns nil",
			text:    `{"tool_calls":[{"id":"call_1"`,
			want:    nil,
			wantLen: 0,
		},
		{
			name:    "empty tool_calls array",
			text:    `{"tool_calls":[]}`,
			want:    nil,
			wantLen: 0,
		},
		{
			name:     "text before and after tool_calls",
			text:     `Before text.` + "\n" + `{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"test","arguments":"{}"}}]}` + "\n" + `After text.`,
			wantLen:  1,
			checkIDs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractToolCallsFromText(tt.text)
			if len(got) != tt.wantLen {
				t.Errorf("ExtractToolCallsFromText() length = %d, want %d", len(got), tt.wantLen)
				return
			}
			if tt.checkIDs && len(got) > 0 {
				// Verify structure of first tool call
				tc := got[0]
				if tc.ID == "" {
					t.Error("ExtractToolCallsFromText() tool call ID is empty")
				}
				if tc.Name == "" {
					t.Error("ExtractToolCallsFromText() tool call Name is empty")
				}
				if tc.Function == nil {
					t.Error("ExtractToolCallsFromText() tool call Function is nil")
				} else {
					if tc.Function.Name == "" {
						t.Error("ExtractToolCallsFromText() Function.Name is empty")
					}
					if tc.Function.Arguments == "" {
						t.Error("ExtractToolCallsFromText() Function.Arguments is empty")
					}
				}
				if tc.Arguments == nil {
					t.Error("ExtractToolCallsFromText() Arguments map is nil")
				}
			}
		})
	}
}

func TestExtractToolCallsFromText_Detailed(t *testing.T) {
	text := `Let me check the weather.` + "\n" + `{"tool_calls":[{"id":"call_123","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Tokyo\",\"unit\":\"celsius\"}"}}]}`

	toolCalls := ExtractToolCallsFromText(text)
	if len(toolCalls) != 1 {
		t.Fatalf("ExtractToolCallsFromText() length = %d, want 1", len(toolCalls))
	}

	tc := toolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_123")
	}
	if tc.Type != "function" {
		t.Errorf("ToolCall.Type = %q, want %q", tc.Type, "function")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.Function == nil {
		t.Fatal("ToolCall.Function is nil")
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("Function.Name = %q, want %q", tc.Function.Name, "get_weather")
	}
	if tc.Function.Arguments != `{"location":"Tokyo","unit":"celsius"}` {
		t.Errorf("Function.Arguments = %q, want %q", tc.Function.Arguments, `{"location":"Tokyo","unit":"celsius"}`)
	}
	if tc.Arguments["location"] != "Tokyo" {
		t.Errorf("Arguments[location] = %v, want Tokyo", tc.Arguments["location"])
	}
	if tc.Arguments["unit"] != "celsius" {
		t.Errorf("Arguments[unit] = %v, want celsius", tc.Arguments["unit"])
	}
}

// --- ParseStructuredToolCalls tests ---

func TestParseStructuredToolCalls(t *testing.T) {
	tests := []struct {
		name         string
		apiToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function *struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		}
		wantLen int
	}{
		{
			name: "empty array returns nil",
			apiToolCalls: []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function *struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}{},
			wantLen: 0,
		},
		{
			name: "single tool call",
			apiToolCalls: []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function *struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}{
				{
					ID:   "call_1",
					Type: "function",
					Function: &struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "get_weather",
						Arguments: `{"location":"NYC"}`,
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "multiple tool calls",
			apiToolCalls: []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function *struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}{
				{
					ID:   "call_1",
					Type: "function",
					Function: &struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "read_file",
						Arguments: `{"path":"/tmp/a.txt"}`,
					},
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: &struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "write_file",
						Arguments: `{"path":"/tmp/b.txt","content":"hello"}`,
					},
				},
			},
			wantLen: 2,
		},
		{
			name: "nil function is skipped",
			apiToolCalls: []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function *struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}{
				{
					ID:       "call_1",
					Type:     "function",
					Function: nil,
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: &struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "valid_call",
						Arguments: `{}`,
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "empty arguments string",
			apiToolCalls: []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function *struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}{
				{
					ID:   "call_1",
					Type: "function",
					Function: &struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "no_args",
						Arguments: "",
					},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStructuredToolCalls(tt.apiToolCalls)
			if tt.wantLen == 0 {
				if len(got) > 0 {
					t.Errorf("ParseStructuredToolCalls() = %v, want nil or empty", got)
				}
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("ParseStructuredToolCalls() length = %d, want %d", len(got), tt.wantLen)
				return
			}
			// Verify structure
			for i, tc := range got {
				if tc.ID == "" {
					t.Errorf("ParseStructuredToolCalls()[%d].ID is empty", i)
				}
				if tc.Name == "" {
					t.Errorf("ParseStructuredToolCalls()[%d].Name is empty", i)
				}
				if tc.Function == nil {
					t.Errorf("ParseStructuredToolCalls()[%d].Function is nil", i)
				}
				if tc.Arguments == nil {
					t.Errorf("ParseStructuredToolCalls()[%d].Arguments is nil", i)
				}
			}
		})
	}
}

func TestParseStructuredToolCalls_Detailed(t *testing.T) {
	apiToolCalls := []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function *struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}{
		{
			ID:   "call_abc",
			Type: "function",
			Function: &struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "get_weather",
				Arguments: `{"city":"SF","unit":"fahrenheit"}`,
			},
		},
	}

	toolCalls := ParseStructuredToolCalls(apiToolCalls)
	if len(toolCalls) != 1 {
		t.Fatalf("ParseStructuredToolCalls() length = %d, want 1", len(toolCalls))
	}

	tc := toolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_abc")
	}
	if tc.Type != "function" {
		t.Errorf("ToolCall.Type = %q, want %q", tc.Type, "function")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.Function == nil {
		t.Fatal("ToolCall.Function is nil")
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("Function.Name = %q, want %q", tc.Function.Name, "get_weather")
	}
	if tc.Function.Arguments != `{"city":"SF","unit":"fahrenheit"}` {
		t.Errorf("Function.Arguments = %q, want %q", tc.Function.Arguments, `{"city":"SF","unit":"fahrenheit"}`)
	}
	if tc.Arguments["city"] != "SF" {
		t.Errorf("Arguments[city] = %v, want SF", tc.Arguments["city"])
	}
	if tc.Arguments["unit"] != "fahrenheit" {
		t.Errorf("Arguments[unit] = %v, want fahrenheit", tc.Arguments["unit"])
	}
}
