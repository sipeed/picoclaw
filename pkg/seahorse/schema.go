package seahorse

import (
	"database/sql"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// SQL statements for FTS5 tables with trigram tokenizer.
const (
	sqlCreateSummariesFTS = `CREATE VIRTUAL TABLE IF NOT EXISTS summaries_fts USING fts5(
		summary_id,
		content,
		tokenize="trigram"
	)`
	sqlCreateMessagesFTS = `CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		message_id,
		content,
		tokenize="trigram"
	)`
	sqlCheckFTS5Available    = `CREATE VIRTUAL TABLE IF NOT EXISTS _fts5_check USING fts5(content)`
	sqlCheckTrigramAvailable = `CREATE VIRTUAL TABLE IF NOT EXISTS _trigram_check USING fts5(content, tokenize="trigram")`
	sqlDropFTS5Check         = `DROP TABLE IF EXISTS _fts5_check`
	sqlDropTrigramCheck      = `DROP TABLE IF EXISTS _trigram_check`
)

// runSchema creates or upgrades the database schema.
// All schemas are idempotent (safe to run multiple times).
func runSchema(db *sql.DB) error {
	// Check FTS5 support before creating tables
	if err := checkFTS5Support(db); err != nil {
		return fmt.Errorf("FTS5 check: %w", err)
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS conversations (
			conversation_id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_key     TEXT NOT NULL UNIQUE,
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS messages (
			message_id      INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL REFERENCES conversations(conversation_id),
			role            TEXT NOT NULL,
			content         TEXT NOT NULL DEFAULT '',
			token_count     INTEGER NOT NULL DEFAULT 0,
			created_at      TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS message_parts (
			part_id     INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id  INTEGER NOT NULL REFERENCES messages(message_id),
			type        TEXT NOT NULL,
			text        TEXT,
			name        TEXT,
			arguments   TEXT,
			tool_call_id TEXT,
			media_uri   TEXT,
			mime_type   TEXT,
			ordinal     INTEGER NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS summaries (
			summary_id                TEXT PRIMARY KEY,
			conversation_id           INTEGER NOT NULL REFERENCES conversations(conversation_id),
			kind                      TEXT NOT NULL,
			depth                     INTEGER NOT NULL DEFAULT 0,
			content                   TEXT NOT NULL,
			token_count               INTEGER NOT NULL DEFAULT 0,
			earliest_at               TEXT,
			latest_at                 TEXT,
			descendant_count          INTEGER NOT NULL DEFAULT 0,
			descendant_token_count    INTEGER NOT NULL DEFAULT 0,
			source_message_token_count INTEGER NOT NULL DEFAULT 0,
			model                     TEXT,
			created_at                TEXT NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS summary_parents (
			summary_id        TEXT NOT NULL,
			parent_summary_id TEXT NOT NULL,
			PRIMARY KEY (summary_id, parent_summary_id)
		)`,

		`CREATE TABLE IF NOT EXISTS summary_messages (
			summary_id TEXT NOT NULL,
			message_id INTEGER NOT NULL,
			ordinal    INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (summary_id, message_id)
		)`,

		`CREATE TABLE IF NOT EXISTS context_items (
			conversation_id INTEGER NOT NULL,
			ordinal         INTEGER NOT NULL,
			item_type       TEXT NOT NULL,
			summary_id      TEXT,
			message_id      INTEGER,
			token_count     INTEGER NOT NULL DEFAULT 0,
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (conversation_id, ordinal)
		)`,

		// FTS5 virtual table with trigram tokenizer for CJK support
		sqlCreateSummariesFTS,

		// FTS5 virtual table for message search with trigram tokenizer
		sqlCreateMessagesFTS,

		// Indexes for common query patterns
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(conversation_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_conversation ON summaries(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_kind_depth ON summaries(conversation_id, kind, depth)`,
		`CREATE INDEX IF NOT EXISTS idx_summary_parents_parent ON summary_parents(parent_summary_id)`,
		`CREATE INDEX IF NOT EXISTS idx_summary_messages_message ON summary_messages(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_context_items_conv ON context_items(conversation_id, ordinal)`,

		// FTS5 triggers to keep summaries_fts in sync with summaries table
		`CREATE TRIGGER IF NOT EXISTS summaries_ai AFTER INSERT ON summaries BEGIN
			INSERT INTO summaries_fts (summary_id, content) VALUES (new.summary_id, new.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS summaries_ad AFTER DELETE ON summaries BEGIN
			INSERT INTO summaries_fts (summaries_fts, summary_id, content) VALUES ('delete', old.summary_id, old.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS summaries_au AFTER UPDATE ON summaries BEGIN
			INSERT INTO summaries_fts (summaries_fts, summary_id, content) VALUES ('delete', old.summary_id, old.content);
			INSERT INTO summaries_fts (summary_id, content) VALUES (new.summary_id, new.content);
		END`,

		// FTS5 triggers to keep messages_fts in sync with messages table
		`CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
			INSERT INTO messages_fts (message_id, content) VALUES (new.message_id, new.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
			DELETE FROM messages_fts WHERE message_id = old.message_id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
			DELETE FROM messages_fts WHERE message_id = old.message_id;
			INSERT INTO messages_fts (message_id, content) VALUES (new.message_id, new.content);
		END`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

// checkFTS5Support verifies that SQLite has FTS5 with trigram tokenizer enabled.
// This is required for full-text search with CJK (Chinese, Japanese, Korean) support.
func checkFTS5Support(db *sql.DB) error {
	// Check if FTS5 is compiled in
	var fts5Enabled int
	err := db.QueryRow(`SELECT sqlite_compileoption_used('ENABLE_FTS5')`).Scan(&fts5Enabled)
	if err != nil {
		// sqlite_compileoption_used might not exist in older SQLite
		// Try a different approach: create a test FTS5 table
		_, testErr := db.Exec(sqlCheckFTS5Available)
		if testErr != nil {
			return fmt.Errorf("SQLite FTS5 not available: %w (required for full-text search)", testErr)
		}
		db.Exec(sqlDropFTS5Check)
	} else if fts5Enabled == 0 {
		return fmt.Errorf("SQLite was compiled without FTS5 support (required for full-text search)")
	}

	// Check if trigram tokenizer is available by trying to create a test table
	// Not all SQLite builds include the trigram tokenizer
	_, err = db.Exec(sqlCheckTrigramAvailable)
	if err != nil {
		logger.WarnCF("seahorse", "SQLite trigram tokenizer not available, CJK search may be limited",
			map[string]any{"error": err.Error()})
		// Trigram is not strictly required, just better for CJK
		// Don't return error, just log warning
	} else {
		db.Exec(sqlDropTrigramCheck)
	}

	return nil
}
