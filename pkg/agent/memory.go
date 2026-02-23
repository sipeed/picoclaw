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
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

// MemoryStore manages persistent memory for the agent.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
// - In-memory cache to reduce file I/O
type MemoryStore struct {
	workspace     string
	memoryDir     string
	memoryFile    string
	longTermCache string
	todayCache    string
	todayCacheKey string
	// Concurrency safety
	mu sync.RWMutex
	// File modification times for cache invalidation
	longTermMtime time.Time
	todayMtime    time.Time
}

// getCacheKey returns a cache key based on the current date.
func (ms *MemoryStore) getCacheKey() string {
	return time.Now().Format("20060102") // YYYYMMDD
}

// NewMemoryStore creates a new MemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0o755)

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
// Uses in-memory cache to reduce file I/O, but checks file mtime for cache invalidation.
func (ms *MemoryStore) ReadLongTerm() string {
	ms.mu.RLock()
	cache := ms.longTermCache
	mtime := ms.longTermMtime
	ms.mu.RUnlock()

	// Check file modification time to invalidate cache if file was edited externally
	if cache != "" {
		if info, err := os.Stat(ms.memoryFile); err == nil {
			if info.ModTime().After(mtime) {
				// File was modified externally, invalidate cache
				cache = ""
			}
		}
	}

	if cache != "" {
		return cache
	}

	// Read from file and update cache
	if data, err := os.ReadFile(ms.memoryFile); err == nil {
		content := string(data)
		ms.mu.Lock()
		ms.longTermCache = content
		if info, err := os.Stat(ms.memoryFile); err == nil {
			ms.longTermMtime = info.ModTime()
		}
		ms.mu.Unlock()
		return content
	}

	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
// Also updates the in-memory cache.
func (ms *MemoryStore) WriteLongTerm(content string) error {
	// Use unified atomic write utility with explicit sync for flash storage reliability.
	// Using 0o600 (owner read/write only) for secure default permissions.
	if err := fileutil.WriteFileAtomic(ms.memoryFile, []byte(content), 0o600); err != nil {
		return err
	}

	// Update cache on successful write
	ms.mu.Lock()
	ms.longTermCache = content
	if info, err := os.Stat(ms.memoryFile); err == nil {
		ms.longTermMtime = info.ModTime()
	}
	ms.mu.Unlock()
	return nil
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
// Uses in-memory cache to reduce file I/O, but checks file mtime for cache invalidation.
func (ms *MemoryStore) ReadToday() string {
	todayKey := ms.getCacheKey()
	todayFile := ms.getTodayFile()

	ms.mu.RLock()
	cacheKey := ms.todayCacheKey
	cache := ms.todayCache
	mtime := ms.todayMtime
	ms.mu.RUnlock()

	// Check if cache is valid for today and not expired
	if cacheKey == todayKey && cache != "" {
		// Check file modification time to invalidate cache if file was edited externally
		if info, err := os.Stat(todayFile); err == nil {
			if info.ModTime().After(mtime) {
				// File was modified externally, invalidate cache
				cache = ""
			}
		}
	}

	if cache != "" {
		return cache
	}

	// Read from file and update cache
	if data, err := os.ReadFile(todayFile); err == nil {
		content := string(data)
		ms.mu.Lock()
		ms.todayCache = content
		ms.todayCacheKey = todayKey
		if info, err := os.Stat(todayFile); err == nil {
			ms.todayMtime = info.ModTime()
		}
		ms.mu.Unlock()
		return content
	}

	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
// Also updates the in-memory cache.
func (ms *MemoryStore) AppendToday(content string) error {
	todayFile := ms.getTodayFile()
	todayKey := ms.getCacheKey()

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return err
	}

	// Get existing content from cache or file
	var existingContent string
	ms.mu.RLock()
	cacheKey := ms.todayCacheKey
	if cacheKey == todayKey {
		existingContent = ms.todayCache
	}
	ms.mu.RUnlock()

	// Fallback to file if cache is not valid
	if existingContent == "" {
		if data, err := os.ReadFile(todayFile); err == nil {
			existingContent = string(data)
		}
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

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	if err := fileutil.WriteFileAtomic(todayFile, []byte(newContent), 0o600); err != nil {
		return err
	}

	// Update cache on successful write
	ms.mu.Lock()
	ms.todayCache = newContent
	ms.todayCacheKey = todayKey
	if info, err := os.Stat(todayFile); err == nil {
		ms.todayMtime = info.ModTime()
	}
	ms.mu.Unlock()
	return nil
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	var sb strings.Builder
	first := true

	for i := range days {
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
