package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// MemSearchTool searches the agent's memory files (MEMORY.md + daily notes)
// using keyword matching. Returns matching lines with context.
type MemSearchTool struct {
	memoryDir string
}

func NewMemSearchTool(workspace string) *MemSearchTool {
	return &MemSearchTool{
		memoryDir: filepath.Join(workspace, "memory"),
	}
}

func (t *MemSearchTool) Name() string { return "mem_search" }

func (t *MemSearchTool) Description() string {
	return "Search through long-term memory and daily notes using keywords. Returns matching entries with surrounding context. Use this to recall specific facts, preferences, or past observations without loading all memory into context."
}

func (t *MemSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query (keywords separated by spaces, all must match). Case-insensitive.",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Optional: filter by category (preferences, facts, projects, decisions, observations). Leave empty to search all.",
				"enum":        []string{"", "preferences", "facts", "projects", "decisions", "observations"},
			},
			"days": map[string]interface{}{
				"type":        "number",
				"description": "How many days of daily notes to search (default: 30, max: 365). Set to 0 to search only MEMORY.md.",
			},
		},
		"required": []string{"query"},
	}
}

func (t *MemSearchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	query, _ := args["query"].(string)
	if query == "" {
		return ErrorResult("query is required")
	}

	category, _ := args["category"].(string)
	days := 30
	if d, ok := args["days"].(float64); ok {
		days = int(d)
		if days > 365 {
			days = 365
		}
	}

	keywords := strings.Fields(strings.ToLower(query))
	if len(keywords) == 0 {
		return ErrorResult("query must contain at least one keyword")
	}

	var results []searchResult

	// Search MEMORY.md
	memFile := filepath.Join(t.memoryDir, "MEMORY.md")
	if memResults := t.searchFile(memFile, keywords, category); len(memResults) > 0 {
		results = append(results, memResults...)
	}

	// Search daily notes
	if days > 0 {
		for i := 0; i < days; i++ {
			date := time.Now().AddDate(0, 0, -i)
			dateStr := date.Format("20060102")
			monthDir := dateStr[:6]
			filePath := filepath.Join(t.memoryDir, monthDir, dateStr+".md")
			if noteResults := t.searchFile(filePath, keywords, ""); len(noteResults) > 0 {
				results = append(results, noteResults...)
			}
		}
	}

	if len(results) == 0 {
		return SilentResult(fmt.Sprintf("No results found for query: %q", query))
	}

	// Cap results
	if len(results) > 20 {
		results = results[:20]
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Memory Search: %q\n\n", query))
	sb.WriteString(fmt.Sprintf("Found %d matches:\n\n", len(results)))

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("### %s (line %d)\n", r.source, r.line))
		if r.category != "" {
			sb.WriteString(fmt.Sprintf("**Category**: %s\n", r.category))
		}
		sb.WriteString(r.context)
		sb.WriteString("\n\n")
	}

	return SilentResult(sb.String())
}

type searchResult struct {
	source   string // file name
	line     int    // line number
	category string // detected category
	context  string // matching text with surrounding context
}

func (t *MemSearchTool) searchFile(filePath string, keywords []string, filterCategory string) []searchResult {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	source := filepath.Base(filePath)
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var results []searchResult
	currentCategory := ""
	categoryRe := regexp.MustCompile(`^##\s+(.+)`)

	for i, line := range lines {
		// Track current category from ## headers
		if m := categoryRe.FindStringSubmatch(line); len(m) > 1 {
			currentCategory = strings.ToLower(strings.TrimSpace(m[1]))
		}

		// Filter by category if specified
		if filterCategory != "" && !strings.Contains(currentCategory, strings.ToLower(filterCategory)) {
			continue
		}

		// Check if all keywords match this line
		lower := strings.ToLower(line)
		allMatch := true
		for _, kw := range keywords {
			if !strings.Contains(lower, kw) {
				allMatch = false
				break
			}
		}

		if !allMatch {
			continue
		}

		// Build context (2 lines before and after)
		start := i - 2
		if start < 0 {
			start = 0
		}
		end := i + 3
		if end > len(lines) {
			end = len(lines)
		}

		contextLines := lines[start:end]
		results = append(results, searchResult{
			source:   source,
			line:     i + 1,
			category: currentCategory,
			context:  strings.Join(contextLines, "\n"),
		})
	}

	return results
}

