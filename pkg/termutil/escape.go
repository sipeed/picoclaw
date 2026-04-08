package termutil

import (
	"fmt"
	"strings"
	"unicode"
)

// EscapeControlChars preserves readable text while escaping control and format
// characters that could alter terminal state, such as ANSI escape codes and
// bidi overrides.
func EscapeControlChars(input string) string {
	var sb strings.Builder
	sb.Grow(len(input))

	for _, r := range input {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			sb.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			sb.WriteString(fmt.Sprintf("\\x%02x", r))
		case unicode.Is(unicode.Cf, r):
			if r <= 0xffff {
				sb.WriteString(fmt.Sprintf("\\u%04x", r))
			} else {
				sb.WriteString(fmt.Sprintf("\\U%08x", r))
			}
		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()
}
