package channels

import (
	"net/http"
)

// APIVersionMiddleware provides version negotiation and versioned routing for APIs.
type APIVersionMiddleware struct {
	Version string
	Handler http.Handler
}

// ServeHTTP implements the http.Handler interface with version negotiation.
func (avm *APIVersionMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set response headers for version negotiation
	w.Header().Set("API-Version", avm.Version)

	// Add CORS headers for API clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, API-Version")
	w.Header().Set("Access-Control-Expose-Headers", "API-Version")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Continue with the wrapped handler
	avm.Handler.ServeHTTP(w, r)
}

// NewVersionedHandler creates a new APIVersionMiddleware instance.
func NewVersionedHandler(version string, handler http.Handler) *APIVersionMiddleware {
	return &APIVersionMiddleware{
		Version: version,
		Handler: handler,
	}
}

// WithVersionPrefix returns a versioned path for a given endpoint.
// For example, if the version is "v1" and the endpoint is "/webhook/telegram",
// it would return "/v1/webhook/telegram".
func WithVersionPrefix(version, endpoint string) string {
	if version == "" {
		return endpoint
	}
	if endpoint == "" {
		return "/" + version
	}
	if endpoint[0] != '/' {
		endpoint = "/" + endpoint
	}
	return "/" + version + endpoint
}
