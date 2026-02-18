package command

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// AgentState provides access to agent internals needed by commands.
type AgentState interface {
	GetModel() string
	SetModel(model string)
	GetChannelManager() interface{} // Returns channels.Manager (interface to avoid circular dep)
}

// Command represents an executable command.
type Command interface {
	Name() string
	Description() string
	Execute(ctx context.Context, agent AgentState, args []string, msg bus.InboundMessage) (string, error)
}