// MemSaveTool saves structured observations to memory with categorization.
type MemSaveTool struct {
	memoryDir  string
	memoryFile string
}

func NewMemSaveTool(workspace string) *MemSaveTool {
	memoryDir := filepath.Join(workspace, "memory")
	return &MemSaveTool{
		memoryDir:  memoryDir,
		memoryFile: filepath.Join(memoryDir, "MEMORY.md"),
	}
}

func (t *MemSaveTool) Name() string { return "mem_save" }

func (t *MemSaveTool) Description() string {
	return "Save a structured observation to long-term memory (MEMORY.md) under a specific category. Use this instead of write_file for memory operations. Categories: preferences, facts, projects, decisions, observations."
}

func (t *MemSaveTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Memory category to save under",
				"enum":        []string{"preferences", "facts", "projects", "decisions", "observations"},
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The observation or fact to save. Be concise but complete.",
			},
			"tags": map[string]interface{}{
				"type":        "string",
				"description": "Optional: comma-separated tags for easier retrieval (e.g. 'python,coding-style,formatting')",
			},
		},
		"required": []string{"category", "content"},
	}
}

func (t *MemSaveTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	category, _ := args["category"].(string)
	content, _ := args["content"].(string)
	tags, _ := args["tags"].(string)

	if category == "" {
		return ErrorResult("category is required")
	}
	if content == "" {
		return ErrorResult("content is required")
	}

	// Validate category
	validCategories := map[string]bool{
		"preferences": true, "facts": true, "projects": true,
		"decisions": true, "observations": true,
	}
	if !validCategories[category] {
		return ErrorResult(fmt.Sprintf("invalid category %q. Use: preferences, facts, projects, decisions, observations", category))
	}

	// Ensure memory directory exists
	os.MkdirAll(t.memoryDir, 0755)

	// Read existing memory
	existing := ""
	if data, err := os.ReadFile(t.memoryFile); err == nil {
		existing = string(data)
	}

	// Format the entry
	timestamp := time.Now().Format("2006-01-02")
	entry := fmt.Sprintf("- %s", content)
	if tags != "" {
		entry += fmt.Sprintf(" `[%s]`", tags)
	}
	entry += fmt.Sprintf(" _%s_", timestamp)

	// Category header (capitalized)
	categoryHeader := fmt.Sprintf("## %s", strings.ToUpper(category[:1])+category[1:])

	// Insert into the right category section
	if existing == "" {
		// Create new memory file with structure
		newContent := t.buildNewMemory(categoryHeader, entry)
		if err := os.WriteFile(t.memoryFile, []byte(newContent), 0644); err != nil {
			return ErrorResult(fmt.Sprintf("failed to write memory: %v", err))
		}
	} else {
		// Insert entry under existing category, or create the category
		newContent := t.insertIntoCategory(existing, categoryHeader, entry)
		if err := os.WriteFile(t.memoryFile, []byte(newContent), 0644); err != nil {
			return ErrorResult(fmt.Sprintf("failed to write memory: %v", err))
		}
	}

	return SilentResult(fmt.Sprintf("Saved to memory [%s]: %s", category, content))
}

