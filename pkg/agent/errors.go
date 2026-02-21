package agent

import (
	"strings"
)

// friendlyError maps a raw Go error to a user-friendly message.
// Errors are checked in priority order: auth > rate-limit > context > network > server > fallback.
func friendlyError(err error) string {
	msg := strings.ToLower(err.Error())

	// 1. Authentication errors (most actionable — check first)
	if strings.Contains(msg, "401") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "invalid api key") ||
		strings.Contains(msg, "authentication") {
		return "I couldn't authenticate with the AI provider. Please check your API key in ~/.picoclaw/config.json"
	}

	// 2. Rate limiting
	if strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "too many requests") {
		return "I'm being rate-limited by the AI provider. Please try again in a moment."
	}

	// 3. Context/token limit exceeded
	if strings.Contains(msg, "context length") ||
		strings.Contains(msg, "token limit") ||
		strings.Contains(msg, "maximum context") {
		return "The conversation is too long for the current model. Try starting a new conversation."
	}

	// 4. Network errors
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "dial tcp") {
		return "I couldn't reach the AI provider. Please check your internet connection."
	}

	// 5. Server errors
	if strings.Contains(msg, "500") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "internal server error") ||
		strings.Contains(msg, "bad gateway") ||
		strings.Contains(msg, "service unavailable") {
		return "The AI provider is experiencing issues. Please try again later."
	}

	// 6. Generic fallback
	return "Something went wrong processing your message. Run 'picoclaw doctor' to diagnose."
}
