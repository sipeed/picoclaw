package providers

import (
	"testing"
)

func TestExtractToolCallsFromText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		want      int // number of tool calls expected
		wantNames []string
	}{
		{
			name: "Single tool call",
			text: `Some thinking here.
{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.txt\"}"}}]}
More text.`,
			want:      1,
			wantNames: []string{"read_file"},
		},
		{
			name: "Multiple tool call blocks",
			text: `First call:
{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.txt\"}"}}]}
Second call:
{"tool_calls":[{"id":"call_2","type":"function","function":{"name":"ls","arguments":"{}"}}]}`,
			want:      2,
			wantNames: []string{"read_file", "ls"},
		},
		{
			name: "Multiple calls in one block",
			text: `{"tool_calls":[
{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"1.txt\"}"}},
{"id":"call_2","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"2.txt\"}"}}
]}`,
			want:      2,
			wantNames: []string{"read_file", "read_file"},
		},
		{
			name: "Broken JSON block and a good one",
			text: `{"tool_calls": [ ... broken ...
{"tool_calls":[{"id":"call_3","type":"function","function":{"name":"ls","arguments":"{}"}}]}`,
			want:      1,
			wantNames: []string{"ls"},
		},
		{
			name: "Braces in arguments",
			text: `{"tool_calls":[{"id":"call_4","type":"function","function":{"name":"grep","arguments":"{\"pattern\":\"{[0-9]+}\"}"}}]}`,
			want:      1,
			wantNames: []string{"grep"},
		},
		{
			name: "JSON in markdown block",
			text: "```json\n" + `{"tool_calls":[{"id":"call_5","type":"function","function":{"name":"ls","arguments":"{}"}}]}` + "\n```",
			want:      1,
			wantNames: []string{"ls"},
		},
		{
			name: "JSON with whitespace",
			text: `{  "tool_calls":  [{"id":"call_6","type":"function","function":{"name":"pwd","arguments":"{}"}}]}`,
			want:      1,
			wantNames: []string{"pwd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolCallsFromText(tt.text)
			if len(got) != tt.want {
				t.Errorf("extractToolCallsFromText() got %v calls, want %v", len(got), tt.want)
			}
			for i, name := range tt.wantNames {
				if i < len(got) && got[i].Name != name {
					t.Errorf("call [%d] name = %v, want %v", i, got[i].Name, name)
				}
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
			name: "Strip single block",
			text: "Intro\n{\"tool_calls\":[]}\nOutro",
			want: "Intro\n\nOutro",
		},
		{
			name: "Strip multiple blocks",
			text: "A\n{\"tool_calls\":[]}\nB\n{\"tool_calls\":[]}\nC",
			want: "A\n\nB\n\nC",
		},
		{
			name: "No tool calls",
			text: "Just plain text.",
			want: "Just plain text.",
		},
		{
			name: "Strip markdown block",
			text: "Intro\n```json\n{\"tool_calls\":[]}\n```\nOutro",
			want: "Intro\n\nOutro",
		},
		{
			name: "Strip with whitespace marker",
			text: "A {  \"tool_calls\":[] } B",
			want: "A\n\nB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripToolCallsFromText(tt.text)
			if got != tt.want {
				t.Errorf("stripToolCallsFromText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindMatchingBraceRobust(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		pos     int
		wantEnd int
	}{
		{"Simple", `{"a": 1}`, 0, 8},
		{"Nested", `{"a": {"b": 2}}`, 0, 15},
		{"InString", `{"a": "}"}`, 0, 10},
		{"Escaped", `{"a": "\""}`, 0, 11},
		{"MultipleEscapes", `{"a": "\\\""}`, 0, 13},
		{"BareBackslashOutsideString", `\ {"a": 1}`, 2, 10},
		{"BracesInStringValue", `{"a":"b{c}d"}`, 0, 13},
		{"NotStarted", `abc`, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMatchingBrace(tt.text, tt.pos)
			if got != tt.wantEnd {
				t.Errorf("findMatchingBrace(%q, %d) = %d, want %d", tt.text, tt.pos, got, tt.wantEnd)
			}
		})
	}
}
