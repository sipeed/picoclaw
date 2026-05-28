package providers

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token-bucket rate limiter for a single key.
// Allows up to RPM requests per minute with a burst equal to RPM.
// Thread-safe.
type RateLimiter struct {
	mu       sync.Mutex
	rpm      int
	tokens   float64
	maxBurst float64
	lastTick time.Time
	nowFunc  func() time.Time // for testing
}

func (rl *RateLimiter) refillLocked(now time.Time) {
	elapsed := now.Sub(rl.lastTick).Seconds()
	rl.lastTick = now

	// Refill tokens proportional to elapsed time.
	refill := elapsed * float64(rl.rpm) / 60.0
	rl.tokens = min(rl.maxBurst, rl.tokens+refill)
}

// newRateLimiter creates a RateLimiter that allows rpm requests/minute.
func newRateLimiter(rpm int) *RateLimiter {
	return &RateLimiter{
		rpm:      rpm,
		tokens:   float64(rpm), // start full
		maxBurst: float64(rpm),
		lastTick: time.Now(),
		nowFunc:  time.Now,
	}
}

// Wait blocks until a token is available or ctx is canceled.
// Returns ctx.Err() if canceled while waiting.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()
		now := rl.nowFunc()
		rl.refillLocked(now)

		if rl.tokens >= 1.0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calculate how long until a token is available.
		deficit := 1.0 - rl.tokens
		waitSec := deficit / (float64(rl.rpm) / 60.0)
		rl.mu.Unlock()

		timer := time.NewTimer(time.Duration(waitSec * float64(time.Second)))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
			// Loop to re-check (another goroutine may have consumed the token).
		}
	}
}

// TryAcquire attempts to consume a token without blocking.
func (rl *RateLimiter) TryAcquire() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillLocked(rl.nowFunc())
	if rl.tokens < 1.0 {
		return false
	}
	rl.tokens--
	return true
}

// RateLimiterRegistry holds per-candidate rate limiters.
// Candidates with RPM=0 are unrestricted.
// Thread-safe for concurrent reads/writes.
type RateLimiterRegistry struct {
	mu       sync.RWMutex
	limiters map[string]*RateLimiter
}

// NewRateLimiterRegistry creates an empty registry.
func NewRateLimiterRegistry() *RateLimiterRegistry {
	return &RateLimiterRegistry{
		limiters: make(map[string]*RateLimiter),
	}
}

// Register adds a rate limiter for the given key at the given RPM.
// If rpm <= 0, no limiter is registered (unrestricted).
func (r *RateLimiterRegistry) Register(key string, rpm int) {
	if rpm <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limiters[key] = newRateLimiter(rpm)
}

// Wait acquires a token for the given key, blocking if needed.
// If no limiter is registered for key, returns immediately.
func (r *RateLimiterRegistry) Wait(ctx context.Context, key string) error {
	r.mu.RLock()
	rl := r.limiters[key]
	r.mu.RUnlock()
	if rl == nil {
		return nil
	}
	return rl.Wait(ctx)
}

// TryAcquire attempts to consume a token for the given key without blocking.
// If no limiter is registered for key, it returns true.
func (r *RateLimiterRegistry) TryAcquire(key string) bool {
	r.mu.RLock()
	rl := r.limiters[key]
	r.mu.RUnlock()
	if rl == nil {
		return true
	}
	return rl.TryAcquire()
}

// RegisterCandidates registers rate limiters for all candidates that have RPM > 0.
// Candidates with RPM == 0 are ignored (no restriction).
func (r *RateLimiterRegistry) RegisterCandidates(candidates []FallbackCandidate) {
	for _, c := range candidates {
		if c.RPM > 0 {
			r.Register(c.StableKey(), c.RPM)
		}
	}
}
