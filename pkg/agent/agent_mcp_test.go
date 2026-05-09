// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/mcp"
	agenttools "github.com/sipeed/picoclaw/pkg/tools"
)

func boolPtr(b bool) *bool { return &b }

func TestMCPRuntimeResetClearsState(t *testing.T) {
	var rt mcpRuntime
	manager := mcp.NewManager()
	rt.setManager(manager)
	rt.setInitErr(errors.New("stale init error"))
	rt.initOnce.Do(func() {})

	got := rt.reset()
	if got != manager {
		t.Fatalf("reset() manager = %p, want %p", got, manager)
	}
	if rt.hasManager() {
		t.Fatal("expected manager to be cleared after reset")
	}
	if err := rt.getInitErr(); err != nil {
		t.Fatalf("getInitErr() = %v, want nil", err)
	}

	reran := false
	rt.initOnce.Do(func() { reran = true })
	if !reran {
		t.Fatal("expected initOnce to be reset")
	}
}

func TestReloadProviderAndConfig_ResetsMCPRuntime(t *testing.T) {
	al, cfg, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()
	defer al.Close()

	manager := mcp.NewManager()
	al.mcp.setManager(manager)
	al.mcp.setInitErr(errors.New("stale init error"))
	al.mcp.initOnce.Do(func() {})

	if !al.mcp.hasManager() {
		t.Fatal("expected MCP manager to exist before reload")
	}

	if err := al.ReloadProviderAndConfig(context.Background(), &mockProvider{}, cfg); err != nil {
		t.Fatalf("ReloadProviderAndConfig() error = %v", err)
	}

	if al.mcp.hasManager() {
		t.Fatal("expected MCP manager to be cleared when reloaded config has MCP disabled")
	}
	if err := al.mcp.getInitErr(); err != nil {
		t.Fatalf("getInitErr() = %v, want nil", err)
	}

	reran := false
	al.mcp.initOnce.Do(func() { reran = true })
	if !reran {
		t.Fatal("expected MCP initOnce to be reset after reload")
	}
}

func TestServerIsDeferred(t *testing.T) {
	tests := []struct {
		name             string
		discoveryEnabled bool
		serverDeferred   *bool
		want             bool
	}{
		// --- global false always wins: per-server deferred is ignored ---
		{
			name:             "global false: per-server deferred=true is ignored",
			discoveryEnabled: false,
			serverDeferred:   boolPtr(true),
			want:             false,
		},
		{
			name:             "global false: per-server deferred=false stays false",
			discoveryEnabled: false,
			serverDeferred:   boolPtr(false),
			want:             false,
		},
		// --- global true: per-server override applies ---
		{
			name:             "global true: per-server deferred=false opts out",
			discoveryEnabled: true,
			serverDeferred:   boolPtr(false),
			want:             false,
		},
		{
			name:             "global true: per-server deferred=true stays true",
			discoveryEnabled: true,
			serverDeferred:   boolPtr(true),
			want:             true,
		},
		// --- no per-server override: fall back to global ---
		{
			name:             "no per-server field, global discovery enabled",
			discoveryEnabled: true,
			serverDeferred:   nil,
			want:             true,
		},
		{
			name:             "no per-server field, global discovery disabled",
			discoveryEnabled: false,
			serverDeferred:   nil,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCfg := config.MCPServerConfig{Deferred: tt.serverDeferred}
			got := serverIsDeferred(tt.discoveryEnabled, serverCfg)
			if got != tt.want {
				t.Errorf("serverIsDeferred(discoveryEnabled=%v, deferred=%v) = %v, want %v",
					tt.discoveryEnabled, tt.serverDeferred, got, tt.want)
			}
		})
	}
}

func TestRegisterMCPServerPromptContributorUsesActualRegisteredToolCount(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	agent := &AgentInstance{ContextBuilder: cb}

	registerMCPServerPromptContributor("research", agent, "github", 0, false)
	messages := cb.BuildMessagesFromPrompt(PromptBuildRequest{CurrentMessage: "hello"})
	if prompt := messages[0].Content; strings.Contains(prompt, "MCP server `github`") {
		t.Fatalf("expected no MCP prompt when no tools were registered, got %q", prompt)
	}

	registerMCPServerPromptContributor("research", agent, "github", 2, false)
	messages = cb.BuildMessagesFromPrompt(PromptBuildRequest{CurrentMessage: "hello"})
	prompt := messages[0].Content
	if !strings.Contains(prompt, "MCP server `github` is connected") {
		t.Fatalf("expected MCP prompt for registered tools, got %q", prompt)
	}
	if !strings.Contains(prompt, "It contributes 2 tool(s)") {
		t.Fatalf("expected actual registered tool count in prompt, got %q", prompt)
	}
}

