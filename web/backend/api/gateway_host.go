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

func firstForwardedValue(raw string) string {
	if raw == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(raw, ",")[0])
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

func requestWSAuthority(r *http.Request) string {
	scheme := requestWSScheme(r)
	authority := firstForwardedValue(r.Header.Get("X-Forwarded-Host"))
	if authority == "" {
		authority = strings.TrimSpace(r.Host)
	}
	if authority == "" {
		return "127.0.0.1"
	}

	parsed, err := url.Parse("//" + authority)
	if err != nil {
		return authority
	}
	if parsed.Port() != "" {
		return authority
	}

	forwardedPort := firstForwardedValue(r.Header.Get("X-Forwarded-Port"))
	if forwardedPort == "" {
		return authority
	}
	if (scheme == "wss" && forwardedPort == "443") || (scheme == "ws" && forwardedPort == "80") {
		return authority
	}
	if parsed.Hostname() == "" {
		return authority
	}
	return net.JoinHostPort(parsed.Hostname(), forwardedPort)
}

func buildWsURL(r *http.Request) string {
	return requestWSScheme(r) + "://" + requestWSAuthority(r) + "/pico/ws"
}
