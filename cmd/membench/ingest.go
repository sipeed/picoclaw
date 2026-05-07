package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sipeed/picoclaw/pkg/seahorse"
)

// ConvMap stores the mapping from sampleID to seahorse ConversationID.
type ConvMap map[string]int64

// SeahorseIngestResult holds the results of ingesting into seahorse.
type SeahorseIngestResult struct {
	Engine  *seahorse.Engine
	ConvMap ConvMap // sampleID → conversationID
}

// IngestSeahorse loads all LOCOMO samples into a seahorse Engine.
// Returns the engine and a mapping from sampleID to conversationID for scoped retrieval.
func IngestSeahorse(ctx context.Context, samples []LocomoSample, dbPath string) (*SeahorseIngestResult, error) {
	noopFn := func(ctx context.Context, prompt string, opts seahorse.CompleteOptions) (string, error) {
		return "", nil
	}

	engine, err := seahorse.NewEngine(seahorse.Config{
		DBPath: dbPath,
	}, noopFn)
	if err != nil {
		return nil, fmt.Errorf("create seahorse engine: %w", err)
	}

	store := engine.GetRetrieval().Store()
	convMap := make(ConvMap)

	for si := range samples {
		sample := &samples[si]
		sessionKey := "locomo-" + sample.SampleID

		// Check if conversation already exists (idempotent)
		existing, _ := store.GetConversationBySessionKey(ctx, sessionKey)
		if existing != nil {
			convMap[sample.SampleID] = existing.ConversationID
			log.Printf("Skipping existing sample %s: convID=%d", sample.SampleID, existing.ConversationID)
			continue
		}

		turns := GetTurns(sample)

		// Convert turns to seahorse messages
		msgs := make([]seahorse.Message, 0, len(turns))
		for _, turn := range turns {
			content := turn.Speaker + ": " + turn.Text
			msgs = append(msgs, seahorse.Message{
				Role:       "user",
				Content:    content,
				TokenCount: len(turn.Text) / 4,
			})
		}

		// Ingest all turns for this sample
		_, err := engine.Ingest(ctx, sessionKey, msgs)
		if err != nil {
			return nil, fmt.Errorf("ingest sample %s: %w", sample.SampleID, err)
		}

		// Get the conversation ID for scoped retrieval
		conv, err := store.GetConversationBySessionKey(ctx, sessionKey)
		if err != nil {
			return nil, fmt.Errorf("get conversation for %s: %w", sample.SampleID, err)
		}
		if conv == nil {
			return nil, fmt.Errorf("conversation not found for %s after ingest", sample.SampleID)
		}
		convMap[sample.SampleID] = conv.ConversationID
		log.Printf("Ingested sample %s: %d turns, convID=%d", sample.SampleID, len(turns), conv.ConversationID)
	}

	log.Printf("Seahorse ingestion complete: %d samples, %d conversations", len(samples), len(convMap))
	return &SeahorseIngestResult{
		Engine:  engine,
		ConvMap: convMap,
	}, nil
}
