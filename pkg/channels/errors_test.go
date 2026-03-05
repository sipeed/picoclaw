package channels

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorsIs(t *testing.T) {
	wrapped := fmt.Errorf("telegram API: %w", ErrRateLimit)
	if !errors.Is(wrapped, ErrRateLimit) {
		t.Error("wrapped ErrRateLimit should match")
	}
	if errors.Is(wrapped, ErrTemporary) {
		t.Error("wrapped ErrRateLimit should not match ErrTemporary")
	}
}

func TestErrorsIsAllTypes(t *testing.T) {
	sentinels := []error{ErrNotRunning, ErrRateLimit, ErrTemporary, ErrSendFailed}

	for _, sentinel := range sentinels {
		wrapped := fmt.Errorf("context: %w", sentinel)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("wrapped %v should match itself", sentinel)
		}

		// Verify it doesn't match other sentinel errors
		for _, other := range sentinels {
			if other == sentinel {
				continue
			}
			if errors.Is(wrapped, other) {
				t.Errorf("wrapped %v should not match %v", sentinel, other)
			}
		}
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrNotRunning, "channel not running"},
		{ErrRateLimit, "rate limited"},
		{ErrTemporary, "temporary failure"},
		{ErrSendFailed, "send failed"},
	}

	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("error message = %q, want %q", got, tt.want)
		}
	}
}
