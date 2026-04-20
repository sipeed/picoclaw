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
	mux.HandleFunc("GET /api/pico/info", h.handleGetPicoInfo)
	mux.HandleFunc("POST /api/pico/token", h.handleRegenPicoToken)
	mux.HandleFunc("POST /api/pico/setup", h.handlePicoSetup)

	// WebSocket proxy: forward /pico/ws to gateway
	// This allows the frontend to connect via the same port as the web UI,
	// avoiding the need to expose extra ports for WebSocket communication.
	mux.HandleFunc("GET /pico/ws", h.handleWebSocketProxy())
}

// createWsProxy creates a reverse proxy to the current gateway WebSocket endpoint.
// The gateway bind host and port are resolved from the latest configuration.
func (h *Handler) createWsProxy(origProtocol string, upstreamProtocol string) *httputil.ReverseProxy {
	wsProxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := h.gatewayProxyURL()
			r.SetURL(target)
			r.Out.Header.Del(protocolKey)
			if upstreamProtocol != "" {
				r.Out.Header.Set(protocolKey, upstreamProtocol)
			}
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

func decodePicoSettings(cfg *config.Config) (config.PicoSettings, bool) {
	if cfg == nil {
		return config.PicoSettings{}, false
	}

	bc := cfg.Channels.GetByType(config.ChannelPico)
	if bc == nil {
		return config.PicoSettings{}, false
	}

	var picoCfg config.PicoSettings
	if err := bc.Decode(&picoCfg); err != nil {
		return config.PicoSettings{}, false
	}

	return picoCfg, bc.Enabled
}

func (h *Handler) writePicoInfoResponse(
	w http.ResponseWriter,
	r *http.Request,
	cfg *config.Config,
	changed *bool,
) {
	picoCfg, enabled := decodePicoSettings(cfg)

	resp := map[string]any{
		"ws_url":  h.buildWsURL(r),
		"enabled": enabled,
	}
	if changed != nil {
		resp["changed"] = *changed
	}
	if picoCfg.Token.String() != "" {
		resp["configured"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleWebSocketProxy wraps a reverse proxy to handle WebSocket connections.
// It relies on launcher dashboard auth, then injects the raw pico token only
// on the upstream gateway request.
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

		upstreamProtocol := picoGatewayProtocol()
		if upstreamProtocol == "" {
			logger.Warn("Pico token unavailable for WebSocket proxy")
			http.Error(w, "Pico channel not configured", http.StatusServiceUnavailable)
			return
		}

		var origProtocol string
		if prot := r.Header.Values(protocolKey); len(prot) > 0 {
			origProtocol = prot[0]
		}

		h.createWsProxy(origProtocol, upstreamProtocol).ServeHTTP(w, r)
	}
}

// handleGetPicoInfo returns non-secret Pico connection info for the launcher UI.
//
//	GET /api/pico/info
func (h *Handler) handleGetPicoInfo(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	h.writePicoInfoResponse(w, r, cfg, nil)
}

// handleRegenPicoToken rotates the raw Pico WebSocket token and returns
// non-secret connection info for the launcher UI.
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

	h.writePicoInfoResponse(w, r, cfg, nil)
}

// EnsurePicoChannel enables the Pico channel with sane defaults if it isn't
// already configured. Returns true when the config was modified.
func (h *Handler) EnsurePicoChannel() (bool, error) {
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
	changed, err := h.EnsurePicoChannel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload config (EnsurePicoChannel may have modified it).
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	h.writePicoInfoResponse(w, r, cfg, &changed)
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
