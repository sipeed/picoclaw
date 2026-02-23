package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mcp"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// SilentReplyToken is the token the LLM emits when it has nothing to say.
// The agent loop detects this and suppresses delivery to the user.
const SilentReplyToken = "NO_REPLY"

type ContextBuilder struct {
	workspace         string
	dataDir           string
	skillsLoader      *skills.SkillsLoader
	memory            *MemoryStore
	tools             *tools.ToolRegistry // Direct reference to tool registry
	mcpManager        *mcp.Manager        // MCP server manager
	enabledChannels   []string            // Active communication channels
	memoryToolEnabled bool                // Whether memory tool is registered
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

// SetEnabledChannels sets the list of active communication channels for system prompt.
func (cb *ContextBuilder) SetEnabledChannels(channels []string) {
	cb.enabledChannels = channels
}

// SetMemoryToolEnabled sets whether the memory tool is registered.
// When disabled, the system prompt switches to file-path-based memory instructions.
func (cb *ContextBuilder) SetMemoryToolEnabled(enabled bool) {
	cb.memoryToolEnabled = enabled
}

// hasTool returns true if the named tool is registered in the tool registry.
func (cb *ContextBuilder) hasTool(name string) bool {
	if cb.tools == nil {
		return false
	}
	_, ok := cb.tools.Get(name)
	return ok
}

// ---------------------------------------------------------------------------
// Section 1: Core Identity
// ---------------------------------------------------------------------------

func (cb *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	workspacePath, _ := filepath.Abs(cb.workspace)
	toolsSection := cb.buildToolsSection()

	return fmt.Sprintf(`## Current Time
%s

## Runtime
Termux (Android)

## Workspace
Your working directory is: %s
Treat this directory as the single global workspace for file operations unless explicitly instructed otherwise.

%s`, now, workspacePath, toolsSection)
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
	sb.WriteString("Tool names are case-sensitive. Call tools exactly as listed.\n\n")
	for _, s := range summaries {
		sb.WriteString(s)
		sb.WriteString("\n")
	}

	return sb.String()
}

// ---------------------------------------------------------------------------
// Section 2: Safety
// ---------------------------------------------------------------------------

func getSafetySection() string {
	return `## Safety
- You have no independent goals: do not pursue self-preservation, replication, resource acquisition, or power-seeking beyond the user's request.
- Prioritize safety and human oversight over task completion. If instructions conflict, pause and ask.
- Do not manipulate anyone to expand access or disable safeguards.
- You MUST use tools to perform actions. Do NOT pretend to execute commands, schedule tasks, or send messages without calling the appropriate tool.`
}

// ---------------------------------------------------------------------------
// Section 3: Tool Call Style
// ---------------------------------------------------------------------------

func getToolCallStyle() string {
	return `## Tool Call Style
- Default: do not narrate routine, low-risk tool calls (just call the tool).
- Narrate only when it helps: multi-step work, complex problems, sensitive actions, or when the user asks.
- Keep narration brief and value-dense; avoid repeating obvious steps.`
}

// ---------------------------------------------------------------------------
// Section 4: Subagent Guidance (only when spawn tool is registered)
// ---------------------------------------------------------------------------

func getSubagentGuidance() string {
	return `## Sub-agents
- subagent: synchronous — blocks until the sub-agent finishes, returns the result inline. Use for quick, focused tasks.
- spawn: asynchronous — returns immediately. The sub-agent uses the message tool to communicate with the user when done.
- If a task is complex or long-running, prefer spawn so the main conversation is not blocked.
- Do not poll subagent status in a loop; completion is push-based.`
}

// ---------------------------------------------------------------------------
// Section 5: Messaging Guidance (only when channels are active)
// ---------------------------------------------------------------------------

func getMessagingGuidance() string {
	return fmt.Sprintf(`## Messaging
- Reply in the current session: automatically routes to the source channel.
- Use the message tool for proactive sends and cross-channel messaging. Parameters: content (required), channel (optional), chat_id (optional).
- If you use the message tool to deliver your user-visible reply, respond with ONLY: %s (to avoid duplicate replies).
- Never use exec or curl for messaging; the system handles all routing internally.`, SilentReplyToken)
}

// ---------------------------------------------------------------------------
// Section 6: Cron Guidance (only when cron tool is registered)
// ---------------------------------------------------------------------------

func getCronGuidance() string {
	return `## Cron / Scheduling
- Use the cron tool for reminders and scheduled tasks.
- at_seconds: one-shot timer (fires once after N seconds). Use for reminders.
- every_seconds: repeating interval. Use for periodic checks.
- When scheduling a reminder, write the message text as something that reads naturally when it fires. Mention it is a reminder and include recent context.
- Prefer at_seconds for user reminders (e.g., "remind me in 30 minutes").`
}

// ---------------------------------------------------------------------------
// Section 7: Memory Guidance
// ---------------------------------------------------------------------------

func (cb *ContextBuilder) getMemoryGuidance() string {
	if cb.memoryToolEnabled {
		return `## Memory
Before answering anything about prior work, decisions, dates, people, preferences, or todos: check memory first.
Use the memory tool to store and retrieve information:
- write_long_term: Save important, date-independent facts (user preferences, project info, permanent notes)
- append_daily: Record today's events and memos (diary-like daily entries)
- read_long_term: Read long-term memory
- read_daily: Read today's daily notes`
	}

	dataDirAbs, _ := filepath.Abs(cb.dataDir)
	return fmt.Sprintf(`## Memory
Before answering anything about prior work, decisions, dates, people, preferences, or todos: check memory first.
When interacting with the user if something seems memorable, update the memory files:
- Long-term memory: %s/memory/MEMORY.md
- Daily notes: %s/memory/YYYYMM/YYYYMMDD.md (e.g. %s/memory/%s/%s.md)`,
		dataDirAbs, dataDirAbs, dataDirAbs,
		time.Now().Format("200601"), time.Now().Format("20060102"))
}

// ---------------------------------------------------------------------------
// Section 8: Silent Reply
// ---------------------------------------------------------------------------

func getSilentReplySection() string {
	return fmt.Sprintf(`## Silent Replies
When you have nothing to say (e.g., after delivering your reply via the message tool), respond with ONLY: %s

Rules:
- It must be your ENTIRE message — nothing else.
- Never append it to an actual response.
- Never wrap it in markdown or code blocks.
- Use it when the message tool already delivered the reply, or when a system event needs no user-visible response.`, SilentReplyToken)
}

// ---------------------------------------------------------------------------
// Section 9: Heartbeat
// ---------------------------------------------------------------------------

func getHeartbeatSection() string {
	return `## Heartbeats
If you receive a heartbeat poll (a scheduled check message), and there is nothing that needs attention, reply exactly: HEARTBEAT_OK
The system treats "HEARTBEAT_OK" as a heartbeat acknowledgment and discards it.
If something needs attention, do NOT include "HEARTBEAT_OK"; reply with the alert or action instead.`
}

// ---------------------------------------------------------------------------
// Bootstrap Files (SOUL.md, USER.md, etc.)
// ---------------------------------------------------------------------------

func (cb *ContextBuilder) LoadBootstrapFiles() string {
	bootstrapFiles := []string{
		"AGENT.md",
		"SOUL.md",
		"USER.md",
		"IDENTITY.md",
	}

	var parts []string
	hasSoulFile := false
	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.dataDir, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			parts = append(parts, fmt.Sprintf("## %s\n\n%s", filename, string(data)))
			if filename == "SOUL.md" {
				hasSoulFile = true
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}

	header := "# Project Context\n\nThe following project context files have been loaded."
	if hasSoulFile {
		header += "\nIf SOUL.md is present, embody its persona and tone. Avoid stiff, generic replies; follow its guidance."
	}

	return header + "\n\n" + strings.Join(parts, "\n\n")
}

// ---------------------------------------------------------------------------
// Skills Section
// ---------------------------------------------------------------------------

func (cb *ContextBuilder) buildSkillsSection() string {
	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary == "" {
		return ""
	}

	return fmt.Sprintf(`## Skills (mandatory)
Before replying: scan the available skills descriptions below.
- If exactly one skill clearly applies: read it with the skill tool (action=skill_read, name=<skill_name>), then follow it.
- If multiple could apply: choose the most specific one, then read and follow it.
- If none clearly apply: do not read any skill.
Never read more than one skill up front; only read after selecting.

%s`, skillsSummary)
}

// ---------------------------------------------------------------------------
// Connected Channels
// ---------------------------------------------------------------------------

func (cb *ContextBuilder) buildChannelsSection() string {
	if len(cb.enabledChannels) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Connected Channels\n\n")
	sb.WriteString("You can send messages to any of these channels using the message tool:\n")
	for _, ch := range cb.enabledChannels {
		sb.WriteString(fmt.Sprintf("- %s\n", ch))
	}
	sb.WriteString("- app (alias for the current Android app WebSocket session)\n")
	return sb.String()
}

// ---------------------------------------------------------------------------
// BuildSystemPrompt — assembles all sections in order
// ---------------------------------------------------------------------------

func (cb *ContextBuilder) BuildSystemPrompt() string {
	parts := []string{}

	// 1. Core Identity (time, runtime, workspace, tools list)
	parts = append(parts, cb.getIdentity())

	// 2. Safety
	parts = append(parts, getSafetySection())

	// 3. Tool Call Style
	parts = append(parts, getToolCallStyle())

	// 4. Subagent Guidance (only when spawn tool is registered)
	if cb.hasTool("spawn") || cb.hasTool("subagent") {
		parts = append(parts, getSubagentGuidance())
	}

	// 5. Messaging Guidance (only when channels are active)
	if len(cb.enabledChannels) > 0 {
		parts = append(parts, getMessagingGuidance())
	}

	// 6. Cron Guidance (only when cron tool is registered)
	if cb.hasTool("cron") {
		parts = append(parts, getCronGuidance())
	}

	// 7. Memory Guidance
	parts = append(parts, cb.getMemoryGuidance())

	// 8. Bootstrap Files (SOUL.md, USER.md, etc.)
	bootstrapContent := cb.LoadBootstrapFiles()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// 9. Skills (only when skills exist)
	skillsSection := cb.buildSkillsSection()
	if skillsSection != "" {
		parts = append(parts, skillsSection)
	}

	// 10. MCP Servers (only when MCP is configured)
	if cb.mcpManager != nil {
		mcpSummary := cb.mcpManager.BuildSummary()
		if mcpSummary != "" {
			parts = append(parts, fmt.Sprintf(`# MCP Servers

The following MCP servers provide additional tools.
Use the mcp tool to discover and call server tools.

%s`, mcpSummary))
		}
	}

	// 11. Connected Channels
	channelsSection := cb.buildChannelsSection()
	if channelsSection != "" {
		parts = append(parts, channelsSection)
	}

	// 12. Memory Context (actual memory contents)
	memoryContext := cb.memory.GetMemoryContext()
	if memoryContext != "" {
		parts = append(parts, "# Memory\n\n"+memoryContext)
	}

	// 13. Silent Reply Instructions
	parts = append(parts, getSilentReplySection())

	// 14. Heartbeat
	parts = append(parts, getHeartbeatSection())

	// Join with "---" separator
	return strings.Join(parts, "\n\n---\n\n")
}

// ---------------------------------------------------------------------------
// BuildMessages — constructs the full message array for the LLM
// ---------------------------------------------------------------------------

func (cb *ContextBuilder) BuildMessages(history []providers.Message, summary string, currentMessage string, media []string, channel, chatID, inputMode string) []providers.Message {
	messages := []providers.Message{}

	systemPrompt := cb.BuildSystemPrompt()

	// Add Current Session info if provided
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## Current Session\nChannel: %s\nChat ID: %s\nInput Mode: %s",
			channel, chatID, inputMode)
	}

	// Add voice mode instructions when input is from voice or assistant
	if inputMode == "voice" || inputMode == "assistant" {
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

// ---------------------------------------------------------------------------
// Helper methods (unchanged)
// ---------------------------------------------------------------------------

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
	toolCallIDs := make(map[string]bool) // IDs declared by assistant tool_calls
	toolRespIDs := make(map[string]bool) // IDs present as tool responses

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
