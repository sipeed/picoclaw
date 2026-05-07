package audio

import (
	"strings"
	"unicode"
)

// SplitSentences splits text into sentence-sized chunks suitable for TTS synthesis.
// It splits on sentence-ending punctuation (.!?\n, as well as CJK 。, ！, ？) while avoiding false splits
// on decimal numbers. Very short fragments are merged with
// the next sentence to prevent choppy playback.
func SplitSentences(text string) []string {
	if text == "" {
		return nil
	}

	var sentences []string
	var current strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\n' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
			continue
		}

		current.WriteRune(r)

		if r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？' {
			// Avoid splitting on decimal numbers like "3.14"
			if r == '.' && i > 0 && unicode.IsDigit(runes[i-1]) &&
				i+1 < len(runes) && unicode.IsDigit(runes[i+1]) {
				continue
			}

			// Consume contiguous punctuation clusters (e.g., "..." or "?!").
			for i+1 < len(runes) && (runes[i+1] == '.' || runes[i+1] == '!' || runes[i+1] == '?' || runes[i+1] == '。' || runes[i+1] == '！' || runes[i+1] == '？') {
				i++
				current.WriteRune(runes[i])
			}

			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	// Flush remaining text
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}

	// Merge very short fragments with the next sentence
	return mergeShorties(sentences, 15)
}

// mergeShorties merges sentences shorter than minLen characters with the following sentence.
func mergeShorties(sentences []string, minLen int) []string {
	if len(sentences) <= 1 {
		return sentences
	}

	var merged []string
	var buf string

	for _, s := range sentences {
		if buf != "" {
			buf += " " + s
			if len([]rune(buf)) >= minLen {
				merged = append(merged, buf)
				buf = ""
			}
		} else if len([]rune(s)) < minLen {
			buf = s
		} else {
			merged = append(merged, s)
		}
	}

	if buf != "" {
		if len(merged) > 0 {
			merged[len(merged)-1] += " " + buf
		} else {
			merged = append(merged, buf)
		}
	}

	return merged
}
