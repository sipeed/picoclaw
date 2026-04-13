package termutil

import "testing"

func TestEscapeControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preserves printable text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "preserves whitespace controls",
			input:    "a\tb\nc\rd",
			expected: "a\tb\nc\rd",
		},
		{
			name:     "escapes C0 controls",
			input:    "\x1b[31mred",
			expected: `\x1b[31mred`,
		},
		{
			name:     "escapes DEL",
			input:    "a\x7fb",
			expected: `a\x7fb`,
		},
		{
			name:     "escapes C1 controls",
			input:    "a\u009bb",
			expected: `a\x9bb`,
		},
		{
			name:     "escapes bidi override",
			input:    "safe\u202edanger",
			expected: `safe\u202edanger`,
		},
		{
			name:     "escapes zero width chars",
			input:    "a\u200bb",
			expected: `a\u200bb`,
		},
		{
			name:     "escapes astral format chars",
			input:    "a\U000e0001b",
			expected: `a\U000e0001b`,
		},
		{
			name:     "escapes BMP private use chars",
			input:    "a\ue000b",
			expected: `a\ue000b`,
		},
		{
			name:     "escapes supplementary private use chars",
			input:    "a\U000f0000b",
			expected: `a\U000f0000b`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeControlChars(tt.input); got != tt.expected {
				t.Fatalf("EscapeControlChars() = %q, want %q", got, tt.expected)
			}
		})
	}
}
