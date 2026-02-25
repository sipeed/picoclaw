// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	vector     *memoryVectorStore
}

var memorySectionOrder = []string{
	"Profile",
	"Long-term Facts",
	"Active Goals",
	"Constraints",
	"Open Threads",
	"Deprecated/Resolved",
}

var memorySectionAliases = map[string]string{
	"profile":             "Profile",
	"long-term memory":    "Long-term Facts",
	"long term memory":    "Long-term Facts",
	"long-term facts":     "Long-term Facts",
	"active goals":        "Active Goals",
	"constraints":         "Constraints",
	"open threads":        "Open Threads",
	"open tasks":          "Open Threads",
	"pending tasks":       "Open Threads",
	"deprecated/resolved": "Deprecated/Resolved",
	"resolved":            "Deprecated/Resolved",
}

// NewMemoryStore creates a new MemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0o755)

	vectorSettings := defaultMemoryVectorSettings()

	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: memoryFile,
		vector:     newMemoryVectorStore(memoryDir, memoryFile, vectorSettings),
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
	if err := os.WriteFile(ms.memoryFile, []byte(content), 0o644); err != nil {
		return err
	}
	ms.refreshVectorIndex()
	return nil
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
	os.MkdirAll(monthDir, 0o755)

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

	if err := os.WriteFile(todayFile, []byte(newContent), 0o644); err != nil {
		return err
	}
	ms.refreshVectorIndex()
	return nil
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	var sb strings.Builder
	first := true

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")

		if data, err := os.ReadFile(filePath); err == nil {
			if !first {
				sb.WriteString("\n\n---\n\n")
			}
			sb.Write(data)
			first = false
		}
	}

	return sb.String()
}

// GetMemoryContext returns formatted memory context for the agent prompt.
// Includes long-term memory and recent daily notes.
func (ms *MemoryStore) GetMemoryContext() string {
	longTerm := ms.ReadLongTerm()
	recentNotes := ms.GetRecentDailyNotes(3)

	if longTerm == "" && recentNotes == "" {
		return ""
	}

	var sb strings.Builder

	if longTerm != "" {
		sb.WriteString("## Long-term Memory\n\n")
		sb.WriteString(longTerm)
	}

	if recentNotes != "" {
		if longTerm != "" {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString("## Recent Daily Notes\n\n")
		sb.WriteString(recentNotes)
	}

	return sb.String()
}

// OrganizeWriteback rewrites MEMORY.md using stable sections and deduplicated bullets.
// It preserves existing content while integrating newly extracted memory notes.
func (ms *MemoryStore) OrganizeWriteback(extracted string) error {
	base := parseMemorySections(ms.ReadLongTerm())
	incoming := parseMemorySections(extracted)

	for section, entries := range incoming {
		base[section] = append(base[section], entries...)
	}
	normalizeMemorySections(base)

	return ms.WriteLongTerm(renderMemorySections(base))
}

func (ms *MemoryStore) SetVectorSettings(settings MemoryVectorSettings) {
	if ms.vector == nil {
		return
	}
	ms.vector.SetSettings(settings)
}

// SearchRelevant runs semantic retrieval over MEMORY.md + recent daily notes.
func (ms *MemoryStore) SearchRelevant(query string, topK int, minScore float64) ([]MemoryVectorHit, error) {
	if ms.vector == nil {
		return nil, nil
	}
	return ms.vector.Search(query, topK, minScore)
}

func (ms *MemoryStore) GetBySource(source string) (MemoryVectorHit, bool, error) {
	if ms.vector == nil {
		return MemoryVectorHit{}, false, nil
	}
	return ms.vector.GetBySource(source)
}

func (ms *MemoryStore) refreshVectorIndex() {
	if ms.vector == nil {
		return
	}
	_ = ms.vector.Rebuild()
}

func parseMemorySections(content string) map[string][]string {
	sections := make(map[string][]string, len(memorySectionOrder))
	if strings.TrimSpace(content) == "" {
		return sections
	}

	current := "Long-term Facts"
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if normalized, ok := normalizeMemorySectionName(heading); ok {
				current = normalized
			}
			continue
		}

		entry := strings.TrimSpace(strings.TrimLeft(line, "-*+"))
		if entry == "" {
			continue
		}
		sections[current] = append(sections[current], entry)
	}
	return sections
}

func normalizeMemorySectionName(name string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return "", false
	}
	if section, ok := memorySectionAliases[key]; ok {
		return section, true
	}
	for _, section := range memorySectionOrder {
		if strings.EqualFold(section, name) {
			return section, true
		}
	}
	return "", false
}

func normalizeMemorySections(sections map[string][]string) {
	for section, entries := range sections {
		seen := map[string]struct{}{}
		deduped := make([]string, 0, len(entries))
		for _, entry := range entries {
			clean := strings.TrimSpace(entry)
			if clean == "" {
				continue
			}
			key := strings.ToLower(clean)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			deduped = append(deduped, clean)
		}
		sort.Strings(deduped)
		sections[section] = deduped
	}
}

func renderMemorySections(sections map[string][]string) string {
	var sb strings.Builder
	sb.WriteString("# MEMORY\n\n")
	sb.WriteString(fmt.Sprintf("_Last organized: %s_\n\n", time.Now().Format("2006-01-02 15:04")))

	wroteSection := false
	for _, section := range memorySectionOrder {
		entries := sections[section]
		if len(entries) == 0 {
			continue
		}
		wroteSection = true
		sb.WriteString("## ")
		sb.WriteString(section)
		sb.WriteString("\n")
		for _, entry := range entries {
			sb.WriteString("- ")
			sb.WriteString(entry)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if !wroteSection {
		sb.WriteString("## Long-term Facts\n")
		sb.WriteString("- (no durable facts recorded yet)\n")
	}

	return strings.TrimSpace(sb.String()) + "\n"
}
