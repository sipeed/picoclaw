package agent

import "github.com/sipeed/picoclaw/pkg/tools"

func resetSentTrackingTools(agent *AgentInstance, sessionKey string) {
	if agent == nil {
		return
	}
	for _, name := range agent.Tools.List() {
		tool, ok := agent.Tools.Get(name)
		if !ok {
			continue
		}
		if resetter, ok := tool.(interface{ ResetSentInRound(sessionKey string) }); ok {
			resetter.ResetSentInRound(sessionKey)
		}
	}
}

func anySentTrackingToolSentTo(agent *AgentInstance, sessionKey, channel, chatID string) bool {
	if agent == nil {
		return false
	}
	for _, name := range agent.Tools.List() {
		tool, ok := agent.Tools.Get(name)
		if !ok {
			continue
		}
		if tracker, ok := tool.(interface {
			HasSentTo(sessionKey, channel, chatID string) bool
		}); ok {
			if tracker.HasSentTo(sessionKey, channel, chatID) {
				return true
			}
		}
	}
	return false
}

var _ tools.Tool = (*tools.MessageTool)(nil)
