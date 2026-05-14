package agent

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/routing"
)

// AgentDescriptor is the structured discovery payload injected into each
// agent's system prompt so the LLM can choose a peer by identity.
type AgentDescriptor struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListAgents returns structured descriptors for every agent in the current
// PicoClaw instance. The current workspace, when provided, is used only to
// order the matching agent first for prompt readability.
func (r *AgentRegistry) ListAgents(workspace string) []AgentDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	selfWorkspace := cleanWorkspacePath(workspace)
	descriptors := make([]AgentDescriptor, 0, len(ids))
	for _, id := range ids {
		agent := r.agents[id]
		if agent == nil {
			continue
		}
		descriptors = append(descriptors, r.buildAgentDescriptorLocked(agent))
	}

	if selfWorkspace == "" {
		return descriptors
	}

	sort.SliceStable(descriptors, func(i, j int) bool {
		leftSelf := cleanWorkspacePath(
			r.workspaceForAgentIDLocked(descriptors[i].ID),
		) == selfWorkspace
		rightSelf := cleanWorkspacePath(
			r.workspaceForAgentIDLocked(descriptors[j].ID),
		) == selfWorkspace
		if leftSelf != rightSelf {
			return leftSelf
		}
		return descriptors[i].ID < descriptors[j].ID
	})

	return descriptors
}

// ListSpawnableAgents returns descriptors only when the current agent can call
// spawn, and only for peers it is allowed to spawn. Restricted peers are
// intentionally omitted from discovery.
func (r *AgentRegistry) ListSpawnableAgents(agentID string) []AgentDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parentID := routing.NormalizeAgentID(agentID)
	parent, ok := r.agents[parentID]
	if !ok || parent == nil {
		return nil
	}
	if !agentHasSpawnTool(parent) {
		return nil
	}

	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		if id == parentID {
			continue
		}
		if !agentAllowsSubagent(parent, id) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)

	descriptors := make([]AgentDescriptor, 0, len(ids))
	for _, id := range ids {
		agent := r.agents[id]
		if agent == nil {
			continue
		}
		descriptors = append(descriptors, r.buildAgentDescriptorLocked(agent))
	}
	return descriptors
}

// GetAgentDescriptor returns the structured discovery payload for one agent.
func (r *AgentRegistry) GetAgentDescriptor(agentID string) (*AgentDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id := routing.NormalizeAgentID(agentID)
	agent, ok := r.agents[id]
	if !ok || agent == nil {
		return nil, false
	}

	descriptor := r.buildAgentDescriptorLocked(agent)
	return &descriptor, true
}

func (r *AgentRegistry) buildAgentDescriptorLocked(agent *AgentInstance) AgentDescriptor {
	definition := loadAgentDefinition(agent.Workspace)
	name, description := descriptorIdentity(agent.ID, definition)

	return AgentDescriptor{
		ID:          agent.ID,
		Name:        name,
		Description: description,
	}
}

func descriptorIdentity(agentID string, definition AgentContextDefinition) (string, string) {
	name := agentID
	description := ""
	if definition.Agent != nil {
		if trimmed := strings.TrimSpace(definition.Agent.Frontmatter.Name); trimmed != "" {
			name = trimmed
		}
		if trimmed := strings.TrimSpace(definition.Agent.Frontmatter.Description); trimmed != "" {
			description = trimmed
		}
	}

	if description == "" &&
		definition.Agent != nil {
		if definition.Source == AgentDefinitionSourceAgent {
			description = firstNonEmptyLine(definition.Agent.Body)
		} else if definition.Source == AgentDefinitionSourceAgents {
			description = firstMeaningfulParagraph(definition.Agent.Body)
		}
	}

	return name, description
}

func firstNonEmptyLine(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstMeaningfulParagraph(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	paragraphs := strings.Split(content, "\n\n")
	for _, paragraph := range paragraphs {
		lines := strings.Split(paragraph, "\n")
		parts := make([]string, 0, len(lines))
		inFence := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inFence = !inFence
				continue
			}
			if inFence || trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
				trimmed = strings.TrimSpace(trimmed[2:])
			}
			parts = append(parts, trimmed)
		}
		if len(parts) == 0 {
			continue
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func (r *AgentRegistry) workspaceForAgentIDLocked(agentID string) string {
	agent, ok := r.agents[routing.NormalizeAgentID(agentID)]
	if !ok || agent == nil {
		return ""
	}
	return agent.Workspace
}

func (r *AgentRegistry) defaultAgentIDLocked() string {
	if _, ok := r.agents[routing.DefaultAgentID]; ok {
		return routing.DefaultAgentID
	}
	if r.cfg != nil && len(r.cfg.Agents.List) > 0 {
		for _, agentCfg := range r.cfg.Agents.List {
			if !agentCfg.Default {
				continue
			}
			id := routing.NormalizeAgentID(agentCfg.ID)
			if _, ok := r.agents[id]; ok {
				return id
			}
		}
		id := routing.NormalizeAgentID(r.cfg.Agents.List[0].ID)
		if _, ok := r.agents[id]; ok {
			return id
		}
	}
	for id := range r.agents {
		return id
	}
	return ""
}

func cleanWorkspacePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func formatAgentDiscoverySection(agents []AgentDescriptor) string {
	if len(agents) == 0 {
		return ""
	}

	payload := struct {
		Agents []AgentDescriptor `json:"agents"`
	}{
		Agents: agents,
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}

	var header strings.Builder
	header.WriteString("# Agent Discovery\n\n")
	header.WriteString("This registry lists the peer agents this agent is permitted to spawn.\n")
	header.WriteString(
		"Choose a peer based on its description. Use only agent IDs listed here when calling spawn.\n\n",
	)
	header.WriteString("```json\n")
	header.Write(encoded)
	header.WriteString("\n```")

	return header.String()
}
