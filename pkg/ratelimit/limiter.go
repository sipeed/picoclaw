// Package ratelimit provides rate limiting for API and tool usage.
// It implements a token bucket algorithm for smooth rate limiting.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Config holds rate limiter configuration.
type Config struct {
	Enabled                 bool
	RequestsPerMinute       int
	ToolExecutionsPerMinute int
	PerUserLimit            bool
}

// DefaultConfig returns the default rate limiting configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:                 false, // Off by default for single-user use
		RequestsPerMinute:       60,
		ToolExecutionsPerMinute: 30,
		PerUserLimit:            true,
	}
}

// Limiter implements a token bucket rate limiter.
type Limiter struct {
	config       Config
	buckets      sync.Map // map[string]*bucket
	globalMu     sync.Mutex
	globalBucket *bucket
}

// bucket represents a token bucket for rate limiting.
type bucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// newBucket creates a new token bucket.
func newBucket(maxTokens, refillRate float64) *bucket {
	return &bucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// refill adds tokens based on elapsed time.
func (b *bucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.lastRefill = now

	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
}

// tryTake attempts to take n tokens from the bucket.
// Returns true if successful, false if not enough tokens.
func (b *bucket) tryTake(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.tokens >= n {
		b.tokens -= n
		return true
	}
	return false
}

// waitUntil blocks until n tokens are available or context is cancelled.
func (b *bucket) waitUntil(ctx context.Context, n float64) error {
	for {
		if b.tryTake(n) {
			return nil
		}

		// Calculate wait time
		b.mu.Lock()
		b.refill()
		deficit := n - b.tokens
		waitTime := time.Duration(deficit/b.refillRate) * time.Second
		b.mu.Unlock()

		if waitTime <= 0 {
			waitTime = 100 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			continue
		}
	}
}

// availableTokens returns the current number of available tokens.
func (b *bucket) availableTokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	return b.tokens
}

// NewLimiter creates a new rate limiter with the given configuration.
func NewLimiter(config Config) *Limiter {
	l := &Limiter{
		config: config,
	}

	if config.Enabled {
		// Create global bucket
		l.globalBucket = newBucket(
			float64(config.RequestsPerMinute),
			float64(config.RequestsPerMinute)/60.0,
		)
	}

	return l
}

// AllowRequest checks if a request is allowed under the rate limit.
// Returns true if allowed, false if rate limit exceeded.
func (l *Limiter) AllowRequest(userID string) bool {
	if !l.config.Enabled {
		return true
	}

	// Check global limit first
	if !l.globalBucket.tryTake(1) {
		return false
	}

	// Check per-user limit if enabled
	if l.config.PerUserLimit && userID != "" {
		userBucket := l.getUserBucket(userID)
		if !userBucket.tryTake(1) {
			return false
		}
	}

	return true
}

// AllowToolExecution checks if a tool execution is allowed under the rate limit.
func (l *Limiter) AllowToolExecution(userID, toolName string) bool {
	if !l.config.Enabled {
		return true
	}

	// Create a bucket key for tool executions
	key := "tool:" + userID
	toolBucket := l.getToolBucket(key)

	return toolBucket.tryTake(1)
}

// WaitForRequest blocks until a request is allowed or context is cancelled.
func (l *Limiter) WaitForRequest(ctx context.Context, userID string) error {
	if !l.config.Enabled {
		return nil
	}

	// Wait for global bucket
	if err := l.globalBucket.waitUntil(ctx, 1); err != nil {
		return err
	}

	// Wait for per-user bucket if enabled
	if l.config.PerUserLimit && userID != "" {
		userBucket := l.getUserBucket(userID)
		if err := userBucket.waitUntil(ctx, 1); err != nil {
			return err
		}
	}

	return nil
}

// getUserBucket gets or creates a bucket for a specific user.
func (l *Limiter) getUserBucket(userID string) *bucket {
	if cached, ok := l.buckets.Load(userID); ok {
		return cached.(*bucket)
	}

	// Create new bucket
	newB := newBucket(
		float64(l.config.RequestsPerMinute),
		float64(l.config.RequestsPerMinute)/60.0,
	)

	actual, _ := l.buckets.LoadOrStore(userID, newB)
	return actual.(*bucket)
}

