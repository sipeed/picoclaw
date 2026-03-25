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

// clientFacingAuthorityFromRequest returns host[:port] for URLs returned to the
// browser. If Host / X-Forwarded-Host has no explicit port, the listen port is
// omitted so default ports for http/https/ws/wss apply (reverse proxy, FRP, etc.).
func clientFacingAuthorityFromRequest(r *http.Request, listenPort int) string {
	raw := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if raw != "" {
		raw = strings.TrimSpace(strings.Split(raw, ",")[0])
	} else {
		raw = strings.TrimSpace(r.Host)
	}
	if raw == "" {
		return net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return raw
	}
	return net.JoinHostPort(host, port)
}

func (h *Handler) picoURLAuthority(r *http.Request, cfg *config.Config) string {
	webPort := h.serverPort
	if webPort == 0 {
		webPort = 18800
	}
	bind := h.effectiveGatewayBindHost(cfg)
	if bind != "" && bind != "0.0.0.0" {
		return net.JoinHostPort(bind, strconv.Itoa(webPort))
	}
	return clientFacingAuthorityFromRequest(r, webPort)
}

// forwardedRFC7239Proto returns the proto value from the first Forwarded element (RFC 7239), or "".
func forwardedRFC7239Proto(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("Forwarded"))
	if raw == "" {
		return ""
	}
	first := strings.TrimSpace(strings.Split(raw, ",")[0])
	for _, part := range strings.Split(first, ";") {
		part = strings.TrimSpace(part)
		idx := strings.IndexByte(part, '=')
		if idx <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(part[:idx]))
		if key != "proto" {
			continue
		}
		v := strings.Trim(strings.TrimSpace(part[idx+1:]), `"'`)
		return strings.ToLower(v)
	}
	return ""
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

	if p := forwardedRFC7239Proto(r); p != "" {
		if p == "https" || p == "wss" {
			return "wss"
		}
		if p == "http" || p == "ws" {
			return "ws"
		}
	}

	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Ssl")), "on") {
		return "wss"
	}

	if r.TLS != nil {
		return "wss"
	}

	return "ws"
}

func requestHTTPScheme(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		proto := strings.ToLower(strings.TrimSpace(strings.Split(forwarded, ",")[0]))
		if proto == "https" {
			return "https"
		}
	}
	if p := forwardedRFC7239Proto(r); p == "https" {
		return "https"
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Ssl")), "on") {
		return "https"
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func (h *Handler) buildWsURL(r *http.Request, cfg *config.Config) string {
	auth := h.picoURLAuthority(r, cfg)
	return requestWSScheme(r) + "://" + auth + "/pico/ws"
}

func (h *Handler) buildPicoEventsURL(r *http.Request, cfg *config.Config) string {
	auth := h.picoURLAuthority(r, cfg)
	return requestHTTPScheme(r) + "://" + auth + "/pico/events"
}

func (h *Handler) buildPicoSendURL(r *http.Request, cfg *config.Config) string {
	auth := h.picoURLAuthority(r, cfg)
	return requestHTTPScheme(r) + "://" + auth + "/pico/send"
}
