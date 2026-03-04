package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"golang.org/x/time/rate"
	"strconv"
)

// RateLimiter stores rate limiters for each user/IP combination
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate rate.Limit, burst int, ttl time.Duration) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate,
		burst:    burst,
		ttl:      ttl,
	}
}

// GetLimiter returns the rate limiter for the provided key (user ID or IP)
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter

		// Clean up the limiter after TTL expires
		go func(k string) {
			time.Sleep(rl.ttl)
			rl.mu.Lock()
			defer rl.mu.Unlock()
			// Double-check if the key still exists to avoid race conditions
			if l, ok := rl.limiters[k]; ok {
				// Only remove if it's the same limiter we set the timer for
				if l == limiter {
					delete(rl.limiters, k)
				}
			}
		}(key)
	}

	return limiter
}

// Allow checks if a request for the provided key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	limiter := rl.GetLimiter(key)
	return limiter.Allow()
}

// Reserve reserves a token for the provided key
func (rl *RateLimiter) Reserve(key string) *rate.Reservation {
	limiter := rl.GetLimiter(key)
	return limiter.Reserve()
}

// Wait waits until a token is available for the provided key
func (rl *RateLimiter) Wait(ctx context.Context, key string) error {
	limiter := rl.GetLimiter(key)
	return limiter.Wait(ctx)
}

// RateLimiterConfig holds the configuration for the rate limiter
type RateLimiterConfig struct {
	Rate     rate.Limit    `json:"rate"`
	Burst    int           `json:"burst"`
	TTL      time.Duration `json:"ttl"`
	Strategy string        `json:"strategy"` // "ip", "user", or "combined"
}

// GetUserIdentifier extracts a unique identifier from the request based on the strategy
func GetUserIdentifier(req *http.Request, strategy string) string {
	switch strategy {
	case "user":
		// Attempt to get user ID from headers or other authentication means
		userID := req.Header.Get("X-User-ID")
		if userID == "" {
			// Get user ID from basic auth username
			username, _, ok := req.BasicAuth()
			if ok {
				userID = username
			}
		}
		if userID == "" {
			// Get OAuth/JWT user info from header if present
			userID = req.Header.Get("X-Forwarded-User")
		}
		if userID == "" {
			// Fallback to request header
			userID = req.Header.Get("Authorization")
			// If Auth Bearer token, extract from format "Bearer <token>"
			if strings.HasPrefix(userID, "Bearer ") || strings.HasPrefix(userID, "Basic ") {
				parts := strings.Split(userID, " ")
				if len(parts) >= 2 {
					userID = parts[1]
				}
			}
		}
		return "user:" + userID
	case "ip":
		// Get real IP by checking various headers (use proxy-forwarded headers first)
		ip := req.Header.Get("X-Real-IP")
		if ip == "" {
			ip = req.Header.Get("X-Forwarded-For")
			// Take first IP if there are multiple values
			if idx := strings.Index(ip, ","); idx != -1 {
				ip = strings.TrimSpace(ip[:idx])
			}
		}
		if ip == "" {
			ip = req.Header.Get("CF-Connecting-IP") // Cloudflare
		}
		if ip == "" {
			ip = strings.Split(req.RemoteAddr, ":")[0] // Fall back to direct RemoteAddr
		}
		return "ip:" + ip
	case "combined":
		// Combine both IP and user ID for identification
		userPart := GetUserIdentifier(req, "user")
		ipPart := GetUserIdentifier(req, "ip")
		return userPart + ":" + ipPart
	default:
		// Default to IP-based limiting
		return GetUserIdentifier(req, "ip")
	}
}

// HTTP middleware for rate limiting
func (rl *RateLimiter) HTTPMiddleware(strategy string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := GetUserIdentifier(r, strategy)

			if !rl.Allow(key) {
				// Rate limit exceeded - return HTTP 429 Too Many Requests
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.burst))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Unix()+int64(rl.ttl.Seconds())))

				http.Error(w, "Rate limit exceeded. Please slow down.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CreateDefaultRateLimiterFromConfig creates a rate limiter with default configuration from app config
func CreateDefaultRateLimiterFromConfig(cfg *config.Config) *RateLimiter {
	rateLimitConfig := cfg.Tools.RateLimiting
	if rateLimitConfig.Enabled {
		return NewRateLimiter(
			rate.Limit(rateLimitConfig.Rate),
			rateLimitConfig.Burst,
			time.Duration(rateLimitConfig.TTL)*time.Second,
		)
	}

	// Create default rate limiter if not configured (default: 10 req/sec, burst of 20, TTL of 60 minutes)
	return NewRateLimiter(
		rate.Limit(10), // 10 requests per second
		20,             // burst of 20
		60*time.Minute, // TTL of 60 minutes
	)
}
