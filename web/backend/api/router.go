package api

import (
	"net/http"
	"sync"

	"github.com/sipeed/picoclaw/pkg/requestlog"
	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

// Handler serves HTTP API requests.
type Handler struct {
	configPath       string
	serverPort       int
	serverPublic     bool
	serverCIDRs      []string
	requestLogReader *requestlog.Reader
	requestLogger    *requestlog.Logger
	oauthMu          sync.Mutex
	oauthFlows       map[string]*oauthFlow
	oauthState       map[string]string
}

// NewHandler creates an instance of the API handler.
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath:       configPath,
		serverPort:       launcherconfig.DefaultPort,
		requestLogReader: nil,
		requestLogger:    nil,
		oauthFlows:       make(map[string]*oauthFlow),
		oauthState:       make(map[string]string),
	}
}

func (h *Handler) SetRequestLogReader(reader *requestlog.Reader) {
	h.requestLogReader = reader
}

func (h *Handler) SetRequestLogger(logger *requestlog.Logger) {
	h.requestLogger = logger
}

// SetServerOptions stores current backend listen options for fallback behavior.
func (h *Handler) SetServerOptions(port int, public bool, allowedCIDRs []string) {
	h.serverPort = port
	h.serverPublic = public
	h.serverCIDRs = append([]string(nil), allowedCIDRs...)
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

	// Channel catalog (for frontend navigation/config pages)
	h.registerChannelRoutes(mux)

	// OS startup / launch-at-login
	h.registerStartupRoutes(mux)

	// Launcher service parameters (port/public)
	h.registerLauncherConfigRoutes(mux)

	if h.requestLogReader != nil {
		h.registerRequestLogRoutes(mux)
	}
}
