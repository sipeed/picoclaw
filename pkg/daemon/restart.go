// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package daemon

import (
	"sync"
	"time"
)

// RestartPolicy defines the strategy for process restart after crashes.
//
// Design rationale:
// - Automatic restart improves reliability for long-running services
// - Exponential backoff prevents rapid restart loops that could indicate systemic issues
// - Maximum attempt limits prevent infinite restart cycles
// - Time window ensures crashes are counted within a meaningful period
type RestartPolicy struct {
	// MaxAttempts is the maximum number of restart attempts before giving up.
	MaxAttempts int

	// WindowDuration is the time window in which restart attempts are counted.
	// Crashes outside this window don't count toward MaxAttempts.
	WindowDuration time.Duration

	// BackoffBase is the initial backoff duration before first restart.
	BackoffBase time.Duration

	// BackoffMultiplier is the factor by which backoff increases after each attempt.
	// A value of 2.0 means each backoff is double the previous one.
	BackoffMultiplier float64

	// BackoffMax is the maximum backoff duration.
	BackoffMax time.Duration
}

// DefaultRestartPolicy returns a restart policy with sensible defaults.
//
// Defaults:
// - Max 3 attempts within 5 minutes
// - Exponential backoff starting at 1 second, capped at 30 seconds
func DefaultRestartPolicy() *RestartPolicy {
	return &RestartPolicy{
		MaxAttempts:       3,
		WindowDuration:    5 * time.Minute,
		BackoffBase:       1 * time.Second,
		BackoffMultiplier: 2.0,
		BackoffMax:        30 * time.Second,
	}
}

// RestartTracker tracks restart attempts and determines when restart should occur.
type RestartTracker struct {
	policy   *RestartPolicy
	attempts []time.Time
	mu       sync.Mutex
}

// NewRestartTracker creates a new restart tracker with the given policy.
func NewRestartTracker(policy *RestartPolicy) *RestartTracker {
	if policy == nil {
		policy = DefaultRestartPolicy()
	}

	return &RestartTracker{
		policy:   policy,
		attempts: make([]time.Time, 0, policy.MaxAttempts),
	}
}

// RecordAttempt records a restart attempt at the current time.
// Returns the duration to wait before the next restart, or an error if
// the maximum number of attempts has been exceeded.
func (rt *RestartTracker) RecordAttempt() (time.Duration, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	now := time.Now()

	// Remove attempts outside the time window
	rt.cleanupOldAttempts(now)

	// Check if we've exceeded max attempts
	if len(rt.attempts) >= rt.policy.MaxAttempts {
		return 0, &MaxRestartsExceededError{
			attempts:      len(rt.attempts),
			maxAttempts:   rt.policy.MaxAttempts,
			Window:        rt.policy.WindowDuration,
			LastAttemptAt: rt.attempts[len(rt.attempts)-1],
		}
	}

	// Record this attempt
	rt.attempts = append(rt.attempts, now)

	// Calculate backoff duration
	backoff := rt.calculateBackoff(len(rt.attempts))

	return backoff, nil
}

// cleanupOldAttempts removes attempts that are outside the time window.
// Must be called with the lock held.
func (rt *RestartTracker) cleanupOldAttempts(now time.Time) {
	cutoff := now.Add(-rt.policy.WindowDuration)

	// Find the first attempt within the window
	firstValid := 0
	for i, attempt := range rt.attempts {
		if attempt.After(cutoff) {
			firstValid = i
			break
		}
	}

	// Remove old attempts
	if firstValid > 0 {
		rt.attempts = rt.attempts[firstValid:]
	}
}

// calculateBackoff computes the backoff duration using exponential backoff.
// Must be called with the lock held.
func (rt *RestartTracker) calculateBackoff(attemptNum int) time.Duration {
	// Exponential backoff: base * multiplier^(attemptNum-1)
	backoff := rt.policy.BackoffBase

	for i := 1; i < attemptNum; i++ {
		backoff = time.Duration(float64(backoff) * rt.policy.BackoffMultiplier)
		if backoff > rt.policy.BackoffMax {
			backoff = rt.policy.BackoffMax
			break
		}
	}

	return backoff
}

// Reset clears all recorded attempts.
// Use this when the process has been running successfully for a while
// and you want to reset the crash counter.
func (rt *RestartTracker) Reset() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.attempts = make([]time.Time, 0, rt.policy.MaxAttempts)
}

// ShouldRestart returns true if another restart attempt should be made.
func (rt *RestartTracker) ShouldRestart() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.cleanupOldAttempts(time.Now())
	return len(rt.attempts) < rt.policy.MaxAttempts
}

// GetAttemptCount returns the number of restart attempts within the current window.
func (rt *RestartTracker) GetAttemptCount() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.cleanupOldAttempts(time.Now())
	return len(rt.attempts)
}

// MaxRestartsExceededError is returned when the maximum number of
// restart attempts has been exceeded within the time window.
type MaxRestartsExceededError struct {
	attempts      int
	maxAttempts   int
	Window        time.Duration
	LastAttemptAt time.Time
}

func (e *MaxRestartsExceededError) Error() string {
	return "maximum restart attempts exceeded"
}

// GetAttempts returns the number of restart attempts made.
func (e *MaxRestartsExceededError) GetAttempts() int {
	return e.attempts
}

// GetMaxAttempts returns the maximum allowed attempts.
func (e *MaxRestartsExceededError) GetMaxAttempts() int {
	return e.maxAttempts
}
