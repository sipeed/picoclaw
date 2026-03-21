package qq

import (
	"context"
	"sync"
	"time"
)

// rateLimiter implements a per-chat token bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	lastSend map[string]time.Time
	interval time.Duration
}

// newRateLimiter creates a rate limiter with the specified minimum interval between sends.
func newRateLimiter(interval time.Duration) *rateLimiter {
	return &rateLimiter{
		lastSend: make(map[string]time.Time),
		interval: interval,
	}
}

// wait waits if necessary until enough time has passed since the last send to this chat.
func (r *rateLimiter) wait(chatID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if lastSend, ok := r.lastSend[chatID]; ok {
		elapsed := now.Sub(lastSend)
		if elapsed < r.interval {
			time.Sleep(r.interval - elapsed)
			now = time.Now()
		}
	}

	r.lastSend[chatID] = now
}

// waitWithContext waits with context cancellation support.
func (r *rateLimiter) waitWithContext(chatID string, ctx context.Context) error {
	r.mu.Lock()
	now := time.Now()
	var waitDuration time.Duration

	if lastSend, ok := r.lastSend[chatID]; ok {
		elapsed := now.Sub(lastSend)
		if elapsed < r.interval {
			waitDuration = r.interval - elapsed
		}
	}
	r.mu.Unlock()

	if waitDuration > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}

		r.mu.Lock()
		r.lastSend[chatID] = time.Now()
		r.mu.Unlock()
	} else {
		r.mu.Lock()
		r.lastSend[chatID] = now
		r.mu.Unlock()
	}

	return nil
}

// clear removes the rate limit entry for a chat (e.g., after reconnection).
func (r *rateLimiter) clear(chatID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lastSend, chatID)
}

// clearAll removes all rate limit entries.
func (r *rateLimiter) clearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSend = make(map[string]time.Time)
}
