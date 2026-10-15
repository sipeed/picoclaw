package main

// Blank import so the launcher process registers every built-in channel type
// (including the first-party qq_c2c channel). Without this the launcher loads
// the config with an empty channel-type registry and refuses to auto-start the
// gateway with: channel "qq_c2c" has unknown type "qq_c2c".
import (
	_ "github.com/sipeed/picoclaw/pkg/channels/builtin"
)