func TestToolRegistryIncludesReportsOnlyRegisteredTools(t *testing.T) {
	registry := agenttools.NewToolRegistry()
	registry.SetAllowlist([]string{"mcp_github_search"})

	registry.RegisterHidden(&allowlistTestTool{name: "mcp_github_search"})
	registry.RegisterHidden(&allowlistTestTool{name: "mcp_github_create_issue"})

	if !toolRegistryIncludes(registry, "mcp_github_search") {
		t.Fatal("expected hidden registered MCP tool to be included")
	}
	if toolRegistryIncludes(registry, "mcp_github_create_issue") {
		t.Fatal("blocked MCP tool should not be included")
	}
}

func TestFilterMCPConfigServersCaseInsensitivePreservesOriginalKeys(t *testing.T) {
	mcpCfg := config.MCPConfig{
		Servers: map[string]config.MCPServerConfig{
			"GitHub":     {Enabled: true},
			"filesystem": {Enabled: true},
			"Slack":      {Enabled: true},
		},
	}
	allowed := map[string]struct{}{
		"github":     {},
		"FILESYSTEM": {},
	}

	filtered := filterMCPConfigServers(mcpCfg, allowed)

	if len(filtered.Servers) != 2 {
		t.Fatalf("filtered.Servers = %v, want 2 entries", filtered.Servers)
	}
	if _, ok := filtered.Servers["GitHub"]; !ok {
		t.Fatal("expected original GitHub config key to be preserved")
	}
	if _, ok := filtered.Servers["filesystem"]; !ok {
		t.Fatal("expected filesystem config key to be preserved")
	}
	if _, ok := filtered.Servers["github"]; ok {
		t.Fatal("did not expect normalized github key to replace original config key")
	}
	if _, ok := filtered.Servers["Slack"]; ok {
		t.Fatal("did not expect unallowed Slack server")
	}
}

func TestAgentHasDiscoverableMCPServers(t *testing.T) {
	deferredFalse := false
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			MCP: config.MCPConfig{
				ToolConfig: config.ToolConfig{Enabled: true},
				Discovery: config.ToolDiscoveryConfig{
					Enabled:  true,
					UseBM25:  true,
					UseRegex: false,
				},
				Servers: map[string]config.MCPServerConfig{
					"github":     {Enabled: true},
					"filesystem": {Enabled: true, Deferred: &deferredFalse},
				},
			},
		},
	}

	tests := []struct {
		name    string
		allowed *PatternPolicy
		want    bool
	}{
		{
			name: "nil allowlist includes discoverable enabled server",
			want: true,
		},
		{
			name:    "empty allowlist denies all servers",
			allowed: &PatternPolicy{Allow: []string{}, form: patternPolicyFormList},
			want:    false,
		},
		{
			name:    "selected server discoverable",
			allowed: &PatternPolicy{Allow: []string{"github"}, form: patternPolicyFormList},
			want:    true,
		},
		{
			name:    "selected server opted out of discovery",
			allowed: &PatternPolicy{Allow: []string{"filesystem"}, form: patternPolicyFormList},
			want:    false,
		},
		{
			name:    "unknown allowlist server matches nothing",
			allowed: &PatternPolicy{Allow: []string{"slack"}, form: patternPolicyFormList},
			want:    false,
		},
		{
			name:    "deny-only policy can exclude discoverable server",
			allowed: &PatternPolicy{Deny: []string{"github"}, form: patternPolicyFormObject},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := agentHasDiscoverableMCPServers(cfg, tt.allowed); got != tt.want {
				t.Fatalf("agentHasDiscoverableMCPServers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureMCPInitialized_LoadFailureSetsInitErr(t *testing.T) {
	al, cfg, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()
	defer al.Close()

	cfg.Tools = config.ToolsConfig{
		MCP: config.MCPConfig{
			ToolConfig: config.ToolConfig{Enabled: true},
			Servers: map[string]config.MCPServerConfig{
				"broken": {
					Enabled: true,
					Command: "picoclaw-command-that-does-not-exist-for-mcp-tests",
				},
			},
		},
	}

	err := al.ensureMCPInitialized(context.Background())
	if err == nil {
		t.Fatal("ensureMCPInitialized() error = nil, want load failure")
	}
	if !strings.Contains(err.Error(), "failed to load MCP servers") {
		t.Fatalf("ensureMCPInitialized() error = %q, want wrapped load failure", err.Error())
	}

	initErr := al.mcp.getInitErr()
	if initErr == nil {
		t.Fatal("getInitErr() = nil, want cached load failure")
	}
	if !strings.Contains(initErr.Error(), "failed to load MCP servers") {
		t.Fatalf("getInitErr() = %q, want wrapped load failure", initErr.Error())
	}
	if al.mcp.getManager() != nil {
		t.Fatal("expected MCP manager to remain nil after load failure")
	}

	err = al.ensureMCPInitialized(context.Background())
	if err == nil {
		t.Fatal("second ensureMCPInitialized() error = nil, want cached load failure")
	}
	if !strings.Contains(err.Error(), "failed to load MCP servers") {
		t.Fatalf("second ensureMCPInitialized() error = %q, want wrapped load failure", err.Error())
	}
}
