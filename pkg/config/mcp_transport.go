package config

import "strings"

// NormalizeMCPTransportType canonicalizes MCP transport names used in config.
// "http" is PicoClaw's streamable HTTP request-response mode, and
// "streamable-http" is accepted as an explicit alias for the same transport.
func NormalizeMCPTransportType(transport string) string {
	normalized := strings.ToLower(strings.TrimSpace(transport))

	switch normalized {
	case "streamable-http", "streamable_http", "streamablehttp":
		return "http"
	default:
		return normalized
	}
}

// EffectiveMCPTransportType returns the normalized configured transport, or the
// inferred default when the config leaves Type empty.
func EffectiveMCPTransportType(server MCPServerConfig) string {
	if transport := NormalizeMCPTransportType(server.Type); transport != "" {
		return transport
	}
	if server.URL != "" {
		return "sse"
	}
	if server.Command != "" {
		return "stdio"
	}
	return ""
}
