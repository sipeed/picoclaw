package memory

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/philippgille/chromem-go"
	"github.com/sipeed/picoclaw/pkg/swarm/core"
)

type ChromemStore struct {
	db         *chromem.DB
	llm        core.LLMClient
	collection *chromem.Collection
}

func NewChromemStore(ctx context.Context, llm core.LLMClient) (*ChromemStore, error) {
	db, err := chromem.NewPersistentDB("./picoclaw_memory", false)
	if err != nil {
		return nil, err
	}

	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		return llm.Embed(ctx, text)
	}

	c, err := db.GetOrCreateCollection("swarm_facts", nil, embeddingFunc)
	if err != nil {
		return nil, err
	}

	return &ChromemStore{db: db, llm: llm, collection: c}, nil
}

func (s *ChromemStore) SaveFact(ctx context.Context, fact core.Fact) error {
	meta := make(map[string]string)
	for k, v := range fact.Metadata {
		meta[k] = fmt.Sprintf("%v", v)
	}
	meta["swarm_id"] = string(fact.SwarmID)
	meta["source"] = fact.Source

	doc := chromem.Document{
		ID:       fmt.Sprintf("%s_%f_%d", fact.SwarmID, fact.Confidence, time.Now().UnixNano()),
		Content:  fact.Content,
		Metadata: meta,
	}

	return s.collection.AddDocuments(ctx, []chromem.Document{doc}, runtime.NumCPU())
}

func (s *ChromemStore) SearchFacts(ctx context.Context, swarmID core.SwarmID, query string, limit int, global bool) ([]core.FactResult, error) {
	var where map[string]string
	if !global {
		where = map[string]string{"swarm_id": string(swarmID)}
	}
	
	docs, err := s.collection.Query(ctx, query, limit, where, nil)
	if err != nil {
		return nil, err
	}

	var results []core.FactResult
	for _, doc := range docs {
		results = append(results, core.FactResult{
			Content: doc.Content,
			Score:   doc.Similarity,
		})
	}

	return results, nil
}

func (s *ChromemStore) ClearMemory(ctx context.Context, swarmID core.SwarmID) error {
	return nil
}
