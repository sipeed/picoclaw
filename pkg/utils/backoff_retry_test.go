package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoffCalculator(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:   5,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.0, // No jitter for predictable testing
	}

	// Test initial attempt (0) - should return BaseDelay
	delay := policy.Backoff(0)
	assert.Equal(t, 100*time.Millisecond, delay)

	// Test subsequent attempts with multiplier
	delay = policy.Backoff(1) // 100ms * 2 = 200ms
	assert.Equal(t, 200*time.Millisecond, delay)

	delay = policy.Backoff(2) // 200ms * 2 = 400ms
	assert.Equal(t, 400*time.Millisecond, delay)

	delay = policy.Backoff(3) // 400ms * 2 = 800ms
	assert.Equal(t, 800*time.Millisecond, delay)

	delay = policy.Backoff(4) // 800ms * 2 = 1600ms, but capped at MaxDelay (1s)
	assert.Equal(t, 1*time.Second, delay)
}

func TestJitteredBackoff(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:   3,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2, // 20% jitter
	}

	// Test that jittered backoffs are within expected ranges
	for i := 0; i < 5; i++ {
		delay := policy.Backoff(1) // Should be ~200ms with jitter
		expected := 200 * time.Millisecond
		margin := time.Duration(float64(expected) * 0.2) // 20% of 200ms

		assert.True(t, delay >= expected-margin && delay <= expected+margin,
			"Expected delay %v to be within [%v, %v]", delay, expected-margin, expected+margin)
	}
}

func TestDoWithRetry_Success(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:   3,
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   1.0,
		JitterFactor: 0.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		if callCount == 1 {
			return errors.New("simulated error")
		}
		return nil
	}

	ctx := context.Background()
	err := policy.DoWithRetry(ctx, fn)

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount) // should succeed on second try
}

func TestDoWithRetry_Exhausted(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:   2,
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   1.0,
		JitterFactor: 0.0,
	}

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("persistent error")
	}

	ctx := context.Background()
	err := policy.DoWithRetry(ctx, fn)

	assert.Error(t, err)
	assert.Equal(t, "persistent error", err.Error())
	assert.Equal(t, 3, callCount) // original + 2 retries
}

func TestDoWithRetry_Cancelled(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:   3,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   1.0,
		JitterFactor: 0.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short time to simulate interruption during backoff
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	errCh := make(chan error, 1)
	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("persistent error")
	}

	go func() {
		err := policy.DoWithRetry(ctx, fn)
		errCh <- err
	}()

	err := <-errCh
	assert.Equal(t, context.Canceled, err)
}

// error that is recognized as non-retryable by our policy function
type testNonRetryableError struct{}

func (e *testNonRetryableError) Error() string {
return "non-retryable error"
}
func TestDoWithRetry_NonRetryableError(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay: 100 * time.Millisecond,
		Multiplier: 1.0,
		JitterFactor: 0.0, // Adding the required parameter
		RetryableFunc: func(err error) bool {
			_, isTestError := err.(*testNonRetryableError) // Non-retryable
			return !isTestError
		},
	}

	callCount := 0
	nonRetryableErr := &testNonRetryableError{}

	fn := func() error {
		callCount++
		return nonRetryableErr
	}

	ctx := context.Background()
	err := policy.DoWithRetry(ctx, fn)

	assert.Equal(t, nonRetryableErr, err)
	assert.Equal(t, 1, callCount) // should only be called once since error is not retryable
}

func TestTemporaryErrorDetection(t *testing.T) {
	temporaryErr := errors.New("connection timeout")

	assert.True(t, IsTemporaryError(temporaryErr), "Should detect timeout error as temporary")
	assert.True(t, IsTemporaryError(errors.New("connection refused")), "Should detect connection refused as temporary")
	assert.True(t, IsTemporaryError(errors.New("too many requests")), "Should detect rate limiting as temporary")

	permanentErr := errors.New("invalid argument")
	assert.False(t, IsTemporaryError(permanentErr), "Should not consider invalid argument as temporary")
}

func TestRateLimitErrorDetection(t *testing.T) {
	assert.True(t, IsRateLimitedError(errors.New("too many requests")), "Should detect rate limit error")
	assert.True(t, IsRateLimitedError(errors.New("rate limit exceeded")), "Should detect rate limit error")
	assert.True(t, IsRateLimitedError(errors.New("429")), "Should detect HTTP 429 as rate limit error")

	regularErr := errors.New("random error")
	assert.False(t, IsRateLimitedError(regularErr), "Should not consider random error as rate limit")
}
