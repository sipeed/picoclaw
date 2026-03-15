package agent

import "github.com/sipeed/picoclaw/pkg/tools"

// contextBuilderExt holds fork-specific fields for ContextBuilder.
// Embedded in ContextBuilder so existing field access (cb.workDir, cb.tools, etc.) continues to work.
// Upstream additions to ContextBuilder won't conflict with these fields.
type contextBuilderExt struct {
	workDir              string              // session-specific working directory (worktree or project subdir)
	tools                *tools.ToolRegistry // Direct reference to tool registry
	peerNote             string              // set per-call from loop.go for peer session awareness
	orchestrationEnabled bool                // set from AgentLoop when --orchestration flag is used
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.
func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
}

// SetWorkDir sets the session-specific working directory (e.g., worktree path
// or project subdirectory). Bootstrap files found here take priority over workspace.
func (cb *ContextBuilder) SetWorkDir(dir string) {
	cb.workDir = dir
}

// SetPeerNote sets the peer session awareness note for the current call.
func (cb *ContextBuilder) SetPeerNote(note string) {
	cb.peerNote = note
}

// SetOrchestrationEnabled sets whether orchestration is enabled.
func (cb *ContextBuilder) SetOrchestrationEnabled(enabled bool) {
	cb.orchestrationEnabled = enabled
}

// extIdentityOverrides returns the orchestration-specific overrides for
// getIdentity: banner prefix, identity string, and plan executing rule.
// When orchestration is disabled, all return values are empty strings.
func (cb *ContextBuilder) extIdentityOverrides() (banner, identity, executingRule string) {
	if !cb.orchestrationEnabled {
		return "", "", ""
	}

	banner = ` /_/_/_/_/_/_/_/_/_/_/_/_/_/_/

 O R C H E S T R A  M O D E

/_/_/_/_/_/_/_/_/_/_/_/_/_/_/



`
	identity = "a conductor AI agent that orchestrates subagents"
	executingRule = `Delegate the current Phase's steps to subagents using spawn.
     For each step: spawn a subagent with the appropriate preset (scout for investigation,
     coder for implementation, analyst for review). Spawn multiple independent steps in parallel.
     When a subagent completes, mark "- [x]" via edit_file and record findings in
     ## Orchestration > Findings in MEMORY.md.
     Only do a step inline if it's a single quick tool call (e.g., reading one file).`
	return banner, identity, executingRule
}

// extPromptSections returns fork-specific prompt sections to append to
// BuildSystemPrompt: orchestration guidance and peer session note.
func (cb *ContextBuilder) extPromptSections() []string {
	var sections []string

	// Orchestration guidance — injected only when spawn tool is registered
	if cb.tools != nil {
		if _, hasSpawn := cb.tools.Get("spawn"); hasSpawn {
			sections = append(sections, orchestrationGuidance)
		}
	}

	// Peer session coordination
	if cb.peerNote != "" {
		sections = append(sections, "## Active Sessions\n\n"+cb.peerNote)
	}

	return sections
}

// Memory returns the underlying MemoryStore for direct plan queries.
func (cb *ContextBuilder) Memory() *MemoryStore {
	return cb.memory
}

// ReadMemory reads the long-term memory (MEMORY.md).
func (cb *ContextBuilder) ReadMemory() string {
	return cb.memory.ReadLongTerm()
}

// WriteMemory writes content to the long-term memory file.
func (cb *ContextBuilder) WriteMemory(content string) error {
	return cb.memory.WriteLongTerm(content)
}

// ClearMemory removes the long-term memory file.
func (cb *ContextBuilder) ClearMemory() error {
	return cb.memory.ClearLongTerm()
}

// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]any {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}
	return map[string]any{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}
}
