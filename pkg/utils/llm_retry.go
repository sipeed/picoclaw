package utils

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// RetryDecision captures whether an LLM error should be retried.
type RetryDecision struct {
	Retryable  bool
	Reason     providers.FailoverReason
	Status     int
	RetryAfter time.Duration
}

// RetryNotice is emitted before waiting for the next retry attempt.
type RetryNotice struct {
	Attempt  int // failed attempt number, starts at 1
	Total    int // total attempt count
	Decision RetryDecision
	Delay    time.Duration
}

type RetryNotifyFunc func(RetryNotice)
type RetrySleepFunc func(context.Context, time.Duration) error
type RetryJitterFunc func(time.Duration) time.Duration

// RetryPolicy defines per-attempt timeouts and backoffs for retry execution.
type RetryPolicy struct {
	AttemptTimeouts []time.Duration
	Backoffs        []time.Duration
	MaxElapsed      time.Duration
	MaxJitter       time.Duration
	Notify          RetryNotifyFunc
	Sleep           RetrySleepFunc
	Jitter          RetryJitterFunc
}

var retryAfterPattern = regexp.MustCompile(`(?i)retry[- ]after[:=]?\s*([^\r\n]+)`)

// DefaultLLMRetryPolicy returns the default retry behavior for LLM calls.
func DefaultLLMRetryPolicy() RetryPolicy {
	return RetryPolicy{
		AttemptTimeouts: []time.Duration{45 * time.Second, 90 * time.Second, 120 * time.Second},
		Backoffs:        []time.Duration{2 * time.Second, 5 * time.Second},
		MaxElapsed:      120 * time.Second,
		MaxJitter:       500 * time.Millisecond,
	}
}

// ClassifyRetryDecision classifies retryability using providers.ClassifyError.
func ClassifyRetryDecision(err error) RetryDecision {
	if err == nil {
		return RetryDecision{}
	}

	classified := providers.ClassifyError(err, "", "")
	if classified == nil {
		return RetryDecision{}
	}

	decision := RetryDecision{
		Reason: classified.Reason,
		Status: classified.Status,
	}

	switch classified.Reason {
	case providers.FailoverTimeout, providers.FailoverRateLimit:
		decision.Retryable = true
	default:
		decision.Retryable = false
	}

	if retryAfter, ok := extractRetryAfter(err, time.Now()); ok {
		decision.RetryAfter = retryAfter
	}

	return decision
}

// DoWithRetry executes fn with retry according to policy.
func DoWithRetry[T any](ctx context.Context, policy RetryPolicy, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	if len(policy.AttemptTimeouts) == 0 {
		return fn(ctx)
	}

	runCtx := ctx
	cancelRun := func() {}
	if policy.MaxElapsed > 0 {
		runCtx, cancelRun = context.WithTimeout(ctx, policy.MaxElapsed)
	}
	defer cancelRun()

	sleepFn := policy.Sleep
	if sleepFn == nil {
		sleepFn = sleepWithCtx
	}
	jitterFn := policy.Jitter
	if jitterFn == nil {
		jitterFn = defaultJitter
	}

	var lastErr error
	totalAttempts := len(policy.AttemptTimeouts)
	for attempt := 0; attempt < totalAttempts; attempt++ {
		if runCtx.Err() != nil {
			return zero, runCtx.Err()
		}

		attemptCtx := runCtx
		cancelAttempt := func() {}
		if attemptTimeout := policy.AttemptTimeouts[attempt]; attemptTimeout > 0 {
			timeout, ok := boundedAttemptTimeout(runCtx, attemptTimeout)
			if !ok {
				if err := runCtx.Err(); err != nil {
					return zero, err
				}
				return zero, context.DeadlineExceeded
			}
			attemptCtx, cancelAttempt = context.WithTimeout(runCtx, timeout)
		}

		val, err := fn(attemptCtx)
		cancelAttempt()
		if err == nil {
			return val, nil
		}
		lastErr = err

		// No retries left.
		if attempt == totalAttempts-1 {
			break
		}

		decision := ClassifyRetryDecision(err)
		if !decision.Retryable {
			break
		}

		delay := retryDelay(policy, attempt, decision, jitterFn)
		if policy.Notify != nil {
			policy.Notify(RetryNotice{
				Attempt:  attempt + 1,
				Total:    totalAttempts,
				Decision: decision,
				Delay:    delay,
			})
		}

		if delay > 0 {
			if err := sleepFn(runCtx, delay); err != nil {
				return zero, err
			}
		}
	}

	return zero, lastErr
}

// FormatLLMRetryNotice formats user-facing retry notice text.
func FormatLLMRetryNotice(notice RetryNotice) string {
	nextAttempt := notice.Attempt + 1
	if nextAttempt > notice.Total {
		nextAttempt = notice.Total
	}

	switch notice.Decision.Reason {
	case providers.FailoverRateLimit:
		if notice.Decision.Status > 0 {
			return fmt.Sprintf("LLM rate limited (%d). Retrying (%d/%d)...", notice.Decision.Status, nextAttempt, notice.Total)
		}
		return fmt.Sprintf("LLM rate limited. Retrying (%d/%d)...", nextAttempt, notice.Total)
	case providers.FailoverTimeout:
		if notice.Decision.Status > 0 {
			return fmt.Sprintf("LLM timeout/server error (%d). Retrying (%d/%d)...", notice.Decision.Status, nextAttempt, notice.Total)
		}
		return fmt.Sprintf("Temporary LLM timeout. Retrying (%d/%d)...", nextAttempt, notice.Total)
	default:
		return fmt.Sprintf("Temporary LLM error. Retrying (%d/%d)...", nextAttempt, notice.Total)
	}
}

func retryDelay(policy RetryPolicy, attempt int, decision RetryDecision, jitterFn RetryJitterFunc) time.Duration {
	if decision.RetryAfter > 0 {
		return decision.RetryAfter
	}

	if attempt < 0 || attempt >= len(policy.Backoffs) {
		return 0
	}

	base := policy.Backoffs[attempt]
	if base <= 0 {
		return 0
	}
	if policy.MaxJitter <= 0 {
		return base
	}

	jitter := jitterFn(policy.MaxJitter)
	if jitter < 0 {
		jitter = 0
	}
	if jitter > policy.MaxJitter {
		jitter = policy.MaxJitter
	}
	return base + jitter
}

func boundedAttemptTimeout(ctx context.Context, configured time.Duration) (time.Duration, bool) {
	if configured <= 0 {
		return 0, false
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return configured, true
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0, false
	}
	if configured > remaining {
		return remaining, true
	}
	return configured, true
}

func defaultJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	//nolint:gosec // Used only for retry backoff jitter.
	n := rand.Int63n(int64(max) + 1)
	return time.Duration(n)
}

func extractRetryAfter(err error, now time.Time) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}

	matches := retryAfterPattern.FindStringSubmatch(err.Error())
	if len(matches) < 2 {
		return 0, false
	}

	value := strings.TrimSpace(matches[1])
	if value == "" {
		return 0, false
	}

	if secs, convErr := strconv.Atoi(value); convErr == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}

	for _, layout := range []string{time.RFC1123, time.RFC1123Z, time.RFC850, time.ANSIC} {
		if t, parseErr := time.Parse(layout, value); parseErr == nil {
			delay := t.Sub(now)
			if delay <= 0 {
				return 0, false
			}
			return delay, true
		}
	}

	return 0, false
}
