package agent

import (
	"fmt"
	"testing"
)

func TestFriendlyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		// Authentication errors
		{
			name: "401 status code",
			err:  fmt.Errorf("LLM call failed after retries: status 401: Unauthorized"),
			want: "I couldn't authenticate with the AI provider. Please check your API key in ~/.picoclaw/config.json",
		},
		{
			name: "unauthorized keyword",
			err:  fmt.Errorf("request failed: unauthorized"),
			want: "I couldn't authenticate with the AI provider. Please check your API key in ~/.picoclaw/config.json",
		},
		{
			name: "invalid api key",
			err:  fmt.Errorf("invalid api key provided"),
			want: "I couldn't authenticate with the AI provider. Please check your API key in ~/.picoclaw/config.json",
		},
		{
			name: "authentication error",
			err:  fmt.Errorf("authentication failed for provider"),
			want: "I couldn't authenticate with the AI provider. Please check your API key in ~/.picoclaw/config.json",
		},

		// Rate limiting errors
		{
			name: "429 status code",
			err:  fmt.Errorf("status 429: Too Many Requests"),
			want: "I'm being rate-limited by the AI provider. Please try again in a moment.",
		},
		{
			name: "rate limit keyword",
			err:  fmt.Errorf("rate limit exceeded"),
			want: "I'm being rate-limited by the AI provider. Please try again in a moment.",
		},
		{
			name: "too many requests",
			err:  fmt.Errorf("too many requests from your organization"),
			want: "I'm being rate-limited by the AI provider. Please try again in a moment.",
		},

		// Context/token limit errors
		{
			name: "context length exceeded",
			err:  fmt.Errorf("context length exceeded: max 128000 tokens"),
			want: "The conversation is too long for the current model. Try starting a new conversation.",
		},
		{
			name: "token limit",
			err:  fmt.Errorf("token limit exceeded for model"),
			want: "The conversation is too long for the current model. Try starting a new conversation.",
		},
		{
			name: "maximum context",
			err:  fmt.Errorf("maximum context window reached"),
			want: "The conversation is too long for the current model. Try starting a new conversation.",
		},

		// Network errors
		{
			name: "connection refused",
			err:  fmt.Errorf("dial tcp: connection refused"),
			want: "I couldn't reach the AI provider. Please check your internet connection.",
		},
		{
			name: "no such host",
			err:  fmt.Errorf("dial tcp: lookup api.anthropic.com: no such host"),
			want: "I couldn't reach the AI provider. Please check your internet connection.",
		},
		{
			name: "timeout",
			err:  fmt.Errorf("request timeout after 30s"),
			want: "I couldn't reach the AI provider. Please check your internet connection.",
		},
		{
			name: "dial tcp",
			err:  fmt.Errorf("dial tcp 1.2.3.4:443: i/o timeout"),
			want: "I couldn't reach the AI provider. Please check your internet connection.",
		},

		// Server errors
		{
			name: "500 status code",
			err:  fmt.Errorf("status 500: internal server error"),
			want: "The AI provider is experiencing issues. Please try again later.",
		},
		{
			name: "502 bad gateway",
			err:  fmt.Errorf("status 502: bad gateway"),
			want: "The AI provider is experiencing issues. Please try again later.",
		},
		{
			name: "503 service unavailable",
			err:  fmt.Errorf("status 503: service unavailable"),
			want: "The AI provider is experiencing issues. Please try again later.",
		},
		{
			name: "internal server error keyword",
			err:  fmt.Errorf("internal server error occurred"),
			want: "The AI provider is experiencing issues. Please try again later.",
		},
		{
			name: "bad gateway keyword",
			err:  fmt.Errorf("bad gateway response"),
			want: "The AI provider is experiencing issues. Please try again later.",
		},
		{
			name: "service unavailable keyword",
			err:  fmt.Errorf("service unavailable"),
			want: "The AI provider is experiencing issues. Please try again later.",
		},

		// Generic fallback
		{
			name: "unknown error",
			err:  fmt.Errorf("something completely unexpected happened"),
			want: "Something went wrong processing your message. Run 'picoclaw doctor' to diagnose.",
		},
		{
			name: "wrapped unknown error",
			err:  fmt.Errorf("LLM call failed after retries: %w", fmt.Errorf("some obscure error")),
			want: "Something went wrong processing your message. Run 'picoclaw doctor' to diagnose.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := friendlyError(tt.err)
			if got != tt.want {
				t.Errorf("friendlyError(%q)\n  got:  %q\n  want: %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestFriendlyError_PriorityOrder(t *testing.T) {
	// Test that when an error matches multiple categories,
	// the more specific match wins (auth before server)
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "401 with server error text",
			err:  fmt.Errorf("status 401: internal server error"),
			want: "I couldn't authenticate with the AI provider. Please check your API key in ~/.picoclaw/config.json",
		},
		{
			name: "429 with timeout text",
			err:  fmt.Errorf("status 429: timeout waiting for rate limit"),
			want: "I'm being rate-limited by the AI provider. Please try again in a moment.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := friendlyError(tt.err)
			if got != tt.want {
				t.Errorf("friendlyError(%q)\n  got:  %q\n  want: %q", tt.err, got, tt.want)
			}
		})
	}
}
