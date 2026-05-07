// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"github.com/sipeed/picoclaw/pkg/agent/interfaces"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Pipeline holds the runtime dependencies used by Pipeline methods.
// It is constructed by runTurn via NewPipeline and passed to sub-methods
// so that the coordinator can delegate phase execution.
type Pipeline struct {
	Bus            interfaces.MessageBus
	Cfg            *config.Config
	ContextManager ContextManager
	Hooks          *HookManager
	Fallback       *providers.FallbackChain
	ChannelManager interfaces.ChannelManager
	MediaStore     media.MediaStore
	Steering       any // TODO: *Steering
	al             *AgentLoop
}

// NewPipeline creates a Pipeline from an AgentLoop instance.
func NewPipeline(al *AgentLoop) *Pipeline {
	return &Pipeline{
		Bus:            al.bus,
		Cfg:            al.GetConfig(),
		ContextManager: al.contextManager,
		Hooks:          al.hooks,
		Fallback:       al.fallback,
		ChannelManager: al.channelManager,
		MediaStore:     al.mediaStore,
		Steering:       al.steering,
		al:             al,
	}
}
