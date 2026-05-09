package agent

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	agenttools "github.com/sipeed/picoclaw/pkg/tools"
)

type allowlistTestTool struct {
	name string
}

func (t *allowlistTestTool) Name() string { return t.name }

func (t *allowlistTestTool) Description() string { return "test tool" }

func (t *allowlistTestTool) Parameters() map[string]any {
	return map[string]any{"type": "object"}
}

func (t *allowlistTestTool) Execute(
	_ context.Context,
	_ map[string]any,
) *agenttools.ToolResult {
	return agenttools.NewToolResult("ok")
}

func TestUnknownAgentToolNames(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
tools: [read_file, web_serach, mcp_github_search]
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	registry := agenttools.NewToolRegistry()
	registry.Register(&allowlistTestTool{name: "read_file"})
	registry.Register(&allowlistTestTool{name: "web_search"})

	unknown := unknownAgentToolNames(registry, loadAgentDefinition(workspace))
	if len(unknown) != 1 || unknown[0] != "web_serach" {
		t.Fatalf("unknownAgentToolNames() = %v, want [web_serach]", unknown)
	}
}

func TestUnknownAgentToolNamesUsesRegisteredRuntimeTools(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
tools: [serial, reaction, send_tts, load_image, delegate, made_up]
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	registry := agenttools.NewToolRegistry()
	for _, name := range []string{"serial", "reaction", "send_tts", "load_image", "delegate"} {
		registry.Register(&allowlistTestTool{name: name})
	}

	unknown := unknownAgentToolNames(registry, loadAgentDefinition(workspace))
	if len(unknown) != 1 || unknown[0] != "made_up" {
		t.Fatalf("unknownAgentToolNames() = %v, want [made_up]", unknown)
	}
}

func TestResolveAgentToolPolicyDistinguishesMissingAndEmptyToolsField(t *testing.T) {
	tests := []struct {
		name      string
		agentMD   string
		wantNil   bool
		wantEmpty bool
	}{
		{
			name: "missing tools field allows all tools",
			agentMD: `---
name: pico
---
# Agent
`,
			wantNil: true,
		},
		{
			name: "explicit empty tools list blocks all tools",
			agentMD: `---
tools: []
---
# Agent
`,
			wantEmpty: true,
		},
		{
			name: "blank tools field blocks all tools",
			agentMD: `---
tools:
---
# Agent
`,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := setupWorkspace(t, map[string]string{
				"AGENT.md": tt.agentMD,
			})
			defer cleanupWorkspace(t, workspace)

			policy := resolveAgentToolPolicy(loadAgentDefinition(workspace))

			if tt.wantNil {
				if policy != nil {
					t.Fatalf("resolveAgentToolPolicy() = %v, want nil", policy)
				}
				return
			}

			if policy == nil {
				t.Fatal("resolveAgentToolPolicy() = nil, want explicit empty allowlist")
			}
			if len(policy.Allow) != 0 {
				t.Fatalf("resolveAgentToolPolicy() = %+v, want empty allowlist", policy)
			}
		})
	}
}

func TestResolveAgentToolPolicy_ObjectAllowDeny(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
tools:
  allow:
    - mcp_*
    - web_fetch
  deny:
    - mcp_gpt_researcher_*
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	policy := resolveAgentToolPolicy(loadAgentDefinition(workspace))
	if policy == nil {
		t.Fatal("resolveAgentToolPolicy() = nil, want policy")
	}
	if got, want := policy.Allow, []string{"mcp_*", "web_fetch"}; len(got) != len(want) ||
		got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("allow = %v, want %v", got, want)
	}
	if got, want := policy.Deny, []string{"mcp_gpt_researcher_*"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("deny = %v, want %v", got, want)
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

func TestUnknownAgentMCPServerNamesMatchesConfigCaseInsensitively(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
mcpServers: [github, FileSystem, slak]
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	cfg := &config.Config{
		Tools: config.ToolsConfig{
			MCP: config.MCPConfig{
				Servers: map[string]config.MCPServerConfig{
					"GitHub":     {Enabled: true},
					"filesystem": {Enabled: true},
				},
			},
		},
	}

	unknown := unknownAgentMCPServerNames(cfg, loadAgentDefinition(workspace))
	if len(unknown) != 1 || unknown[0] != "slak" {
		t.Fatalf("unknownAgentMCPServerNames() = %v, want [slak]", unknown)
	}
}

func TestUnknownDeclarationsIgnoreGlobPatterns(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
tools:
  allow:
    - mcp_*
    - web_*
    - read_file
mcpServers:
  allow:
    - git*
    - filesystem
---
# Agent
`,
	})
	defer cleanupWorkspace(t, workspace)

	registry := agenttools.NewToolRegistry()
	registry.Register(&allowlistTestTool{name: "read_file"})
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			MCP: config.MCPConfig{
				Servers: map[string]config.MCPServerConfig{
					"github":     {Enabled: true},
					"filesystem": {Enabled: true},
				},
			},
		},
	}

	if unknown := unknownAgentToolNames(registry, loadAgentDefinition(workspace)); len(unknown) != 0 {
		t.Fatalf("unknownAgentToolNames() = %v, want none", unknown)
	}
	if unknown := unknownAgentMCPServerNames(cfg, loadAgentDefinition(workspace)); len(unknown) != 0 {
		t.Fatalf("unknownAgentMCPServerNames() = %v, want none", unknown)
	}
}
