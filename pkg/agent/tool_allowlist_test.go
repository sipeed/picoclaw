package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestUnknownAgentToolNames(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
tools: [read_file, web_serach, mcp_github_search]
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{Enabled: true},
			Web: config.WebToolsConfig{
				ToolConfig: config.ToolConfig{Enabled: true},
			},
		},
	}

	unknown := unknownAgentToolNames(cfg, loadAgentDefinition(workspace))
	if len(unknown) != 1 || unknown[0] != "web_serach" {
		t.Fatalf("unknownAgentToolNames() = %v, want [web_serach]", unknown)
	}
}

func TestUnknownAgentMCPServerNames(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
mcpServers: [github, githb]
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			MCP: config.MCPConfig{
				Servers: map[string]config.MCPServerConfig{
					"github": {Enabled: true},
				},
			},
		},
	}

	unknown := unknownAgentMCPServerNames(cfg, loadAgentDefinition(workspace))
	if len(unknown) != 1 || unknown[0] != "githb" {
		t.Fatalf("unknownAgentMCPServerNames() = %v, want [githb]", unknown)
	}
}
