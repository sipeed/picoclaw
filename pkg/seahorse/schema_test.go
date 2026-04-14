package seahorse

import (
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	_ "modernc.org/sqlite"
)

var testDBCounter uint64

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	n := atomic.AddUint64(&testDBCounter, 1)
	testName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	// Use a shared in-memory database so concurrent goroutines/connections in tests
	// observe the same schema/data.
	dsn := fmt.Sprintf("file:seahorse_test_%s_%d?mode=memory&cache=shared", testName, n)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRunMigrations(t *testing.T) {
	db := openTestDB(t)

	if err := runSchema(db); err != nil {
		t.Fatalf("runSchema: %v", err)
	}

	// Verify all tables exist
	tables := []string{
		"conversations",
		"messages",
		"message_parts",
		"summaries",
		"summary_parents",
		"summary_messages",
		"context_items",
	}
	for _, tbl := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}

	// Verify FTS5 virtual table exists
	var ftsName string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='summaries_fts'",
	).Scan(&ftsName)
	if err != nil {
		t.Errorf("FTS5 table summaries_fts not found: %v", err)
	}
}

func TestRunMigrationsIdempotent(t *testing.T) {
	db := openTestDB(t)

	// Run migrations twice — should succeed both times
	if err := runSchema(db); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := runSchema(db); err != nil {
		t.Fatalf("second migration (idempotent): %v", err)
	}

	// Verify we can still insert data after double migration
	res, err := db.Exec(
		"INSERT INTO conversations (session_key, created_at, updated_at) VALUES (?, datetime('now'), datetime('now'))",
		"test-session",
	)
	if err != nil {
		t.Fatalf("insert after double migration: %v", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		t.Error("expected non-zero conversation id")
	}
}

func TestMigrationConversationUnique(t *testing.T) {
	db := openTestDB(t)
	if err := runSchema(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	// Insert first
	_, err := db.Exec(
		"INSERT INTO conversations (session_key, created_at, updated_at) VALUES (?, datetime('now'), datetime('now'))",
		"unique-key",
	)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// Duplicate should fail
	_, err = db.Exec(
		"INSERT INTO conversations (session_key, created_at, updated_at) VALUES (?, datetime('now'), datetime('now'))",
		"unique-key",
	)
	if err == nil {
		t.Error("expected unique constraint violation for duplicate session_key")
	}
}

func TestMigrationSummaryFTSInsert(t *testing.T) {
	db := openTestDB(t)
	if err := runSchema(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	// Insert a conversation first
	_, err := db.Exec(
		"INSERT INTO conversations (session_key, created_at, updated_at) VALUES (?, datetime('now'), datetime('now'))",
		"fts-test",
	)
	if err != nil {
		t.Fatalf("insert conversation: %v", err)
	}

	// Insert a summary
	_, err = db.Exec(
		`INSERT INTO summaries (summary_id, conversation_id, kind, depth, content, token_count, created_at)
		 VALUES ('sum_test1', 1, 'leaf', 0, '你好世界 hello world', 10, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert summary: %v", err)
	}

	// FTS should find it — trigram tokenizer requires >= 3 chars
	rows, err := db.Query(
		"SELECT summary_id FROM summaries_fts WHERE summaries_fts MATCH ?",
		"你好世",
	)
	if err != nil {
		t.Fatalf("FTS query: %v", err)
	}
	defer rows.Close()

	var found string
	if rows.Next() {
		if err := rows.Scan(&found); err != nil {
			t.Fatalf("scan: %v", err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	if found != "sum_test1" {
		t.Errorf("FTS: expected 'sum_test1', got %q", found)
	}
}

func TestMigrationSummaryParentsPK(t *testing.T) {
	db := openTestDB(t)
	if err := runSchema(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	// Insert two summaries
	for _, id := range []string{"sum_a", "sum_b"} {
		_, err := db.Exec(
			`INSERT INTO summaries (summary_id, conversation_id, kind, depth, content, token_count, created_at)
			 VALUES (?, 1, 'leaf', 0, 'content', 5, datetime('now'))`, id)
		if err != nil {
			t.Fatalf("insert summary %s: %v", id, err)
		}
	}

	// Link child to parent
	_, err := db.Exec(
		"INSERT INTO summary_parents (summary_id, parent_summary_id) VALUES ('sum_a', 'sum_b')")
	if err != nil {
		t.Fatalf("link: %v", err)
	}

	// Duplicate link should fail (composite PK)
	_, err = db.Exec(
		"INSERT INTO summary_parents (summary_id, parent_summary_id) VALUES ('sum_a', 'sum_b')")
	if err == nil {
		t.Error("expected unique constraint violation for duplicate summary_parents link")
	}
}

func TestTriggerMigration(t *testing.T) {
	db := openTestDB(t)

	// Run schema once to create tables and (correct) triggers
	if err := runSchema(db); err != nil {
		t.Fatalf("runSchema: %v", err)
	}

	// Drop correct triggers and recreate them with the old buggy body.
	// The old trigger used INSERT INTO fts VALUES('delete', ...) which is wrong
	// for non-external-content FTS5 tables.
	oldSummariesDelete := `CREATE TRIGGER summaries_ad AFTER DELETE ON summaries BEGIN
		INSERT INTO summaries_fts (summaries_fts, summary_id, content) VALUES('delete', old.summary_id, old.content);
	END`
	oldMessagesDelete := `CREATE TRIGGER messages_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts (messages_fts, message_id, content) VALUES('delete', old.message_id, old.content);
	END`

	for _, sql := range []string{
		`DROP TRIGGER IF EXISTS summaries_ad`,
		`DROP TRIGGER IF EXISTS messages_ad`,
		oldSummariesDelete,
		oldMessagesDelete,
	} {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("setup old trigger: %v", err)
		}
	}

	// Insert a conversation and summary so we have something to delete
	_, err := db.Exec(`INSERT INTO conversations (session_key) VALUES ('old-db-test')`)
	if err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	_, err = db.Exec(`INSERT INTO summaries (summary_id, conversation_id, kind, depth, content, token_count)
		VALUES ('old-sum', 1, 'leaf', 0, 'old content', 5)`)
	if err != nil {
		t.Fatalf("insert summary: %v", err)
	}

	// The old trigger body is wrong for normal FTS5 — DELETE should fail.
	_, err = db.Exec(`DELETE FROM summaries WHERE summary_id = 'old-sum'`)
	if err == nil {
		t.Error("expected error from old buggy trigger, but DELETE succeeded")
	} else {
		t.Logf("old trigger correctly causes error: %v", err)
	}

	// Now runSchema again — this drops and recreates the triggers with correct bodies.
	err = runSchema(db)
	if err != nil {
		t.Fatalf("runSchema migration: %v", err)
	}

	// Insert again so we have data to delete
	_, err = db.Exec(`INSERT INTO summaries (summary_id, conversation_id, kind, depth, content, token_count)
		VALUES ('migrated-sum', 1, 'leaf', 0, 'new content', 5)`)
	if err != nil {
		t.Fatalf("insert after migration: %v", err)
	}

	// DELETE should now work with the corrected trigger body.
	_, err = db.Exec(`DELETE FROM summaries WHERE summary_id = 'migrated-sum'`)
	if err != nil {
		t.Fatalf("DELETE after migration failed (trigger not corrected): %v", err)
	}

	// Verify the summary is gone
	var count int
	err = db.QueryRow(`SELECT count(*) FROM summaries WHERE summary_id = 'migrated-sum'`).Scan(&count)
	if err != nil {
		t.Fatalf("query after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("summary should be gone after DELETE, got count=%d", count)
	}
}

func TestFTS5SQLConstants(t *testing.T) {
	db := openTestDB(t)

	// Verify FTS5 check SQL executes without error
	_, err := db.Exec(sqlCheckFTS5Available)
	if err != nil {
		t.Errorf("sqlCheckFTS5Available failed: %v", err)
	}

	// Verify trigram check SQL executes without error
	_, err = db.Exec(sqlCheckTrigramAvailable)
	if err != nil {
		t.Errorf("sqlCheckTrigramAvailable failed: %v", err)
	}

	// Verify summaries_fts SQL executes without error
	_, err = db.Exec(sqlCreateSummariesFTS)
	if err != nil {
		t.Errorf("sqlCreateSummariesFTS failed: %v", err)
	}

	// Verify messages_fts SQL executes without error
	_, err = db.Exec(sqlCreateMessagesFTS)
	if err != nil {
		t.Errorf("sqlCreateMessagesFTS failed: %v", err)
	}
}
