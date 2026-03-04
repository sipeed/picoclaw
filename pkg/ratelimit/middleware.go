package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware defines the middleware that integrates with the ratelimit package
type RateLimitMiddleware struct {
	limiter *RateLimiter
}

// NewRateLimitMiddleware creates a new middleware using a rate limiter
func NewRateLimitMiddleware(limiter *RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// CreateMiddleware returns an HTTP middleware function configured with the strategy from config
func (rlm *RateLimitMiddleware) CreateMiddleware(strategy string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := GetUserIdentifier(r, strategy)

			if !rlm.limiter.Allow(key) {
				// Rate limit exceeded - return HTTP 429 Too Many Requests
				w.Header().Set("X-RateLimit-Limit", strconv.FormatFloat(float64(rlm.limiter.rate), 'f', -1, 64))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Unix()+int64(rlm.limiter.ttl.Seconds())))
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", rlm.limiter.ttl.Seconds()))

				http.Error(w, "Rate limit exceeded. Please slow down.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper function to create middleware directly from config
func CreateMiddlewareFromConfig(cfg *config.Config) func(http.Handler) http.Handler {
	if !cfg.Tools.RateLimiting.Enabled {
		// If rate limiting is disabled, return a noop middleware
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	limiter := CreateDefaultRateLimiterFromConfig(cfg)
	middleware := NewRateLimitMiddleware(limiter)

	strategy := cfg.Tools.RateLimiting.Strategy
	if strategy == "" {
		strategy = "ip" // default to IP-based limiting
	}

	return middleware.CreateMiddleware(strategy)
}

// RateLimitedHandler wraps an http.Handler with rate limiting
func RateLimitedHandler(handler http.Handler, limiter *RateLimiter, strategy string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := GetUserIdentifier(r, strategy)

		if !limiter.Allow(key) {
			w.Header().Set("X-RateLimit-Limit", strconv.FormatFloat(float64(limiter.rate), 'f', -1, 64))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Unix()+int64(limiter.ttl.Seconds())))
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", limiter.ttl.Seconds()))

			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// ContextKey type for storing rate limit info in request context
type ContextKey string

const RateLimitInfoKey ContextKey = "rate_limit_info"

// RateLimitInfo holds information about the current rate limit status
type RateLimitInfo struct {
	Limit     int `json:"limit"`
	Remaining int `json:"remaining"`
	Reset     int `json:"reset"` // Unix timestamp when the rate limit resets
}

// WithRateLimitInfo adds rate limit info to the request context
func (rlm *RateLimitMiddleware) WithRateLimitInfo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := GetUserIdentifier(r, "ip") // Use IP by default for info purposes

		limiter := rlm.limiter.GetLimiter(key)

		bucketState := limiter.TokensAt(time.Now())
		remaining := rlm.limiter.burst - int(bucketState)
		if remaining < 0 {
			remaining = 0
		}

		info := RateLimitInfo{
			Limit:     rlm.limiter.burst,
			Remaining: remaining,
			Reset:     int(time.Now().Add(rlm.limiter.ttl).Unix()),
		}

		// Add rate limit info to context
		ctx := context.WithValue(r.Context(), RateLimitInfoKey, info)

		// Add headers with rate limit info
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rlm.limiter.burst))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(time.Now().Add(rlm.limiter.ttl).Unix())))

		// Continue with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRateLimitInfo retrieves rate limit info from request context
func GetRateLimitInfo(ctx context.Context) *RateLimitInfo {
	info, ok := ctx.Value(RateLimitInfoKey).(RateLimitInfo)
	if !ok {
		return nil
	}
	return &info
}
