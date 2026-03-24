package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadAgentDefinitionParsesFrontmatterAndSoul(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: pico
description: Structured agent
model: claude-3-7-sonnet
tools:
  - shell
  - search
maxTurns: 8
skills:
  - review
  - search-docs
mcpServers:
  - github
metadata:
  mode: strict
---
# Agent

Act directly and use tools first.
`,
		"SOUL.md": "# Soul\nStay precise.",
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)
	definition := cb.LoadAgentDefinition()

	if definition.Source != AgentDefinitionSourceAgent {
		t.Fatalf("expected source %q, got %q", AgentDefinitionSourceAgent, definition.Source)
	}
	if definition.Agent == nil {
		t.Fatal("expected AGENT.md definition to be loaded")
	}
	if definition.Agent.Body == "" || !strings.Contains(definition.Agent.Body, "Act directly") {
		t.Fatalf("expected AGENT.md body to be preserved, got %q", definition.Agent.Body)
	}
	if definition.Agent.Frontmatter.Name != "pico" {
		t.Fatalf("expected name to be parsed, got %q", definition.Agent.Frontmatter.Name)
	}
	if definition.Agent.Frontmatter.Model != "claude-3-7-sonnet" {
		t.Fatalf("expected model to be parsed, got %q", definition.Agent.Frontmatter.Model)
	}
	if len(definition.Agent.Frontmatter.Tools) != 2 {
		t.Fatalf("expected tools to be parsed, got %v", definition.Agent.Frontmatter.Tools)
	}
	if definition.Agent.Frontmatter.MaxTurns == nil || *definition.Agent.Frontmatter.MaxTurns != 8 {
		t.Fatalf("expected maxTurns to be parsed, got %v", definition.Agent.Frontmatter.MaxTurns)
	}
	if len(definition.Agent.Frontmatter.Skills) != 2 {
		t.Fatalf("expected skills to be parsed, got %v", definition.Agent.Frontmatter.Skills)
	}
	if len(definition.Agent.Frontmatter.MCPServers) != 1 || definition.Agent.Frontmatter.MCPServers[0] != "github" {
		t.Fatalf("expected mcpServers to be parsed, got %v", definition.Agent.Frontmatter.MCPServers)
	}
	if definition.Agent.Frontmatter.Fields["metadata"] == nil {
		t.Fatal("expected arbitrary frontmatter fields to remain available")
	}

	if definition.Soul == nil {
		t.Fatal("expected SOUL.md to be loaded")
	}
	if !strings.Contains(definition.Soul.Content, "Stay precise") {
		t.Fatalf("expected soul content to be loaded, got %q", definition.Soul.Content)
	}
	if definition.Soul.Path != filepath.Join(tmpDir, "SOUL.md") {
		t.Fatalf("expected default SOUL.md path, got %q", definition.Soul.Path)
	}
}

func TestLoadAgentDefinitionFallsBackToLegacyAgentsMarkdown(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENTS.md": "# Legacy Agent\nKeep compatibility.",
		"SOUL.md":   "# Soul\nLegacy soul.",
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)
	definition := cb.LoadAgentDefinition()

	if definition.Source != AgentDefinitionSourceAgents {
		t.Fatalf("expected source %q, got %q", AgentDefinitionSourceAgents, definition.Source)
	}
	if definition.Agent == nil {
		t.Fatal("expected AGENTS.md to be loaded")
	}
	if definition.Agent.RawFrontmatter != "" {
		t.Fatalf("legacy AGENTS.md should not have frontmatter, got %q", definition.Agent.RawFrontmatter)
	}
	if !strings.Contains(definition.Agent.Body, "Keep compatibility") {
		t.Fatalf("expected legacy body to be preserved, got %q", definition.Agent.Body)
	}
	if definition.Soul == nil || !strings.Contains(definition.Soul.Content, "Legacy soul") {
		t.Fatal("expected default SOUL.md to be loaded for legacy format")
	}
}

func TestLoadAgentDefinitionLoadsWorkspaceUserMarkdown(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENT.md": "# Agent\nStructured agent.",
		"USER.md":  "# User\nWorkspace preferences.",
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)
	definition := cb.LoadAgentDefinition()

	if definition.User == nil {
		t.Fatal("expected USER.md to be loaded")
	}
	if definition.User.Path != filepath.Join(tmpDir, "USER.md") {
		t.Fatalf("expected workspace USER.md path, got %q", definition.User.Path)
	}
	if !strings.Contains(definition.User.Content, "Workspace preferences") {
		t.Fatalf("expected workspace USER.md content, got %q", definition.User.Content)
	}
}

func TestLoadAgentDefinitionInvalidFrontmatterFallsBackToEmptyStructuredFields(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: pico
tools:
  - shell
  broken
---
# Agent

Keep going.
`,
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)
	definition := cb.LoadAgentDefinition()

	if definition.Agent == nil {
		t.Fatal("expected AGENT.md definition to be loaded")
	}
	if !strings.Contains(definition.Agent.Body, "Keep going.") {
		t.Fatalf("expected AGENT.md body to be preserved, got %q", definition.Agent.Body)
	}
	if definition.Agent.Frontmatter.Name != "" ||
		definition.Agent.Frontmatter.Description != "" ||
		definition.Agent.Frontmatter.Model != "" ||
		definition.Agent.Frontmatter.MaxTurns != nil ||
		len(definition.Agent.Frontmatter.Tools) != 0 ||
		len(definition.Agent.Frontmatter.Skills) != 0 ||
		len(definition.Agent.Frontmatter.MCPServers) != 0 ||
		len(definition.Agent.Frontmatter.Fields) != 0 {
		t.Fatalf("expected invalid frontmatter to decode as empty struct, got %+v", definition.Agent.Frontmatter)
	}
}

