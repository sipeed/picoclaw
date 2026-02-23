package tools

import (
	"sync"
	"time"
)

// rateBucket is a sliding-window rate limiter that tracks timestamps
// of recent calls within a 1-minute window. Zero external dependencies.
type rateBucket struct {
	mu      sync.Mutex
	max     int
	calls   []time.Time
	nowFunc func() time.Time
}

func newRateBucket(maxPerMinute int, nowFunc func() time.Time) *rateBucket {
	return &rateBucket{
		max:     maxPerMinute,
		calls:   make([]time.Time, 0, maxPerMinute),
		nowFunc: nowFunc,
	}
}

// Allow returns true if the call is within the rate limit.
func (rb *rateBucket) Allow() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	now := rb.nowFunc()
	cutoff := now.Add(-time.Minute)

	// Prune expired entries
	valid := 0
	for _, t := range rb.calls {
		if t.After(cutoff) {
			rb.calls[valid] = t
			valid++
		}
	}
	rb.calls = rb.calls[:valid]

	if len(rb.calls) >= rb.max {
		return false
	}

	rb.calls = append(rb.calls, now)
	return true
}
