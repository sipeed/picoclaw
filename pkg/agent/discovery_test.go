package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestAgentRegistry_ListAgentsBuildsStructuredDescriptors(t *testing.T) {
	mainWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: Main Frontmatter Name
description: Structured main agent
---
# Agent

Handle general requests.
`,
	})
	defer cleanupWorkspace(t, mainWorkspace)

	supportWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: Support Frontmatter Name
description: Support frontmatter description
---
# Agent

Handle support tickets carefully.
`,
	})
	defer cleanupWorkspace(t, supportWorkspace)

	cfg := testCfg([]config.AgentConfig{
		{ID: "main", Default: true, Name: "Configured Main", Workspace: mainWorkspace},
		{ID: "support", Workspace: supportWorkspace},
	})

	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	descriptors := registry.ListAgents(mainWorkspace)
	if len(descriptors) != 2 {
		t.Fatalf("expected 2 descriptors, got %d", len(descriptors))
	}

	if descriptors[0].ID != "main" {
		t.Fatalf("expected current workspace agent first, got %q", descriptors[0].ID)
	}
	if descriptors[0].Name != "Main Frontmatter Name" {
		t.Fatalf("expected frontmatter name to drive discovery, got %q", descriptors[0].Name)
	}
	if descriptors[0].Description != "Structured main agent" {
		t.Fatalf("expected frontmatter description, got %q", descriptors[0].Description)
	}

	support, ok := registry.GetAgentDescriptor("support")
	if !ok || support == nil {
		t.Fatal("expected support descriptor lookup to succeed")
	}
	if support.Name != "Support Frontmatter Name" {
		t.Fatalf("expected support frontmatter name, got %q", support.Name)
	}
	if support.Description != "Support frontmatter description" {
		t.Fatalf("expected support frontmatter description, got %q", support.Description)
	}
}

func TestAgentRegistry_ListSpawnableAgentsRespectsPermissions(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{
			ID:      "parent",
			Default: true,
			Subagents: &config.SubagentsConfig{
				AllowAgents: []string{"child2", "child1"},
			},
		},
		{ID: "child1"},
		{ID: "child2"},
		{ID: "restricted"},
	})
	cfg.Tools.Spawn.Enabled = true
	cfg.Tools.Subagent.Enabled = true

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockRegistryProvider{})
	defer al.Close()

	descriptors := al.GetRegistry().ListSpawnableAgents("parent")
	if len(descriptors) != 2 {
		t.Fatalf("expected 2 spawnable descriptors, got %d: %+v", len(descriptors), descriptors)
	}
	if descriptors[0].ID != "child1" || descriptors[1].ID != "child2" {
		t.Fatalf("expected sorted spawnable peers only, got %+v", descriptors)
	}
}

func TestAgentRegistry_ListSpawnableAgentsRequiresSpawnTool(t *testing.T) {
	cfg := testCfg([]config.AgentConfig{
		{
			ID:      "parent",
			Default: true,
			Subagents: &config.SubagentsConfig{
				AllowAgents: []string{"child"},
			},
		},
		{ID: "child"},
	})
	cfg.Tools.Subagent.Enabled = true

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockRegistryProvider{})
	defer al.Close()

	if descriptors := al.GetRegistry().ListSpawnableAgents("parent"); len(descriptors) != 0 {
		t.Fatalf("expected no spawnable descriptors without spawn tool, got %+v", descriptors)
	}
}

func TestContextBuilder_BuildMessagesIncludesAgentDiscoverySection(t *testing.T) {
	mainWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
description: Main agent
---
# Agent

Generalist.
`,
	})
	defer cleanupWorkspace(t, mainWorkspace)

	researchWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: Research Agent
description: Research specialist
---
# Agent

Investigate deeply.
`,
	})
	defer cleanupWorkspace(t, researchWorkspace)

	restrictedWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: Restricted Agent
description: Restricted specialist
---
# Agent

