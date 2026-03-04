package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"golang.org/x/time/rate"
)

// TestRateLimiter tests the core rate limiting functionality
func TestRateLimiter(t *testing.T) {
	// Create rate limiter with 1 request per second and burst of 1
	limiter := NewRateLimiter(rate.Limit(1), 1, time.Hour)

	key := "test-user"

	// First request should be allowed
	if !limiter.Allow(key) {
		t.Errorf("First request should be allowed")
	}

	// Second request should be denied (rate limit reached)
	if limiter.Allow(key) {
		t.Errorf("Second request should be denied due to rate limit")
	}

	// Wait for refill and third request should be allowed
	time.Sleep(1100 * time.Millisecond) // Wait slightly more than 1 second

	if !limiter.Allow(key) {
		t.Errorf("Third request should be allowed after waiting")
	}
}

// TestGetUserIdentifier tests extraction of user identifiers
func TestGetUserIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		setupReq func() *http.Request
		expected string
	}{
		{
			name:     "test ip strategy",
			strategy: "ip",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "192.168.1.1")
				return req
			},
			expected: "ip:192.168.1.1",
		},
		{
			name:     "test user strategy with X-User-ID",
			strategy: "user",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-ID", "test_user")
				return req
			},
			expected: "user:test_user",
		},
		{
			name:     "test combined strategy",
			strategy: "combined",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-ID", "test_user")
				req.Header.Set("X-Real-IP", "192.168.1.1")
				return req
			},
			expected: "user:test_user:ip:192.168.1.1",
		},
		{
			name:     "test default (ip) strategy",
			strategy: "invalid_strategy", // Invalid, should default to ip
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			expected: "ip:10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			result := GetUserIdentifier(req, tt.strategy)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestHTTPMiddleware tests the HTTP middleware functionality
func TestHTTPMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.RateLimiting.Enabled = true
	cfg.Tools.RateLimiting.Rate = 1  // 1 request per second
	cfg.Tools.RateLimiting.Burst = 1 // burst of 1
	cfg.Tools.RateLimiting.Strategy = "ip"

	// Create a rate-limited handler
	limitedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := CreateMiddlewareFromConfig(cfg)
	rateLimitedHandler := middleware(limitedHandler)

	// Test first request - should succeed
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	recorder1 := httptest.NewRecorder()

	rateLimitedHandler.ServeHTTP(recorder1, req1)

	if recorder1.Code != http.StatusOK {
		t.Errorf("First request should succeed, got status %d", recorder1.Code)
	}

	// Test second request - should fail with 429
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.1:12345" // Same IP as first request
	recorder2 := httptest.NewRecorder()

	rateLimitedHandler.ServeHTTP(recorder2, req2)

	if recorder2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should fail with 429, got status %d", recorder2.Code)
	}

	// Test third request from a different IP - should succeed
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "192.168.1.2:12345" // Different IP
	recorder3 := httptest.NewRecorder()

	rateLimitedHandler.ServeHTTP(recorder3, req3)

	if recorder3.Code != http.StatusOK {
		t.Errorf("Requests from different IPs should not affect each other, got status %d", recorder3.Code)
	}
}

// TestWithDisabledRateLimit tests behavior when rate limit is disabled
func TestWithDisabledRateLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.RateLimiting.Enabled = false // Disable rate limiting

	// Create a handler that should work without being limited
	callCount := 0
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := CreateMiddlewareFromConfig(cfg)
	wrappedHandler := middleware(testHandler)

	// Even 10 rapid requests should succeed when rate limiting is disabled
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		recorder := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Request %d should succeed when rate limiting is disabled, got status %d", i+1, recorder.Code)
		}
	}

	if callCount != 10 {
		t.Errorf("Expected 10 successful requests, got %d", callCount)
	}
}

// TestWaitMethod tests the Wait method with context
func TestWaitMethod(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(1), 1, time.Hour)
	ctx := context.Background()

	// First request should pass immediately
	err := limiter.Wait(ctx, "test-ip")
	if err != nil {
		t.Errorf("First request should not return an error: %v", err)
	}

	// Second request should be delayed and context should not timeout
	startTime := time.Now()
	err = limiter.Wait(ctx, "test-ip")
	duration := time.Since(startTime)

	// The wait should take approximately 1 second (for 1 request per second rate)
	// but shouldn't fail
	if err != nil {
		t.Errorf("Wait should not return an error: %v", err)
	}
	if duration < 500*time.Millisecond { // 500ms as threshold
		t.Errorf("Second request should be delayed, but took only %v", duration)
	}
}
