// ABOUTME: Semantic memory store backed by chromem-go vector database.
// ABOUTME: Provides Remember/Recall operations with Ollama embeddings for persistent memory.
package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	chromem "github.com/philippgille/chromem-go"
)

// MemoryEntry represents a single memory to be stored.
type MemoryEntry struct {
	ID        string
	Content   string
	Category  string   // "preference", "fact", "decision", etc.
	Tags      []string
	Source    string   // "agent", "auto-extract"
	Timestamp time.Time
}

// RecallResult is a memory entry returned from a search, with similarity score.
type RecallResult struct {
	MemoryEntry
	Similarity float32
}

// Store is the interface for semantic memory operations.
// Using an interface allows easy testing with mock implementations.
type Store interface {
	IsAvailable() bool
	Count() int
	Remember(ctx context.Context, entry MemoryEntry) error
	Recall(ctx context.Context, query string, topK int) ([]RecallResult, error)
}

// SemanticStore implements Store using chromem-go for vector storage
// and Ollama for embedding generation.
type SemanticStore struct {
	db         *chromem.DB
	collection *chromem.Collection
	available  bool
}

// NewSemanticStore creates a persistent vector store at persistDir.
// It connects to Ollama at ollamaURL for embeddings using the given model.
// If Ollama is unreachable, the store is created in a degraded state
// where IsAvailable() returns false and all operations return clean errors.
func NewSemanticStore(persistDir, ollamaURL, embeddingModel string) (*SemanticStore, error) {
	if ollamaURL == "" {
		ollamaURL = DefaultOllamaURL
	}
	if embeddingModel == "" {
		embeddingModel = DefaultEmbeddingModel
	}

	// chromem-go expects the Ollama API base URL (ending in /api), not
	// the server root. Normalize so users can provide either form.
	ollamaAPIBase := normalizeOllamaURL(ollamaURL)
	embedFn := chromem.NewEmbeddingFuncOllama(embeddingModel, ollamaAPIBase)

	db, err := chromem.NewPersistentDB(persistDir, false)
	if err != nil {
		return &SemanticStore{available: false}, fmt.Errorf("creating vector db: %w", err)
	}

	collection, err := db.GetOrCreateCollection("memories", nil, embedFn)
	if err != nil {
		return &SemanticStore{available: false}, fmt.Errorf("creating collection: %w", err)
	}

	return &SemanticStore{
		db:         db,
		collection: collection,
		available:  true,
	}, nil
}

// NewSemanticStoreWithEmbedding creates a store with a custom embedding function.
// This is primarily for testing without requiring Ollama.
func NewSemanticStoreWithEmbedding(persistDir string, embedFn chromem.EmbeddingFunc) (*SemanticStore, error) {
	db, err := chromem.NewPersistentDB(persistDir, false)
	if err != nil {
		return nil, fmt.Errorf("creating vector db: %w", err)
	}

	collection, err := db.GetOrCreateCollection("memories", nil, embedFn)
	if err != nil {
		return nil, fmt.Errorf("creating collection: %w", err)
	}

	return &SemanticStore{
		db:         db,
		collection: collection,
		available:  true,
	}, nil
}

func (s *SemanticStore) IsAvailable() bool {
	return s.available
}

func (s *SemanticStore) Count() int {
	if !s.available {
		return 0
	}
	return s.collection.Count()
}

func (s *SemanticStore) Remember(ctx context.Context, entry MemoryEntry) error {
	if !s.available {
		return fmt.Errorf("semantic memory is not available")
	}

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("mem_%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	metadata := map[string]string{
		"category":  entry.Category,
		"source":    entry.Source,
		"timestamp": entry.Timestamp.Format(time.RFC3339),
	}
	if len(entry.Tags) > 0 {
		metadata["tags"] = strings.Join(entry.Tags, ",")
	}

	doc := chromem.Document{
		ID:       entry.ID,
		Content:  entry.Content,
		Metadata: metadata,
	}

	return s.collection.AddDocument(ctx, doc)
}

func (s *SemanticStore) Recall(ctx context.Context, query string, topK int) ([]RecallResult, error) {
	if !s.available {
		return nil, fmt.Errorf("semantic memory is not available")
	}
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	// chromem-go returns an error if topK > collection count, so we
	// cap it to the collection size.
	count := s.collection.Count()
	if count == 0 {
		return nil, nil
	}
	if topK > count {
		topK = count
	}

	results, err := s.collection.Query(ctx, query, topK, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("querying memories: %w", err)
	}

	var recalls []RecallResult
	for _, r := range results {
		entry := MemoryEntry{
			ID:       r.ID,
			Content:  r.Content,
			Category: r.Metadata["category"],
			Source:   r.Metadata["source"],
		}
		if ts, ok := r.Metadata["timestamp"]; ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				entry.Timestamp = t
			}
		}
		if tagStr, ok := r.Metadata["tags"]; ok && tagStr != "" {
			entry.Tags = strings.Split(tagStr, ",")
		}

		recalls = append(recalls, RecallResult{
			MemoryEntry: entry,
			Similarity:  r.Similarity,
		})
	}

	return recalls, nil
}

// normalizeOllamaURL ensures the URL ends with /api as required by
// chromem-go's NewEmbeddingFuncOllama. Users typically configure the
// Ollama server root (http://localhost:11434) without the /api suffix.
func normalizeOllamaURL(url string) string {
	url = strings.TrimRight(url, "/")
	if url == "" {
		return "" // let chromem-go use its default
	}
	if !strings.HasSuffix(url, "/api") {
		url += "/api"
	}
	return url
}
