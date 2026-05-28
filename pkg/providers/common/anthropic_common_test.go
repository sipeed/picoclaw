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
