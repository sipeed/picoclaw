package channels

import (
	"net/http"
	"strings"
)

// APIVersionNegotiator handles API version negotiation based on request headers, path, or query params
type APIVersionNegotiator struct {
	// ValidVersions holds the list of API versions the server supports
	ValidVersions []string
	// DefaultVersion specifies which version to use when none is indicated
	DefaultVersion string
}

// NewAPIVersionNegotiator creates a new instance with valid versions and a default
func NewAPIVersionNegotiator(validVersions []string, defaultVersion string) *APIVersionNegotiator {
	if len(validVersions) == 0 {
		validVersions = []string{"v1"}
	}
	if defaultVersion == "" {
		defaultVersion = "v1"
	}
	return &APIVersionNegotiator{
		ValidVersions:  validVersions,
		DefaultVersion: defaultVersion,
	}
}

// DetermineVersion extracts and validates API version from the request
func (avn *APIVersionNegotiator) DetermineVersion(r *http.Request) string {
	// Check for API-Version header first
	if version := r.Header.Get("API-Version"); version != "" {
		if avn.isValidVersion(version) {
			return version
		}
	}

	// Check for X-API-Version header as fallback
	if version := r.Header.Get("X-API-Version"); version != "" {
		if avn.isValidVersion(version) {
			return version
		}
	}

	// Check URL path for version prefix (e.g. /v2/webhook/telegram)
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 1 {
		pathVersion := parts[1]
		if avn.isValidVersion(pathVersion) {
			return pathVersion
		}
	}

	// Check query parameter as last resort
	if version := r.URL.Query().Get("api-version"); version != "" {
		if avn.isValidVersion(version) {
			return version
		}
	}

	// Return default if no version could be negotiated
	return avn.DefaultVersion
}

// isValidVersion checks if the given version string is one of the supported versions
func (avn *APIVersionNegotiator) isValidVersion(version string) bool {
	for _, v := range avn.ValidVersions {
		if v == version {
			return true
		}
	}
	return false
}

// VersionedHandlerWithNegotiation creates an HTTP handler that performs version negotiation
func VersionedHandlerWithNegotiation(negotiator *APIVersionNegotiator, v1Handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := negotiator.DetermineVersion(r)
		// Set the negotiated version on the response
		w.Header().Set("API-Version", version)
		
		// For now, we'll serve v1 regardless of negotiated version
		// In future, could route to different handlers based on version
		v1Handler.ServeHTTP(w, r)
	})
}
