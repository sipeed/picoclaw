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
	if definition.Agent == nil || definition.Agent.Frontmatter.ToolPolicy == nil {
		return nil
	}

	known := registeredRuntimeToolNames(registry)
	unknown := make(map[string]struct{})
	for _, raw := range definition.Agent.Frontmatter.ToolPolicy.Allow {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || strings.HasPrefix(name, dynamicMCPToolPrefix) || containsGlobMeta(name) {
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
	if cfg == nil || definition.Agent == nil || definition.Agent.Frontmatter.MCPPolicy == nil {
		return nil
	}

	knownServers := normalizedMCPServerNameSet(cfg.Tools.MCP.Servers)
	unknown := make(map[string]struct{})
	for _, raw := range definition.Agent.Frontmatter.MCPPolicy.Allow {
		name := normalizeMCPServerName(raw)
		if name == "" || containsGlobMeta(name) {
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

func resolveAgentToolPolicy(definition AgentContextDefinition) *PatternPolicy {
	if frontmatterParseFailed(definition) {
		return &PatternPolicy{Allow: []string{}, form: patternPolicyFormList}
	}
	if definition.Agent == nil || !frontmatterDeclaresField(definition, "tools") {
		return nil
	}
	return normalizePatternPolicy(definition.Agent.Frontmatter.ToolPolicy)
}

func resolveAgentMCPServerPolicy(definition AgentContextDefinition) *PatternPolicy {
	if frontmatterParseFailed(definition) {
		return &PatternPolicy{Allow: []string{}, form: patternPolicyFormList}
	}
	if definition.Agent == nil || !frontmatterDeclaresField(definition, "mcpServers") {
		return nil
	}
	return normalizePatternPolicy(definition.Agent.Frontmatter.MCPPolicy)
}

func normalizePatternPolicy(policy *PatternPolicy) *PatternPolicy {
	if policy == nil {
		return nil
	}

	normalized := &PatternPolicy{form: policy.form}
	normalized.Allow = normalizePatterns(policy.Allow)
	normalized.Deny = normalizePatterns(policy.Deny)

	if policy.form == patternPolicyFormList && len(normalized.Allow) == 0 {
		normalized.Allow = []string{}
	}

	return normalized
}

func normalizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(patterns))
	result := make([]string, 0, len(patterns))
	for _, raw := range patterns {
		trimmed := strings.ToLower(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	sort.Strings(result)
	return result
}

func containsGlobMeta(value string) bool {
	return strings.ContainsAny(value, "*?[")
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
