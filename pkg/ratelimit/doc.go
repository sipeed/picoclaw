// Package ratelimit provides token bucket rate limiting functionality with support for
// user/IP-based limiting, configurable parameters, and integration with HTTP handlers
//
// This package provides:
//   - Token bucket rate limiting algorithm implementation
//   - Support for different identification strategies (IP, User ID, Combined)
//   - Configurable rate, burst, and TTL parameters
//   - HTTP middleware for easy integration
//   - Global applicator for application-wide rate limiting
//
// Example usage:
//
//	// Basic rate limiter
//	limiter := ratelimit.NewRateLimiter(rate.Limit(5), 10, time.Minute) // 5 req/s, burst 10, 1 min TTL
//
//	// Check if request should be allowed
//	if limiter.Allow("user-key") {
//	    // Process request
//	} else {
//	    // Rate limit exceeded
//	}
//
//	// HTTP Middleware usage
//	middleware := ratelimit.NewRateLimitMiddleware(limiter)
//	rateLimitedHandler := middleware.CreateMiddleware("ip")(handler)  // IP-based limiting
//
//	// Or with configuration
//	cfg := config.DefaultConfig()
//	httpHandler := middleware.CreateMiddleware(cfg.Tools.RateLimiting.Strategy)(handler)
package ratelimit
