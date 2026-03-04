package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const sqliteDriver = "sqlite"

const schema = `



CREATE TABLE IF NOT EXISTS sessions (



	key         TEXT PRIMARY KEY,



	parent_key  TEXT NOT NULL DEFAULT '',



	fork_turn_id TEXT NOT NULL DEFAULT '',



	status      TEXT NOT NULL DEFAULT 'active',



	label       TEXT NOT NULL DEFAULT '',



	summary     TEXT NOT NULL DEFAULT '',



	created_at  TEXT NOT NULL,



	updated_at  TEXT NOT NULL



);



CREATE TABLE IF NOT EXISTS turns (



	id          TEXT PRIMARY KEY,



	session_key TEXT NOT NULL REFERENCES sessions(key) ON DELETE CASCADE,



	seq         INTEGER NOT NULL,



	kind        INTEGER NOT NULL DEFAULT 0,



	messages    TEXT NOT NULL DEFAULT '[]',



	origin_key  TEXT NOT NULL DEFAULT '',



	summary     TEXT NOT NULL DEFAULT '',



	author      TEXT NOT NULL DEFAULT '',



	created_at  TEXT NOT NULL,



	meta        TEXT NOT NULL DEFAULT '{}'



);



CREATE INDEX IF NOT EXISTS idx_turns_session_seq ON turns(session_key, seq);



CREATE INDEX IF NOT EXISTS idx_sessions_parent ON sessions(parent_key);



`

// SQLiteStore implements SessionStore backed by a single SQLite file.

type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLiteStore opens (or creates) a SQLite session database at dbPath.

func OpenSQLiteStore(dbPath string) (*SQLiteStore, error) {
	connStr := "file:" + dbPath + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"

	db, err := sql.Open(sqliteDriver, connStr)
	if err != nil {
		return nil, fmt.Errorf("open session store: %w", err)
	}

	db.SetMaxOpenConns(1)

	db.SetMaxIdleConns(1)

	// Ensure foreign keys are enabled (connection string param may not suffice).

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)

	return t
}

// --- Session CRUD ---

func (s *SQLiteStore) Create(key string, opts *CreateOpts) error {
	now := nowUTC()

	parentKey, forkTurnID, label := "", "", ""

	if opts != nil {
		parentKey = opts.ParentKey

		forkTurnID = opts.ForkTurnID

		label = opts.Label
	}

	_, err := s.db.Exec(

		`INSERT INTO sessions (key, parent_key, fork_turn_id, label, created_at, updated_at)



		 VALUES (?, ?, ?, ?, ?, ?)`,

		key, parentKey, forkTurnID, label, now, now,
	)

	return err
}

func (s *SQLiteStore) Get(key string) (*SessionInfo, error) {
	row := s.db.QueryRow(

		`SELECT key, parent_key, fork_turn_id, status, label, summary, created_at, updated_at



		 FROM sessions WHERE key = ?`, key,
	)

	return scanSessionInfo(row)
}

