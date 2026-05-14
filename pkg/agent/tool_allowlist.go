package agent

import (
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/tools"
)

const dynamicMCPToolPrefix = "mcp_"

func normalizeMCPServerName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizedMCPServerNameSet(
	servers map[string]config.MCPServerConfig,
) map[string]struct{} {
	normalized := make(map[string]struct{}, len(servers))
	for serverName := range servers {
		name := normalizeMCPServerName(serverName)
		if name == "" {
			continue
		}
		normalized[name] = struct{}{}
	}
	return normalized
}

func warnOnUnknownAgentToolDeclarations(
	agentID, workspace string,
	definition AgentContextDefinition,
	registry *tools.ToolRegistry,
) {
	if registry == nil || frontmatterParseFailed(definition) {
		return
	}

	if unknownTools := unknownAgentToolNames(registry, definition); len(unknownTools) > 0 {
		logger.WarnCF("agent", "AGENT.md declares unregistered tool names",
			map[string]any{
				"agent_id":  agentID,
				"workspace": workspace,
				"tools":     unknownTools,
			})
	}
}

func warnOnUnknownAgentMCPServerDeclarations(
	agentID, workspace string,
	cfg *config.Config,
	definition AgentContextDefinition,
) {
	if cfg == nil || frontmatterParseFailed(definition) {
		return
	}

	if unknownServers := unknownAgentMCPServerNames(cfg, definition); len(unknownServers) > 0 {
		logger.WarnCF("agent", "AGENT.md declares unknown MCP server names",
			map[string]any{
				"agent_id":    agentID,
				"workspace":   workspace,
				"mcp_servers": unknownServers,
			})
	}
}

func unknownAgentToolNames(
	registry *tools.ToolRegistry,
	definition AgentContextDefinition,
) []string {
	if definition.Agent == nil || definition.Agent.Frontmatter.Tools == nil {
		return nil
	}

	known := registeredRuntimeToolNames(registry)
	unknown := make(map[string]struct{})
	for _, raw := range definition.Agent.Frontmatter.Tools {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || strings.HasPrefix(name, dynamicMCPToolPrefix) {
			continue
		}
		if _, ok := known[name]; ok {
			continue
		}
		unknown[name] = struct{}{}
	}

	return sortedKeys(unknown)
}

func registeredRuntimeToolNames(registry *tools.ToolRegistry) map[string]struct{} {
	known := make(map[string]struct{})
	if registry == nil {
		return known
	}
	for _, raw := range registry.List() {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		known[name] = struct{}{}
	}
	return known
}

func unknownAgentMCPServerNames(cfg *config.Config, definition AgentContextDefinition) []string {
	if cfg == nil || definition.Agent == nil || definition.Agent.Frontmatter.MCPServers == nil {
		return nil
	}

	knownServers := normalizedMCPServerNameSet(cfg.Tools.MCP.Servers)
	unknown := make(map[string]struct{})
	for _, raw := range definition.Agent.Frontmatter.MCPServers {
		name := normalizeMCPServerName(raw)
		if name == "" {
			continue
		}
		if _, ok := knownServers[name]; ok {
			continue
		}
		unknown[name] = struct{}{}
	}

	return sortedKeys(unknown)
}

func sortedKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func resolveAgentToolAllowlist(definition AgentContextDefinition) []string {
	if frontmatterParseFailed(definition) {
		return []string{}
	}
	if definition.Agent == nil || !frontmatterDeclaresField(definition, "tools") {
		return nil
	}

	allowlist := make(map[string]struct{}, len(definition.Agent.Frontmatter.Tools))
	for _, raw := range definition.Agent.Frontmatter.Tools {
		trimmed := strings.ToLower(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}
		allowlist[trimmed] = struct{}{}
	}

	if len(allowlist) == 0 {
		return []string{}
	}

	return sortedKeys(allowlist)
}

func resolveAgentMCPServerAllowlist(definition AgentContextDefinition) map[string]struct{} {
	if frontmatterParseFailed(definition) {
		return map[string]struct{}{}
	}
	if definition.Agent == nil || !frontmatterDeclaresField(definition, "mcpServers") {
		return nil
	}

	allowlist := make(map[string]struct{}, len(definition.Agent.Frontmatter.MCPServers))
	for _, raw := range definition.Agent.Frontmatter.MCPServers {
		trimmed := strings.ToLower(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}
		allowlist[trimmed] = struct{}{}
	}

	return allowlist
}

func frontmatterDeclaresField(definition AgentContextDefinition, field string) bool {
	if definition.Agent == nil || definition.Agent.Frontmatter.Fields == nil {
		return false
	}
	_, ok := definition.Agent.Frontmatter.Fields[field]
	return ok
}

func frontmatterParseFailed(definition AgentContextDefinition) bool {
	if definition.Agent == nil {
		return false
	}
	if strings.TrimSpace(definition.Agent.RawFrontmatter) == "" {
		return false
	}
	return strings.TrimSpace(definition.Agent.FrontmatterErr) != ""
}
