package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubValueRunner struct {
	errors []error
	vals   []string
	calls  int
}

func (s *stubValueRunner) Run(ctx context.Context) (string, error) {
	s.calls++
	idx := s.calls - 1
	if idx < len(s.errors) && s.errors[idx] != nil {
		return "", s.errors[idx]
	}
	if idx < len(s.vals) {
		return s.vals[idx], nil
	}
	return "", errors.New("no value")
}

func TestLLMRetry_IsRetryableError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want RetryDecision
	}{
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			want: RetryDecision{Retryable: true, Reason: RetryReasonTimeout},
		},
		{
			name: "client timeout string",
			err:  errors.New("failed to read response: context deadline exceeded (Client.Timeout)"),
			want: RetryDecision{Retryable: true, Reason: RetryReasonTimeout},
		},
		{
			name: "server 502",
			err:  errors.New("API request failed:\n  Status: 502\n  Body:   bad"),
			want: RetryDecision{Retryable: true, Status: 502, Reason: RetryReasonServerError},
		},
		{
			name: "client 400",
			err:  errors.New("API request failed:\n  Status: 400\n  Body:   bad"),
			want: RetryDecision{Retryable: false, Status: 400},
		},
		{
			name: "other error",
			err:  errors.New("something else"),
			want: RetryDecision{Retryable: false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsRetryableError(tc.err)
			if got.Retryable != tc.want.Retryable || got.Status != tc.want.Status || got.Reason != tc.want.Reason {
				t.Fatalf("IsRetryableError(%v) = %+v, want %+v", tc.err, got, tc.want)
			}
		})
	}
}

func TestLLMRetry_DoWithRetry_TimeoutThenSuccess(t *testing.T) {
	runner := &stubValueRunner{
		errors: []error{context.DeadlineExceeded, nil},
		vals:   []string{"", "ok"},
	}

	notices := 0
	retryCfg := RetryConfig{
		Timeouts: []time.Duration{5 * time.Millisecond, 5 * time.Millisecond},
		Backoffs: []time.Duration{},
		Notify: func(attempt, total int, decision RetryDecision) {
			notices++
			if decision.Reason != RetryReasonTimeout {
				t.Fatalf("expected timeout reason, got %v", decision.Reason)
			}
		},
	}

	val, err := DoWithRetry(context.Background(), retryCfg, runner.Run)
	if err != nil {
		t.Fatalf("DoWithRetry error: %v", err)
	}
	if val != "ok" {
		t.Fatalf("val = %q, want ok", val)
	}
	if runner.calls != 2 {
		t.Fatalf("runner.calls = %d, want 2", runner.calls)
	}
	if notices != 1 {
		t.Fatalf("notices = %d, want 1", notices)
	}
}

func TestLLMRetry_DoWithRetry_ServerErrorThenSuccess(t *testing.T) {
	runner := &stubValueRunner{
		errors: []error{errors.New("API request failed:\n  Status: 502\n  Body: bad"), nil},
		vals:   []string{"", "ok"},
	}

	retryCfg := RetryConfig{
		Timeouts: []time.Duration{5 * time.Millisecond, 5 * time.Millisecond},
		Backoffs: []time.Duration{},
	}

	val, err := DoWithRetry(context.Background(), retryCfg, runner.Run)
	if err != nil {
		t.Fatalf("DoWithRetry error: %v", err)
	}
	if val != "ok" {
		t.Fatalf("val = %q, want ok", val)
	}
	if runner.calls != 2 {
		t.Fatalf("runner.calls = %d, want 2", runner.calls)
	}
}

func TestLLMRetry_DoWithRetry_NoRetryOnClientError(t *testing.T) {
	runner := &stubValueRunner{
		errors: []error{errors.New("API request failed:\n  Status: 400\n  Body: bad")},
		vals:   []string{""},
	}

	retryCfg := RetryConfig{
		Timeouts: []time.Duration{5 * time.Millisecond, 5 * time.Millisecond},
		Backoffs: []time.Duration{},
	}

	_, err := DoWithRetry(context.Background(), retryCfg, runner.Run)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if runner.calls != 1 {
		t.Fatalf("runner.calls = %d, want 1", runner.calls)
	}
}
