package providers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

func TestClassifyError_HTTPErrorStatus(t *testing.T) {
	err := fmt.Errorf("provider request: %w", &common.HTTPError{
		StatusCode:  httpStatusUnauthorized,
		BodyPreview: `{"error":"unauthorized"}`,
		ContentType: "application/json",
		APIBase:     "https://api.example.com",
	})

	result := ClassifyError(err, "openai", "gpt-4")
	if result == nil {
		t.Fatal("expected classified error")
	}
	if result.Reason != FailoverAuth {
		t.Fatalf("reason = %q, want %q", result.Reason, FailoverAuth)
	}
	if result.Status != httpStatusUnauthorized {
		t.Fatalf("status = %d, want %d", result.Status, httpStatusUnauthorized)
	}
}

func TestClassifyAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want AuthErrorKind
	}{
		{
			name: "invalid api key",
			err: errors.New(
				`API request failed: Status: 401 Body: {"error":{"message":"Incorrect API key provided"}}`,
			),
			want: AuthErrorInvalidAPIKey,
		},
		{
			name: "missing api key",
			err:  errors.New("API key not configured"),
			want: AuthErrorMissingAPIKey,
		},
		{
			name: "expired token",
			err:  errors.New("oauth token refresh failed: token has expired"),
			want: AuthErrorExpiredToken,
		},
		{
			name: "structured generic auth",
			err: &common.HTTPError{
				StatusCode:  httpStatusUnauthorized,
				BodyPreview: `{"error":"unauthorized"}`,
				ContentType: "application/json",
				APIBase:     "https://api.example.com",
			},
			want: AuthErrorGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClassifyAuthError(tt.err)
			if !ok {
				t.Fatal("expected auth classification")
			}
			if got != tt.want {
				t.Fatalf("kind = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyAuthError_FallbackExhaustedAllAuth(t *testing.T) {
	err := &FallbackExhaustedError{
		Attempts: []FallbackAttempt{
			{
				Reason: FailoverAuth,
				Error: &FailoverError{
					Reason:  FailoverAuth,
					Wrapped: errors.New("invalid api key"),
				},
			},
			{
				Reason: FailoverAuth,
				Error: &FailoverError{
					Reason:  FailoverAuth,
					Wrapped: errors.New("unauthorized"),
				},
			},
		},
	}

	got, ok := ClassifyAuthError(err)
	if !ok {
		t.Fatal("expected auth classification")
	}
	if got != AuthErrorInvalidAPIKey {
		t.Fatalf("kind = %q, want %q", got, AuthErrorInvalidAPIKey)
	}
}

func TestClassifyAuthError_FallbackExhaustedMixedFailures(t *testing.T) {
	err := &FallbackExhaustedError{
		Attempts: []FallbackAttempt{
			{
				Reason: FailoverAuth,
				Error: &FailoverError{
					Reason:  FailoverAuth,
					Wrapped: errors.New("invalid api key"),
				},
			},
			{
				Reason: FailoverRateLimit,
				Error: &FailoverError{
					Reason:  FailoverRateLimit,
					Wrapped: errors.New("rate limit exceeded"),
				},
			},
		},
	}

	if got, ok := ClassifyAuthError(err); ok {
		t.Fatalf("kind = %q, want no auth classification for mixed failures", got)
	}
}

const httpStatusUnauthorized = 401
