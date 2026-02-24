package utils

import (
	"regexp"
	"strings"
)

// Repetition detection constants.
const (
	repetitionSampleSize      = 2000 // runes to sample from the tail
	repetitionNgramSize       = 10   // sliding window length
	repetitionUniqueThreshold = 0.1  // unique ratio below this → repetition
)

var (
	thinkBlockClosedRe = regexp.MustCompile(`(?is)<think>.*?</think>`)
	thinkBlockOpenRe   = regexp.MustCompile(`(?is)<think>.*$`)
)

// StripThinkBlocks removes <think>…</think> blocks (including unclosed ones)
// from s and returns the trimmed result.
func StripThinkBlocks(s string) string {
	s = thinkBlockClosedRe.ReplaceAllString(s, "")
	s = thinkBlockOpenRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// TailPad returns a fixed-height block of n visual lines built from the
// tail of s.  Long lines are wrapped at wrapWidth runes so the result
// never exceeds the chat bubble width.  If fewer than n visual lines
// exist, Braille-blank lines (\u2800) are prepended as padding.
func TailPad(s string, n, wrapWidth int) string {
	// Wrap each raw line into visual lines respecting wrapWidth.
	var visual []string
	for _, raw := range strings.Split(s, "\n") {
		visual = append(visual, wrapLine(raw, wrapWidth)...)
	}
	if len(visual) > n {
		visual = visual[len(visual)-n:]
	}
	for len(visual) < n {
		visual = append([]string{"\u2800"}, visual...)
	}
	return strings.Join(visual, "\n")
}

// wrapLine splits a single line into segments of at most width runes.
// An empty line produces one empty string (preserving blank lines).
func wrapLine(line string, width int) []string {
	// ASCII fast path: byte length == rune length for pure ASCII
	if len(line) <= width {
		return []string{line}
	}
	runes := []rune(line)
	if len(runes) <= width {
		return []string{line}
	}
	var segs []string
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		segs = append(segs, string(runes[:end]))
		runes = runes[end:]
	}
	return segs
}

// DetectRepetitionLoop checks if text contains degenerate repetition
// by computing the unique N-gram ratio on the last repetitionSampleSize runes.
// Returns true if the ratio of unique N-grams to total N-grams
// falls below repetitionUniqueThreshold (i.e., 90%+ are duplicates).
func DetectRepetitionLoop(text string) bool {
	runes := []rune(text)

	// Sample the tail
	if len(runes) > repetitionSampleSize {
		runes = runes[len(runes)-repetitionSampleSize:]
	}

	total := len(runes) - repetitionNgramSize + 1
	if total <= 0 {
		return false
	}

	unique := make(map[string]struct{}, total/repetitionNgramSize)
	for i := 0; i < total; i++ {
		ng := string(runes[i : i+repetitionNgramSize])
		unique[ng] = struct{}{}
	}

	ratio := float64(len(unique)) / float64(total)
	return ratio < repetitionUniqueThreshold
}

// Truncate returns a truncated version of s with at most maxLen runes.
// Handles multi-byte Unicode characters properly.
// If the string is truncated, "..." is appended to indicate truncation.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	// ASCII fast path: byte length == rune length for pure ASCII
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	// Reserve 3 chars for "..."
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// DerefStr dereferences a pointer to a string and
// returns the value or a fallback if the pointer is nil.
func DerefStr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
