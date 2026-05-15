// Package identity provides unified user identity utilities for PicoClaw.
// It introduces a canonical "platform:id" format and matching logic
// that is backward-compatible with all legacy allow-list formats.
package identity

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// BuildCanonicalID constructs a canonical "platform:id" identifier.
// Both platform and platformID are lowercased and trimmed.
func BuildCanonicalID(platform, platformID string) string {
	p := strings.ToLower(strings.TrimSpace(platform))
	id := strings.TrimSpace(platformID)
	if p == "" || id == "" {
		return ""
	}
	return p + ":" + id
}

// ParseCanonicalID splits a canonical ID ("platform:id") into its parts.
// Returns ok=false if the input does not contain a colon separator.
func ParseCanonicalID(canonical string) (platform, id string, ok bool) {
	canonical = strings.TrimSpace(canonical)
	idx := strings.Index(canonical, ":")
	if idx <= 0 || idx == len(canonical)-1 {
		return "", "", false
	}
	return canonical[:idx], canonical[idx+1:], true
}

// MatchAllowed checks whether the given sender matches a single allow-list entry.
// It is backward-compatible with all legacy formats:
//
//   - "123456"              → matches sender.PlatformID
//   - "@alice"              → matches sender.Username
//   - "123456|alice"        → matches PlatformID or Username
//   - "telegram:123456"     → exact match on sender.CanonicalID
func MatchAllowed(sender bus.SenderInfo, allowed string) bool {
	allowed = strings.TrimSpace(allowed)
	if allowed == "" {
		return false
	}

	// Keep track of explicit username format
	isAtUsername := strings.HasPrefix(allowed, "@")

	// Strip leading "@" for username matching
	trimmed := strings.TrimPrefix(allowed, "@")

	// Try canonical match first: "platform:id" format.
	// Skip when the entry starts with "@" — the "@" prefix signals a
	// username or Matrix MXID (e.g. "@alice:matrix.org"), not a platform:id pair.
	if !isAtUsername {
		if platform, id, ok := ParseCanonicalID(allowed); ok {
			// Only treat as canonical if the platform portion looks like a known platform name
			// (not a pure-numeric string, which could be a compound ID)
			if !isNumeric(platform) {
				candidate := BuildCanonicalID(platform, id)
				if candidate != "" && sender.CanonicalID != "" {
					return strings.EqualFold(sender.CanonicalID, candidate)
				}
				// If sender has no canonical ID, try matching platform + platformID
				return strings.EqualFold(platform, sender.Platform) &&
					sender.PlatformID == id
			}
		}
	}

	// Split compound "id|username" format
	allowedID := trimmed
	allowedUser := ""
	if idx := strings.Index(trimmed, "|"); idx > 0 {
		allowedID = trimmed[:idx]
		allowedUser = trimmed[idx+1:]
	}

	// Match against PlatformID
	if sender.PlatformID != "" && sender.PlatformID == allowedID {
		return true
	}

	// Match against Username only when explicitly requested via "@username"
	if isAtUsername && sender.Username != "" && sender.Username == trimmed {
		return true
	}

	// Match the full "@"-prefixed entry against PlatformID or Username.
	// Needed for Matrix MXIDs where both the config entry and the
	// sender identifiers include the "@" prefix (e.g. "@alice:matrix.org").
	if isAtUsername {
		if sender.PlatformID != "" && sender.PlatformID == allowed {
			return true
		}
		if sender.Username != "" && sender.Username == allowed {
			return true
		}
	}

	// Match compound sender format against allowed parts
	if allowedUser != "" && sender.PlatformID != "" && sender.PlatformID == allowedID {
		return true
	}
	if allowedUser != "" && sender.Username != "" && sender.Username == allowedUser {
		return true
	}

	return false
}

// isNumeric returns true if s consists entirely of digits, allowing for an optional leading minus sign
// (required for Telegram group/channel IDs like -1001234567890).
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	start := 0
	if s[0] == '-' && len(s) > 1 {
		start = 1
	}
	for i := start; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
