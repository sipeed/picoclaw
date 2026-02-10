package memory

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/sipeed/picoclaw/pkg/swarm/core"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil { return nil, err }
	s := &SQLiteStore{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) init() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS swarms (id TEXT PRIMARY KEY, goal TEXT, status TEXT, created_at DATETIME, origin_channel TEXT, origin_chat_id TEXT);`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS nodes (id TEXT PRIMARY KEY, swarm_id TEXT, parent_id TEXT, role JSON, task TEXT, status TEXT, output TEXT, stats JSON);`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS facts (swarm_id TEXT, content TEXT, confidence REAL, source TEXT, metadata JSON);`); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) CreateSwarm(ctx context.Context, sw *core.Swarm) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO swarms (id, goal, status, created_at, origin_channel, origin_chat_id) VALUES (?, ?, ?, ?, ?, ?)", sw.ID, sw.Goal, sw.Status, sw.CreatedAt, sw.OriginChannel, sw.OriginChatID)
	return err
}

func (s *SQLiteStore) GetSwarm(ctx context.Context, id core.SwarmID) (*core.Swarm, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, goal, status, created_at, origin_channel, origin_chat_id FROM swarms WHERE id=?", id)
	var sw core.Swarm
	if err := row.Scan(&sw.ID, &sw.Goal, &sw.Status, &sw.CreatedAt, &sw.OriginChannel, &sw.OriginChatID); err != nil {
		return nil, err
	}
	return &sw, nil
}

func (s *SQLiteStore) UpdateSwarm(ctx context.Context, sw *core.Swarm) error {
	_, err := s.db.ExecContext(ctx, "UPDATE swarms SET status=? WHERE id=?", sw.Status, sw.ID)
	return err
}

func (s *SQLiteStore) ListSwarms(ctx context.Context, status core.SwarmStatus) ([]*core.Swarm, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, goal, status, created_at, origin_channel, origin_chat_id FROM swarms WHERE status=?", status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*core.Swarm
	for rows.Next() {
		var sw core.Swarm
		if err := rows.Scan(&sw.ID, &sw.Goal, &sw.Status, &sw.CreatedAt, &sw.OriginChannel, &sw.OriginChatID); err != nil {
			return nil, err
		}
		list = append(list, &sw)
	}
	return list, nil
}

func (s *SQLiteStore) CreateNode(ctx context.Context, n *core.Node) error {
	role, err := json.Marshal(n.Role)
	if err != nil {
		return err
	}
	stats, err := json.Marshal(n.Stats)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO nodes (id, swarm_id, parent_id, role, task, status, output, stats) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", n.ID, n.SwarmID, n.ParentID, role, n.Task, n.Status, n.Output, stats)
	return err
}

func (s *SQLiteStore) GetNode(ctx context.Context, id core.NodeID) (*core.Node, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, swarm_id, parent_id, role, task, status, output, stats FROM nodes WHERE id=?", id)
	var n core.Node
	var role, stats []byte
	if err := row.Scan(&n.ID, &n.SwarmID, &n.ParentID, &role, &n.Task, &n.Status, &n.Output, &stats); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(role, &n.Role); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(stats, &n.Stats); err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *SQLiteStore) UpdateNode(ctx context.Context, n *core.Node) error {
	stats, err := json.Marshal(n.Stats)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE nodes SET status=?, output=?, stats=? WHERE id=?", n.Status, n.Output, stats, n.ID)
	return err
}

func (s *SQLiteStore) GetSwarmNodes(ctx context.Context, id core.SwarmID) ([]*core.Node, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, swarm_id, parent_id, role, task, status, output, stats FROM nodes WHERE swarm_id=?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*core.Node
	for rows.Next() {
		var n core.Node
		var role, stats []byte
		if err := rows.Scan(&n.ID, &n.SwarmID, &n.ParentID, &role, &n.Task, &n.Status, &n.Output, &stats); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(role, &n.Role); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(stats, &n.Stats); err != nil {
			return nil, err
		}
		list = append(list, &n)
	}
	return list, nil
}

func (s *SQLiteStore) SaveFact(ctx context.Context, f core.Fact) error {
	meta, err := json.Marshal(f.Metadata)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO facts (swarm_id, content, confidence, source, metadata) VALUES (?, ?, ?, ?, ?)", f.SwarmID, f.Content, f.Confidence, f.Source, meta)
	return err
}

func (s *SQLiteStore) SearchFacts(ctx context.Context, id core.SwarmID, q string, limit int, global bool) ([]core.FactResult, error) {
	query := "SELECT content FROM facts WHERE swarm_id=? AND content LIKE ? LIMIT ?"
	args := []any{id, "%" + q + "%", limit}
	
	if global {
		query = "SELECT content FROM facts WHERE content LIKE ? LIMIT ?"
		args = []any{"%" + q + "%", limit}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []core.FactResult
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		res = append(res, core.FactResult{Content: content, Score: 1.0})
	}
	return res, nil
}

func (s *SQLiteStore) ClearMemory(ctx context.Context, id core.SwarmID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM facts WHERE swarm_id=?", id)
	return err
}