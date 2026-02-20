package agent

import (
	"fmt"
	"sync"
	"time"
)

// rateLimiter provides simple sliding-window rate limiting for tool calls and requests.
type rateLimiter struct {
	maxToolCallsPerMinute int
	maxRequestsPerMinute  int

	mu            sync.Mutex
	toolCallTimes []time.Time
	requestTimes  []time.Time
}

func newRateLimiter(maxToolCalls, maxRequests int) *rateLimiter {
	return &rateLimiter{
		maxToolCallsPerMinute: maxToolCalls,
		maxRequestsPerMinute:  maxRequests,
	}
}

// checkToolCall checks if a tool call is allowed under the rate limit.
// Returns nil if allowed, error if rate limited.
func (rl *rateLimiter) checkToolCall() error {
	if rl.maxToolCallsPerMinute <= 0 {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// Remove expired entries
	rl.toolCallTimes = pruneOld(rl.toolCallTimes, cutoff)

	if len(rl.toolCallTimes) >= rl.maxToolCallsPerMinute {
		return fmt.Errorf("tool call limit exceeded (%d/min)", rl.maxToolCallsPerMinute)
	}

	rl.toolCallTimes = append(rl.toolCallTimes, now)
	return nil
}

// checkRequest checks if a request is allowed under the rate limit.
// Returns nil if allowed, error if rate limited.
func (rl *rateLimiter) checkRequest() error {
	if rl.maxRequestsPerMinute <= 0 {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// Remove expired entries
	rl.requestTimes = pruneOld(rl.requestTimes, cutoff)

	if len(rl.requestTimes) >= rl.maxRequestsPerMinute {
		return fmt.Errorf("request limit exceeded (%d/min)", rl.maxRequestsPerMinute)
	}

	rl.requestTimes = append(rl.requestTimes, now)
	return nil
}

// pruneOld removes timestamps older than cutoff.
func pruneOld(times []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(times) && times[i].Before(cutoff) {
		i++
	}
	if i == 0 {
		return times
	}
	return times[i:]
}
