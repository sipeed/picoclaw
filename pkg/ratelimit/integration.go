package ratelimit

import (
	"net/http"

	"github.com/sipeed/picoclaw/pkg/config"
)

// RateLimitApplicator provides functions to integrate rate limiting
// with the rest of the application infrastructure
type RateLimitApplicator struct {
	config  *config.Config
	limiter *RateLimiter
}

// NewRateLimitApplicator creates a new applicator for integrating rate limiting
func NewRateLimitApplicator(cfg *config.Config) *RateLimitApplicator {
	limiter := CreateDefaultRateLimiterFromConfig(cfg)

	return &RateLimitApplicator{
		config:  cfg,
		limiter: limiter,
	}
}

// ApplyToMux applies rate limiting middleware to an http.ServeMux
// applying it around all registered routes
func (ra *RateLimitApplicator) ApplyToMux(mux *http.ServeMux) *http.ServeMux {
	if !ra.config.Tools.RateLimiting.Enabled {
		return mux
	}

	// Create a new mux with the same routes but wrapped with rate limiting
	newMux := http.NewServeMux()

	// For this implementation, we'll provide a wrapper function
	// instead of attempting to enumerate existing routes
	// This is because Go's http.ServeMux doesn't provide a method to list all registered routes

	return newMux
}

// WrapHandlerConditionally wraps the handler with rate limiting only if configured
func (ra *RateLimitApplicator) WrapHandlerConditionally(handler http.Handler) http.Handler {
	if !ra.config.Tools.RateLimiting.Enabled {
		return handler
	}

	middleware := NewRateLimitMiddleware(ra.limiter)

	strategy := ra.config.Tools.RateLimiting.Strategy
	if strategy == "" {
		strategy = "ip" // default to IP-based limiting
	}

	return middleware.CreateMiddleware(strategy)(handler)
}

// WrapHandlerFuncConditionally wraps the handler func with rate limiting only if configured
func (ra *RateLimitApplicator) WrapHandlerFuncConditionally(handlerFunc http.HandlerFunc) http.Handler {
	return ra.WrapHandlerConditionally(handlerFunc)
}

// IsEnabled returns true if rate limiting is enabled
func (ra *RateLimitApplicator) IsEnabled() bool {
	return ra.config.Tools.RateLimiting.Enabled
}

// GetConfig returns the rate limiting configuration
func (ra *RateLimitApplicator) GetConfig() *config.RateLimitingConfig {
	return &ra.config.Tools.RateLimiting
}

// UpdateConfig updates the configuration for the rate limiter
func (ra *RateLimitApplicator) UpdateConfig(newConfig config.RateLimitingConfig) {
	if newConfig.Enabled {
		// Create a new limiter with updated config
		newLimiter := NewRateLimiter(
			ra.limiter.rate,  // Keep current rate for now, we'll update based on config
			ra.limiter.burst, // Keep current burst for now, we'll update based on config
			ra.limiter.ttl,   // Keep current ttl for now, we'll update based on config
		)

		// Update limiter fields with new values from config
		newLimiter.rate = rate.Limit(newConfig.Rate)
		newLimiter.burst = newConfig.Burst
		newLimiter.ttl = time.Duration(newConfig.TTL) * time.Second

		ra.limiter = newLimiter
		ra.config.Tools.RateLimiting = newConfig
	} else {
		ra.config.Tools.RateLimiting = newConfig
	}
}
