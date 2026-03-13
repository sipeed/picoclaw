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
