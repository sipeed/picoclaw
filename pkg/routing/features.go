package routing

import (
	"strings"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// lookbackWindow is the number of recent history entries scanned for tool calls.
// Six entries covers roughly one full tool-use round-trip (user → assistant+tool_call → tool_result → assistant).
const lookbackWindow = 6

// Features holds the structural signals extracted from a message and its session context.
// Every dimension is language-agnostic by construction — no keyword or pattern matching
// against natural-language content. This ensures consistent routing for all locales.
type Features struct {
	// TokenEstimate is a conservative proxy for token count.
	// Computed as utf8.RuneCountInString(msg) / 3, which handles CJK characters
	// (each rune ≈ 1 token for CJK, ≈ 0.25 tokens for ASCII) without any API call.
	TokenEstimate int

	// CodeBlockCount is the number of fenced code blocks (``` pairs) in the message.
	// Coding tasks almost always require the heavy model.
	CodeBlockCount int

	// RecentToolCalls is the count of tool_call messages in the last lookbackWindow
	// history entries. A high density indicates an active agentic workflow.
	RecentToolCalls int

	// ConversationDepth is the total number of messages in the session history.
	// Deep sessions tend to carry implicit complexity built up over many turns.
	ConversationDepth int

	// HasAttachments is true when the message appears to contain media (images,
	// audio, video). Multi-modal inputs require vision-capable heavy models.
	HasAttachments bool
}

// ExtractFeatures computes the structural feature vector for a message.
// It is a pure function with no side effects and zero allocations beyond
// the returned struct.
func ExtractFeatures(msg string, history []providers.Message) Features {
	return Features{
		TokenEstimate:     estimateTokens(msg),
		CodeBlockCount:    countCodeBlocks(msg),
		RecentToolCalls:   countRecentToolCalls(history),
		ConversationDepth: len(history),
		HasAttachments:    hasAttachments(msg),
	}
}

// estimateTokens returns a conservative token count proxy.
// Using rune count / 3 rather than / 4 because CJK characters each map to
// roughly one token, while ASCII words average ~1.3 chars/token. Dividing
// by 3 is a safe middle ground that slightly over-estimates for Latin text
// (errs toward routing to the heavy model) and is accurate for CJK.
func estimateTokens(msg string) int {
	rc := utf8.RuneCountInString(msg)
	return rc / 3
}

// countCodeBlocks counts the number of complete fenced code blocks.
// Each ``` delimiter increments a counter; pairs of delimiters form one block.
// An unclosed opening fence (odd count) is treated as zero complete blocks
// since it may just be an inline code span or a typo.
func countCodeBlocks(msg string) int {
	n := strings.Count(msg, "```")
	return n / 2
}

// countRecentToolCalls counts messages with tool calls in the last lookbackWindow
// entries of history. It examines the ToolCalls field rather than parsing
// the content string, so it is robust to any message format.
func countRecentToolCalls(history []providers.Message) int {
	start := len(history) - lookbackWindow
	if start < 0 {
		start = 0
	}

	count := 0
	for _, msg := range history[start:] {
		if len(msg.ToolCalls) > 0 {
			count += len(msg.ToolCalls)
		}
	}
	return count
}

// hasAttachments returns true when the message content contains embedded media.
// It checks for base64 data URIs (data:image/, data:audio/, data:video/) and
// common image/audio URL extensions. This is intentionally conservative —
// false negatives (missing an attachment) just mean the routing falls back to
// the primary model anyway.
func hasAttachments(msg string) bool {
	lower := strings.ToLower(msg)

	// Base64 data URIs embedded directly in the message
	if strings.Contains(lower, "data:image/") ||
		strings.Contains(lower, "data:audio/") ||
		strings.Contains(lower, "data:video/") {
		return true
	}

	// Common image/audio extensions in URLs or file references
	mediaExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp",
		".mp3", ".wav", ".ogg", ".m4a", ".flac",
		".mp4", ".avi", ".mov", ".webm",
	}
	for _, ext := range mediaExts {
		if strings.Contains(lower, ext) {
			return true
		}
	}

	return false
}
