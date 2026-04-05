// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"regexp"
	"strings"
)

// MessageSplitMarker is the delimiter used to split a message into multiple outbound messages.
// When SplitOnMarker is enabled in config, the Manager will split messages on this marker
// and send each part as a separate message.
const MessageSplitMarker = "<|[SPLIT]|>"

// splitMarkerRe matches the split marker with optional spaces (LLMs sometimes add them).
// Matches: <|[SPLIT]|>  <| [SPLIT] |>  <|[ SPLIT ]|>  etc.
var splitMarkerRe = regexp.MustCompile(`<\|\s*\[\s*SPLIT\s*\]\s*\|>`)

// SplitByMarker splits a message by the MessageSplitMarker and returns the parts.
// Empty parts (including from consecutive markers) are filtered out.
// If no marker is found, returns a single-element slice containing the original content.
// Tolerates whitespace variations in the marker that LLMs may produce.
func SplitByMarker(content string) []string {
	if content == "" {
		return nil
	}
	// Normalize any whitespace variations to the canonical marker
	normalized := splitMarkerRe.ReplaceAllString(content, MessageSplitMarker)
	parts := strings.Split(normalized, MessageSplitMarker)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return []string{content}
	}
	return result
}
