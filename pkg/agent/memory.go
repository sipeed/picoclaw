// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"

	_ "modernc.org/sqlite"
)

// MemoryStore manages persistent memory for the agent using SQLite.
//
// Schema:
//   - long_term: single-row table holding the long-term memory content
//   - daily_notes: one row per day (key = "YYYYMMDD")
//   - memory_entries: individually tagged memory items
//
// The database file is stored at workspace/memory.db.
type MemoryStore struct {
	workspace string
	db        *sql.DB
	mu        sync.Mutex // serialise writes
}

// NewMemoryStore creates a new MemoryStore backed by SQLite.
// It creates the database and tables if they do not exist.
func NewMemoryStore(workspace string) *MemoryStore {
	dbPath := filepath.Join(workspace, "memory.db")

	// Ensure workspace directory exists.
	os.MkdirAll(workspace, 0o755)

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		logger.DebugCF("memory", "Failed to open memory DB", map[string]any{"error": err.Error()})
		// Return a store that degrades gracefully (methods return empty / no-op).
		return &MemoryStore{workspace: workspace}
	}

	// Create tables.
	ddl := `
CREATE TABLE IF NOT EXISTS long_term (
	id      INTEGER PRIMARY KEY CHECK (id = 1),
	content TEXT NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO long_term (id, content) VALUES (1, '');

CREATE TABLE IF NOT EXISTS daily_notes (
	day     TEXT PRIMARY KEY,  -- YYYYMMDD
	content TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS memory_entries (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	content    TEXT    NOT NULL,
	tags       TEXT    NOT NULL DEFAULT '',  -- comma-separated, lowercase
	created_at TEXT    NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS cot_usage (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	intent      TEXT    NOT NULL DEFAULT '',
	tags        TEXT    NOT NULL DEFAULT '',  -- comma-separated tags from message analysis
	cot_prompt  TEXT    NOT NULL DEFAULT '',  -- LLM-generated thinking strategy
	message     TEXT    NOT NULL DEFAULT '',  -- first 200 chars of user message
	feedback    INTEGER NOT NULL DEFAULT 0,   -- -1=bad, 0=neutral, 1=good
	created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);
`
	if _, err := db.Exec(ddl); err != nil {
		logger.DebugCF("memory", "Failed to initialise memory DB tables", map[string]any{"error": err.Error()})
		db.Close()
		return &MemoryStore{workspace: workspace}
	}

	ms := &MemoryStore{
		workspace: workspace,
		db:        db,
	}

	// Migrate from legacy file-based storage if memory.db was just created.
	ms.migrateFromFiles()

	return ms
}

// Close closes the underlying database. Safe to call multiple times.
func (ms *MemoryStore) Close() {
	if ms.db != nil {
		ms.db.Close()
	}
}

// --- Long-term memory -------------------------------------------------------

// ReadLongTerm reads the long-term memory content.
// Returns empty string if the database is unavailable.
func (ms *MemoryStore) ReadLongTerm() string {
	if ms.db == nil {
		return ""
	}
	var content string
	err := ms.db.QueryRow("SELECT content FROM long_term WHERE id = 1").Scan(&content)
	if err != nil {
		return ""
	}
	return content
}

