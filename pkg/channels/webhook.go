package channels

import "net/http"

// WebhookHandler is an optional interface for channels that receive messages
// via HTTP webhooks. Manager discovers channels implementing this interface
// and registers them on the shared HTTP server.
type WebhookHandler interface {
	// WebhookPath returns the path to mount this handler on the shared server.
	// Examples: "/webhook/line", "/webhook/wecom"
	WebhookPath() string
	http.Handler // ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// HealthChecker is an optional interface for channels that expose
// a health check endpoint on the shared HTTP server.
type HealthChecker interface {
	HealthPath() string
	HealthHandler(w http.ResponseWriter, r *http.Request)
}
