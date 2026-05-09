// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"fmt"

	"github.com/sipeed/picoclaw/pkg/audio/asr"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	registry := al.GetRegistry()
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok {
			agent.Tools.Register(tool)
		}
	}
}

func (al *AgentLoop) SetChannelManager(cm *channels.Manager) {
	al.channelManager = cm
}

func (al *AgentLoop) GetRegistry() *AgentRegistry {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.registry
}

// GrantPermission grants permission for a path to a specific agent.
// agentID is the normalized agent ID (e.g. "main"), path is the file path,
// duration is "once" or "session".
// Returns an error if the agent is not found or the permission cache is not initialized.
func (al *AgentLoop) GrantPermission(agentID, path, duration string) error {
	registry := al.GetRegistry()
	if registry == nil {
		return fmt.Errorf("agent registry not initialized")
	}
	agent, ok := registry.GetAgent(agentID)
	if !ok {
		return fmt.Errorf("agent %q not found", agentID)
	}
	return agent.GrantPermission(path, duration)
}

func (al *AgentLoop) GetConfig() *config.Config {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.cfg
}

func (al *AgentLoop) SetMediaStore(s media.MediaStore) {
	al.mediaStore = s

	// Propagate store to all registered tools that can emit media.
	registry := al.GetRegistry()
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok {
			agent.Tools.SetMediaStore(s)
		}
	}
	registry.ForEachTool("send_tts", func(t tools.Tool) {
		if st, ok := t.(*tools.SendTTSTool); ok {
			st.SetMediaStore(s)
		}
	})
}

func (al *AgentLoop) SetTranscriber(t asr.Transcriber) {
	al.transcriber = t
}

func (al *AgentLoop) SetReloadFunc(fn func() error) {
	al.reloadFunc = fn
}

func (al *AgentLoop) RecordLastChannel(channel string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChannel(channel)
}

func (al *AgentLoop) RecordLastChatID(chatID string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChatID(chatID)
}

func (al *AgentLoop) GetStartupInfo() map[string]any {
	info := make(map[string]any)

	registry := al.GetRegistry()
	agent := registry.GetDefaultAgent()
	if agent == nil {
		return info
	}

	// Tools info
	toolsList := agent.Tools.List()
	info["tools"] = map[string]any{
		"count": len(toolsList),
		"names": toolsList,
	}

	// Skills info
	info["skills"] = agent.ContextBuilder.GetSkillsInfo()

	// Agents info
	info["agents"] = map[string]any{
		"count": len(registry.ListAgentIDs()),
		"ids":   registry.ListAgentIDs(),
	}

	return info
}

func (al *AgentLoop) GetMainSubagentTasks(channel, chatID string) []tools.SubagentTask {
	al.mu.RLock()
	manager := al.subagents
	al.mu.RUnlock()
	if manager == nil {
		return nil
	}

	all := manager.ListTaskCopies()
	if channel == "" && chatID == "" {
		return all
	}

	filtered := make([]tools.SubagentTask, 0, len(all))
	for _, task := range all {
		if channel != "" && task.OriginChannel != "" && task.OriginChannel != channel {
			continue
		}
		if chatID != "" && task.OriginChatID != "" && task.OriginChatID != chatID {
			continue
		}
		filtered = append(filtered, task)
	}
	return filtered
}
