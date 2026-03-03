// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"

	_ "modernc.org/sqlite"
)

// TurnRecord captures everything that happened during a single completed turn.
// It is persisted to turns.db for use by MemoryDigest and instant-memory assembly.
type TurnRecord struct {
	ID         string          // ULID or time-based unique ID
	Ts         int64           // Unix timestamp (seconds)
	ChannelKey string          // "channel:chatID"
	Score      int             // Phase 3 CalcTurnScore result
	Intent     string          // Phase 1 detected intent
	Tags       []string        // Phase 1 detected tags
	Tokens     int             // rough token estimate (chars / 3)
	Status     string          // "pending" | "processed" | "archived"
	UserMsg    string          // original user message
	Reply      string          // assistant final response
	ToolCalls  []ToolCallRecord // serialised as JSON in DB
}

// TurnStore manages persistent Turn storage in SQLite.
// The DB lives at {workspace}/turns.db, mirroring the memory.db pattern.
type TurnStore struct {
	db *sql.DB
}

const turnsDDL = `
CREATE TABLE IF NOT EXISTS turns (
    id          TEXT    PRIMARY KEY,
    ts          INTEGER NOT NULL,
    channel_key TEXT    NOT NULL DEFAULT '',
    score       INTEGER NOT NULL DEFAULT 0,
    intent      TEXT    NOT NULL DEFAULT '',
    tags        TEXT    NOT NULL DEFAULT '[]',
    tokens      INTEGER NOT NULL DEFAULT 0,
    status      TEXT    NOT NULL DEFAULT 'pending',
    user_msg    TEXT    NOT NULL DEFAULT '',
    reply       TEXT    NOT NULL DEFAULT '',
    tool_calls  TEXT    NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_turns_status  ON turns(status);
CREATE INDEX IF NOT EXISTS idx_turns_ts      ON turns(ts);
CREATE INDEX IF NOT EXISTS idx_turns_channel ON turns(channel_key);
CREATE INDEX IF NOT EXISTS idx_turns_score   ON turns(score);
`

// NewTurnStore creates (or opens) turns.db in the given workspace directory.
func NewTurnStore(workspace string) (*TurnStore, error) {
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return nil, fmt.Errorf("turn_store: mkdir %s: %w", workspace, err)
	}
	dbPath := filepath.Join(workspace, "turns.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("turn_store: open %s: %w", dbPath, err)
	}
	if _, err := db.Exec(turnsDDL); err != nil {
		db.Close()
		return nil, fmt.Errorf("turn_store: init schema: %w", err)
	}
	return &TurnStore{db: db}, nil
}

// Close shuts down the underlying DB connection.
func (s *TurnStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func marshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func unmarshalTags(raw string) []string {
	var tags []string
	_ = json.Unmarshal([]byte(raw), &tags)
	return tags
}

func unmarshalToolCalls(raw string) []ToolCallRecord {
	var tcs []ToolCallRecord
	_ = json.Unmarshal([]byte(raw), &tcs)
	return tcs
}

// estimateTokens gives a cheap estimate: characters / 3.
func estimateTokens(r TurnRecord) int {
	chars := len(r.UserMsg) + len(r.Reply)
	for _, tc := range r.ToolCalls {
		chars += len(tc.Name) + len(tc.Error)
	}
	if chars < 3 {
		return 1
	}
	return chars / 3
}

// NewTurnID generates a time-sortable unique ID without external dependencies.
// Format: unixMilli-randomSuffix using millisecond precision.
func NewTurnID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixMilli(), time.Now().Nanosecond()%1_000_000)
}

// ---------------------------------------------------------------------------
// Writes
// ---------------------------------------------------------------------------

