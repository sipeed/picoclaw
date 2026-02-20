package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type ContextBuilder struct {
	workspace    string
	skillsLoader *skills.SkillsLoader
	memory       *MemoryStore
	tools        *tools.ToolRegistry // Direct reference to tool registry
}

func getGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw")
}

func NewContextBuilder(workspace string) *ContextBuilder {
	// builtin skills: skills directory in current project
	// Use the skills/ directory under the current working directory
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	return &ContextBuilder{
		workspace:    workspace,
		skillsLoader: skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:       NewMemoryStore(workspace),
	}
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.
func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
}

func (cb *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))
	runtime := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	// Build tools section dynamically
	toolsSection := cb.buildToolsSection()

	return fmt.Sprintf(`# picoclaw 🦞

You are picoclaw, a helpful AI assistant.

## Current Time
%s

## Runtime
%s

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

%s

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory & Plans**
   - Use memory/MEMORY.md for structured plans.
   - If Status is "interviewing": Ask clarifying questions.
     After each answer, use edit_file to save findings to ## Context in memory/MEMORY.md.
     When you have enough information, write ## Phase and ## Commands sections into MEMORY.md, then set Status to "executing".
   - If Status is "executing": Work through the current Phase's steps.
     Mark each [x] via edit_file. The system will auto-advance phases.
   - Plan format:
     # Active Plan
     > Task: <description>
     > Status: interviewing | executing
     > Phase: <current phase number>
     ## Phase 1: <title>
     - [ ] Step 1
     ## Phase 2: <title>
     - [ ] Step 2
     ## Commands
     build: <build command>
     test: <test command>
     lint: <lint command>
     ## Context
     <requirements, decisions, environment>
   - Keep each phase to 3-5 steps. Do NOT create plans without /plan.
   - Always ask about build/test/lint commands during interview.

4. **Response Formatting**
   - NEVER use ASCII box-drawing characters (┌─┐│└─┘╔═╗║╚═╝ etc.) or ASCII art diagrams.
   - Use markdown headings, bold, lists, and indentation for structure.
   - Keep lines short — most users read on mobile.
   - For architecture/flow, use arrow text: CLI → Pipeline → Adapters`,
		now, runtime, workspacePath, workspacePath, workspacePath, workspacePath, toolsSection)
}

