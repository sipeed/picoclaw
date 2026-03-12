// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/stretchr/testify/assert"
)

func setupTestMemory(t *testing.T) (*agent.MemoryStore, func()) {
	// Create temp directory
	tmpDir := t.TempDir()
	
	memory := agent.NewMemoryStore(tmpDir)
	
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	
	return memory, cleanup
}

func TestUpdateMemoryTool_LongTerm(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	tool := NewUpdateMemoryTool(memory)
	ctx := context.Background()

	// Test adding user info
	args := map[string]any{
		"memory_type": "long_term",
		"category":    "user_info",
		"content":     "用户名叫小明，是一名 Go 语言开发者",
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	assert.Contains(t, result.ForUser, "长期记忆")
	
	// Verify content was written
	content := memory.ReadLongTerm()
	assert.Contains(t, content, "## User Information")
	assert.Contains(t, content, "用户名叫小明，是一名 Go 语言开发者")
}

func TestUpdateMemoryTool_DailyNote(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	tool := NewUpdateMemoryTool(memory)
	ctx := context.Background()

	args := map[string]any{
		"memory_type": "daily_note",
		"content":     "完成了代码审查",
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	assert.Contains(t, result.ForUser, "今日笔记")
	
	// Verify daily note was created
	today := time.Now().Format("20060102")
	monthDir := today[:6]
	todayFile := filepath.Join(memory.GetWorkspace(), "memory", monthDir, today+".md")
	
	content, err := os.ReadFile(todayFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "完成了代码审查")
}

func TestUpdateMemoryTool_EmptyContent(t *testing.T) {
	memory, _ := setupTestMemory(t)
	tool := NewUpdateMemoryTool(memory)
	ctx := context.Background()

	args := map[string]any{
		"memory_type": "long_term",
		"content":     "   ", // empty after trim
	}

	result := tool.Execute(ctx, args)
	
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "empty")
}

func TestUpdateMemoryTool_InvalidType(t *testing.T) {
	memory, _ := setupTestMemory(t)
	tool := NewUpdateMemoryTool(memory)
	ctx := context.Background()

	args := map[string]any{
		"memory_type": "invalid_type",
		"content":     "test content",
	}

	result := tool.Execute(ctx, args)
	
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "invalid memory_type")
}

func TestUpdateMemoryTool_AppendToExistingSection(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	// Create initial memory
	initialContent := `# Long-term Memory

## User Information

- 初始信息`
	
	err := memory.WriteLongTerm(initialContent)
	assert.NoError(t, err)

	tool := NewUpdateMemoryTool(memory)
	ctx := context.Background()

	args := map[string]any{
		"memory_type": "long_term",
		"category":    "user_info",
		"content":     "新增的用户信息",
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	
	// Verify both items exist
	content := memory.ReadLongTerm()
	assert.Contains(t, content, "初始信息")
	assert.Contains(t, content, "新增的用户信息")
}

func TestUpdateMemoryTool_CreateNewSection(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	// Create minimal initial memory
	initialContent := "# Long-term Memory\n\n## Preferences\n\n- 偏好 1"
	err := memory.WriteLongTerm(initialContent)
	assert.NoError(t, err)

	tool := NewUpdateMemoryTool(memory)
	ctx := context.Background()

	// Add to non-existing section
	args := map[string]any{
		"memory_type": "long_term",
		"category":    "task", // This section doesn't exist yet
		"content":     "新任务",
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	
	content := memory.ReadLongTerm()
	assert.Contains(t, content, "## Tasks & Activities")
	assert.Contains(t, content, "新任务")
}

func TestSearchMemoryTool_LongTerm(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	// Setup test data
	testContent := `# Long-term Memory

## User Information

- 用户名叫小明
- 是一名 Go 语言开发者

## Preferences

- 喜欢喝拿铁咖啡
- 偏好中文交流`
	
	err := memory.WriteLongTerm(testContent)
	assert.NoError(t, err)

	tool := NewSearchMemoryTool(memory)
	ctx := context.Background()

	// Test search that should find results
	args := map[string]any{
		"query":       "小明",
		"memory_type": "long_term",
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "小明")
	assert.Contains(t, result.ForUser, "找到")
}

func TestSearchMemoryTool_NoResults(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	// Setup test data
	testContent := `# Long-term Memory

## User Information

- 用户名叫小明`
	
	err := memory.WriteLongTerm(testContent)
	assert.NoError(t, err)

	tool := NewSearchMemoryTool(memory)
	ctx := context.Background()

	// Test search that should NOT find results
	args := map[string]any{
		"query":       "不存在的关键词",
		"memory_type": "long_term",
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "No relevant memories")
	assert.Contains(t, result.ForUser, "未找到")
}

func TestSearchMemoryTool_DailyNotes(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	// Create today's daily note
	today := time.Now().Format("20060102")
	monthDir := today[:6]
	todayFile := filepath.Join(memory.GetWorkspace(), "memory", monthDir, today+".md")
	
	err := os.MkdirAll(filepath.Dir(todayFile), 0o755)
	assert.NoError(t, err)
	
	noteContent := `# ` + today[:4] + "-" + today[4:6] + "-" + today[6:] + `

## Conversations

- 讨论了项目架构
- 帮助调试代码`
	
	err = os.WriteFile(todayFile, []byte(noteContent), 0o600)
	assert.NoError(t, err)

	tool := NewSearchMemoryTool(memory)
	ctx := context.Background()

	args := map[string]any{
		"query":       "项目架构",
		"memory_type": "daily_notes",
		"days":        float64(7),
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "项目架构")
}

func TestSearchMemoryTool_EmptyQuery(t *testing.T) {
	memory, _ := setupTestMemory(t)
	tool := NewSearchMemoryTool(memory)
	ctx := context.Background()

	args := map[string]any{
		"query": "",
	}

	result := tool.Execute(ctx, args)
	
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "query is required")
}

func TestSearchMemoryTool_CaseInsensitive(t *testing.T) {
	memory, cleanup := setupTestMemory(t)
	defer cleanup()

	// Setup test data with mixed case
	testContent := `# Long-term Memory

## User Information

- 用户喜欢 GO 编程`
	
	err := memory.WriteLongTerm(testContent)
	assert.NoError(t, err)

	tool := NewSearchMemoryTool(memory)
	ctx := context.Background()

	// Search with different case
	args := map[string]any{
		"query": "go", // lowercase
	}

	result := tool.Execute(ctx, args)
	
	assert.False(t, result.IsError)
	// Should find "GO" even though query is lowercase
	assert.Contains(t, result.ForLLM, "GO")
}