// Insert persists a TurnRecord to the DB.
// The record's ID and Ts are set if empty/zero.
func (s *TurnStore) Insert(r TurnRecord) error {
	if r.ID == "" {
		r.ID = NewTurnID()
	}
	if r.Ts == 0 {
		r.Ts = time.Now().Unix()
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	if r.Tokens == 0 {
		r.Tokens = estimateTokens(r)
	}

	tagsJSON := marshalJSON(r.Tags)
	tcJSON := marshalJSON(r.ToolCalls)

	_, err := s.db.Exec(`
		INSERT INTO turns (id, ts, channel_key, score, intent, tags, tokens, status, user_msg, reply, tool_calls)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING`,
		r.ID, r.Ts, r.ChannelKey, r.Score, r.Intent,
		tagsJSON, r.Tokens, r.Status, r.UserMsg, r.Reply, tcJSON,
	)
	if err != nil {
		return fmt.Errorf("turn_store: insert %s: %w", r.ID, err)
	}

	logger.DebugCF("turn_store", "Turn inserted",
		map[string]any{"id": r.ID, "score": r.Score, "tokens": r.Tokens, "status": r.Status})
	return nil
}

// SetStatus updates the status of a turn by ID.
func (s *TurnStore) SetStatus(id, status string) error {
	_, err := s.db.Exec("UPDATE turns SET status = ? WHERE id = ?", status, id)
	return err
}

// ---------------------------------------------------------------------------
// Queries — used by MemoryDigest and instant-memory assembly
// ---------------------------------------------------------------------------

// QueryPending returns up to limit turns with status = 'pending', ordered oldest first.
func (s *TurnStore) QueryPending(limit int) ([]TurnRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, ts, channel_key, score, intent, tags, tokens, status, user_msg, reply, tool_calls
		FROM turns WHERE status = 'pending'
		ORDER BY ts ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTurns(rows)
}

// QueryByScore returns all turns with score >= highThreshold (always_keep),
// ordered by ts ASC.
func (s *TurnStore) QueryByScore(highThreshold int) ([]TurnRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, ts, channel_key, score, intent, tags, tokens, status, user_msg, reply, tool_calls
		FROM turns WHERE score >= ? AND status != 'archived'
		ORDER BY ts ASC`, highThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTurns(rows)
}

// QueryByTags returns turns whose tags JSON contains at least one of the given tags
// and score > 0, ordered by ts ASC.
func (s *TurnStore) QueryByTags(tags []string) ([]TurnRecord, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	// Build LIKE conditions for simple JSON array matching.
	conds := make([]string, 0, len(tags))
	args := make([]any, 0, len(tags)*2)
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		conds = append(conds, `(tags LIKE ? OR tags LIKE ?)`)
		args = append(args, `%"`+t+`"%`, `%'`+t+`'%`)
	}
	if len(conds) == 0 {
		return nil, nil
	}
	// Append non-archived filter.
	query := fmt.Sprintf(`
		SELECT id, ts, channel_key, score, intent, tags, tokens, status, user_msg, reply, tool_calls
		FROM turns
		WHERE score > 0 AND status != 'archived' AND (%s)
		ORDER BY ts ASC`, strings.Join(conds, " OR "))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTurns(rows)
}

// QueryRecent returns the n most-recent non-archived turns for a channelKey,
// ordered by ts ASC (oldest first, so they can be appended naturally).
func (s *TurnStore) QueryRecent(channelKey string, n int) ([]TurnRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, ts, channel_key, score, intent, tags, tokens, status, user_msg, reply, tool_calls
		FROM turns
		WHERE channel_key = ? AND status != 'archived'
		ORDER BY ts DESC LIMIT ?`, channelKey, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	turns, err := scanTurns(rows)
	if err != nil {
		return nil, err
	}
	// Reverse to ascending order.
	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}
	return turns, nil
}

// ArchiveOldProcessed marks processed turns older than olderThanDays as 'archived'.
// At most 100 rows are archived per call to limit lock time.
func (s *TurnStore) ArchiveOldProcessed(olderThanDays int) error {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays).Unix()
	_, err := s.db.Exec(`
		UPDATE turns SET status = 'archived'
		WHERE id IN (
			SELECT id FROM turns
			WHERE status = 'processed' AND ts < ?
			ORDER BY ts ASC LIMIT 100
		)`, cutoff)
	return err
}

// ---------------------------------------------------------------------------
// Internal scanner
// ---------------------------------------------------------------------------

func scanTurns(rows *sql.Rows) ([]TurnRecord, error) {
	var out []TurnRecord
	for rows.Next() {
		var r TurnRecord
		var tagsJSON, tcJSON string
		if err := rows.Scan(
			&r.ID, &r.Ts, &r.ChannelKey, &r.Score, &r.Intent,
			&tagsJSON, &r.Tokens, &r.Status,
			&r.UserMsg, &r.Reply, &tcJSON,
		); err != nil {
			return out, err
		}
		r.Tags = unmarshalTags(tagsJSON)
		r.ToolCalls = unmarshalToolCalls(tcJSON)
		out = append(out, r)
	}
	return out, rows.Err()
}
