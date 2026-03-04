package ratelimit

import (
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"golang.org/x/time/rate"
)

// GlobalRateLimiter is an instance that can be accessed throughout the application
var GlobalRateLimiter *RateLimiter

// GlobalApplicator is an instance that provides methods to integrate rate limiting globally
var GlobalApplicator *GlobalApplicatorType

// GlobalApplicatorType provides application-wide rate limiting functions
type GlobalApplicatorType struct {
	config  *config.Config
	limiter *RateLimiter
}

// InitGlobalRateLimiter initializes the global rate limiter with configuration
func InitGlobalRateLimiter(cfg *config.Config) {
	GlobalRateLimiter = CreateDefaultRateLimiterFromConfig(cfg)
	GlobalApplicator = &GlobalApplicatorType{
		config:  cfg,
		limiter: GlobalRateLimiter,
	}
}

// IsRateLimited checks if a given key is currently rate limited
func (ga *GlobalApplicatorType) IsRateLimited(key string) bool {
	if ga.config == nil || !ga.config.Tools.RateLimiting.Enabled {
		return false
	}
	return !ga.limiter.Allow(key)
}

// ApplyToRoute conditionally applies rate limiting to a route based on configuration
func (ga *GlobalApplicatorType) ApplyToRoute(pattern string, handler http.Handler) (string, http.Handler) {
	if ga.config == nil || !ga.config.Tools.RateLimiting.Enabled {
		return pattern, handler
	}

	middleware := NewRateLimitMiddleware(ga.limiter)

	strategy := ga.config.Tools.RateLimiting.Strategy
	if strategy == "" {
		strategy = "ip" // default to IP-based limiting
	}

	wrapped := middleware.CreateMiddleware(strategy)(handler)
	return pattern, wrapped
}

// ApplyToRouteFunc conditionally applies rate limiting to a route function
func (ga *GlobalApplicatorType) ApplyToRouteFunc(pattern string, handlerFunc http.HandlerFunc) (string, http.Handler) {
	return ga.ApplyToRoute(pattern, handlerFunc)
}

// ApplyToHandler conditionally applies rate limiting to a handler
func (ga *GlobalApplicatorType) ApplyToHandler(handler http.Handler) http.Handler {
	if ga.config == nil || !ga.config.Tools.RateLimiting.Enabled {
		return handler
	}

	middleware := NewRateLimitMiddleware(ga.limiter)

	strategy := ga.config.Tools.RateLimiting.Strategy
	if strategy == "" {
		strategy = "ip" // default to IP-based limiting
	}

	return middleware.CreateMiddleware(strategy)(handler)
}

// GetRateLimiter returns the global rate limiter
func (ga *GlobalApplicatorType) GetRateLimiter() *RateLimiter {
	return ga.limiter
}

// GetCurrentConfig returns the current rate limiting configuration
func (ga *GlobalApplicatorType) GetCurrentConfig() config.RateLimitingConfig {
	return ga.config.Tools.RateLimiting
}

// UpdateConfig updates the global configuration
func (ga *GlobalApplicatorType) UpdateConfig(newConfig config.RateLimitingConfig) {
	if newConfig.Enabled {
		newLimiter := NewRateLimiter(
			rate.Limit(newConfig.Rate),
			newConfig.Burst,
			time.Duration(newConfig.TTL)*time.Second,
		)

		ga.limiter = newLimiter
		ga.config.Tools.RateLimiting = newConfig
	} else {
		ga.config.Tools.RateLimiting = newConfig
	}
}

// GetRateLimitStatus returns status information for a specific key
func (ga *GlobalApplicatorType) GetRateLimitStatus(key string) map[string]interface{} {
	if ga.config == nil || !ga.config.Tools.RateLimiting.Enabled {
		return map[string]interface{}{
			"enabled": false,
			"limited": false,
		}
	}

	limiter := ga.limiter.GetLimiter(key)

	// Get remaining tokens
	bucketState := limiter.TokensAt(time.Now())
	remaining := ga.limiter.burst - int(bucketState)
	if remaining < 0 {
		remaining = 0
	}

	return map[string]interface{}{
		"enabled":   true,
		"limited":   !limiter.Allow(),
		"limit":     ga.limiter.burst,
		"remaining": remaining,
		"reset":     time.Now().Add(ga.limiter.ttl).Unix(),
		"key":       key,
		"strategy":  ga.config.Tools.RateLimiting.Strategy,
	}
}
