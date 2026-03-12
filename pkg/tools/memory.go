// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MemoryStore manages persistent memory for the agent.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
type MemoryStore struct {
	workspace  string
	memoryDir  string
	memoryFile string
}

// NewMemoryStore creates a new MemoryStore with the given workspace path.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	os.MkdirAll(memoryDir, 0755)
	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: filepath.Join(memoryDir, "MEMORY.md"),
	}
}

// ReadLongTerm reads the long-term memory file.
func (m *MemoryStore) ReadLongTerm() string {
	data, err := os.ReadFile(m.memoryFile)
	if err != nil {
		return ""
	}
	return string(data)
}

// WriteLongTerm writes the long-term memory file.
func (m *MemoryStore) WriteLongTerm(content string) error {
	return os.WriteFile(m.memoryFile, []byte(content), 0644)
}

// GetTodayFilePath returns the path for today's daily note file.
func (m *MemoryStore) GetTodayFilePath() string {
	now := time.Now()
	yearMonth := now.Format("200601")
	dayFile := now.Format("20060102.md")
	return filepath.Join(m.memoryDir, yearMonth, dayFile)
}

// AppendToday appends content to today's daily note.
func (m *MemoryStore) AppendToday(content string) error {
	now := time.Now()
	yearMonth := now.Format("200601")
	monthDir := filepath.Join(m.memoryDir, yearMonth)
	os.MkdirAll(monthDir, 0755)

	filePath := m.GetTodayFilePath()
	var existingContent string
	if data, err := os.ReadFile(filePath); err == nil {
		existingContent = string(data)
	}

	// Add timestamp header if file is new
	if existingContent == "" {
		existingContent = "# " + now.Format("2006-01-02") + "\n\n"
	}

	updated := existingContent + content + "\n"
	return os.WriteFile(filePath, []byte(updated), 0644)
}

// ReadToday reads today's daily note.
func (m *MemoryStore) ReadToday() string {
	filePath := m.GetTodayFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return string(data)
}

// GetWorkspace returns the workspace path.
func (m *MemoryStore) GetWorkspace() string {
	return m.workspace
}

// UpdateMemoryTool updates long-term memory or daily notes.
type UpdateMemoryTool struct {
	memory *MemoryStore
}

// NewUpdateMemoryTool creates a new UpdateMemoryTool.
func NewUpdateMemoryTool(memory *MemoryStore) *UpdateMemoryTool {
	return &UpdateMemoryTool{
		memory: memory,
	}
}

// Name returns the tool name.
func (t *UpdateMemoryTool) Name() string {
	return "update_memory"
}

// Description returns the tool description.
func (t *UpdateMemoryTool) Description() string {
	return `Save important information to memory. Use this when:
- User shares personal info (name, job, location, relationships)
- User expresses preferences (language, timezone, habits, likes/dislikes)
- Important events, deadlines, or plans are mentioned
- Task completions worth remembering

Memory types:
- long_term: Permanent storage (user info, preferences, important notes)
- daily_note: Temporary note for today's activities`
}

// Parameters returns the tool parameters.
func (t *UpdateMemoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"memory_type": map[string]any{
				"type":        "string",
				"enum":        []string{"long_term", "daily_note"},
				"description": "Type of memory: 'long_term' for permanent storage, 'daily_note' for today's temporary note",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to remember (concise and clear)",
			},
			"category": map[string]any{
				"type":        "string",
				"enum":        []string{"user_info", "preference", "important_note", "task", "configuration"},
				"description": "Category for long_term memory (ignored for daily_note)",
			},
		},
		"required": []string{"memory_type", "content"},
	}
}

