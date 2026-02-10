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
	s.init()
	return s, nil
}

func (s *SQLiteStore) init() {
	s.db.Exec(`CREATE TABLE IF NOT EXISTS swarms (id TEXT PRIMARY KEY, goal TEXT, status TEXT, created_at DATETIME);`)
	s.db.Exec(`CREATE TABLE IF NOT EXISTS nodes (id TEXT PRIMARY KEY, swarm_id TEXT, parent_id TEXT, role JSON, task TEXT, status TEXT, output TEXT, stats JSON);`)
	s.db.Exec(`CREATE TABLE IF NOT EXISTS facts (swarm_id TEXT, content TEXT, confidence REAL, source TEXT, metadata JSON);`)
}

func (s *SQLiteStore) CreateSwarm(ctx context.Context, sw *core.Swarm) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO swarms VALUES (?, ?, ?, ?)", sw.ID, sw.Goal, sw.Status, sw.CreatedAt)
	return err
}

func (s *SQLiteStore) GetSwarm(ctx context.Context, id core.SwarmID) (*core.Swarm, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, goal, status, created_at FROM swarms WHERE id=?", id)
	var sw core.Swarm
	err := row.Scan(&sw.ID, &sw.Goal, &sw.Status, &sw.CreatedAt)
	return &sw, err
}

func (s *SQLiteStore) UpdateSwarm(ctx context.Context, sw *core.Swarm) error {
	_, err := s.db.ExecContext(ctx, "UPDATE swarms SET status=? WHERE id=?", sw.Status, sw.ID)
	return err
}

func (s *SQLiteStore) ListSwarms(ctx context.Context, status core.SwarmStatus) ([]*core.Swarm, error) {
	rows, _ := s.db.QueryContext(ctx, "SELECT id, goal, status, created_at FROM swarms WHERE status=?", status)
	defer rows.Close()
	var list []*core.Swarm
	for rows.Next() {
		var sw core.Swarm
		rows.Scan(&sw.ID, &sw.Goal, &sw.Status, &sw.CreatedAt)
		list = append(list, &sw)
	}
	return list, nil
}

func (s *SQLiteStore) CreateNode(ctx context.Context, n *core.Node) error {
	role, _ := json.Marshal(n.Role)
	stats, _ := json.Marshal(n.Stats)
	_, err := s.db.ExecContext(ctx, "INSERT INTO nodes VALUES (?, ?, ?, ?, ?, ?, ?, ?)", n.ID, n.SwarmID, n.ParentID, role, n.Task, n.Status, n.Output, stats)
	return err
}

func (s *SQLiteStore) GetNode(ctx context.Context, id core.NodeID) (*core.Node, error) {
	row := s.db.QueryRowContext(ctx, "SELECT * FROM nodes WHERE id=?", id)
	var n core.Node
	var role, stats []byte
	row.Scan(&n.ID, &n.SwarmID, &n.ParentID, &role, &n.Task, &n.Status, &n.Output, &stats)
	json.Unmarshal(role, &n.Role)
	json.Unmarshal(stats, &n.Stats)
	return &n, nil
}

func (s *SQLiteStore) UpdateNode(ctx context.Context, n *core.Node) error {
	stats, _ := json.Marshal(n.Stats)
	_, err := s.db.ExecContext(ctx, "UPDATE nodes SET status=?, output=?, stats=? WHERE id=?", n.Status, n.Output, stats, n.ID)
	return err
}

func (s *SQLiteStore) GetSwarmNodes(ctx context.Context, id core.SwarmID) ([]*core.Node, error) {
	rows, _ := s.db.QueryContext(ctx, "SELECT * FROM nodes WHERE swarm_id=?", id)
	defer rows.Close()
	var list []*core.Node
	for rows.Next() {
		var n core.Node
		var role, stats []byte
		rows.Scan(&n.ID, &n.SwarmID, &n.ParentID, &role, &n.Task, &n.Status, &n.Output, &stats)
		json.Unmarshal(role, &n.Role)
		json.Unmarshal(stats, &n.Stats)
		list = append(list, &n)
	}
	return list, nil
}

func (s *SQLiteStore) SaveFact(ctx context.Context, f core.Fact) error {
	meta, _ := json.Marshal(f.Metadata)
	_, err := s.db.ExecContext(ctx, "INSERT INTO facts VALUES (?, ?, ?, ?, ?)", f.SwarmID, f.Content, f.Confidence, f.Source, meta)
	return err
}

func (s *SQLiteStore) SearchFacts(ctx context.Context, id core.SwarmID, q string, limit int, global bool) ([]core.FactResult, error) {
	query := "SELECT content FROM facts WHERE swarm_id=? AND content LIKE ? LIMIT ?"
	args := []any{id, "%" + q + "%", limit}
	
	if global {
		query = "SELECT content FROM facts WHERE content LIKE ? LIMIT ?"
		args = []any{"%" + q + "%", limit}
	}

	rows, _ := s.db.QueryContext(ctx, query, args...)
	defer rows.Close()
	var res []core.FactResult
	for rows.Next() {
		var content string
		rows.Scan(&content)
		res = append(res, core.FactResult{Content: content, Score: 1.0})
	}
	return res, nil
}

func (s *SQLiteStore) ClearMemory(ctx context.Context, id core.SwarmID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM facts WHERE swarm_id=?", id)
	return err
}