func (t *MemSaveTool) buildNewMemory(categoryHeader, entry string) string {
	// Create structured memory file
	categories := []string{
		"## Preferences", "## Facts", "## Projects",
		"## Decisions", "## Observations",
	}

	var sb strings.Builder
	sb.WriteString("# Memory\n\n")

	for _, cat := range categories {
		sb.WriteString(cat + "\n\n")
		if cat == categoryHeader {
			sb.WriteString(entry + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (t *MemSaveTool) insertIntoCategory(existing, categoryHeader, entry string) string {
	lines := strings.Split(existing, "\n")

	// Find the category section
	categoryIdx := -1
	nextSectionIdx := -1

	for i, line := range lines {
		if strings.TrimSpace(line) == categoryHeader {
			categoryIdx = i
			// Find the next ## section
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "## ") {
					nextSectionIdx = j
					break
				}
			}
			break
		}
	}

	if categoryIdx == -1 {
		// Category doesn't exist, append it
		return existing + "\n\n" + categoryHeader + "\n\n" + entry + "\n"
	}

	// Insert entry before the next section (or at end)
	insertIdx := nextSectionIdx
	if insertIdx == -1 {
		insertIdx = len(lines)
	}

	// Find the last non-empty line before the next section to insert after it
	insertAt := insertIdx
	for i := insertIdx - 1; i > categoryIdx; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			insertAt = i + 1
			break
		}
	}

	// Insert the entry
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertAt]...)
	newLines = append(newLines, entry)
	newLines = append(newLines, lines[insertAt:]...)

	return strings.Join(newLines, "\n")
}

// MemIndexTool returns a compact index of all memory content.
// Used internally by progressive disclosure.
type MemIndexTool struct {
	memoryDir string
}

func NewMemIndexTool(workspace string) *MemIndexTool {
	return &MemIndexTool{
		memoryDir: filepath.Join(workspace, "memory"),
	}
}

func (t *MemIndexTool) Name() string { return "mem_index" }

func (t *MemIndexTool) Description() string {
	return "Get a compact index of all memory content (categories, entry counts, recent daily notes). Use this to understand what's in memory before searching for specific items."
}

func (t *MemIndexTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *MemIndexTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	index := t.buildIndex()
	return SilentResult(index)
}

func (t *MemIndexTool) buildIndex() string {
	var sb strings.Builder
	sb.WriteString("## Memory Index\n\n")

	// Index MEMORY.md categories
	memFile := filepath.Join(t.memoryDir, "MEMORY.md")
	if data, err := os.ReadFile(memFile); err == nil {
		content := string(data)
		categories := t.extractCategories(content)
		if len(categories) > 0 {
			sb.WriteString("### Long-term Memory\n")
			for _, cat := range categories {
				sb.WriteString(fmt.Sprintf("- **%s**: %d entries\n", cat.name, cat.count))
				// Show first 3 entries as preview
				for i, preview := range cat.previews {
					if i >= 3 {
						if cat.count > 3 {
							sb.WriteString(fmt.Sprintf("  - ... and %d more\n", cat.count-3))
						}
						break
					}
					sb.WriteString(fmt.Sprintf("  - %s\n", preview))
				}
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("### Long-term Memory\n_Empty - no MEMORY.md yet_\n\n")
	}

	// Index recent daily notes
	sb.WriteString("### Recent Daily Notes\n")
	noteCount := 0
	for i := 0; i < 7; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102")
		monthDir := dateStr[:6]
		filePath := filepath.Join(t.memoryDir, monthDir, dateStr+".md")
		if info, err := os.Stat(filePath); err == nil {
			noteCount++
			sb.WriteString(fmt.Sprintf("- **%s** (%d bytes)\n",
				date.Format("2006-01-02"), info.Size()))
		}
	}
	if noteCount == 0 {
		sb.WriteString("_No recent daily notes_\n")
	}

	return sb.String()
}

type categoryInfo struct {
	name     string
	count    int
	previews []string
}

func (t *MemIndexTool) extractCategories(content string) []categoryInfo {
	lines := strings.Split(content, "\n")
	var categories []categoryInfo
	var current *categoryInfo

	categoryRe := regexp.MustCompile(`^##\s+(.+)`)

	for _, line := range lines {
		if m := categoryRe.FindStringSubmatch(line); len(m) > 1 {
			if current != nil {
				categories = append(categories, *current)
			}
			current = &categoryInfo{name: strings.TrimSpace(m[1])}
			continue
		}

		if current != nil && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			current.count++
			// Create preview: first 80 chars of the entry
			preview := strings.TrimSpace(line)
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			current.previews = append(current.previews, preview)
		}
	}

	if current != nil {
		categories = append(categories, *current)
	}

	// Sort by count descending
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].count > categories[j].count
	})

	return categories
}
