package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mcp"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type ContextBuilder struct {
	workspace    string
	dataDir      string
	skillsLoader *skills.SkillsLoader
	memory       *MemoryStore
	tools        *tools.ToolRegistry // Direct reference to tool registry
	mcpManager   *mcp.Manager        // MCP server manager
}

func getGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw")
}

func NewContextBuilder(workspace string, dataDir string) *ContextBuilder {
	// builtin skills: skills directory in current project
	// Use the skills/ directory under the current working directory
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	return &ContextBuilder{
		workspace:    workspace,
		dataDir:      dataDir,
		skillsLoader: skills.NewSkillsLoader(dataDir, globalSkillsDir, builtinSkillsDir),
		memory:       NewMemoryStore(dataDir),
	}
}

// GetMemory returns the memory store for tool registration.
func (cb *ContextBuilder) GetMemory() *MemoryStore {
	return cb.memory
}

// GetSkillsLoader returns the skills loader for tool registration.
func (cb *ContextBuilder) GetSkillsLoader() *skills.SkillsLoader {
	return cb.skillsLoader
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.
func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
}

// SetMCPManager sets the MCP manager for system prompt integration.
func (cb *ContextBuilder) SetMCPManager(manager *mcp.Manager) {
	cb.mcpManager = manager
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

%s

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory** - Use the memory tool to store and retrieve information.
   - write_long_term: Save important, date-independent facts (user preferences, project info, permanent notes)
   - append_daily: Record today's events and memos (diary-like daily entries)
   - read_long_term: Read long-term memory
   - read_daily: Read today's daily notes`,
		now, runtime, workspacePath, toolsSection)
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

	// Skills - show summary, AI can read full content with skill_read tool
	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, call the skill_read tool with the skill name.

%s`, skillsSummary))
	}

	// MCP Servers - show summary, AI uses mcp tool to discover and call
	if cb.mcpManager != nil {
		mcpSummary := cb.mcpManager.BuildSummary()
		if mcpSummary != "" {
			parts = append(parts, fmt.Sprintf(`# MCP Servers

The following MCP servers provide additional tools.
Use the mcp tool to discover and call server tools.

%s`, mcpSummary))
		}
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
		filePath := filepath.Join(cb.dataDir, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			result += fmt.Sprintf("## %s\n\n%s\n\n", filename, string(data))
		}
	}

	return result
}

func (cb *ContextBuilder) BuildMessages(history []providers.Message, summary string, currentMessage string, media []string, channel, chatID, inputMode string) []providers.Message {
	messages := []providers.Message{}

	systemPrompt := cb.BuildSystemPrompt()

	// Add Current Session info if provided
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## Current Session\nChannel: %s\nChat ID: %s\nInput Mode: %s",
			channel, chatID, inputMode)
	}

	// Add voice mode instructions when input is from voice
	if inputMode == "voice" {
		systemPrompt += voiceModePrompt()
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

	// Sanitize tool messages: remove orphaned tool responses and strip
	// tool_calls whose responses are missing. This prevents API errors
	// after context compression or mid-execution cancellation.
	history = sanitizeToolMessages(history)

	messages = append(messages, providers.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	messages = append(messages, history...)

	userMsg := providers.Message{
		Role:    "user",
		Content: currentMessage,
	}
	if len(media) > 0 {
		userMsg.Media = media
	}
	messages = append(messages, userMsg)

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

// sanitizeToolMessages ensures tool_calls and tool responses are consistent.
// It performs two passes:
//  1. Remove orphaned tool messages whose matching tool_calls assistant is missing.
//  2. Strip individual tool_calls entries from assistant messages when the
//     corresponding tool response is missing (e.g. cancelled mid-execution).
//
// This prevents API errors like "messages with role 'tool' must be a response
// to a preceding message with 'tool_calls'" after compression or cancellation.
func sanitizeToolMessages(history []providers.Message) []providers.Message {
	// Pass 1: collect IDs present in each direction
	toolCallIDs := make(map[string]bool)  // IDs declared by assistant tool_calls
	toolRespIDs := make(map[string]bool)  // IDs present as tool responses

	for _, msg := range history {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				toolCallIDs[tc.ID] = true
			}
		}
		if msg.Role == "tool" && msg.ToolCallID != "" {
			toolRespIDs[msg.ToolCallID] = true
		}
	}

	// Pass 2: rebuild history with fixes
	result := make([]providers.Message, 0, len(history))
	for _, msg := range history {
		// Remove orphaned tool responses (no matching tool_calls)
		if msg.Role == "tool" && msg.ToolCallID != "" {
			if !toolCallIDs[msg.ToolCallID] {
				logger.DebugCF("agent", "Removing orphaned tool message", map[string]interface{}{
					"tool_call_id": msg.ToolCallID,
				})
				continue
			}
		}

		// Strip tool_calls entries that have no corresponding tool response
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			kept := make([]providers.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				if toolRespIDs[tc.ID] {
					kept = append(kept, tc)
				} else {
					logger.DebugCF("agent", "Removing tool_call with missing response", map[string]interface{}{
						"tool_call_id": tc.ID,
					})
				}
			}
			if len(kept) != len(msg.ToolCalls) {
				// Create a copy so we don't mutate the original slice
				fixed := msg
				fixed.ToolCalls = kept
				result = append(result, fixed)
				continue
			}
		}

		result = append(result, msg)
	}

	return result
}
