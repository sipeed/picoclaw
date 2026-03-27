package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	mcp_pkg "github.com/sipeed/picoclaw/pkg/mcp"
)

func TestMultiUserMCPPropagation(t *testing.T) {
	cfg := &config.Config{}
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Tools.MCP.Enabled = true
	cfg.Tools.MCP.Servers = map[string]config.MCPServerConfig{
		"test-server": {Enabled: true},
	}

	msgBus := bus.NewMessageBus()
	provider := &mockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	// Mock initialized MCP manager
	mcpManager := mcp_pkg.NewManager()
	al.mcp.setManager(mcpManager)

	// 1. Create a transient agent instance
	agent := NewAgentInstance(&config.AgentConfig{ID: "test"}, &cfg.Agents.Defaults, cfg, provider, "user-123")
	require.NotNil(t, agent)

	// 2. Register tools initially (should be nothing)
	al.RegisterMCPToolsToAgent("test", agent)

	// Verify no MCP tools yet
	_, ok := agent.Tools.Get("mcp_test_tool")
	assert.False(t, ok)

	// 3. Test Discovery tools registration
	cfg.Tools.MCP.Discovery.Enabled = true
	cfg.Tools.MCP.Discovery.UseRegex = true

	t.Logf("Config before registration: MCP.Enabled=%v, Discovery.Enabled=%v, UseRegex=%v",
		cfg.Tools.MCP.Enabled, cfg.Tools.MCP.Discovery.Enabled, cfg.Tools.MCP.Discovery.UseRegex)

	// Call registration again - it should now add the discovery tool
	al.RegisterMCPToolsToAgent("test", agent)

	t.Logf("Registered tools: %v", agent.Tools.List())

	_, ok = agent.Tools.Get("tool_search_tool_regex")
	assert.True(t, ok, "Discovery tool (tool_search_tool_regex) should be registered after enabling it")
}