// Execute executes the tool.
func (t *UpdateMemoryTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	memoryType, ok := args["memory_type"].(string)
	if !ok {
		return ErrorResult("memory_type is required and must be 'long_term' or 'daily_note'").
			WithError(fmt.Errorf("invalid memory_type"))
	}

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required and must be a string").
			WithError(fmt.Errorf("invalid content"))
	}

	// Validate content length
	if strings.TrimSpace(content) == "" {
		return ErrorResult("content cannot be empty").
			WithError(fmt.Errorf("empty content"))
	}

	switch memoryType {
	case "long_term":
		return t.updateLongTerm(content, args)
	case "daily_note":
		return t.updateDailyNote(content)
	default:
		return ErrorResult("memory_type must be 'long_term' or 'daily_note'").
			WithError(fmt.Errorf("invalid memory_type: %s", memoryType))
	}
}

// updateLongTerm updates the long-term memory file (MEMORY.md).
func (t *UpdateMemoryTool) updateLongTerm(content string, args map[string]any) *ToolResult {
	category, _ := args["category"].(string)
	if category == "" {
		category = "important_note" // default category
	}

	// Read current memory
	currentMemory := t.memory.ReadLongTerm()

	// Append to the appropriate section
	updatedMemory := t.appendToSection(currentMemory, category, content)

	// Write back atomically
	err := t.memory.WriteLongTerm(updatedMemory)
	if err != nil {
		return ErrorResult("Failed to write long-term memory: " + err.Error()).
			WithError(err)
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Successfully saved to long-term memory under '%s' category.", category),
		ForUser: fmt.Sprintf("✅ 已保存到长期记忆 (%s)", category),
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

// updateDailyNote appends content to today's daily note.
func (t *UpdateMemoryTool) updateDailyNote(content string) *ToolResult {
	// Format with timestamp
	timestamp := time.Now().Format("15:04")
	formattedContent := fmt.Sprintf("- [%s] %s", timestamp, content)

	err := t.memory.AppendToday(formattedContent)
	if err != nil {
		return ErrorResult("Failed to write daily note: " + err.Error()).
			WithError(err)
	}

	return &ToolResult{
		ForLLM:  "Successfully added to today's daily note.",
		ForUser: "✅ 已添加到今日笔记",
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

// appendToSection appends content to a specific section in MEMORY.md.
func (t *UpdateMemoryTool) appendToSection(currentMemory, category, content string) string {
	// Define section headers
	sectionHeaders := map[string]string{
		"user_info":      "## User Information",
		"preference":     "## Preferences",
		"important_note": "## Important Notes",
		"task":           "## Tasks & Activities",
		"configuration":  "## Configuration",
	}

	header, exists := sectionHeaders[category]
	if !exists {
		header = "## Important Notes"
	}

	// Check if section exists
	if strings.Contains(currentMemory, header) {
		// Append to existing section
		return t.appendAfterHeader(currentMemory, header, content)
	}

	// Create section if it doesn't exist
	return t.createNewSection(currentMemory, header, content)
}

// appendAfterHeader appends content after a section header.
func (t *UpdateMemoryTool) appendAfterHeader(memory, header, content string) string {
	lines := strings.Split(memory, "\n")
	var result []string
	sectionFound := false

	for i, line := range lines {
		result = append(result, line)

		// Find the header
		if strings.TrimSpace(line) == header {
			sectionFound = true
			// Skip empty lines after header
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				result = append(result, lines[j])
				j++
			}
			// Add content as bullet point
			result = append(result, fmt.Sprintf("- %s", content))
			result = append(result, "") // Add spacing
		}
	}

	// Fallback if header not found (shouldn't happen)
	if !sectionFound {
		return memory + "\n\n" + header + "\n\n- " + content + "\n"
	}

	return strings.Join(result, "\n")
}

// createNewSection creates a new section in MEMORY.md.
func (t *UpdateMemoryTool) createNewSection(currentMemory, header, content string) string {
	if currentMemory == "" {
		// Create new file with header
		return "# Long-term Memory\n\n" + header + "\n\n- " + content + "\n"
	}

	// Append to end of file
	return strings.TrimRight(currentMemory, "\n") + "\n\n" + header + "\n\n- " + content + "\n"
}
