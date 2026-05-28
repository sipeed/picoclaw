package commands

import "testing"

func TestHasCommandPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/help", true},
		{"!help", true},
		{"/switch model to gpt-4", true},
		{"!switch model to gpt-4", true},
		{"hello", false},
		{"", false},
		{"   ", false},
		{"hello /world", false},
		{"/", true},
		{"!", true},
		{"  /help", true},
	}
	for _, tt := range tests {
		got := HasCommandPrefix(tt.input)
		if got != tt.want {
			t.Errorf("HasCommandPrefix(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
