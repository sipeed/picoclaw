package telegram

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_markdownToTelegramHTML(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "bold",
			input:    "**bold text**",
			expected: "<b>bold text</b>",
		},
		{
			name:     "italic",
			input:    "_italic text_",
			expected: "<i>italic text</i>",
		},
		{
			name:     "link without underscores in URL",
			input:    "[click here](https://example.com/path)",
			expected: `<a href="https://example.com/path">click here</a>`,
		},
		{
			name: "link with underscores in URL is not corrupted by italic regex",
			// Google Flights URLs use URL-safe base64 with underscores in the tfs param.
			// Previously reItalic ran after reLink, matching _text_ inside href and injecting
			// <i> tags into the URL, which broke the link in Telegram.
			input:    "[3 → 10 сентября — от $202](https://www.google.com/travel/flights/search?tfs=CBwQAho_EgoyURL_safe_base64)",
			expected: `<a href="https://www.google.com/travel/flights/search?tfs=CBwQAho_EgoyURL_safe_base64">3 → 10 сентября — от $202</a>`,
		},
		{
			name:     "multiple links all survive",
			input:    "[first](https://a.com/path_one) and [second](https://b.com/path_two_x)",
			expected: `<a href="https://a.com/path_one">first</a> and <a href="https://b.com/path_two_x">second</a>`,
		},
		{
			name:     "link label with HTML special chars is escaped",
			input:    "[a & b](https://example.com)",
			expected: `<a href="https://example.com">a &amp; b</a>`,
		},
		{
			name:     "HTML special chars in plain text are escaped",
			input:    "a & b < c > d",
			expected: "a &amp; b &lt; c &gt; d",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := markdownToTelegramHTML(tc.input)
			require.Equal(t, tc.expected, actual)
		})
	}
}