// WriteLongTerm replaces the long-term memory content.
func (ms *MemoryStore) WriteLongTerm(content string) error {
	if ms.db == nil {
		return fmt.Errorf("memory DB not available")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	_, err := ms.db.Exec("UPDATE long_term SET content = ? WHERE id = 1", content)
	return err
}

// --- Daily notes ------------------------------------------------------------

// todayKey returns today's date as "YYYYMMDD".
func todayKey() string {
	return time.Now().Format("20060102")
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist or the database is unavailable.
func (ms *MemoryStore) ReadToday() string {
	if ms.db == nil {
		return ""
	}
	var content string
	err := ms.db.QueryRow("SELECT content FROM daily_notes WHERE day = ?", todayKey()).Scan(&content)
	if err != nil {
		return ""
	}
	return content
}

// AppendToday appends content to today's daily note.
// If no note exists for today, a new one is created with a date header.
func (ms *MemoryStore) AppendToday(content string) error {
	if ms.db == nil {
		return fmt.Errorf("memory DB not available")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	key := todayKey()

	var existing string
	err := ms.db.QueryRow("SELECT content FROM daily_notes WHERE day = ?", key).Scan(&existing)
	if err == sql.ErrNoRows || existing == "" {
		// New day — add header.
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		content = header + content
		_, err = ms.db.Exec(
			"INSERT OR REPLACE INTO daily_notes (day, content) VALUES (?, ?)",
			key, content,
		)
	} else if err == nil {
		// Append to existing.
		content = existing + "\n" + content
		_, err = ms.db.Exec("UPDATE daily_notes SET content = ? WHERE day = ?", content, key)
	}
	return err
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	if ms.db == nil {
		return ""
	}

	var sb strings.Builder
	first := true

	for i := range days {
		date := time.Now().AddDate(0, 0, -i)
		key := date.Format("20060102")

		var content string
		err := ms.db.QueryRow("SELECT content FROM daily_notes WHERE day = ?", key).Scan(&content)
		if err == nil && content != "" {
			if !first {
				sb.WriteString("\n\n---\n\n")
			}
			sb.WriteString(content)
			first = false
		}
	}

	return sb.String()
}

// --- Tagged memory entries ---------------------------------------------------

// MemoryEntry represents a single tagged memory item.
type MemoryEntry struct {
	ID        int64
	Content   string
	Tags      []string
	CreatedAt string
	UpdatedAt string
}

// normaliseTags lowercases, trims, deduplicates, and sorts tags.
func normaliseTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

// joinTags joins tags with "," for storage.
func joinTags(tags []string) string {
	return strings.Join(normaliseTags(tags), ",")
}

// splitTags splits a stored tag string back into a slice.
func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// AddEntry inserts a new tagged memory entry. Returns the new entry ID.
func (ms *MemoryStore) AddEntry(content string, tags []string) (int64, error) {
	if ms.db == nil {
		return 0, fmt.Errorf("memory DB not available")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	res, err := ms.db.Exec(
		"INSERT INTO memory_entries (content, tags) VALUES (?, ?)",
		content, joinTags(tags),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateEntry updates the content and tags of an existing entry.
func (ms *MemoryStore) UpdateEntry(id int64, content string, tags []string) error {
	if ms.db == nil {
		return fmt.Errorf("memory DB not available")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	_, err := ms.db.Exec(
		"UPDATE memory_entries SET content = ?, tags = ?, updated_at = datetime('now') WHERE id = ?",
		content, joinTags(tags), id,
	)
	return err
}

// DeleteEntry removes a memory entry by ID.
func (ms *MemoryStore) DeleteEntry(id int64) error {
	if ms.db == nil {
		return fmt.Errorf("memory DB not available")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	_, err := ms.db.Exec("DELETE FROM memory_entries WHERE id = ?", id)
	return err
}

// GetEntry retrieves a single memory entry by ID.
func (ms *MemoryStore) GetEntry(id int64) (*MemoryEntry, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	var e MemoryEntry
	var tagsStr string
	err := ms.db.QueryRow(
		"SELECT id, content, tags, created_at, updated_at FROM memory_entries WHERE id = ?", id,
	).Scan(&e.ID, &e.Content, &tagsStr, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Tags = splitTags(tagsStr)
	return &e, nil
}

// SearchByTag returns all entries that contain the given tag.
// Tag matching is case-insensitive (tags are stored lowercase).
func (ms *MemoryStore) SearchByTag(tag string) ([]MemoryEntry, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return nil, nil
	}

	// Match: exact tag as whole string, at start, at end, or in the middle.
	// Pattern: tag OR tag,... OR ...,tag OR ...,tag,...
	rows, err := ms.db.Query(
		`SELECT id, content, tags, created_at, updated_at FROM memory_entries
		 WHERE tags = ? OR tags LIKE ? OR tags LIKE ? OR tags LIKE ?
		 ORDER BY updated_at DESC`,
		tag, tag+",%", "%,"+tag, "%,"+tag+",%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// SearchByTags returns entries that contain ALL of the given tags.
func (ms *MemoryStore) SearchByTags(tags []string) ([]MemoryEntry, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	tags = normaliseTags(tags)
	if len(tags) == 0 {
		return nil, nil
	}

	// Build WHERE clause: each tag must match.
	conds := make([]string, 0, len(tags))
	args := make([]any, 0, len(tags)*4)
	for _, tag := range tags {
		conds = append(conds,
			"(tags = ? OR tags LIKE ? OR tags LIKE ? OR tags LIKE ?)")
		args = append(args, tag, tag+",%", "%,"+tag, "%,"+tag+",%")
	}

	query := fmt.Sprintf(
		"SELECT id, content, tags, created_at, updated_at FROM memory_entries WHERE %s ORDER BY updated_at DESC",
		strings.Join(conds, " AND "),
	)

	rows, err := ms.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// SearchByAnyTag returns entries that contain ANY of the given tags (OR logic).
// Results are deduplicated and ordered by updated_at DESC, limited to 20 entries.
func (ms *MemoryStore) SearchByAnyTag(tags []string) ([]MemoryEntry, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	tags = normaliseTags(tags)
	if len(tags) == 0 {
		return nil, nil
	}

	// Build WHERE clause: any tag may match (OR).
	conds := make([]string, 0, len(tags))
	args := make([]any, 0, len(tags)*4)
	for _, tag := range tags {
		conds = append(conds,
			"(tags = ? OR tags LIKE ? OR tags LIKE ? OR tags LIKE ?)")
		args = append(args, tag, tag+",%", "%,"+tag, "%,"+tag+",%")
	}

	query := fmt.Sprintf(
		"SELECT id, content, tags, created_at, updated_at FROM memory_entries WHERE %s ORDER BY updated_at DESC LIMIT 20",
		strings.Join(conds, " OR "),
	)

	rows, err := ms.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// ListAllTags returns all unique tags used across memory entries.
func (ms *MemoryStore) ListAllTags() ([]string, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}

	rows, err := ms.db.Query("SELECT DISTINCT tags FROM memory_entries WHERE tags != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	for rows.Next() {
		var tagsStr string
		if err := rows.Scan(&tagsStr); err != nil {
			continue
		}
		for _, t := range splitTags(tagsStr) {
			seen[t] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for t := range seen {
		result = append(result, t)
	}
	return result, nil
}

// ListEntries returns the most recent N entries (all tags), ordered newest first.
func (ms *MemoryStore) ListEntries(limit int) ([]MemoryEntry, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := ms.db.Query(
		"SELECT id, content, tags, created_at, updated_at FROM memory_entries ORDER BY updated_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// scanEntries is a helper to scan rows into MemoryEntry slices.
func scanEntries(rows *sql.Rows) ([]MemoryEntry, error) {
	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var tagsStr string
		if err := rows.Scan(&e.ID, &e.Content, &tagsStr, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return entries, err
		}
		e.Tags = splitTags(tagsStr)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// --- Composite context ------------------------------------------------------

// GetMemoryContext returns formatted memory context for the agent prompt.
// Includes long-term memory, recent daily notes, and recent tagged entries.
func (ms *MemoryStore) GetMemoryContext() string {
	longTerm := ms.ReadLongTerm()
	recentNotes := ms.GetRecentDailyNotes(3)

	var sb strings.Builder
	hasContent := false

	if longTerm != "" {
		sb.WriteString("## Long-term Memory\n\n")
		sb.WriteString(longTerm)
		hasContent = true
	}

	if recentNotes != "" {
		if hasContent {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString("## Recent Daily Notes\n\n")
		sb.WriteString(recentNotes)
		hasContent = true
	}

	// Include recent tagged memory entries.
	entries, _ := ms.ListEntries(10)
	if len(entries) > 0 {
		if hasContent {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString("## Tagged Memories\n\n")
		for _, e := range entries {
			tagLabel := ""
			if len(e.Tags) > 0 {
				tagLabel = " [" + strings.Join(e.Tags, ", ") + "]"
			}
			fmt.Fprintf(&sb, "- (#%d%s) %s\n", e.ID, tagLabel, e.Content)
		}
		hasContent = true
	}

	if !hasContent {
		return ""
	}
	return sb.String()
}

// --- CoT usage tracking (learning) ------------------------------------------

// CotUsageRecord represents a single CoT usage entry.
type CotUsageRecord struct {
	ID        int64
	Intent    string
	Tags      []string // Tags from the message analysis
	CotPrompt string   // LLM-generated thinking strategy
	Message   string
	Feedback  int // -1=bad, 0=neutral, 1=good
	CreatedAt string
}

// CotStats holds aggregated statistics for an intent.
type CotStats struct {
	Intent    string
	TotalUses int
	AvgScore  float64 // Average feedback score
	LastUsed  string
}

// RecordCotUsage logs a CoT usage event with the LLM-generated prompt and tags.
// messagePreview is truncated to 200 characters.
func (ms *MemoryStore) RecordCotUsage(intent string, tags []string, cotPrompt, message string) (int64, error) {
	if ms.db == nil {
		return 0, fmt.Errorf("memory DB not available")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Truncate message preview.
	if len(message) > 200 {
		message = message[:200]
	}

	tagStr := strings.Join(tags, ",")
	res, err := ms.db.Exec(
		"INSERT INTO cot_usage (intent, tags, cot_prompt, message) VALUES (?, ?, ?, ?)",
		intent, tagStr, cotPrompt, message,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateCotFeedback updates the feedback score for a CoT usage record.
// score: -1=bad, 0=neutral, 1=good.
func (ms *MemoryStore) UpdateCotFeedback(id int64, score int) error {
	if ms.db == nil {
		return fmt.Errorf("memory DB not available")
	}
	if score < -1 || score > 1 {
		return fmt.Errorf("feedback score must be -1, 0, or 1")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	_, err := ms.db.Exec("UPDATE cot_usage SET feedback = ? WHERE id = ?", score, id)
	return err
}

// UpdateLatestCotFeedback updates the feedback score for the most recent
// CoT usage record. This is useful when the user provides feedback after
// the main LLM has responded (at which point the usage ID may not be tracked).
func (ms *MemoryStore) UpdateLatestCotFeedback(score int) error {
	if ms.db == nil {
		return fmt.Errorf("memory DB not available")
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	_, err := ms.db.Exec(
		"UPDATE cot_usage SET feedback = ? WHERE id = (SELECT MAX(id) FROM cot_usage)",
		score,
	)
	return err
}

// GetCotStats returns aggregated statistics per intent,
// based on usage in the last N days. Ordered by total uses descending.
func (ms *MemoryStore) GetCotStats(days int) ([]CotStats, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	if days <= 0 {
		days = 30
	}

	rows, err := ms.db.Query(`
		SELECT
			intent,
			COUNT(*) as total_uses,
			COALESCE(AVG(CASE WHEN feedback != 0 THEN CAST(feedback AS REAL) END), 0.0) as avg_score,
			MAX(created_at) as last_used
		FROM cot_usage
		WHERE created_at >= datetime('now', ? || ' days')
		GROUP BY intent
		ORDER BY total_uses DESC
	`, fmt.Sprintf("-%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []CotStats
	for rows.Next() {
		var s CotStats
		if err := rows.Scan(&s.Intent, &s.TotalUses, &s.AvgScore, &s.LastUsed); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetCotIntentStats returns usage stats per intent.
// This is a simpler version that just counts per intent.
func (ms *MemoryStore) GetCotIntentStats(days int) ([]CotStats, error) {
	return ms.GetCotStats(days)
}

// GetTopRatedCotPrompts returns the highest-rated generated CoT prompts.
// If filterTags is non-empty, prioritises prompts that share tags with the query.
// These serve as proven examples for future LLM generation.
func (ms *MemoryStore) GetTopRatedCotPrompts(days, limit int, filterTags []string) ([]CotUsageRecord, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	if days <= 0 {
		days = 30
	}
	if limit <= 0 {
		limit = 5
	}

	rows, err := ms.db.Query(`
		SELECT id, intent, tags, cot_prompt, message, feedback, created_at
		FROM cot_usage
		WHERE feedback > 0
		  AND cot_prompt != ''
		  AND created_at >= datetime('now', ? || ' days')
		ORDER BY feedback DESC, created_at DESC
		LIMIT ?
	`, fmt.Sprintf("-%d", days), limit*3) // Over-fetch to filter by tags later.
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []CotUsageRecord
	for rows.Next() {
		var r CotUsageRecord
		var tagStr string
		if err := rows.Scan(&r.ID, &r.Intent, &tagStr, &r.CotPrompt, &r.Message, &r.Feedback, &r.CreatedAt); err != nil {
			continue
		}
		if tagStr != "" {
			r.Tags = strings.Split(tagStr, ",")
		}
		all = append(all, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// If filter tags provided, sort by tag overlap (most relevant first).
	if len(filterTags) > 0 && len(all) > 0 {
		tagSet := make(map[string]bool, len(filterTags))
		for _, t := range filterTags {
			tagSet[strings.ToLower(t)] = true
		}

		// Partition: matching first, then non-matching.
		var matching, rest []CotUsageRecord
		for _, r := range all {
			hasOverlap := false
			for _, t := range r.Tags {
				if tagSet[strings.ToLower(t)] {
					hasOverlap = true
					break
				}
			}
			if hasOverlap {
				matching = append(matching, r)
			} else {
				rest = append(rest, r)
			}
		}
		all = append(matching, rest...)
	}

	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// GetRecentCotUsage returns the N most recent CoT usage records.
func (ms *MemoryStore) GetRecentCotUsage(limit int) ([]CotUsageRecord, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("memory DB not available")
	}
	if limit <= 0 {
		limit = 20
	}

	rows, err := ms.db.Query(
		"SELECT id, intent, tags, cot_prompt, message, feedback, created_at FROM cot_usage ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CotUsageRecord
	for rows.Next() {
		var r CotUsageRecord
		var tagStr string
		if err := rows.Scan(&r.ID, &r.Intent, &tagStr, &r.CotPrompt, &r.Message, &r.Feedback, &r.CreatedAt); err != nil {
			continue
		}
		if tagStr != "" {
			r.Tags = strings.Split(tagStr, ",")
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// FormatCotLearningContext formats CoT usage history and top-rated prompts
// into a string for the pre-LLM to learn from past generations.
// currentTags are the tags extracted from the current message, used to
// prioritise relevant proven strategies.
func (ms *MemoryStore) FormatCotLearningContext(days int, currentTags []string) string {
	var sb strings.Builder
	hasContent := false

	// 1. Usage stats per intent.
	stats, err := ms.GetCotStats(days)
	if err == nil && len(stats) > 0 {
		sb.WriteString("## Historical Usage Stats\n\n")
		for _, s := range stats {
			scoreLabel := "neutral"
			if s.AvgScore > 0.3 {
				scoreLabel = "good"
			} else if s.AvgScore < -0.3 {
				scoreLabel = "poor"
			}
			fmt.Fprintf(&sb, "- Intent '%s': %d uses, avg feedback=%s (%.1f)\n",
				s.Intent, s.TotalUses, scoreLabel, s.AvgScore)
		}
		sb.WriteString("\n")
		hasContent = true
	}

	// 2. Top-rated generated prompts as proven examples (filtered by current tags).
	topPrompts, err := ms.GetTopRatedCotPrompts(days, 3, currentTags)
	if err == nil && len(topPrompts) > 0 {
		sb.WriteString("## Proven Strategies (from past sessions with positive feedback)\n\n")
		sb.WriteString("These generated strategies received positive feedback. Use similar approaches for similar intents.\n\n")
		for i, r := range topPrompts {
			msgPreview := r.Message
			if len(msgPreview) > 80 {
				msgPreview = msgPreview[:80] + "..."
			}
			tagLabel := ""
			if len(r.Tags) > 0 {
				tagLabel = fmt.Sprintf(", tags: [%s]", strings.Join(r.Tags, ", "))
			}
			fmt.Fprintf(&sb, "### Proven #%d (intent: %s%s, message: \"%s\")\n%s\n\n",
				i+1, r.Intent, tagLabel, msgPreview, r.CotPrompt)
		}
		hasContent = true
	}

	if !hasContent {
		return ""
	}
	return sb.String()
}

// --- Migration from legacy files --------------------------------------------

// migrateFromFiles imports data from the old file-based storage
// (memory/MEMORY.md and memory/YYYYMM/YYYYMMDD.md) into SQLite.
// It only runs if the long_term content is empty (fresh DB) AND the
// legacy directory exists. After a successful migration the legacy
// directory is renamed to memory_backup.
func (ms *MemoryStore) migrateFromFiles() {
	if ms.db == nil {
		return
	}

	memoryDir := filepath.Join(ms.workspace, "memory")

	// Check if the legacy directory exists.
	info, err := os.Stat(memoryDir)
	if err != nil || !info.IsDir() {
		return // nothing to migrate
	}

	// Only migrate if the DB is empty (fresh).
	longTerm := ms.ReadLongTerm()
	if longTerm != "" {
		return // already has data
	}

	logger.DebugCF("memory", "Migrating legacy file-based memory to SQLite", nil)

	// 1. Long-term memory.
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")
	if data, err := os.ReadFile(memoryFile); err == nil && len(data) > 0 {
		ms.WriteLongTerm(string(data))
	}

	// 2. Daily notes — walk YYYYMM/YYYYMMDD.md files.
	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		monthDir := filepath.Join(memoryDir, entry.Name())
		dayFiles, err := os.ReadDir(monthDir)
		if err != nil {
			continue
		}
		for _, df := range dayFiles {
			name := df.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}
			day := strings.TrimSuffix(name, ".md") // YYYYMMDD
			if len(day) != 8 {
				continue
			}
			data, err := os.ReadFile(filepath.Join(monthDir, name))
			if err != nil || len(data) == 0 {
				continue
			}
			ms.mu.Lock()
			ms.db.Exec(
				"INSERT OR IGNORE INTO daily_notes (day, content) VALUES (?, ?)",
				day, string(data),
			)
			ms.mu.Unlock()
		}
	}

	// Rename legacy dir so we don't migrate again.
	backupDir := filepath.Join(ms.workspace, "memory_backup")
	if err := os.Rename(memoryDir, backupDir); err != nil {
		logger.DebugCF("memory", "Could not rename legacy memory dir", map[string]any{"error": err.Error()})
	} else {
		logger.DebugCF("memory", "Legacy memory migrated and backed up", map[string]any{"backup": backupDir})
	}
}
