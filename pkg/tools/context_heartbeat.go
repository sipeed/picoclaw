package tools

import (
	"context"
	"sync/atomic"
)

type heartbeatKey struct{}

// WithHeartbeatContext marks the context as a heartbeat execution.
func WithHeartbeatContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, heartbeatKey{}, true)
}

// IsHeartbeatContext returns true if the context was created by a heartbeat execution.
func IsHeartbeatContext(ctx context.Context) bool {
	v, _ := ctx.Value(heartbeatKey{}).(bool)
	return v
}

// WebSearchQuota tracks remaining web searches for a heartbeat execution.
type WebSearchQuota struct {
	max       int32
	remaining atomic.Int32
}

// TryConsume atomically decrements the quota. Returns false if exhausted.
func (q *WebSearchQuota) TryConsume() bool {
	for {
		cur := q.remaining.Load()
		if cur <= 0 {
			return false
		}
		if q.remaining.CompareAndSwap(cur, cur-1) {
			return true
		}
	}
}

// Max returns the initial quota limit.
func (q *WebSearchQuota) Max() int32 { return q.max }

// Remaining returns the current remaining quota.
func (q *WebSearchQuota) Remaining() int32 { return q.remaining.Load() }

type searchQuotaKey struct{}

// WithWebSearchQuota attaches a web search quota to the context.
func WithWebSearchQuota(ctx context.Context, quota int) context.Context {
	q := &WebSearchQuota{max: int32(quota)}
	q.remaining.Store(int32(quota))
	return context.WithValue(ctx, searchQuotaKey{}, q)
}

// GetWebSearchQuota returns the search quota from the context, or nil if not set.
func GetWebSearchQuota(ctx context.Context) *WebSearchQuota {
	q, _ := ctx.Value(searchQuotaKey{}).(*WebSearchQuota)
	return q
}
