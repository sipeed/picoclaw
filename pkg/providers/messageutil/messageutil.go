package messageutil

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// nameSanitizer matches characters that are not valid in the OpenAI Chat
// Completions `name` field. The OpenAI spec requires `^[a-zA-Z0-9_-]{1,64}$`.
var nameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// maxMessageNameLen mirrors the OpenAI Chat Completions hard limit on the
// `name` field. Values longer than this are truncated.
const maxMessageNameLen = 64

// SanitizeMessageName normalizes a raw sender identifier into a value safe
// for the OpenAI Chat Completions `name` field and stable across providers.
// Disallowed characters are coalesced into a single underscore; leading and
// trailing underscores are trimmed; the result is truncated to 64 bytes.
//
// Returns "" when raw is empty or collapses to nothing after sanitization.
// The output is pure ASCII so the truncation is byte-safe.
func SanitizeMessageName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	cleaned := nameSanitizer.ReplaceAllString(raw, "_")
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return ""
	}
	if len(cleaned) > maxMessageNameLen {
		cleaned = cleaned[:maxMessageNameLen]
	}
	return cleaned
}

// IsSystemSenderID reports whether senderID identifies an internal trigger
// (cron, heartbeat, async tool callback, system channel) rather than a real
// human user. These should not propagate as message-level sender attribution
// because they don't represent distinct actors in a multi-user conversation.
func IsSystemSenderID(senderID string) bool {
	id := strings.ToLower(strings.TrimSpace(senderID))
	switch id {
	case "", "cron", "heartbeat", "system":
		return true
	}
	return strings.HasPrefix(id, "async:")
}

// ApplyUserNamePrefix returns the message content with a `[name] ` prefix
// when msg carries sender attribution that the calling adapter cannot send
// natively (Anthropic, Bedrock, etc.). The persisted msg is not mutated.
//
// Returns msg.Content unchanged when:
//   - the role is not "user"
//   - msg.Name is empty
//   - msg.ToolCallID is set (tool result, not a user utterance)
//   - msg.Content is empty (avoid producing a bare "[name] ")
func ApplyUserNamePrefix(msg protocoltypes.Message) string {
	if msg.Role != "user" || msg.Name == "" || msg.ToolCallID != "" || msg.Content == "" {
		return msg.Content
	}
	return fmt.Sprintf("[%s] %s", msg.Name, msg.Content)
}

// IsTransientAssistantThoughtMessage reports whether msg is an invalid
// reasoning-only assistant history record. These "hanging" thought messages
// are not a canonical persisted format and should be discarded instead of
// replayed or reconstructed.
func IsTransientAssistantThoughtMessage(msg protocoltypes.Message) bool {
	return msg.Role == "assistant" &&
		strings.TrimSpace(msg.Content) == "" &&
		strings.TrimSpace(msg.ReasoningContent) != "" &&
		len(msg.ToolCalls) == 0 &&
		len(msg.Media) == 0 &&
		len(msg.Attachments) == 0 &&
		strings.TrimSpace(msg.ToolCallID) == ""
}

// FilterInvalidHistoryMessages removes invalid persisted history records such
// as transient assistant thought-only messages.
func FilterInvalidHistoryMessages(history []protocoltypes.Message) []protocoltypes.Message {
	if len(history) == 0 {
		return []protocoltypes.Message{}
	}

	filtered := make([]protocoltypes.Message, 0, len(history))
	for _, msg := range history {
		if IsTransientAssistantThoughtMessage(msg) {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}
