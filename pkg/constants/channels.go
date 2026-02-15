// Package constants provides shared constants across the codebase.
package constants

// InternalChannels defines channels that are used for internal communication
// and should not be exposed to external users or recorded as last active channel.
var InternalChannels = map[string]bool{
	"cli":      true,
	"system":   true,
	"subagent": true,
}

// IsInternalChannel returns true if the channel is an internal channel.
func IsInternalChannel(channel string) bool {
	return InternalChannels[channel]
}