func TestLoadBootstrapFilesUsesAgentBodyNotFrontmatter(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENT.md": `---
name: pico
model: codex-mini
---
# Agent

Follow the body prompt.
`,
		"SOUL.md":     "# Soul\nSpeak plainly.",
		"IDENTITY.md": "# Identity\nWorkspace identity.",
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)
	bootstrap := cb.LoadBootstrapFiles()

	if !strings.Contains(bootstrap, "Follow the body prompt") {
		t.Fatalf("expected AGENT.md body in bootstrap, got %q", bootstrap)
	}
	if !strings.Contains(bootstrap, "Speak plainly") {
		t.Fatalf("expected resolved soul content in bootstrap, got %q", bootstrap)
	}
	if strings.Contains(bootstrap, "name: pico") {
		t.Fatalf("bootstrap should not expose raw frontmatter, got %q", bootstrap)
	}
	if strings.Contains(bootstrap, "model: codex-mini") {
		t.Fatalf("bootstrap should not expose raw frontmatter, got %q", bootstrap)
	}
	if !strings.Contains(bootstrap, "SOUL.md") {
		t.Fatalf("expected bootstrap to label SOUL.md, got %q", bootstrap)
	}
	if strings.Contains(bootstrap, "Workspace identity") {
		t.Fatalf("structured bootstrap should ignore IDENTITY.md, got %q", bootstrap)
	}
}

func TestLoadBootstrapFilesIncludesWorkspaceUserMarkdown(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENT.md": "# Agent\nFollow the new structure.",
		"SOUL.md":  "# Soul\nSpeak plainly.",
		"USER.md":  "# User\nShared profile.",
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)
	bootstrap := cb.LoadBootstrapFiles()

	if !strings.Contains(bootstrap, "Shared profile") {
		t.Fatalf("expected workspace USER.md in bootstrap, got %q", bootstrap)
	}
	if !strings.Contains(bootstrap, "## USER.md") {
		t.Fatalf("expected USER.md heading in bootstrap, got %q", bootstrap)
	}
}

func TestStructuredAgentIgnoresIdentityChanges(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENT.md":    "# Agent\nFollow the new structure.",
		"SOUL.md":     "# Soul\nVersion one.",
		"IDENTITY.md": "# Identity\nLegacy identity.",
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)

	promptV1 := cb.BuildSystemPromptWithCache()
	if strings.Contains(promptV1, "Legacy identity") {
		t.Fatalf("structured prompt should not include IDENTITY.md, got %q", promptV1)
	}

	identityPath := filepath.Join(tmpDir, "IDENTITY.md")
	if err := os.WriteFile(identityPath, []byte("# Identity\nVersion two."), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(identityPath, future, future); err != nil {
		t.Fatal(err)
	}

	cb.systemPromptMutex.RLock()
	changed := cb.sourceFilesChangedLocked()
	cb.systemPromptMutex.RUnlock()
	if changed {
		t.Fatal("IDENTITY.md should not invalidate cache for structured agent definitions")
	}

	promptV2 := cb.BuildSystemPromptWithCache()
	if promptV1 != promptV2 {
		t.Fatal("structured prompt should remain stable after IDENTITY.md changes")
	}
}

func TestStructuredAgentUserChangesInvalidateCache(t *testing.T) {
	tmpDir := setupWorkspace(t, map[string]string{
		"AGENT.md": "# Agent\nFollow the new structure.",
		"SOUL.md":  "# Soul\nVersion one.",
		"USER.md":  "# User\nInitial workspace preferences.",
	})
	defer cleanupWorkspace(t, tmpDir)

	cb := NewContextBuilder(tmpDir)

	promptV1 := cb.BuildSystemPromptWithCache()
	if !strings.Contains(promptV1, "Initial workspace preferences") {
		t.Fatalf("expected workspace USER.md in prompt, got %q", promptV1)
	}

	userPath := filepath.Join(tmpDir, "USER.md")
	if err := os.WriteFile(userPath, []byte("# User\nUpdated workspace preferences."), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(userPath, future, future); err != nil {
		t.Fatal(err)
	}

	cb.systemPromptMutex.RLock()
	changed := cb.sourceFilesChangedLocked()
	cb.systemPromptMutex.RUnlock()
	if !changed {
		t.Fatal("workspace USER.md changes should invalidate cache")
	}

	promptV2 := cb.BuildSystemPromptWithCache()
	if !strings.Contains(promptV2, "Updated workspace preferences") {
		t.Fatalf("expected updated workspace USER.md in prompt, got %q", promptV2)
	}
}

func cleanupWorkspace(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("failed to clean up workspace %s: %v", path, err)
	}
}