// getToolBucket gets or creates a bucket for tool executions.
func (l *Limiter) getToolBucket(key string) *bucket {
	if cached, ok := l.buckets.Load(key); ok {
		return cached.(*bucket)
	}

	// Create new bucket with tool execution limits
	newB := newBucket(
		float64(l.config.ToolExecutionsPerMinute),
		float64(l.config.ToolExecutionsPerMinute)/60.0,
	)

	actual, _ := l.buckets.LoadOrStore(key, newB)
	return actual.(*bucket)
}

// Status returns the current rate limit status for a user.
type Status struct {
	UserID        string
	RequestsUsed  int
	RequestsLimit int
	ToolsUsed     int
	ToolsLimit    int
	ResetIn       time.Duration
	GlobalUsed    int
	GlobalLimit   int
}

// GetStatus returns the current rate limit status for a user.
func (l *Limiter) GetStatus(userID string) Status {
	if !l.config.Enabled {
		return Status{}
	}

	status := Status{
		UserID:        userID,
		RequestsLimit: l.config.RequestsPerMinute,
		ToolsLimit:    l.config.ToolExecutionsPerMinute,
		GlobalLimit:   l.config.RequestsPerMinute,
	}

	// Get global bucket status
	if l.globalBucket != nil {
		status.GlobalUsed = int(l.globalBucket.maxTokens - l.globalBucket.availableTokens())
	}

	// Get user bucket status
	if userID != "" {
		if userBucket, ok := l.buckets.Load(userID); ok {
			b := userBucket.(*bucket)
			status.RequestsUsed = int(b.maxTokens - b.availableTokens())
		}

		// Get tool bucket status
		toolKey := "tool:" + userID
		if toolBucket, ok := l.buckets.Load(toolKey); ok {
			b := toolBucket.(*bucket)
			status.ToolsUsed = int(b.maxTokens - b.availableTokens())
		}
	}

	// Calculate reset time (approximately 1 minute)
	status.ResetIn = time.Minute

	return status
}

// Reset resets all rate limiters.
func (l *Limiter) Reset() {
	l.buckets = sync.Map{}
	if l.globalBucket != nil {
		l.globalBucket.tokens = l.globalBucket.maxTokens
		l.globalBucket.lastRefill = time.Now()
	}
}

// Cleanup removes old unused buckets to free memory.
func (l *Limiter) Cleanup(maxAge time.Duration) {
	now := time.Now()

	l.buckets.Range(func(key, value interface{}) bool {
		bucket := value.(*bucket)
		bucket.mu.Lock()
		if now.Sub(bucket.lastRefill) > maxAge {
			l.buckets.Delete(key)
		}
		bucket.mu.Unlock()
		return true
	})
}

// SetConfig updates the rate limiter configuration.
func (l *Limiter) SetConfig(config Config) {
	l.config = config

	// Recreate global bucket if enabled
	if config.Enabled {
		l.globalBucket = newBucket(
			float64(config.RequestsPerMinute),
			float64(config.RequestsPerMinute)/60.0,
		)
	}
}

// Global rate limiter instance
var globalLimiter *Limiter
var globalOnce sync.Once

// InitGlobal initializes the global rate limiter.
func InitGlobal(config Config) {
	globalOnce.Do(func() {
		globalLimiter = NewLimiter(config)
	})
}

// Allow checks if a request is allowed using the global limiter.
func Allow(userID string) bool {
	if globalLimiter == nil {
		return true
	}
	return globalLimiter.AllowRequest(userID)
}

// AllowTool checks if a tool execution is allowed using the global limiter.
func AllowTool(userID, toolName string) bool {
	if globalLimiter == nil {
		return true
	}
	return globalLimiter.AllowToolExecution(userID, toolName)
}

// GetGlobalStatus returns the rate limit status using the global limiter.
func GetGlobalStatus(userID string) Status {
	if globalLimiter == nil {
		return Status{}
	}
	return globalLimiter.GetStatus(userID)
}
