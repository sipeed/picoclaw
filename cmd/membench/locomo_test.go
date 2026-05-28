package main

import (
	"encoding/json"
	"testing"
)

func TestAnswerString(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			"string answer",
			`{"question":"Q","answer":"Paris","evidence":[],"category":1}`,
			"Paris",
		},
		{
			"int answer",
			`{"question":"Q","answer":42,"evidence":[],"category":1}`,
			"42",
		},
		{
			"adversarial answer (category 5)",
			`{"question":"Q","evidence":[],"category":5,"adversarial_answer":"self-care is important"}`,
			"self-care is important",
		},
		{
			"both answer and adversarial_answer present",
			`{"question":"Q","answer":"normal","evidence":[],"category":5,"adversarial_answer":"adversarial"}`,
			"normal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var qa LocomoQA
			if err := json.Unmarshal([]byte(tt.json), &qa); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got := qa.AnswerString()
			if got != tt.want {
				t.Errorf("AnswerString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSessionNames(t *testing.T) {
	conv := map[string]json.RawMessage{
		"session_2":           {},
		"session_1":           {},
		"session_10":          {},
		"session_1_date_time": {},
		"speaker_a":           {},
	}
	names := GetSessionNames(conv)
	want := []string{"session_1", "session_2", "session_10"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, n, want[i])
		}
	}
}
