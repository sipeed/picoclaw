// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0755)

	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: memoryFile,
	}
}

// getTodayFile returns the path to today's daily note file (memory/YYYYMM/YYYYMMDD.md).
func (ms *MemoryStore) getTodayFile() string {
	today := time.Now().Format("20060102") // YYYYMMDD
	monthDir := today[:6]                  // YYYYMM
	filePath := filepath.Join(ms.memoryDir, monthDir, today+".md")
	return filePath
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadLongTerm() string {
	if data, err := os.ReadFile(ms.memoryFile); err == nil {
		return string(data)
	}
	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *MemoryStore) WriteLongTerm(content string) error {
	return os.WriteFile(ms.memoryFile, []byte(content), 0644)
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadToday() string {
	todayFile := ms.getTodayFile()
	if data, err := os.ReadFile(todayFile); err == nil {
		return string(data)
	}
	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
func (ms *MemoryStore) AppendToday(content string) error {
	todayFile := ms.getTodayFile()

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	os.MkdirAll(monthDir, 0755)

	var existingContent string
	if data, err := os.ReadFile(todayFile); err == nil {
		existingContent = string(data)
	}

	var newContent string
	if existingContent == "" {
		// Add header for new day
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		// Append to existing content
		newContent = existingContent + "\n" + content
	}

	return os.WriteFile(todayFile, []byte(newContent), 0644)
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	var notes []string

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")

		if data, err := os.ReadFile(filePath); err == nil {
			notes = append(notes, string(data))
		}
	}

	if len(notes) == 0 {
		return ""
	}

	// Join with separator
	var result string
	for i, note := range notes {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += note
	}
	return result
}

// GetMemoryContext returns formatted memory context for the agent prompt.
// Uses progressive disclosure: injects a compact index instead of full content.
// The agent can use mem_search and mem_index tools to retrieve details on demand.
func (ms *MemoryStore) GetMemoryContext() string {
	var sb strings.Builder
	sb.WriteString("# Memory\n\n")

	// Progressive disclosure: compact index of long-term memory
	longTerm := ms.ReadLongTerm()
	if longTerm != "" {
		index := ms.buildCompactIndex(longTerm)
		sb.WriteString("## Long-term Memory (Index)\n\n")
		sb.WriteString(index)
		sb.WriteString("\n_Use `mem_search` to find specific entries or `mem_index` for full index._\n\n")
	} else {
		sb.WriteString("## Long-term Memory\n_Empty — use `mem_save` to store important facts and preferences._\n\n")
	}

	// Recent daily notes: only today's content (compact)
	todayNote := ms.ReadToday()
	if todayNote != "" {
		// Truncate if too long (progressive disclosure)
		if len(todayNote) > 500 {
			todayNote = todayNote[:500] + "\n... (truncated, use `read_file` for full content)"
		}
		sb.WriteString("## Today's Notes\n\n")
		sb.WriteString(todayNote)
		sb.WriteString("\n\n")
	}

	// Show which days have notes (compact list)
	recentDays := ms.getRecentNoteDays(7)
	if len(recentDays) > 0 {
		sb.WriteString("## Recent Notes Available\n")
		for _, day := range recentDays {
			sb.WriteString(fmt.Sprintf("- %s\n", day))
		}
		sb.WriteString("_Use `mem_search` to search across all notes._\n")
	}

	return sb.String()
}

// GetFullMemoryContext returns the full (non-progressive) memory context.
// Used when the agent explicitly needs all memory content.
func (ms *MemoryStore) GetFullMemoryContext() string {
	var parts []string

	longTerm := ms.ReadLongTerm()
	if longTerm != "" {
		parts = append(parts, "## Long-term Memory\n\n"+longTerm)
	}

	recentNotes := ms.GetRecentDailyNotes(3)
	if recentNotes != "" {
		parts = append(parts, "## Recent Daily Notes\n\n"+recentNotes)
	}

	if len(parts) == 0 {
		return ""
	}

	var result string
	for i, part := range parts {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += part
	}
	return fmt.Sprintf("# Memory\n\n%s", result)
}

// buildCompactIndex creates a compact index of the memory file.
// Shows categories with entry counts and first-line previews.
// This is the core of progressive disclosure — ~70% token reduction.
func (ms *MemoryStore) buildCompactIndex(content string) string {
	categoryRe := regexp.MustCompile(`^##\s+(.+)`)

	type catEntry struct {
		name     string
		count    int
		previews []string
	}

	var categories []catEntry
	var current *catEntry

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()

		if m := categoryRe.FindStringSubmatch(line); len(m) > 1 {
			if current != nil {
				categories = append(categories, *current)
			}
			current = &catEntry{name: strings.TrimSpace(m[1])}
			continue
		}

		if current != nil && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			current.count++
			// Only keep first 2 as preview
			if len(current.previews) < 2 {
				preview := strings.TrimSpace(line)
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				current.previews = append(current.previews, preview)
			}
		}
	}
	if current != nil {
		categories = append(categories, *current)
	}

	if len(categories) == 0 {
		return "_No structured entries found._"
	}

	var sb strings.Builder
	totalEntries := 0
	for _, cat := range categories {
		totalEntries += cat.count
		sb.WriteString(fmt.Sprintf("**%s** (%d entries)", cat.name, cat.count))
		if len(cat.previews) > 0 {
			sb.WriteString(": ")
			sb.WriteString(cat.previews[0])
			if cat.count > 1 {
				sb.WriteString(fmt.Sprintf(" (+%d more)", cat.count-1))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("\n_Total: %d entries across %d categories_", totalEntries, len(categories)))

	return sb.String()
}

// getRecentNoteDays returns a list of recent dates that have daily notes.
func (ms *MemoryStore) getRecentNoteDays(days int) []string {
	var result []string
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102")
		monthDir := dateStr[:6]
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")
		if _, err := os.Stat(filePath); err == nil {
			result = append(result, date.Format("2006-01-02 (Monday)"))
		}
	}
	return result
}
