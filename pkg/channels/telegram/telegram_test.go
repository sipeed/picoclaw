package telegram

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_markdownToTelegramMarkdownV2(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    `## HeadingH2 #`,
			expected: "*HeadingH2 \\#*",
		},
		{
			input:    "~strikethroughMD~",
			expected: "~strikethroughMD~",
		},
		{
			input:    "[inline URL](http://www.example.com/)",
			expected: "[inline URL](http://www.example.com/)",
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("formating %s -> %s", tc.input, tc.expected), func(t *testing.T) {
			actual := markdownToTelegramMarkdownV2(tc.input)

			require.EqualValues(t, tc.expected, actual)
		})
	}
}
