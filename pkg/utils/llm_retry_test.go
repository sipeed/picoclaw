package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestLLMRetry_ClassifyRetryDecision_429WithRetryAfter(t *testing.T) {
	err := errors.New("API request failed:\n  Status: 429\n  Retry-After: 7")
	decision := ClassifyRetryDecision(err)

	if !decision.Retryable {
		t.Fatal("expected 429 to be retryable")
	}
	if decision.Reason != providers.FailoverRateLimit {
		t.Fatalf("reason = %q, want %q", decision.Reason, providers.FailoverRateLimit)
	}
	if decision.Status != 429 {
		t.Fatalf("status = %d, want 429", decision.Status)
	}
	if decision.RetryAfter != 7*time.Second {
		t.Fatalf("retry-after = %v, want 7s", decision.RetryAfter)
	}
}

func TestLLMRetry_DoWithRetry_ParentDeadlineDoesNotBurnAttempts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	calls := 0
	_, err := DoWithRetry(ctx, RetryPolicy{
		AttemptTimeouts: []time.Duration{time.Second, time.Second},
	}, func(callCtx context.Context) (string, error) {
		calls++
		<-callCtx.Done()
		return "", callCtx.Err()
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context deadline exceeded", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestLLMRetry_DoWithRetry_CancelDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	calls := 0
	sleepCalled := false

	_, err := DoWithRetry(ctx, RetryPolicy{
		AttemptTimeouts: []time.Duration{time.Second, time.Second},
		Backoffs:        []time.Duration{time.Hour},
		Sleep: func(waitCtx context.Context, _ time.Duration) error {
			sleepCalled = true
			<-waitCtx.Done()
			return waitCtx.Err()
		},
	}, func(context.Context) (string, error) {
		calls++
		cancel()
		return "", errors.New("API request failed: status: 502 body: bad gateway")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context canceled", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if !sleepCalled {
		t.Fatal("expected sleep path to be called")
	}
}

func TestLLMRetry_DoWithRetry_JitterBoundedBackoff(t *testing.T) {
	calls := 0
	var slept time.Duration

	_, err := DoWithRetry(context.Background(), RetryPolicy{
		AttemptTimeouts: []time.Duration{time.Second, time.Second},
		Backoffs:        []time.Duration{100 * time.Millisecond},
		MaxJitter:       50 * time.Millisecond,
		Jitter: func(max time.Duration) time.Duration {
			if max != 50*time.Millisecond {
				t.Fatalf("max jitter = %v, want 50ms", max)
			}
			return 37 * time.Millisecond
		},
		Sleep: func(_ context.Context, d time.Duration) error {
			slept = d
			return nil
		},
	}, func(context.Context) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("API request failed: status: 502 body: bad gateway")
		}
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if slept != 137*time.Millisecond {
		t.Fatalf("slept = %v, want 137ms", slept)
	}
}

func TestLLMRetry_DoWithRetry_UsesRetryAfterFor429(t *testing.T) {
	calls := 0
	var slept time.Duration

	_, err := DoWithRetry(context.Background(), RetryPolicy{
		AttemptTimeouts: []time.Duration{time.Second, time.Second},
		Backoffs:        []time.Duration{100 * time.Millisecond},
		MaxJitter:       80 * time.Millisecond,
		Jitter: func(_ time.Duration) time.Duration {
			return 50 * time.Millisecond
		},
		Sleep: func(_ context.Context, d time.Duration) error {
			slept = d
			return nil
		},
	}, func(context.Context) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("API request failed:\n  Status: 429\n  Retry-After: 3")
		}
		return "ok", nil
	})

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if slept != 3*time.Second {
		t.Fatalf("slept = %v, want 3s", slept)
	}
}

func TestLLMRetry_ExtractRetryAfter_HTTPDate(t *testing.T) {
	now := time.Date(2015, 10, 21, 7, 27, 0, 0, time.UTC)
	err := errors.New("API request failed:\n  Status: 429\n  Retry-After: Wed, 21 Oct 2015 07:28:00 GMT")

	delay, ok := extractRetryAfter(err, now)
	if !ok {
		t.Fatal("expected retry-after date to parse")
	}
	if delay != time.Minute {
		t.Fatalf("delay = %v, want 1m", delay)
	}
}
