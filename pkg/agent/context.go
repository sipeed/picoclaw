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

3. **Memory** - When remembering something, write to %s/memory/MEMORY.md`,
		now, runtime, workspacePath, workspacePath, workspacePath, workspacePath, toolsSection, workspacePath)
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

	// Sanitize history to ensure valid turn ordering for all providers
	history = sanitizeHistory(history)

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
		parts := []providers.ContentPart{
			{Type: "text", Text: currentMessage},
		}
		for _, url := range media {
			parts = append(parts, providers.ContentPart{
				Type:     "image_url",
				ImageURL: &providers.ImageURL{URL: url},
			})
		}
		userMsg.ContentParts = parts
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

// sanitizeHistory ensures valid turn ordering for all LLM providers.
// It handles corruption from truncation, failed LLM calls, or race conditions.
func sanitizeHistory(history []providers.Message) []providers.Message {
	if len(history) == 0 {
		return history
	}

	// 1. Remove leading non-user messages (tool results, assistant with tool_calls)
	for len(history) > 0 && history[0].Role != "user" {
		logger.DebugCF("agent", "Removing leading non-user message from history",
			map[string]interface{}{"role": history[0].Role})
		history = history[1:]
	}

	if len(history) == 0 {
		return history
	}

	// 2. Walk through and build a valid sequence
	sanitized := make([]providers.Message, 0, len(history))
	for i := 0; i < len(history); i++ {
		msg := history[i]

		// Skip consecutive user messages (keep the last one in a run)
		if msg.Role == "user" && i+1 < len(history) && history[i+1].Role == "user" {
			logger.DebugCF("agent", "Removing duplicate consecutive user message",
				map[string]interface{}{"index": i})
			continue
		}

		// Skip tool messages that don't follow an assistant message with tool_calls
		if msg.Role == "tool" {
			if len(sanitized) == 0 || sanitized[len(sanitized)-1].Role != "assistant" || len(sanitized[len(sanitized)-1].ToolCalls) == 0 {
				// Check if the preceding message (allowing for other tool messages) was an assistant with tool_calls
				hasMatchingAssistant := false
				for j := len(sanitized) - 1; j >= 0; j-- {
					if sanitized[j].Role == "tool" {
						continue
					}
					if sanitized[j].Role == "assistant" && len(sanitized[j].ToolCalls) > 0 {
						hasMatchingAssistant = true
					}
					break
				}
				if !hasMatchingAssistant {
					logger.DebugCF("agent", "Removing orphaned tool message from history",
						map[string]interface{}{"index": i, "tool_call_id": msg.ToolCallID})
					continue
				}
			}
		}

		sanitized = append(sanitized, msg)
	}

	// 3. Remove trailing incomplete tool-call sequences
	// (assistant with tool_calls at the end without all corresponding tool results)
	for len(sanitized) > 0 {
		last := sanitized[len(sanitized)-1]
		if last.Role == "assistant" && len(last.ToolCalls) > 0 {
			logger.DebugCF("agent", "Removing trailing assistant with unanswered tool_calls",
				map[string]interface{}{"tool_calls": len(last.ToolCalls)})
			sanitized = sanitized[:len(sanitized)-1]
			continue
		}
		// Also check if we end with tool results but the preceding assistant
		// doesn't have all its tool_calls answered
		if last.Role == "tool" {
			// Find the preceding assistant message
			assistantIdx := -1
			for j := len(sanitized) - 2; j >= 0; j-- {
				if sanitized[j].Role == "assistant" && len(sanitized[j].ToolCalls) > 0 {
					assistantIdx = j
					break
				}
				if sanitized[j].Role != "tool" {
					break
				}
			}
			if assistantIdx >= 0 {
				// Count tool results after the assistant
				expectedCount := len(sanitized[assistantIdx].ToolCalls)
				actualCount := 0
				for j := assistantIdx + 1; j < len(sanitized); j++ {
					if sanitized[j].Role == "tool" {
						actualCount++
					}
				}
				if actualCount < expectedCount {
					// Incomplete sequence — remove the assistant and all its tool results
					logger.DebugCF("agent", "Removing trailing incomplete tool-call sequence",
						map[string]interface{}{"expected": expectedCount, "actual": actualCount})
					sanitized = sanitized[:assistantIdx]
					continue
				}
			}
		}
		break
	}

	return sanitized
}
