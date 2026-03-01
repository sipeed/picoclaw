package agent

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestUserFriendlyError_NilError(t *testing.T) {
	result := userFriendlyError(nil)
	if result != "" {
		t.Errorf("expected empty string for nil error, got %q", result)
	}
}

func TestUserFriendlyError_AuthErrors(t *testing.T) {
	cases := []string{
		"API request failed:\n  Status: 401\n  Body:   {\"error\":{\"code\":\"401\",\"message\":\"Unauthorized\"}}",
		"API request failed:\n  Status: 403\n  Body:   Forbidden",
		"invalid api key provided",
		"token has expired",
		"no credentials found for anthropic",
		"oauth token refresh failed",
	}

	for _, errMsg := range cases {
		result := userFriendlyError(errors.New(errMsg))
		if result == "" {
			t.Errorf("expected non-empty result for auth error %q", errMsg)
		}
		if result == genericErrorMessage {
			t.Errorf("expected auth-specific message for %q, got generic", errMsg)
		}
		// Should mention API key or auth login
		if !contains(result, "API key") && !contains(result, "auth") && !contains(result, "authenticate") {
			t.Errorf("expected auth guidance in message for %q, got %q", errMsg, result)
		}
	}
}

func TestUserFriendlyError_RateLimitErrors(t *testing.T) {
	cases := []string{
		"rate limit exceeded",
		"too many requests",
		"API request failed:\n  Status: 429\n  Body:   rate limited",
		"exceeded your current quota",
		"resource_exhausted",
	}

	for _, errMsg := range cases {
		result := userFriendlyError(errors.New(errMsg))
		if result == genericErrorMessage {
			t.Errorf("expected rate-limit-specific message for %q, got generic", errMsg)
		}
		if !contains(result, "rate") && !contains(result, "try again") {
			t.Errorf("expected rate-limit guidance in message for %q, got %q", errMsg, result)
		}
	}
}

func TestUserFriendlyError_TimeoutErrors(t *testing.T) {
	cases := []string{
		"failed to send request: context deadline exceeded (Client.Timeout exceeded while awaiting headers)",
		"timeout waiting for response",
		"request timed out",
	}

	for _, errMsg := range cases {
		result := userFriendlyError(errors.New(errMsg))
		if result == genericErrorMessage {
			t.Errorf("expected timeout-specific message for %q, got generic", errMsg)
		}
		if !contains(result, "timed out") && !contains(result, "internet") {
			t.Errorf("expected timeout guidance in message for %q, got %q", errMsg, result)
		}
	}
}

func TestUserFriendlyError_BillingErrors(t *testing.T) {
	cases := []string{
		"API request failed:\n  Status: 402\n  Body:   Payment Required",
		"insufficient credits",
		"insufficient balance",
	}

	for _, errMsg := range cases {
		result := userFriendlyError(errors.New(errMsg))
		if result == genericErrorMessage {
			t.Errorf("expected billing-specific message for %q, got generic", errMsg)
		}
		if !contains(result, "billing") && !contains(result, "balance") && !contains(result, "plan") {
			t.Errorf("expected billing guidance in message for %q, got %q", errMsg, result)
		}
	}
}

func TestUserFriendlyError_FormatErrors(t *testing.T) {
	cases := []string{
		"string should match pattern",
		"invalid request format",
	}

	for _, errMsg := range cases {
		result := userFriendlyError(errors.New(errMsg))
		if result == genericErrorMessage {
			t.Errorf("expected format-specific message for %q, got generic", errMsg)
		}
		if !contains(result, "doctor") {
			t.Errorf("expected doctor guidance in message for %q, got %q", errMsg, result)
		}
	}
}

func TestUserFriendlyError_UnknownErrors(t *testing.T) {
	cases := []string{
		"some completely random internal error",
		"unexpected nil pointer dereference",
		"goroutine stack overflow",
	}

	for _, errMsg := range cases {
		result := userFriendlyError(errors.New(errMsg))
		if result != genericErrorMessage {
			t.Errorf("expected generic message for unclassified error %q, got %q", errMsg, result)
		}
		// Must NOT contain the raw error text
		if contains(result, errMsg) {
			t.Errorf("generic message should not contain raw error text %q", errMsg)
		}
	}
}

func TestUserFriendlyError_FailoverError(t *testing.T) {
	// When the fallback chain wraps errors in FailoverError, we should
	// still produce a user-friendly message.
	foErr := &providers.FailoverError{
		Reason:   providers.FailoverAuth,
		Provider: "openai",
		Model:    "gpt-4",
		Wrapped:  errors.New("status 401: Unauthorized"),
	}

	result := userFriendlyError(foErr)
	if result == genericErrorMessage {
		t.Errorf("expected auth-specific message for FailoverError, got generic")
	}
	if !contains(result, "API key") && !contains(result, "authenticate") {
		t.Errorf("expected auth guidance for FailoverError, got %q", result)
	}
}

func TestUserFriendlyError_WrappedErrors(t *testing.T) {
	// Errors wrapped with fmt.Errorf %w should still be classified
	inner := errors.New("rate limit exceeded")
	wrapped := fmt.Errorf("LLM call failed after retries: %w", inner)

	result := userFriendlyError(wrapped)
	if result == genericErrorMessage {
		t.Errorf("expected rate-limit message for wrapped error, got generic")
	}
}

func TestUserFriendlyError_NeverLeaksRawError(t *testing.T) {
	// The key security property: raw error text should never appear
	// in the user-facing message.
	sensitiveErrors := []string{
		"API request failed:\n  Status: 401\n  Body:   {\"error\":{\"code\":\"401\"}}",
		"failed to send request: dial tcp 10.0.0.1:443: connect: connection refused",
		"codex API call: stream ended without completed response",
		"loading auth credentials: open /root/.picoclaw/auth.json: permission denied",
	}

	for _, errMsg := range sensitiveErrors {
		result := userFriendlyError(errors.New(errMsg))
		// The result should NOT contain any of the raw technical details
		if contains(result, "Status:") ||
			contains(result, "dial tcp") ||
			contains(result, "stream ended") ||
			contains(result, "/root/") ||
			contains(result, "permission denied") ||
			contains(result, "connection refused") {
			t.Errorf("user-friendly message leaked raw error for %q: got %q", errMsg, result)
		}
	}
}

func TestReasonToUserMessage_AllReasons(t *testing.T) {
	reasons := []providers.FailoverReason{
		providers.FailoverAuth,
		providers.FailoverRateLimit,
		providers.FailoverBilling,
		providers.FailoverTimeout,
		providers.FailoverOverloaded,
		providers.FailoverFormat,
		providers.FailoverUnknown,
	}

	for _, reason := range reasons {
		msg := reasonToUserMessage(reason)
		if msg == "" {
			t.Errorf("expected non-empty message for reason %q", reason)
		}
	}
}

// contains is a case-insensitive helper for test assertions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		(s == substr || containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if eqFoldSlice(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func eqFoldSlice(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