Handle restricted work.
`,
	})
	defer cleanupWorkspace(t, restrictedWorkspace)

	cfg := testCfg([]config.AgentConfig{
		{
			ID:        "main",
			Default:   true,
			Workspace: mainWorkspace,
			Subagents: &config.SubagentsConfig{
				AllowAgents: []string{"research"},
			},
		},
		{ID: "research", Workspace: researchWorkspace},
		{ID: "restricted", Workspace: restrictedWorkspace},
	})
	cfg.Tools.ReadFile.Enabled = true
	cfg.Tools.WriteFile.Enabled = true
	cfg.Tools.Spawn.Enabled = true
	cfg.Tools.Subagent.Enabled = true

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockRegistryProvider{})
	defer al.Close()

	mainAgent, ok := al.GetRegistry().GetAgent("main")
	if !ok || mainAgent == nil {
		t.Fatal("expected main agent")
	}

	messages := mainAgent.ContextBuilder.BuildMessages(
		nil,
		"",
		"delegate wisely",
		nil,
		"telegram",
		"chat-1",
		"",
		"",
	)
	if len(messages) == 0 {
		t.Fatal("expected messages")
	}

	systemPrompt := messages[0].Content
	if !strings.Contains(systemPrompt, "# Agent Discovery") {
		t.Fatalf("expected discovery section in system prompt, got %q", systemPrompt)
	}
	if strings.Contains(systemPrompt, `"id": "main"`) {
		t.Fatalf("did not expect self descriptor in discovery section, got %q", systemPrompt)
	}
	if !strings.Contains(systemPrompt, `"id": "research"`) ||
		!strings.Contains(systemPrompt, `"description": "Research specialist"`) {
		t.Fatalf("expected allowed peer descriptor in discovery section, got %q", systemPrompt)
	}
	if strings.Contains(systemPrompt, `"id": "restricted"`) ||
		strings.Contains(systemPrompt, `"description": "Restricted specialist"`) {
		t.Fatalf("did not expect restricted peer descriptor in discovery section, got %q", systemPrompt)
	}
	for _, forbidden := range []string{`"current_agent_id"`, `"available_tools"`, `"model"`, `"channels"`, `"skills"`, `"mcpServers"`, `"tools"`} {
		if strings.Contains(systemPrompt, forbidden) {
			t.Fatalf("did not expect %s in discovery section, got %q", forbidden, systemPrompt)
		}
	}
}

func TestContextBuilder_BuildMessagesOmitsAgentDiscoveryWithoutSpawnPermissions(t *testing.T) {
	mainWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
description: Main agent
---
# Agent

Generalist.
`,
	})
	defer cleanupWorkspace(t, mainWorkspace)

	researchWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
description: Research specialist
---
# Agent

Investigate deeply.
`,
	})
	defer cleanupWorkspace(t, researchWorkspace)

	cfg := testCfg([]config.AgentConfig{
		{ID: "main", Default: true, Workspace: mainWorkspace},
		{ID: "research", Workspace: researchWorkspace},
	})
	cfg.Tools.ReadFile.Enabled = true
	cfg.Tools.Spawn.Enabled = true
	cfg.Tools.Subagent.Enabled = true

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockRegistryProvider{})
	defer al.Close()

	mainAgent, ok := al.GetRegistry().GetAgent("main")
	if !ok || mainAgent == nil {
		t.Fatal("expected main agent")
	}

	messages := mainAgent.ContextBuilder.BuildMessages(
		nil,
		"",
		"handle locally",
		nil,
		"telegram",
		"chat-1",
		"",
		"",
	)
	if len(messages) == 0 {
		t.Fatal("expected messages")
	}

	systemPrompt := messages[0].Content
	if strings.Contains(systemPrompt, "# Agent Discovery") {
		t.Fatalf("did not expect discovery section without spawn permissions, got %q", systemPrompt)
	}
	if strings.Contains(systemPrompt, `"id": "research"`) {
		t.Fatalf("did not expect unauthorized peer identity in system prompt, got %q", systemPrompt)
	}
}

func TestContextBuilder_BuildMessagesOmitsAgentDiscoveryWithoutSpawnTool(t *testing.T) {
	mainWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
description: Main agent
tools: [read_file]
---
# Agent

Generalist.
`,
	})
	defer cleanupWorkspace(t, mainWorkspace)

	researchWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