func scanSessionInfo(row *sql.Row) (*SessionInfo, error) {
	var info SessionInfo

	var createdAt, updatedAt string

	err := row.Scan(&info.Key, &info.ParentKey, &info.ForkTurnID, &info.Status,

		&info.Label, &info.Summary, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	info.CreatedAt = parseTime(createdAt)

	info.UpdatedAt = parseTime(updatedAt)

	return &info, nil
}

func (s *SQLiteStore) List(filter *ListFilter) ([]*SessionInfo, error) {
	query := `SELECT key, parent_key, fork_turn_id, status, label, summary, created_at, updated_at FROM sessions WHERE 1=1`

	var args []any

	if filter != nil {
		if filter.ParentKey != "" {
			query += ` AND parent_key = ?`

			args = append(args, filter.ParentKey)
		}

		if filter.Status != "" {
			query += ` AND status = ?`

			args = append(args, filter.Status)
		}
	}

	query += ` ORDER BY created_at`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var result []*SessionInfo

	for rows.Next() {
		var info SessionInfo

		var createdAt, updatedAt string

		if err := rows.Scan(&info.Key, &info.ParentKey, &info.ForkTurnID, &info.Status,

			&info.Label, &info.Summary, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		info.CreatedAt = parseTime(createdAt)

		info.UpdatedAt = parseTime(updatedAt)

		result = append(result, &info)
	}

	return result, rows.Err()
}

func (s *SQLiteStore) SetStatus(key, status string) error {
	res, err := s.db.Exec(`UPDATE sessions SET status = ?, updated_at = ? WHERE key = ?`, status, nowUTC(), key)
	if err != nil {
		return err
	}

	return checkRowAffected(res, key)
}

func (s *SQLiteStore) SetSummary(key, summary string) error {
	res, err := s.db.Exec(`UPDATE sessions SET summary = ?, updated_at = ? WHERE key = ?`, summary, nowUTC(), key)
	if err != nil {
		return err
	}

	return checkRowAffected(res, key)
}

func (s *SQLiteStore) Delete(key string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE key = ?`, key)

	return err
}

func (s *SQLiteStore) Children(key string) ([]*SessionInfo, error) {
	return s.List(&ListFilter{ParentKey: key})
}

// --- Turn operations ---

func (s *SQLiteStore) Append(sessionKey string, turn *Turn) error {
	if turn.ID == "" {
		turn.ID = uuid.New().String()
	}

	if turn.CreatedAt.IsZero() {
		turn.CreatedAt = time.Now().UTC()
	}

	messagesJSON, err := json.Marshal(turn.Messages)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	metaJSON, err := json.Marshal(turn.Meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	// Auto-assign seq if not set

	if turn.Seq == 0 {
		var maxSeq sql.NullInt64

		_ = s.db.QueryRow(`SELECT MAX(seq) FROM turns WHERE session_key = ?`, sessionKey).Scan(&maxSeq)

		if maxSeq.Valid {
			turn.Seq = int(maxSeq.Int64) + 1
		} else {
			turn.Seq = 1
		}
	}

	_, err = s.db.Exec(

		`INSERT INTO turns (id, session_key, seq, kind, messages, origin_key, summary, author, created_at, meta)



		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,

		turn.ID, sessionKey, turn.Seq, int(turn.Kind),

		string(messagesJSON), turn.OriginKey, turn.Summary, turn.Author,

		turn.CreatedAt.UTC().Format(time.RFC3339Nano), string(metaJSON),
	)
	if err != nil {
		return err
	}

	// Update session's updated_at

	_, _ = s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE key = ?`, nowUTC(), sessionKey)

	return nil
}

func (s *SQLiteStore) Turns(sessionKey string, sinceSeq int) ([]*Turn, error) {
	rows, err := s.db.Query(

		`SELECT id, session_key, seq, kind, messages, origin_key, summary, author, created_at, meta



		 FROM turns WHERE session_key = ? AND seq > ? ORDER BY seq`,

		sessionKey, sinceSeq,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var result []*Turn

	for rows.Next() {
		t, err := scanTurn(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, t)
	}

	return result, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTurn(row scanner) (*Turn, error) {
	var t Turn

	var kind int

	var messagesJSON, metaJSON, createdAt string

	err := row.Scan(&t.ID, &t.SessionKey, &t.Seq, &kind, &messagesJSON,

		&t.OriginKey, &t.Summary, &t.Author, &createdAt, &metaJSON)
	if err != nil {
		return nil, err
	}

	t.Kind = TurnKind(kind)

	t.CreatedAt = parseTime(createdAt)

	if err := json.Unmarshal([]byte(messagesJSON), &t.Messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}

	if metaJSON != "" && metaJSON != "{}" {
		if err := json.Unmarshal([]byte(metaJSON), &t.Meta); err != nil {
			return nil, fmt.Errorf("unmarshal meta: %w", err)
		}
	}

	return &t, nil
}

func (s *SQLiteStore) LastTurn(sessionKey string) (*Turn, error) {
	row := s.db.QueryRow(

		`SELECT id, session_key, seq, kind, messages, origin_key, summary, author, created_at, meta



		 FROM turns WHERE session_key = ? ORDER BY seq DESC LIMIT 1`,

		sessionKey,
	)

	var t Turn

	var kind int

	var messagesJSON, metaJSON, createdAt string

	err := row.Scan(&t.ID, &t.SessionKey, &t.Seq, &kind, &messagesJSON,

		&t.OriginKey, &t.Summary, &t.Author, &createdAt, &metaJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	t.Kind = TurnKind(kind)

	t.CreatedAt = parseTime(createdAt)

	if err := json.Unmarshal([]byte(messagesJSON), &t.Messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}

	if metaJSON != "" && metaJSON != "{}" {
		if err := json.Unmarshal([]byte(metaJSON), &t.Meta); err != nil {
			return nil, fmt.Errorf("unmarshal meta: %w", err)
		}
	}

	return &t, nil
}

func (s *SQLiteStore) TurnCount(sessionKey string) (int, error) {
	var count int

	err := s.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE session_key = ?`, sessionKey).Scan(&count)

	return count, err
}

func (s *SQLiteStore) Compact(sessionKey string, upToSeq int, summary string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM turns WHERE session_key = ? AND seq <= ?`, sessionKey, upToSeq)
	if err != nil {
		return err
	}

	if summary != "" {
		_, err = tx.Exec(`UPDATE sessions SET summary = ?, updated_at = ? WHERE key = ?`,

			summary, nowUTC(), sessionKey)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// --- DAG operations ---

func (s *SQLiteStore) Fork(parentKey, childKey string, opts *CreateOpts) error {
	if opts == nil {
		opts = &CreateOpts{}
	}

	opts.ParentKey = parentKey

	return s.Create(childKey, opts)
}

// --- Maintenance ---

func (s *SQLiteStore) Prune(olderThan time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339Nano)

	res, err := s.db.Exec(`DELETE FROM sessions WHERE updated_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}

	n, _ := res.RowsAffected()

	return int(n), nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func checkRowAffected(res sql.Result, key string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return fmt.Errorf("session not found: %s", key)
	}

	return nil
}
