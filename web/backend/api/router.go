package api

import (
	"net/http"
	"sync"
)

// Handler serves HTTP API requests.
type Handler struct {
	configPath string
	oauthMu    sync.Mutex
	oauthFlows map[string]*oauthFlow
	oauthState map[string]string
}

// NewHandler creates an instance of the API handler.
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath: configPath,
		oauthFlows: make(map[string]*oauthFlow),
		oauthState: make(map[string]string),
	}
}

// RegisterRoutes binds all API endpoint handlers to the ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Config CRUD
	h.registerConfigRoutes(mux)

	// Pico Channel (WebSocket chat)
	h.registerPicoRoutes(mux)

	// Gateway process lifecycle
	h.registerGatewayRoutes(mux)

	// Session history
	h.registerSessionRoutes(mux)

	// OAuth login and credential management
	h.registerOAuthRoutes(mux)

	// Model list management
	h.registerModelRoutes(mux)

	// Channel management
	h.registerChannelRoutes(mux)
}