description: Research specialist
---
# Agent

Investigate deeply.
`,
	})
	defer cleanupWorkspace(t, researchWorkspace)

	cfg := testCfg([]config.AgentConfig{
		{
			ID:        "main",
			Default:   true,
			Workspace: mainWorkspace,
			Subagents: &config.SubagentsConfig{
				AllowAgents: []string{"research"},
			},
		},
		{ID: "research", Workspace: researchWorkspace},
	})
	cfg.Tools.ReadFile.Enabled = true
	cfg.Tools.Spawn.Enabled = true
	cfg.Tools.Subagent.Enabled = true

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockRegistryProvider{})
	defer al.Close()

	mainAgent, ok := al.GetRegistry().GetAgent("main")
	if !ok || mainAgent == nil {
		t.Fatal("expected main agent")
	}

	messages := mainAgent.ContextBuilder.BuildMessages(
		nil,
		"",
		"handle locally",
		nil,
		"telegram",
		"chat-1",
		"",
		"",
	)
	if len(messages) == 0 {
		t.Fatal("expected messages")
	}

	systemPrompt := messages[0].Content
	if strings.Contains(systemPrompt, "# Agent Discovery") {
		t.Fatalf("did not expect discovery section without spawn tool, got %q", systemPrompt)
	}
	if strings.Contains(systemPrompt, `"id": "research"`) {
		t.Fatalf("did not expect peer identity without spawn tool, got %q", systemPrompt)
	}
}

func TestContextBuilder_BuildMessagesOmitsAgentDiscoverySectionForSingleton(t *testing.T) {
	mainWorkspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
description: Main agent
---
# Agent

Generalist.
`,
	})
	defer cleanupWorkspace(t, mainWorkspace)

	cfg := testCfg([]config.AgentConfig{
		{ID: "main", Default: true, Workspace: mainWorkspace},
	})
	cfg.Tools.ReadFile.Enabled = true
	cfg.Tools.Spawn.Enabled = true
	cfg.Tools.Subagent.Enabled = true

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &mockRegistryProvider{})
	defer al.Close()

	mainAgent, ok := al.GetRegistry().GetAgent("main")
	if !ok || mainAgent == nil {
		t.Fatal("expected main agent")
	}

	messages := mainAgent.ContextBuilder.BuildMessages(
		nil,
		"",
		"handle locally",
		nil,
		"telegram",
		"chat-1",
		"",
		"",
	)
	if len(messages) == 0 {
		t.Fatal("expected messages")
	}

	systemPrompt := messages[0].Content
	if strings.Contains(systemPrompt, "# Agent Discovery") {
		t.Fatalf("did not expect discovery section for singleton registry, got %q", systemPrompt)
	}
}

func TestAgentRegistry_ListAgentsFallsBackToFirstNonEmptyAgentLine(t *testing.T) {
	workspace := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: Research Agent
---


First useful line.
Second line.
`,
	})
	defer cleanupWorkspace(t, workspace)

	cfg := testCfg([]config.AgentConfig{
		{ID: "research", Default: true, Workspace: workspace},
	})

	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})
	descriptor, ok := registry.GetAgentDescriptor("research")
	if !ok || descriptor == nil {
		t.Fatal("expected research descriptor lookup to succeed")
	}
	if descriptor.Description != "First useful line." {
		t.Fatalf("descriptor.Description = %q, want %q", descriptor.Description, "First useful line.")
	}
}
