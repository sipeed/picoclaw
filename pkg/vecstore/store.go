package vecstore

import (
	"encoding/gob"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Chunk represents a text chunk with its embedding vector.
type Chunk struct {
	ID        string
	Text      string
	Source    string // file path the chunk came from
	Embedding []float32
	UpdatedAt time.Time
}

// Result is a search result with similarity score.
type Result struct {
	Chunk
	Score float32
}

// VectorStore is an in-memory vector store with gob persistence.
type VectorStore struct {
	path   string
	chunks []Chunk
	mu     sync.RWMutex
}

// NewVectorStore creates a store that persists to the given path.
func NewVectorStore(path string) *VectorStore {
	return &VectorStore{path: path}
}

// Load reads the store from disk. Returns nil if file doesn't exist.
func (vs *VectorStore) Load() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	f, err := os.Open(vs.path)
	if err != nil {
		if os.IsNotExist(err) {
			vs.chunks = nil
			return nil
		}
		return err
	}
	defer f.Close()

	var chunks []Chunk
	if err := gob.NewDecoder(f).Decode(&chunks); err != nil {
		// Corrupt file — start fresh
		vs.chunks = nil
		return nil
	}
	vs.chunks = chunks
	return nil
}

// Save writes the store to disk.
func (vs *VectorStore) Save() error {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(vs.path), 0755); err != nil {
		return err
	}

	f, err := os.Create(vs.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(vs.chunks)
}

// Search returns the top-K chunks most similar to the query embedding.
func (vs *VectorStore) Search(query []float32, topK int) []Result {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	if len(vs.chunks) == 0 {
		return nil
	}

	results := make([]Result, 0, len(vs.chunks))
	for _, c := range vs.chunks {
		if len(c.Embedding) == 0 {
			continue
		}
		score := cosine(query, c.Embedding)
		results = append(results, Result{Chunk: c, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

// Upsert adds or replaces chunks by ID.
func (vs *VectorStore) Upsert(chunks []Chunk) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	idx := make(map[string]int, len(vs.chunks))
	for i, c := range vs.chunks {
		idx[c.ID] = i
	}

	for _, c := range chunks {
		if i, ok := idx[c.ID]; ok {
			vs.chunks[i] = c
		} else {
			vs.chunks = append(vs.chunks, c)
		}
	}
}

// DeleteBySource removes all chunks from a given source.
func (vs *VectorStore) DeleteBySource(source string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	filtered := vs.chunks[:0]
	for _, c := range vs.chunks {
		if c.Source != source {
			filtered = append(filtered, c)
		}
	}
	vs.chunks = filtered
}

// Len returns the number of chunks in the store.
func (vs *VectorStore) Len() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.chunks)
}

// cosine computes cosine similarity between two vectors.
func cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}
