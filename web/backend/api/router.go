package api

import (
	"net/http"
	"sync"

	"github.com/sipeed/picoclaw/pkg/credential"
	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

// passphraseState tracks what happened with the last passphrase attempt.
// ""             → no passphrase submitted yet (or just cleared)
// "pending"      → passphrase set, gateway starting
// "failed"       → gateway exited; passphrase likely wrong
type passphraseState string

const (
	passphraseStateNone    passphraseState = ""
	passphraseStatePending passphraseState = "pending"
	passphraseStateFailed  passphraseState = "failed"
)

// Handler serves HTTP API requests.
type Handler struct {
	configPath           string
	serverPort           int
	serverPublic         bool
	serverPublicExplicit bool
	serverCIDRs          []string
	oauthMu              sync.Mutex
	oauthFlows           map[string]*oauthFlow
	oauthState           map[string]string
	passphraseStore      *credential.SecureStore
	passphraseMu         sync.Mutex
	passphraseLastState  passphraseState
}

// NewHandler creates an instance of the API handler.
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath:      configPath,
		serverPort:      launcherconfig.DefaultPort,
		oauthFlows:      make(map[string]*oauthFlow),
		oauthState:      make(map[string]string),
		passphraseStore: credential.NewSecureStore(),
	}
}

// SetServerOptions stores current backend listen options for fallback behavior.
func (h *Handler) SetServerOptions(port int, public bool, publicExplicit bool, allowedCIDRs []string) {
	h.serverPort = port
	h.serverPublic = public
	h.serverPublicExplicit = publicExplicit
	h.serverCIDRs = append([]string(nil), allowedCIDRs...)
}

// SeedPassphrase pre-loads the passphrase into the in-memory SecureStore.
// Call this at startup when the passphrase was supplied via an environment
// variable; after seeding, the caller should clear the env var so it is no
// longer visible in the process environment.
func (h *Handler) SeedPassphrase(passphrase string) {
	h.passphraseStore.SetString(passphrase)
}

// GetPassphrase returns the currently stored passphrase, or "" if not set.
// This satisfies the credential.PassphraseProvider signature so all LoadConfig
// calls in the launcher automatically use the in-memory store.
func (h *Handler) GetPassphrase() string {
	return h.passphraseStore.Get()
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

	// Passphrase management (in-memory store for encrypted credentials)
	h.registerPassphraseRoutes(mux)

	// Model list management
	h.registerModelRoutes(mux)

	// Channel catalog (for frontend navigation/config pages)
	h.registerChannelRoutes(mux)

	// Skills and tools support/actions
	h.registerSkillRoutes(mux)
	h.registerToolRoutes(mux)

	// OS startup / launch-at-login
	h.registerStartupRoutes(mux)

	// Launcher service parameters (port/public)
	h.registerLauncherConfigRoutes(mux)
}
