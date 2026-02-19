// Package constants provides shared constants across the codebase.
package constants

// internalChannels defines channels that are used for internal communication
// and should not be exposed to external users or recorded as last active channel.
var internalChannels = map[string]struct{}{
	"cli":      {},
	"system":   {},
	"subagent": {},
}

// IsInternalChannel returns true if the channel is an internal channel.
func IsInternalChannel(channel string) bool {
	_, found := internalChannels[channel]
	return found
}
