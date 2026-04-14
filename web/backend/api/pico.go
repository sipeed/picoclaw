package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
)

// registerPicoRoutes binds Pico Channel management endpoints to the ServeMux.
func (h *Handler) registerPicoRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/pico/token", h.handleGetPicoToken)
	mux.HandleFunc("POST /api/pico/token", h.handleRegenPicoToken)
	mux.HandleFunc("POST /api/pico/setup", h.handlePicoSetup)

	// WebSocket proxy: forward /pico/ws to gateway
	// This allows the frontend to connect via the same port as the web UI,
	// avoiding the need to expose extra ports for WebSocket communication.
	mux.HandleFunc("GET /pico/ws", h.handleWebSocketProxy())
}

// createWsProxy creates a reverse proxy to the current gateway WebSocket endpoint.
// The gateway bind host and port are resolved from the latest configuration.
func (h *Handler) createWsProxy(origProtocol string, token string) *httputil.ReverseProxy {
	wsProxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := h.gatewayProxyURL()
			r.SetURL(target)
			r.Out.Header.Set(protocolKey, tokenPrefix+token)
		},
		ModifyResponse: func(r *http.Response) error {
			if prot := r.Header.Values(protocolKey); len(prot) > 0 {
				r.Header.Del(protocolKey)
				if origProtocol != "" {
					r.Header.Set(protocolKey, origProtocol)
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Errorf("Failed to proxy WebSocket: %v", err)
			http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
		},
	}
	return wsProxy
}

// handleWebSocketProxy wraps a reverse proxy to handle WebSocket connections.
// It validates the client token before forwarding; rejects immediately on failure.
func (h *Handler) handleWebSocketProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gateway.mu.Lock()
		ensurePicoTokenCachedLocked(h.configPath)
		cachedPID := gateway.pidData
		trackedCmd := gateway.cmd
		gateway.mu.Unlock()

		gatewayAvailable := false
		// Prefer fresh PID file data when available.
		if pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), nil); pidData != nil {
			gateway.mu.Lock()
			gateway.pidData = pidData
			setGatewayRuntimeStatusLocked("running")
			gatewayAvailable = true
			gateway.mu.Unlock()
		} else if cachedPID != nil {
			// No PID file now: keep availability only while tracked process is
			// still alive (covers short PID-file races at startup/restart).
			if isCmdProcessAliveLocked(trackedCmd) {
				gatewayAvailable = true
			} else {
				gateway.mu.Lock()
				if gateway.cmd == trackedCmd {
					gateway.pidData = nil
					setGatewayRuntimeStatusLocked("stopped")
				}
				gatewayAvailable = gateway.pidData != nil
				gateway.mu.Unlock()
			}
		}

		if !gatewayAvailable {
			logger.Warnf("Gateway not available for WebSocket proxy")
			http.Error(w, "Gateway not available", http.StatusServiceUnavailable)
			return
		}
		prot := r.Header.Values(protocolKey)
		if len(prot) > 0 {
			origProtocol := prot[0]
			newToken := picoComposedToken(prot[0])
			if newToken != "" {
				h.createWsProxy(origProtocol, newToken).ServeHTTP(w, r)
				return
			}
		}

		logger.Warnf("Invalid Pico token: %v", prot)
		http.Error(w, "Invalid Pico token", http.StatusForbidden)
	}
}

// handleGetPicoToken returns the current WS token and URL for the frontend.
//
//	GET /api/pico/token
func (h *Handler) handleGetPicoToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	wsURL := h.buildWsURL(r)

	w.Header().Set("Content-Type", "application/json")
	bc := cfg.Channels.GetByType(config.ChannelPico)
	var picoCfg config.PicoSettings
	if bc != nil {
		bc.Decode(&picoCfg)
	}
	enabled := false
	if bc != nil {
		enabled = bc.Enabled
	}
	json.NewEncoder(w).Encode(map[string]any{
		"token":   picoCfg.Token.String(),
		"ws_url":  wsURL,
		"enabled": enabled,
	})
}

// handleRegenPicoToken generates a new Pico WebSocket token and saves it.
//
//	POST /api/pico/token
func (h *Handler) handleRegenPicoToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	token := generateSecureToken()
	if bc := cfg.Channels.GetByType(config.ChannelPico); bc != nil {
		decoded, err := bc.GetDecoded()
		if err == nil && decoded != nil {
			if settings, ok := decoded.(*config.PicoSettings); ok {
				settings.Token = *config.NewSecureString(token)
			}
		}
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	// Refresh cached pico token.
	gateway.mu.Lock()
	gateway.picoToken = token
	gateway.mu.Unlock()

	wsURL := h.buildWsURL(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":  token,
		"ws_url": wsURL,
	})
}

// EnsurePicoChannel enables the Pico channel with sane defaults if it isn't
// already configured. Returns true when the config was modified.
//
// callerOrigin is the Origin header from the setup request. If non-empty and
// no origins are configured yet, it's written as the allowed origin so the
// WebSocket handshake works for whatever host the caller is on (LAN, custom
// port, etc.). Pass "" when there's no request context.
func (h *Handler) EnsurePicoChannel(callerOrigin string) (bool, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %w", err)
	}

	changed := false

	bc := cfg.Channels.GetByType(config.ChannelPico)
	if bc == nil {
		bc = &config.Channel{Type: config.ChannelPico}
		cfg.Channels["pico"] = bc
	}

	if !bc.Enabled {
		bc.Enabled = true
		changed = true
	}

	if decoded, err := bc.GetDecoded(); err == nil && decoded != nil {
		if picoCfg, ok := decoded.(*config.PicoSettings); ok {
			if picoCfg.Token.String() == "" {
				picoCfg.Token = *config.NewSecureString(generateSecureToken())
				changed = true
			}

			// Seed origins from the request instead of hardcoding ports.
			if len(picoCfg.AllowOrigins) == 0 && callerOrigin != "" {
				picoCfg.AllowOrigins = []string{callerOrigin}
				changed = true
			}
		}
	}

	if changed {
		if err := config.SaveConfig(h.configPath, cfg); err != nil {
			return false, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return changed, nil
}

// handlePicoSetup automatically configures everything needed for the Pico Channel to work.
//
//	POST /api/pico/setup
func (h *Handler) handlePicoSetup(w http.ResponseWriter, r *http.Request) {
	changed, err := h.EnsurePicoChannel(r.Header.Get("Origin"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload config (EnsurePicoChannel may have modified it) and refresh cache.
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	if changed {
		refreshPicoToken(cfg)
	}

	wsURL := h.buildWsURL(r)

	var picoCfg2 config.PicoSettings
	if bc := cfg.Channels.GetByType(config.ChannelPico); bc != nil {
		if decoded, err := bc.GetDecoded(); err == nil && decoded != nil {
			picoCfg2 = *decoded.(*config.PicoSettings)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":   picoCfg2.Token.String(),
		"ws_url":  wsURL,
		"enabled": true,
		"changed": changed,
	})
}

// generateSecureToken creates a random 32-character hex string.
func generateSecureToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to something pseudo-random if crypto/rand fails
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
