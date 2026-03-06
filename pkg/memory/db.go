// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package memory provides a structured, persistent memory database for the agent.
// It stores entries as a JSON file with full-text search and tag-based organization.
package memory

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry represents a stored memory.
type Entry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// brainFile is the on-disk format.
type brainFile struct {
	Version int      `json:"version"`
	Entries []*Entry `json:"entries"`
}

// DB is a lightweight in-memory database backed by a JSON file.
type DB struct {
	mu      sync.RWMutex
	path    string
	entries []*Entry
}

// Open opens or creates the memory database at workspace/memory/brain.db.
func Open(workspace string) (*DB, error) {
	memDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		return nil, fmt.Errorf("memory: failed to create directory: %w", err)
	}

	dbPath := filepath.Join(memDir, "brain.db")
	db := &DB{path: dbPath}

	if err := db.load(); err != nil {
		return nil, err
	}
	return db, nil
}

// Store saves a new memory entry and returns it.
func (db *DB) Store(content, category string, tags []string) (*Entry, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("memory: content must not be empty")
	}
	if category == "" {
		category = "general"
	}
	if tags == nil {
		tags = []string{}
	}
	now := time.Now().UTC()
	e := &Entry{
		ID:        newID(),
		Content:   content,
		Category:  category,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	db.mu.Lock()
	db.entries = append(db.entries, e)
	db.mu.Unlock()

	if err := db.save(); err != nil {
		return nil, err
	}
	return e, nil
}

// Search performs full-text search across content and tags.
// Returns up to limit entries ordered by relevance then recency.
func (db *DB) Search(query string, limit int) []*Entry {
	if limit <= 0 {
		limit = 10
	}
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return db.List("", limit)
	}

	type scored struct {
		entry *Entry
		score int
	}

	db.mu.RLock()
	results := make([]scored, 0, len(db.entries))
	for _, e := range db.entries {
		score := scoreEntry(e, tokens)
		if score > 0 {
			results = append(results, scored{e, score})
		}
	}
	db.mu.RUnlock()

	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].entry.CreatedAt.After(results[j].entry.CreatedAt)
	})

	out := make([]*Entry, 0, limit)
	for i, r := range results {
		if i >= limit {
			break
		}
		out = append(out, r.entry)
	}
	return out
}

// List returns recent entries, optionally filtered by category.
func (db *DB) List(category string, limit int) []*Entry {
	if limit <= 0 {
		limit = 20
	}

	db.mu.RLock()
	snapshot := make([]*Entry, len(db.entries))
	copy(snapshot, db.entries)
	db.mu.RUnlock()

	// Sort descending by creation time
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].CreatedAt.After(snapshot[j].CreatedAt)
	})

	out := make([]*Entry, 0, limit)
	for _, e := range snapshot {
		if category != "" && e.Category != category {
			continue
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// Delete removes an entry by ID. Returns error if not found.
func (db *DB) Delete(id string) error {
	db.mu.Lock()
	idx := -1
	for i, e := range db.entries {
		if e.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		db.mu.Unlock()
		return fmt.Errorf("memory: entry %q not found", id)
	}
	db.entries = append(db.entries[:idx], db.entries[idx+1:]...)
	db.mu.Unlock()

	return db.save()
}

// Count returns the total number of stored entries.
func (db *DB) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.entries)
}

// GetRecent returns the n most recent entries for context injection.
func (db *DB) GetRecent(n int) []*Entry {
	return db.List("", n)
}

// --- persistence ---

func (db *DB) load() error {
	data, err := os.ReadFile(db.path)
	if os.IsNotExist(err) {
		return nil // fresh start
	}
	if err != nil {
		return fmt.Errorf("memory: failed to read %s: %w", db.path, err)
	}

	var bf brainFile
	if err := json.Unmarshal(data, &bf); err != nil {
		return fmt.Errorf("memory: failed to parse %s: %w", db.path, err)
	}

	db.mu.Lock()
	db.entries = bf.Entries
	if db.entries == nil {
		db.entries = []*Entry{}
	}
	db.mu.Unlock()
	return nil
}

func (db *DB) save() error {
	db.mu.RLock()
	bf := brainFile{
		Version: 1,
		Entries: db.entries,
	}
	db.mu.RUnlock()

	data, err := json.MarshalIndent(bf, "", "  ")
	if err != nil {
		return fmt.Errorf("memory: failed to marshal: %w", err)
	}

	tmp := db.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("memory: failed to write temp: %w", err)
	}
	if err := os.Rename(tmp, db.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("memory: failed to rename: %w", err)
	}
	return nil
}

// --- helpers ---

func newID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

// tokenize splits a query into lowercase tokens.
func tokenize(s string) []string {
	s = strings.ToLower(s)
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == ';' || r == '.' || r == '!'
	})
	out := parts[:0]
	for _, p := range parts {
		if len(p) > 1 {
			out = append(out, p)
		}
	}
	return out
}

// scoreEntry returns a relevance score for an entry given query tokens.
// Higher = more relevant.
func scoreEntry(e *Entry, tokens []string) int {
	contentLow := strings.ToLower(e.Content)
	tagsLow := strings.ToLower(strings.Join(e.Tags, " "))
	categoryLow := strings.ToLower(e.Category)

	score := 0
	for _, tok := range tokens {
		// Exact word boundary in content scores highest
		if strings.Contains(contentLow, tok) {
			score += 2
		}
		// Tag match is high value
		if strings.Contains(tagsLow, tok) {
			score += 3
		}
		// Category match
		if strings.Contains(categoryLow, tok) {
			score += 1
		}
	}
	return score
}
