package agent

import (
	"errors"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// userFriendlyError converts a raw Go error into a message safe to display
// to end users in chat. Internal details (HTTP status codes, wrapped error
// chains, Go formatting) are replaced with actionable, plain-language
// guidance. The original error is still logged server-side by the caller.
func userFriendlyError(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's already a classified FailoverError from the fallback chain.
	var foErr *providers.FailoverError
	if errors.As(err, &foErr) {
		return reasonToUserMessage(foErr.Reason)
	}

	// Classify the raw error using the same patterns the fallback chain uses.
	classified := providers.ClassifyError(err, "", "")
	if classified != nil {
		return reasonToUserMessage(classified.Reason)
	}

	// Unclassified error -- return a generic message.
	// Never expose the raw error string (it may contain API keys, internal
	// paths, or Go stack traces).
	return genericErrorMessage
}

// reasonToUserMessage maps a FailoverReason to a user-facing message.
func reasonToUserMessage(reason providers.FailoverReason) string {
	switch reason {
	case providers.FailoverAuth:
		return "I couldn't authenticate with the AI provider. " +
			"Please check your API key in ~/.picoclaw/config.json or run 'picoclaw auth login'."
	case providers.FailoverRateLimit:
		return "The AI provider is rate-limiting requests. Please try again in a moment."
	case providers.FailoverBilling:
		return "The AI provider rejected the request due to billing. " +
			"Please check your account balance or plan."
	case providers.FailoverTimeout:
		return "The request to the AI provider timed out. " +
			"Please check your internet connection and try again."
	case providers.FailoverOverloaded:
		return "The AI provider is currently overloaded. Please try again in a moment."
	case providers.FailoverFormat:
		return "The request format was rejected by the AI provider. " +
			"Run 'picoclaw doctor' to diagnose."
	default:
		return genericErrorMessage
	}
}

const genericErrorMessage = "Something went wrong processing your message. " +
	"Run 'picoclaw doctor' to diagnose."
