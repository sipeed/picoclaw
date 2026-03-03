// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// VectorMemoryStore provides semantic memory search backed by SQLite.
// Embeddings are stored as raw float32 blobs; cosine similarity is computed in Go.
//
// The DB schema is intentionally minimal:
//
//	CREATE TABLE memories (
//	    id      TEXT PRIMARY KEY,  -- content hash or sequential key
//	    content TEXT NOT NULL,     -- raw text of the memory entry
//	    vector  BLOB NOT NULL      -- float32 little-endian array
//	)
type VectorMemoryStore struct {
	db             *sql.DB
	embeddingModel string
	topK           int
}

// NewVectorMemoryStore opens (or creates) the SQLite DB at dbPath.
func NewVectorMemoryStore(dbPath string, embeddingModel string, topK int) (*VectorMemoryStore, error) {
	if topK <= 0 {
		topK = 5
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("vector memory: open db: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS memories (
		id      TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		vector  BLOB NOT NULL
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("vector memory: create table: %w", err)
	}
	return &VectorMemoryStore{
		db:             db,
		embeddingModel: embeddingModel,
		topK:           topK,
	}, nil
}

// Close closes the underlying database.
func (vs *VectorMemoryStore) Close() error {
	return vs.db.Close()
}

// Upsert stores a memory entry along with its embedding vector.
func (vs *VectorMemoryStore) Upsert(id, content string, vector []float32) error {
	blob := float32SliceToBytes(vector)
	_, err := vs.db.Exec(
		`INSERT INTO memories (id, content, vector) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET content=excluded.content, vector=excluded.vector`,
		id, content, blob,
	)
	return err
}

// Search returns the top-K memory entries most semantically similar to queryVec.
func (vs *VectorMemoryStore) Search(queryVec []float32) ([]string, error) {
	rows, err := vs.db.Query(`SELECT content, vector FROM memories`)
	if err != nil {
		return nil, fmt.Errorf("vector memory: query: %w", err)
	}
	defer rows.Close()

	type scored struct {
		content string
		score   float64
	}
	var results []scored

	for rows.Next() {
		var content string
		var blob []byte
		if err := rows.Scan(&content, &blob); err != nil {
			continue
		}
		vec := bytesToFloat32Slice(blob)
		if len(vec) == 0 {
			continue
		}
		sim := cosineSimilarity(queryVec, vec)
		results = append(results, scored{content: content, score: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	topK := vs.topK
	if topK > len(results) {
		topK = len(results)
	}
	out := make([]string, topK)
	for i := range topK {
		out[i] = results[i].content
	}
	return out, nil
}

// Count returns the number of stored memory entries.
func (vs *VectorMemoryStore) Count() int {
	var n int
	vs.db.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&n) //nolint:errcheck
	return n
}

// SyncFromDirectory parses all .md files in the given directory into individual entries
// and upserts any that are not already indexed, using the provided embedder to vectorize them.
// It also deletes entries from the database that are no longer present in any of the files.
func (vs *VectorMemoryStore) SyncFromDirectory(ctx context.Context, dirPath string, embedder func(string) ([]float32, error)) {
	// Gather complete content from all .md files in the directory
	var allEntries []string
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			if data, readErr := os.ReadFile(path); readErr == nil {
				allEntries = append(allEntries, splitMemoryEntries(string(data))...)
			}
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		log.Printf("vector memory: sync directory walk error: %v", err)
	}

	// Map to track current chunks by hash
	currentHashes := make(map[string]string)

	for _, entry := range allEntries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		hash := md5.Sum([]byte(entry))
		id := hex.EncodeToString(hash[:])
		currentHashes[id] = entry
	}

	// Fetch existing IDs to find what to add/delete
	existingIDs := make(map[string]bool)
	rows, err := vs.db.Query(`SELECT id FROM memories`)
	if err == nil {
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil {
				existingIDs[id] = true
			}
		}
		rows.Close()
	}

	// Insert new entries
	for id, entry := range currentHashes {
		if existingIDs[id] {
			continue // Already indexed
		}
		vec, err := embedder(entry)
		if err != nil {
			log.Printf("vector memory: sync embed error for %s: %v", id[:8], err)
			continue
		}
		if err := vs.Upsert(id, entry, vec); err != nil {
			log.Printf("vector memory: sync upsert error for %s: %v", id[:8], err)
		}
	}

	// Delete stale entries
	for id := range existingIDs {
		if _, ok := currentHashes[id]; !ok {
			vs.db.Exec(`DELETE FROM memories WHERE id=?`, id) //nolint:errcheck
		}
	}
}

// splitMemoryEntries splits a MEMORY.md file into individual memory chunks.
// Splits on markdown headings (##) or blank-line separated paragraphs.
func splitMemoryEntries(content string) []string {
	var entries []string
	// Split by "##" headings first
	if strings.Contains(content, "\n## ") || strings.HasPrefix(content, "## ") {
		parts := strings.Split(content, "\n## ")
		for i, p := range parts {
			if i > 0 {
				p = "## " + p
			}
			if strings.TrimSpace(p) != "" {
				entries = append(entries, p)
			}
		}
		return entries
	}
	// Fallback: split on blank lines
	for _, block := range strings.Split(content, "\n\n") {
		if strings.TrimSpace(block) != "" {
			entries = append(entries, block)
		}
	}
	return entries
}

// --- Vector math helpers ---

func cosineSimilarity(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, normA, normB float64
	for i := range n {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func float32SliceToBytes(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func bytesToFloat32Slice(b []byte) []float32 {
	n := len(b) / 4
	out := make([]float32, n)
	for i := range n {
		bits := binary.LittleEndian.Uint32(b[i*4:])
		out[i] = math.Float32frombits(bits)
	}
	return out
}
