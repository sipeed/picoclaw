package ratelimit

import (
	"net/http"

	"github.com/sipeed/picoclaw/pkg/config"
)

// HTTPServer integrates rate limiting with an HTTP server
type HTTPServer struct {
	config  *config.Config
	server  *http.Server
	limiter *RateLimiter
}

// NewHTTPServer creates a new HTTP server with integrated rate limiting
func NewHTTPServer(cfg *config.Config) *HTTPServer {
	limiter := CreateDefaultRateLimiterFromConfig(cfg)

	return &HTTPServer{
		config:  cfg,
		limiter: limiter,
	}
}

// ApplyRateLimit wraps an HTTP handler with rate limiting based on configuration
func (hs *HTTPServer) ApplyRateLimit(handler http.Handler) http.Handler {
	if !hs.config.Tools.RateLimiting.Enabled {
		return handler
	}

	middleware := NewRateLimitMiddleware(hs.limiter)

	strategy := hs.config.Tools.RateLimiting.Strategy
	if strategy == "" {
		strategy = "ip" // default to IP-based limiting
	}

	return middleware.CreateMiddleware(strategy)(handler)
}

// ApplyRateLimitFunc wraps an HTTP handler function with rate limiting
func (hs *HTTPServer) ApplyRateLimitFunc(handlerFunc http.HandlerFunc) http.Handler {
	return hs.ApplyRateLimit(handlerFunc)
}

// WrapRoute provides an easy way to wrap individual routes with rate limiting
func (hs *HTTPServer) WrapRoute(pattern string, handler http.Handler) (string, http.Handler) {
	return pattern, hs.ApplyRateLimit(handler)
}

// WrapRouteFunc provides an easy way to wrap individual routes with rate limiting
func (hs *HTTPServer) WrapRouteFunc(pattern string, handlerFunc http.HandlerFunc) (string, http.Handler) {
	return pattern, hs.ApplyRateLimit(handlerFunc)
}

// GetLimiter returns the rate limiter instance
func (hs *HTTPServer) GetLimiter() *RateLimiter {
	return hs.limiter
}
