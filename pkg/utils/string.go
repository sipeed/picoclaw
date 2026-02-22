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