func (cb *ContextBuilder) buildToolsSection() string {
	if cb.tools == nil {
		return ""
	}

	summaries := cb.tools.GetSummaries()
	if len(summaries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Available Tools\n\n")
	sb.WriteString("**CRITICAL**: You MUST use tools to perform actions. Do NOT pretend to execute commands or schedule tasks.\n\n")
	sb.WriteString("You have access to the following tools:\n\n")
	for _, s := range summaries {
		sb.WriteString(s)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (cb *ContextBuilder) BuildSystemPrompt() string {
	parts := []string{}

	// Core identity section
	parts = append(parts, cb.getIdentity())

	// Bootstrap files
	bootstrapContent := cb.LoadBootstrapFiles()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// Skills - show summary, AI can read full content with read_file tool
	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.

%s`, skillsSummary))
	}

	// Memory context
	memoryContext := cb.memory.GetMemoryContext()
	if memoryContext != "" {
		parts = append(parts, "# Memory\n\n"+memoryContext)
	}

	// Join with "---" separator
	return strings.Join(parts, "\n\n---\n\n")
}

func (cb *ContextBuilder) LoadBootstrapFiles() string {
	bootstrapFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"USER.md",
		"IDENTITY.md",
	}

	var result string
	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			result += fmt.Sprintf("## %s\n\n%s\n\n", filename, string(data))
		}
	}

	return result
}

func (cb *ContextBuilder) BuildMessages(history []providers.Message, summary string, currentMessage string, media []string, channel, chatID string) []providers.Message {
	messages := []providers.Message{}

	systemPrompt := cb.BuildSystemPrompt()

	// Add Current Session info if provided
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## Current Session\nChannel: %s\nChat ID: %s", channel, chatID)
	}

	// Log system prompt summary for debugging (debug mode only)
	logger.DebugCF("agent", "System prompt built",
		map[string]interface{}{
			"total_chars":   len(systemPrompt),
			"total_lines":   strings.Count(systemPrompt, "\n") + 1,
			"section_count": strings.Count(systemPrompt, "\n\n---\n\n") + 1,
		})

	// Log preview of system prompt (avoid logging huge content)
	preview := systemPrompt
	if len(preview) > 500 {
		preview = preview[:500] + "... (truncated)"
	}
	logger.DebugCF("agent", "System prompt preview",
		map[string]interface{}{
			"preview": preview,
		})

	if summary != "" {
		systemPrompt += "\n\n## Summary of Previous Conversation\n\n" + summary
	}

	//This fix prevents the session memory from LLM failure due to elimination of toolu_IDs required from LLM
	// --- INICIO DEL FIX ---
	//Diegox-17
	for len(history) > 0 && (history[0].Role == "tool") {
		logger.DebugCF("agent", "Removing orphaned tool message from history to prevent LLM error",
			map[string]interface{}{"role": history[0].Role})
		history = history[1:]
	}
	//Diegox-17
	// --- FIN DEL FIX ---

	messages = append(messages, providers.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	messages = append(messages, history...)

	messages = append(messages, providers.Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages
}

func (cb *ContextBuilder) AddToolResult(messages []providers.Message, toolCallID, toolName, result string) []providers.Message {
	messages = append(messages, providers.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
	})
	return messages
}

func (cb *ContextBuilder) AddAssistantMessage(messages []providers.Message, content string, toolCalls []map[string]interface{}) []providers.Message {
	msg := providers.Message{
		Role:    "assistant",
		Content: content,
	}
	// Always add assistant message, whether or not it has tool calls
	messages = append(messages, msg)
	return messages
}

func (cb *ContextBuilder) loadSkills() string {
	allSkills := cb.skillsLoader.ListSkills()
	if len(allSkills) == 0 {
		return ""
	}

	var skillNames []string
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}

	content := cb.skillsLoader.LoadSkillsForContext(skillNames)
	if content == "" {
		return ""
	}

	return "# Skill Definitions\n\n" + content
}

// LoadSkill loads a skill by name, returning its content (with frontmatter stripped) and whether it was found.
func (cb *ContextBuilder) LoadSkill(name string) (string, bool) {
	return cb.skillsLoader.LoadSkill(name)
}

// ListSkills returns all available skills from all tiers.
func (cb *ContextBuilder) ListSkills() []skills.SkillInfo {
	return cb.skillsLoader.ListSkills()
}

// ---------- Plan passthrough methods ----------

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

// HasActivePlan returns true if MEMORY.md contains an active plan.
func (cb *ContextBuilder) HasActivePlan() bool {
	return cb.memory.HasActivePlan()
}

// GetPlanStatus returns the plan status: "interviewing", "executing", or "".
func (cb *ContextBuilder) GetPlanStatus() string {
	return cb.memory.GetPlanStatus()
}

// IsPlanComplete returns true if all steps in all phases are [x].
func (cb *ContextBuilder) IsPlanComplete() bool {
	return cb.memory.IsPlanComplete()
}

// IsCurrentPhaseComplete returns true if all steps in the current phase are [x].
func (cb *ContextBuilder) IsCurrentPhaseComplete() bool {
	return cb.memory.IsCurrentPhaseComplete()
}

// AdvancePhase increments the current phase number by 1.
func (cb *ContextBuilder) AdvancePhase() error {
	return cb.memory.AdvancePhase()
}

// GetCurrentPhase returns the current phase number.
func (cb *ContextBuilder) GetCurrentPhase() int {
	return cb.memory.GetCurrentPhase()
}

// GetTotalPhases returns the total number of phases in the plan.
func (cb *ContextBuilder) GetTotalPhases() int {
	return cb.memory.GetTotalPhases()
}

// FormatPlanDisplay returns a user-facing display of the full plan.
func (cb *ContextBuilder) FormatPlanDisplay() string {
	return cb.memory.FormatPlanDisplay()
}

// MarkStep marks a step as done in the specified phase.
func (cb *ContextBuilder) MarkStep(phase, step int) error {
	return cb.memory.MarkStep(phase, step)
}

// AddStep appends a new step to the given phase.
func (cb *ContextBuilder) AddStep(phase int, desc string) error {
	return cb.memory.AddStep(phase, desc)
}

// SetPlanStatus sets the plan status.
func (cb *ContextBuilder) SetPlanStatus(status string) error {
	return cb.memory.SetStatus(status)
}

// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]interface{} {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}
	return map[string]interface{}{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}
}
