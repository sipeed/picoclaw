// Package constants provides shared constants across the codebase.
package constants

import "strings"

// internalChannels defines channels that are used for internal communication
// and should not be exposed to external users or recorded as last active channel.
var internalChannels = map[string]struct{}{
	"cli":      {},
	"system":   {},
	"subagent": {},
	"launcher": {},
}

// IsInternalChannel returns true if the channel is an internal channel.
// Supports compound names like "launcher:chat" by checking the prefix before ":".
func IsInternalChannel(channel string) bool {
	if _, found := internalChannels[channel]; found {
		return true
	}
	// Check prefix for compound channel names (e.g. "launcher:chat")
	if idx := strings.IndexByte(channel, ':'); idx > 0 {
		_, found := internalChannels[channel[:idx]]
		return found
	}
	return false
}
