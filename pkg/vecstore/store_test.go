package vecstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCosine(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float32
		tol  float32
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0, 0.001},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0, 0.001},
		{"opposite", []float32{1, 0}, []float32{-1, 0}, -1.0, 0.001},
		{"similar", []float32{1, 1}, []float32{1, 0.9}, 0.998, 0.01},
		{"empty", []float32{}, []float32{}, 0.0, 0.001},
		{"mismatched", []float32{1, 2}, []float32{1, 2, 3}, 0.0, 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosine(tt.a, tt.b)
			if diff := got - tt.want; diff > tt.tol || diff < -tt.tol {
				t.Errorf("cosine(%v, %v) = %f, want %f (tol %f)", tt.a, tt.b, got, tt.want, tt.tol)
			}
		})
	}
}

func TestSearchReturnsTopK(t *testing.T) {
	store := NewVectorStore("")
	now := time.Now()

	store.Upsert([]Chunk{
		{ID: "a", Text: "alpha", Embedding: []float32{1, 0, 0}, UpdatedAt: now},
		{ID: "b", Text: "beta", Embedding: []float32{0, 1, 0}, UpdatedAt: now},
		{ID: "c", Text: "gamma", Embedding: []float32{0.9, 0.1, 0}, UpdatedAt: now},
	})

	results := store.Search([]float32{1, 0, 0}, 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "a" {
		t.Errorf("expected first result 'a', got %q", results[0].ID)
	}
	if results[1].ID != "c" {
		t.Errorf("expected second result 'c', got %q", results[1].ID)
	}
}

func TestUpsertReplacesExisting(t *testing.T) {
	store := NewVectorStore("")
	now := time.Now()

	store.Upsert([]Chunk{
		{ID: "a", Text: "original", Embedding: []float32{1, 0}, UpdatedAt: now},
	})
	store.Upsert([]Chunk{
		{ID: "a", Text: "replaced", Embedding: []float32{0, 1}, UpdatedAt: now},
	})

	if store.Len() != 1 {
		t.Fatalf("expected 1 chunk, got %d", store.Len())
	}

	results := store.Search([]float32{0, 1}, 1)
	if results[0].Text != "replaced" {
		t.Errorf("expected replaced text, got %q", results[0].Text)
	}
}

func TestDeleteBySource(t *testing.T) {
	store := NewVectorStore("")
	now := time.Now()

	store.Upsert([]Chunk{
		{ID: "a", Text: "a", Source: "file1.md", Embedding: []float32{1, 0}, UpdatedAt: now},
		{ID: "b", Text: "b", Source: "file2.md", Embedding: []float32{0, 1}, UpdatedAt: now},
		{ID: "c", Text: "c", Source: "file1.md", Embedding: []float32{1, 1}, UpdatedAt: now},
	})

	store.DeleteBySource("file1.md")
	if store.Len() != 1 {
		t.Fatalf("expected 1 chunk after delete, got %d", store.Len())
	}

	results := store.Search([]float32{0, 1}, 10)
	if results[0].Source != "file2.md" {
		t.Errorf("expected remaining chunk from file2.md, got %q", results[0].Source)
	}
}

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.gob")
	now := time.Now()

	// Save
	store1 := NewVectorStore(path)
	store1.Upsert([]Chunk{
		{ID: "x", Text: "hello", Source: "src", Embedding: []float32{0.5, 0.5}, UpdatedAt: now},
	})
	if err := store1.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load into new store
	store2 := NewVectorStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if store2.Len() != 1 {
		t.Fatalf("expected 1 chunk after load, got %d", store2.Len())
	}

	results := store2.Search([]float32{0.5, 0.5}, 1)
	if results[0].Text != "hello" {
		t.Errorf("expected 'hello', got %q", results[0].Text)
	}
}

func TestLoadMissingFile(t *testing.T) {
	store := NewVectorStore(filepath.Join(t.TempDir(), "nonexistent.gob"))
	if err := store.Load(); err != nil {
		t.Fatalf("load missing file should not error: %v", err)
	}
	if store.Len() != 0 {
		t.Fatalf("expected 0 chunks, got %d", store.Len())
	}
}

func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.gob")
	os.WriteFile(path, []byte("not valid gob"), 0644)

	store := NewVectorStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("load corrupt file should not error: %v", err)
	}
	if store.Len() != 0 {
		t.Fatalf("expected 0 chunks after corrupt load, got %d", store.Len())
	}
}
