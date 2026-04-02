package agent

import (
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const dynamicMCPToolPrefix = "mcp_"

func warnOnUnknownAgentDeclarations(
	agentID, workspace string,
	cfg *config.Config,
	definition AgentContextDefinition,
) {
	if cfg == nil || frontmatterParseFailed(definition) {
		return
	}

	if unknownTools := unknownAgentToolNames(cfg, definition); len(unknownTools) > 0 {
		logger.WarnCF("agent", "AGENT.md declares unknown tool names",
			map[string]any{
				"agent_id":  agentID,
				"workspace": workspace,
				"tools":     unknownTools,
			})
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

func unknownAgentToolNames(cfg *config.Config, definition AgentContextDefinition) []string {
	if definition.Agent == nil || definition.Agent.Frontmatter.Tools == nil {
		return nil
	}

	known := knownRuntimeToolNames(cfg)
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

func unknownAgentMCPServerNames(cfg *config.Config, definition AgentContextDefinition) []string {
	if cfg == nil || definition.Agent == nil || definition.Agent.Frontmatter.MCPServers == nil {
		return nil
	}

	unknown := make(map[string]struct{})
	for _, raw := range definition.Agent.Frontmatter.MCPServers {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if _, ok := cfg.Tools.MCP.Servers[name]; ok {
			continue
		}
		unknown[name] = struct{}{}
	}

	return sortedKeys(unknown)
}

func knownRuntimeToolNames(cfg *config.Config) map[string]struct{} {
	known := make(map[string]struct{})
	if cfg == nil {
		return known
	}

	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("read_file"), "read_file")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("write_file"), "write_file")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("list_dir"), "list_dir")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("exec"), "exec")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("edit_file"), "edit_file")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("append_file"), "append_file")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("cron"), "cron")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("web"), "web_search")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("web_fetch"), "web_fetch")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("i2c"), "i2c")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("spi"), "spi")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("message"), "message")
	addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("send_file"), "send_file")
	addKnownToolIfEnabled(
		known,
		cfg.Tools.IsToolEnabled("skills") && cfg.Tools.IsToolEnabled("find_skills"),
		"find_skills",
	)
	addKnownToolIfEnabled(
		known,
		cfg.Tools.IsToolEnabled("skills") && cfg.Tools.IsToolEnabled("install_skill"),
		"install_skill",
	)
	if cfg.Tools.IsToolEnabled("subagent") {
		addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("spawn"), "spawn")
		addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("subagent"), "subagent")
		addKnownToolIfEnabled(known, cfg.Tools.IsToolEnabled("spawn_status"), "spawn_status")
	}
	if cfg.Tools.IsToolEnabled("mcp") && cfg.Tools.MCP.Discovery.Enabled {
		addKnownToolIfEnabled(known, cfg.Tools.MCP.Discovery.UseRegex, "tool_search_tool_regex")
		addKnownToolIfEnabled(known, cfg.Tools.MCP.Discovery.UseBM25, "tool_search_tool_bm25")
	}

	return known
}

func addKnownToolIfEnabled(known map[string]struct{}, enabled bool, name string) {
	if !enabled {
		return
	}
	known[name] = struct{}{}
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
	if definition.Agent == nil || definition.Agent.Frontmatter.Tools == nil {
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

	return sortedKeys(allowlist)
}

func resolveAgentMCPServerAllowlist(definition AgentContextDefinition) map[string]struct{} {
	if frontmatterParseFailed(definition) {
		return map[string]struct{}{}
	}
	if definition.Agent == nil || definition.Agent.Frontmatter.MCPServers == nil {
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

func frontmatterParseFailed(definition AgentContextDefinition) bool {
	if definition.Agent == nil {
		return false
	}
	if strings.TrimSpace(definition.Agent.RawFrontmatter) == "" {
		return false
	}
	return strings.TrimSpace(definition.Agent.FrontmatterErr) != ""
}
