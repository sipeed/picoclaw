package api

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

func (h *Handler) effectiveLauncherPublic() bool {
	if h.serverPublicExplicit {
		return h.serverPublic
	}

	cfg, err := h.loadLauncherConfig()
	if err == nil {
		return cfg.Public
	}

	return h.serverPublic
}

func (h *Handler) gatewayHostOverride() string {
	if h.effectiveLauncherPublic() {
		return "0.0.0.0"
	}
	return ""
}

func (h *Handler) effectiveGatewayBindHost(cfg *config.Config) string {
	if override := h.gatewayHostOverride(); override != "" {
		return override
	}
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Gateway.Host)
}

func gatewayProbeHost(bindHost string) string {
	if bindHost == "" || bindHost == "0.0.0.0" {
		return "127.0.0.1"
	}
	return bindHost
}

func (h *Handler) gatewayProxyURL() *url.URL {
	cfg, err := config.LoadConfig(h.configPath)
	port := 18790
	bindHost := ""
	if err == nil && cfg != nil {
		if cfg.Gateway.Port != 0 {
			port = cfg.Gateway.Port
		}
		bindHost = h.effectiveGatewayBindHost(cfg)
	}

	return &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(gatewayProbeHost(bindHost), strconv.Itoa(port)),
	}
}

func requestHostName(r *http.Request) string {
	reqHost, _, err := net.SplitHostPort(r.Host)
	if err == nil {
		return reqHost
	}
	if strings.TrimSpace(r.Host) != "" {
		return r.Host
	}
	return "127.0.0.1"
}

func requestWSScheme(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		proto := strings.ToLower(strings.TrimSpace(strings.Split(forwarded, ",")[0]))
		if proto == "https" || proto == "wss" {
			return "wss"
		}
		if proto == "http" || proto == "ws" {
			return "ws"
		}
	}

	if r.TLS != nil {
		return "wss"
	}

	return "ws"
}

// buildWsURL returns the correct WebSocket URL for the frontend.
// It intelligently detects the external port it is using
// so custom ports (Docker, Tailscale, etc.) work correctly.
func (h *Handler) buildWsURL(r *http.Request, cfg *config.Config) string {
	// Get host from request (respects X-Forwarded-Host, Host header, etc.)
	host := requestHostName(r)

	// Determine the correct WebSocket port
	var wsPort int

	// Priority 1: Use the actual port from the incoming request (best for custom ports)
	if forwardedPort := strings.TrimSpace(r.Header.Get("X-Forwarded-Port")); forwardedPort != "" {
		if p, err := strconv.Atoi(forwardedPort); err == nil && p > 0 {
			wsPort = p
		}
	}

	// Priority 2: Use the port from the Host header
	if wsPort == 0 {
		if _, portStr, err := net.SplitHostPort(r.Host); err == nil {
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
				wsPort = p
			}
		}
	}

	// Priority 3: Fall back to the web server port (launcher port)
	if wsPort == 0 {
		wsPort = h.serverPort
	}

	// Priority 4: Ultimate fallback to default launcher port
	if wsPort == 0 {
		wsPort = 18800
	}

	scheme := requestWSScheme(r)

	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(wsPort)) + "/pico/ws"
}
