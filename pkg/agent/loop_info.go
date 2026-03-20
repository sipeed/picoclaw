package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/mediacache"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/stats"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func (al *AgentLoop) GetStartupInfo() map[string]any {
	info := make(map[string]any)

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return info
	}

	// Tools info

	toolsList := agent.Tools.List()

	toolsMap := map[string]any{
		"count": len(toolsList),

		"names": toolsList,
	}

	// Report web search provider if registered

	if t, ok := agent.Tools.Get("web_search"); ok {
		if wst, ok := t.(*tools.WebSearchTool); ok {
			toolsMap["web_search_provider"] = wst.ProviderName()
		}
	}

	info["tools"] = toolsMap

	// Skills info

	info["skills"] = agent.ContextBuilder.GetSkillsInfo()

	// Agents info

	info["agents"] = map[string]any{
		"count": len(al.registry.ListAgentIDs()),

		"ids": al.registry.ListAgentIDs(),
	}

	return info
}

// ListSkills returns all available skills from the default agent.

func (al *AgentLoop) ListSkills() []skills.SkillInfo {
	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return nil
	}

	return agent.ContextBuilder.ListSkills()
}

// GetPlanInfo returns plan state from the default agent's memory store.

func (al *AgentLoop) GetPlanInfo() (hasPlan bool, status string, currentPhase, totalPhases int, display string, memory string) {
	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return false, "", 0, 0, "No agent available.", ""
	}

	mem := agent.ContextBuilder.Memory()

	if mem == nil {
		return false, "", 0, 0, "No memory store.", ""
	}

	hasPlan = mem.HasActivePlan()

	status = mem.GetPlanStatus()

	currentPhase = mem.GetCurrentPhase()

	totalPhases = mem.GetTotalPhases()

	display = mem.FormatPlanDisplay()

	memory = mem.ReadLongTerm()

	return hasPlan, status, currentPhase, totalPhases, display, memory
}

// GetPlanStatus returns the current plan status ("interviewing", "executing", "review", etc.) or "".

func (al *AgentLoop) GetPlanStatus() string {
	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return ""
	}

	return agent.ContextBuilder.GetPlanStatus()
}

// GetPlanPhases returns structured phase/step data from the default agent's plan.

func (al *AgentLoop) GetPlanPhases() []PlanPhase {
	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return nil
	}

	mem := agent.ContextBuilder.Memory()

	if mem == nil {
		return nil
	}

	return mem.GetPlanPhases()
}

// GetActiveSessions returns currently active sessions for the mini app API.

func (al *AgentLoop) GetActiveSessions() []SessionEntry {
	return al.sessions.ListActive()
}

// GetSessionStats returns the current session statistics snapshot, or nil if stats tracking is disabled.

func (al *AgentLoop) GetSessionStats() *stats.Stats {
	if al.stats == nil {
		return nil
	}

	s := al.stats.GetStats()

	return &s
}

// GetContextInfo returns the bootstrap file resolution and directory context for the default agent.

func (al *AgentLoop) GetContextInfo() (workDir, planWorkDir, workspace string, bootstrap []BootstrapFileInfo) {
	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return "", "", "", nil
	}

	workspace = agent.Workspace

	planWorkDir = agent.ContextBuilder.GetPlanWorkDir()

	// Use the most recent active session's touch_dir (tool-detected project directory)

	if active := al.sessions.ListActive(); len(active) > 0 && active[0].TouchDir != "" {
		workDir = active[0].TouchDir
	} else {
		workDir = agent.ContextBuilder.workDir
	}

	bootstrap = agent.ContextBuilder.ResolveBootstrapPaths()

	return workDir, planWorkDir, workspace, bootstrap
}

// GetSystemPrompt returns the system prompt last sent to the LLM.

// If the prompt is dirty (state changed since last capture), it rebuilds

// from current state. Falls back to building if no LLM call has occurred yet.

func (al *AgentLoop) GetSystemPrompt() string {
	if !al.promptDirty.Load() {
		if v := al.lastSystemPrompt.Load(); v != nil {
			return v.(string)
		}
	}

	// Rebuild from current state

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return ""
	}

	prompt := agent.ContextBuilder.BuildSystemPrompt()

	al.lastSystemPrompt.Store(prompt)

	al.promptDirty.Store(false)

	return prompt
}

// formatMessagesForLog formats messages for logging

func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var sb strings.Builder

	sb.WriteString("[\n")

	for i, msg := range messages {
		fmt.Fprintf(&sb, "  [%d] Role: %s\n", i, msg.Role)

		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  ToolCalls:\n")

			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&sb, "    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)

				args := tc.Arguments

				if len(args) == 0 && tc.Function != nil {
					args = tc.Function.Arguments
				}

				if len(args) > 0 {
					argsJSON, _ := json.Marshal(args)

					fmt.Fprintf(&sb, "      Arguments: %s\n", utils.Truncate(string(argsJSON), 200))
				}
			}
		}

		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)

			fmt.Fprintf(&sb, "  Content: %s\n", content)
		}

		if msg.ToolCallID != "" {
			fmt.Fprintf(&sb, "  ToolCallID: %s\n", msg.ToolCallID)
		}

		sb.WriteString("\n")
	}

	sb.WriteString("]")

	return sb.String()
}

// formatToolsForLog formats tool definitions for logging

func formatToolsForLog(toolDefs []providers.ToolDefinition) string {
	if len(toolDefs) == 0 {
		return "[]"
	}

	var sb strings.Builder

	sb.WriteString("[\n")

	for i, tool := range toolDefs {
		fmt.Fprintf(&sb, "  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)

		fmt.Fprintf(&sb, "      Description: %s\n", tool.Function.Description)

		if len(tool.Function.Parameters) > 0 {
			fmt.Fprintf(&sb, "      Parameters: %s\n", utils.Truncate(string(tool.Function.Parameters), 200))
		}
	}

	sb.WriteString("]")

	return sb.String()
}

// ListMediaCache returns all media cache entries, optionally filtered by type.
func (al *AgentLoop) ListMediaCache(entryType string) []mediacache.ListEntry {
	if al.mediaCache == nil {
		return nil
	}
	entries, err := al.mediaCache.List(entryType)
	if err != nil {
		return nil
	}
	return entries
}

// DeleteMediaCache deletes all cache entries for the given hash and cleans up files.
func (al *AgentLoop) DeleteMediaCache(hash string) error {
	if al.mediaCache == nil {
		return nil
	}
	for _, t := range []string{mediacache.TypePDFOCR, mediacache.TypePDFText, mediacache.TypeImageDesc} {
		entry, err := al.mediaCache.Delete(hash, t)
		if err != nil {
			continue
		}
		if entry.FilePath != "" {
			dir := filepath.Dir(entry.FilePath)
			if filepath.Base(dir) == hash {
				os.RemoveAll(dir)
			} else {
				os.Remove(entry.FilePath)
			}
		}
	}
	return nil
}

// DeleteAllMediaCache deletes all cache entries and cleans up the OCR output directory.
func (al *AgentLoop) DeleteAllMediaCache() (int64, error) {
	if al.mediaCache == nil {
		return 0, nil
	}
	n, err := al.mediaCache.DeleteAll()
	if err != nil {
		return 0, err
	}
	// Clean up .ocr_cache directory
	registry := al.GetRegistry()
	if agent := registry.GetDefaultAgent(); agent != nil {
		os.RemoveAll(filepath.Join(agent.Workspace, ".ocr_cache"))
	}
	return n, nil
}
