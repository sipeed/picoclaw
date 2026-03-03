package commands

import "github.com/sipeed/picoclaw/pkg/config"

// Deps provides runtime data to command handlers without importing
// agent or channel packages. Function fields are called at handler
// invocation time, not at construction time, so late-bound values
// (e.g. channelManager set after NewAgentLoop) are visible.
type Deps struct {
	Config             *config.Config
	GetModelInfo       func() (name, provider string)
	ListAgentIDs       func() []string
	GetEnabledChannels func() []string
	SwitchModel        func(value string) (oldModel string, err error)
	SwitchChannel      func(value string) error
}
