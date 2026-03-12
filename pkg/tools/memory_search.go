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

// SearchMemoryTool searches through memory files.
type SearchMemoryTool struct {
	memory *MemoryStore
}

// NewSearchMemoryTool creates a new SearchMemoryTool.
func NewSearchMemoryTool(memory *MemoryStore) *SearchMemoryTool {
	return &SearchMemoryTool{
		memory: memory,
	}
}

// Name returns the tool name.
func (t *SearchMemoryTool) Name() string {
	return "search_memory"
}

// Description returns the tool description.
func (t *SearchMemoryTool) Description() string {
	return `Search through long-term memory and daily notes by keywords.
Use this to find previously stored information.

Search scope:
- long_term: Search only MEMORY.md
- daily_notes: Search daily note files (YYYYMMDD.md)
- all: Search both (default)

Returns matching excerpts with context.`
}

// Parameters returns the tool parameters.
func (t *SearchMemoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query (keywords or phrases)",
			},
			"memory_type": map[string]any{
				"type":        "string",
				"enum":        []string{"all", "long_term", "daily_notes"},
				"description": "Scope of search: 'all' (default), 'long_term', or 'daily_notes'",
			},
			"days": map[string]any{
				"type":        "number",
				"description": "Number of recent days to search for daily_notes (default: 7)",
			},
		},
		"required": []string{"query"},
	}
}

// Execute executes the tool.
func (t *SearchMemoryTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return ErrorResult("query is required and cannot be empty").
			WithError(fmt.Errorf("invalid query"))
	}

	memoryType, _ := args["memory_type"].(string)
	if memoryType == "" {
		memoryType = "all"
	}

	daysFloat, _ := args["days"].(float64)
	days := int(daysFloat)
	if days <= 0 {
		days = 7 // default to 7 days
	}

	var results []string

	// Search long-term memory
	if memoryType == "all" || memoryType == "long_term" {
		lmResults := t.searchLongTerm(query)
		if lmResults != "" {
			results = append(results, lmResults)
		}
	}

	// Search daily notes
	if memoryType == "all" || memoryType == "daily_notes" {
		dnResults := t.searchDailyNotes(query, days)
		if dnResults != "" {
			results = append(results, dnResults)
		}
	}

	if len(results) == 0 {
		return &ToolResult{
			ForLLM:  "No relevant memories found for the query.",
			ForUser: "🔍 未找到相关记忆",
			Silent:  false,
			IsError: false,
			Async:   false,
		}
	}

	// Format results
	formattedResults := strings.Join(results, "\n\n---\n\n")

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Found %d result(s):\n\n%s", len(results), formattedResults),
		ForUser: fmt.Sprintf("🔍 找到 %d 条相关记忆", len(results)),
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

// searchLongTerm searches in MEMORY.md.
func (t *SearchMemoryTool) searchLongTerm(query string) string {
	content := t.memory.ReadLongTerm()
	if content == "" {
		return ""
	}

	// Case-insensitive search
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)

	if !strings.Contains(contentLower, queryLower) {
		return ""
	}

	// Extract relevant paragraphs
	excerpts := t.extractRelevantExcerpts(content, queryLower)

	if len(excerpts) == 0 {
		return ""
	}

	return "**长期记忆**\n\n" + strings.Join(excerpts, "\n\n")
}

// searchDailyNotes searches in daily note files.
func (t *SearchMemoryTool) searchDailyNotes(query string, days int) string {
	var results []string
	queryLower := strings.ToLower(query)

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		
		// Get workspace from memory store
		workspace := t.getWorkspacePath()
		filePath := filepath.Join(workspace, "memory", monthDir, dateStr+".md")

		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // File doesn't exist, skip
		}

		contentStr := string(content)
		contentLower := strings.ToLower(contentStr)

		if strings.Contains(contentLower, queryLower) {
			excerpts := t.extractRelevantExcerpts(contentStr, queryLower)
			if len(excerpts) > 0 {
				header := fmt.Sprintf("**%s 的笔记**", date.Format("2006-01-02"))
				results = append(results, header+"\n\n"+strings.Join(excerpts, "\n\n"))
			}
		}
	}

	if len(results) == 0 {
		return ""
	}

	return "**日常笔记**\n\n" + strings.Join(results, "\n\n")
}

// extractRelevantExcerpts extracts paragraphs containing the query.
func (t *SearchMemoryTool) extractRelevantExcerpts(content, queryLower string) []string {
	var excerpts []string
	
	// Split into sections by headers (## or #)
	sections := strings.Split(content, "\n#")
	
	for _, section := range sections {
		sectionLower := strings.ToLower(section)
		
		// Check if section contains query
		if strings.Contains(sectionLower, queryLower) {
			// Clean up section
			cleanSection := strings.TrimSpace(section)
			if !strings.HasPrefix(cleanSection, "#") {
				cleanSection = "#" + cleanSection
			}
			
			// Limit excerpt length
			if len(cleanSection) > 500 {
				cleanSection = cleanSection[:500] + "..."
			}
			
			excerpts = append(excerpts, cleanSection)
		}
	}

	// If no section matches, try paragraph-level search
	if len(excerpts) == 0 {
		paragraphs := strings.Split(content, "\n\n")
		for _, para := range paragraphs {
			if strings.Contains(strings.ToLower(para), queryLower) {
				cleanPara := strings.TrimSpace(para)
				if len(cleanPara) > 300 {
					cleanPara = cleanPara[:300] + "..."
				}
				excerpts = append(excerpts, cleanPara)
				
				// Limit to 3 paragraphs
				if len(excerpts) >= 3 {
					break
				}
			}
		}
	}

	return excerpts
}

// getWorkspacePath extracts workspace path from MemoryStore.
func (t *SearchMemoryTool) getWorkspacePath() string {
	return t.memory.GetWorkspace()
}
