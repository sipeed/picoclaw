// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"jane/pkg/health"
	"jane/pkg/logger"
)

// telemetryMiddleware wraps an http.Handler to track request metrics using OpenTelemetry.
func telemetryMiddleware(next http.Handler) http.Handler {
	meter := otel.Meter("jane/pkg/channels")

	// Create metrics
	requestCounter, err := meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		logger.ErrorCF("channels", "Failed to create request counter metric", map[string]any{"error": err.Error()})
	}

	durationHistogram, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
	)
	if err != nil {
		logger.ErrorCF("channels", "Failed to create duration histogram metric", map[string]any{"error": err.Error()})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate Trace ID and attach to context and response header
		traceID := uuid.New().String()
		w.Header().Set("X-Trace-Id", traceID)
		ctx := context.WithValue(r.Context(), logger.TraceIDKey, traceID)
		r = r.WithContext(ctx)

		// Wrap ResponseWriter to capture the status code
		rw := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		durationMs := float64(time.Since(start).Microseconds()) / 1000.0

		if requestCounter != nil {
			requestCounter.Add(r.Context(), 1)
		}
		if durationHistogram != nil {
			durationHistogram.Record(r.Context(), durationMs)
		}
	})
}

// responseWriterWrapper captures the HTTP status code for metrics and logging
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements the http.Hijacker interface, required for WebSockets
func (rw *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return hijacker.Hijack()
}

// Flush implements the http.Flusher interface, required for streaming
func (rw *responseWriterWrapper) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// SetupHTTPServer creates a shared HTTP server with the given listen address.
// It registers health endpoints from the health server and discovers channels
// that implement WebhookHandler and/or HealthChecker to register their handlers.
func (m *Manager) SetupHTTPServer(addr string, healthServer *health.Server) {
	m.mux = http.NewServeMux()

	// Register health endpoints
	if healthServer != nil {
		healthServer.RegisterOnMux(m.mux)
	}

	// Discover and register webhook handlers and health checkers
	for name, ch := range m.channels {
		if wh, ok := ch.(WebhookHandler); ok {
			m.mux.Handle(wh.WebhookPath(), wh)
			logger.InfoCF("channels", "Webhook handler registered", map[string]any{
				"channel": name,
				"path":    wh.WebhookPath(),
			})
		}
		if hc, ok := ch.(HealthChecker); ok {
			m.mux.HandleFunc(hc.HealthPath(), hc.HealthHandler)
			logger.InfoCF("channels", "Health endpoint registered", map[string]any{
				"channel": name,
				"path":    hc.HealthPath(),
			})
		}
	}

	m.httpServer = &http.Server{
		Addr:              addr,
		Handler:           telemetryMiddleware(m.mux),
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}
