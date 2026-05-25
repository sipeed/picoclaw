package common

import "testing"

func TestNormalizeAnthropicBaseURL(t *testing.T) {
	const defaultURL = "https://api.anthropic.com"
	const defaultURLWithV1 = "https://api.anthropic.com/v1"

	tests := []struct {
		name           string
		apiBase        string
		defaultBase    string
		appendV1Suffix bool
		expected       string
	}{
		{"empty with v1", "", defaultURLWithV1, true, defaultURLWithV1},
		{"empty without v1", "", defaultURL, false, defaultURL},
		{
			"URL without v1 gets it appended",
			"https://api.example.com/anthropic", defaultURLWithV1,
			true, "https://api.example.com/anthropic/v1",
		},
		{
			"URL without v1 stays as-is",
			"https://api.example.com/anthropic", defaultURL,
			false, "https://api.example.com/anthropic",
		},
		{
			"URL with v1 remains unchanged when appending",
			"https://api.example.com/v1", defaultURLWithV1,
			true, "https://api.example.com/v1",
		},
		{
			"URL with v1 gets it stripped when not appending",
			"https://api.example.com/v1", defaultURL,
			false, "https://api.example.com",
		},
		{
			"trailing slash cleaned with v1",
			"https://api.example.com/anthropic/", defaultURLWithV1,
			true, "https://api.example.com/anthropic/v1",
		},
		{
			"trailing slash cleaned without v1",
			"https://api.example.com/anthropic/", defaultURL,
			false, "https://api.example.com/anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeBaseURL(tt.apiBase, tt.defaultBase, tt.appendV1Suffix)
			if got != tt.expected {
				t.Errorf("NormalizeAnthropicBaseURL(%q, %q, %v) = %q, want %q",
					tt.apiBase, tt.defaultBase, tt.appendV1Suffix, got, tt.expected)
			}
		})
	}
}

func TestModelOmitsTemperature(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{"opus 4-7 bare", "claude-opus-4-7", true},
		{"opus 4-7 dated", "claude-opus-4-7-20260601", true},
		{"opus 4-7 uppercase", "Claude-Opus-4-7", true},
		{"opus 4-7 with whitespace", "  claude-opus-4-7  ", true},
		{"gpt-5.5 bare", "gpt-5.5", true},
		{"gpt-5.5 dated", "gpt-5.5-20260601", true},
		{"gpt-5.5 uppercase", "GPT-5.5", true},
		{"opus 4-5 not affected", "claude-opus-4-5", false},
		{"sonnet 4-6 not affected", "claude-sonnet-4-6", false},
		{"haiku 4-5 not affected", "claude-haiku-4-5", false},
		{"gpt-5.4 not affected", "gpt-5.4", false},
		{"gpt-5.4-mini not affected", "gpt-5.4-mini", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ModelOmitsTemperature(tt.model); got != tt.want {
				t.Errorf("ModelOmitsTemperature(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
