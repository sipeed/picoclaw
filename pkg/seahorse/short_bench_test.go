package seahorse

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newBenchStore creates a test store for benchmarks.
func newBenchStore(b *testing.B) (*Store, func()) {
	b.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("open test db: %v", err)
	}
	if err := runSchema(db); err != nil {
		db.Close()
		b.Fatalf("migration: %v", err)
	}
	return &Store{db: db}, func() { db.Close() }
}

// --- Ingest benchmarks ---

func BenchmarkIngest_SingleMessage(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "bench:ingest")
	convID := conv.ConversationID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.AddMessage(ctx, convID, "user", "Test message content", 15)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIngest_BatchMessages(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conv, _ := s.GetOrCreateConversation(ctx, fmt.Sprintf("bench:ingest-batch:%d", i))
		convID := conv.ConversationID

		for j := 0; j < 10; j++ {
			added, err := s.AddMessage(ctx, convID, "user",
				fmt.Sprintf("Message %d in batch", j), 10)
			if err != nil {
				b.Fatal(err)
			}
			s.AppendContextMessage(ctx, convID, added.ID)
		}
	}
}

// --- Assemble benchmarks ---

func BenchmarkAssemble_MessagesOnly(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "bench:assemble-msgs")
	convID := conv.ConversationID

	// Add 100 messages
	for i := 0; i < 100; i++ {
		m, _ := s.AddMessage(ctx, convID, "user",
			fmt.Sprintf("Message content %d with some text", i), 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	a := &Assembler{store: s}
	input := AssembleInput{Budget: 50000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.Assemble(ctx, convID, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssemble_WithSummaries(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "bench:assemble-sums")
	convID := conv.ConversationID

	now := time.Now().UTC()

	// Add 10 leaf summaries
	for i := 0; i < 10; i++ {
		sum, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("Leaf summary %d", i),
			TokenCount:     500,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		s.AppendContextSummary(ctx, convID, sum.SummaryID)
	}

	// Add 20 fresh messages
	for i := 0; i < 20; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", fmt.Sprintf("Fresh message %d", i), 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	a := &Assembler{store: s}
	input := AssembleInput{Budget: 10000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.Assemble(ctx, convID, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssemble_BudgetEviction(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "bench:assemble-evict")
	convID := conv.ConversationID

	now := time.Now().UTC()

	// Add 50 leaf summaries (more than budget can hold)
	for i := 0; i < 50; i++ {
		sum, _ := s.CreateSummary(ctx, CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("Summary %d", i),
			TokenCount:     300,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		s.AppendContextSummary(ctx, convID, sum.SummaryID)
	}

	// Add fresh tail
	for i := 0; i < FreshTailCount; i++ {
		m, _ := s.AddMessage(ctx, convID, "user", "fresh", 10)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	a := &Assembler{store: s}
	input := AssembleInput{Budget: 5000} // Force eviction

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.Assemble(ctx, convID, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Search (FTS5) benchmarks ---

// benchSeedSummaries adds n summaries to a conversation for search benchmarks.
func benchSeedSummaries(b *testing.B, s *Store, convID int64, n int, contentTpl string) {
	b.Helper()
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		sum, err := s.CreateSummary(context.Background(), CreateSummaryInput{
			ConversationID: convID,
			Kind:           SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf(contentTpl, i),
			TokenCount:     200,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		if err != nil {
			b.Fatalf("create summary: %v", err)
		}
		s.AppendContextSummary(context.Background(), convID, sum.SummaryID)
	}
}

func BenchmarkSearchSummaries_FTS5(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "bench:search-fts")
	convID := conv.ConversationID

	benchSeedSummaries(b, s, convID, 100, "Summary about database configuration and API endpoints %d")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.SearchSummaries(ctx, SearchInput{
			Pattern:        "database",
			Mode:           "full_text",
			ConversationID: convID,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchSummaries_Like(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "bench:search-like")
	convID := conv.ConversationID

	benchSeedSummaries(b, s, convID, 100, "Summary about configuration %d")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.SearchSummaries(ctx, SearchInput{
			Pattern:        "config",
			Mode:           "like",
			ConversationID: convID,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchMessages_FTS5(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()
	conv, _ := s.GetOrCreateConversation(ctx, "bench:search-msg-fts")
	convID := conv.ConversationID

	// Add 500 messages
	for i := 0; i < 500; i++ {
		m, _ := s.AddMessage(ctx, convID, "user",
			fmt.Sprintf("User message about API and database integration %d", i), 20)
		s.AppendContextMessage(ctx, convID, m.ID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.SearchMessages(ctx, SearchInput{
			Pattern:        "API database",
			Mode:           "full_text",
			ConversationID: convID,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Bootstrap benchmarks ---

func BenchmarkBootstrap_Empty(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conv, _ := s.GetOrCreateConversation(ctx, fmt.Sprintf("bench:bootstrap-empty:%d", i))
		convID := conv.ConversationID
		_ = convID // Bootstrap with empty history
	}
}

func BenchmarkBootstrap_100Messages(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()

	// Prepare 100 messages
	msgs := make([]Message, 100)
	for i := 0; i < 100; i++ {
		msgs[i] = Message{
			Role:       "user",
			Content:    fmt.Sprintf("Bootstrap message %d", i),
			TokenCount: 15,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conv, _ := s.GetOrCreateConversation(ctx, fmt.Sprintf("bench:bootstrap-100:%d", i))
		convID := conv.ConversationID

		for _, m := range msgs {
			added, _ := s.AddMessage(ctx, convID, m.Role, m.Content, m.TokenCount)
			s.AppendContextMessage(ctx, convID, added.ID)
		}
	}
}

func BenchmarkBootstrap_500Messages(b *testing.B) {
	s, cleanup := newBenchStore(b)
	defer cleanup()
	ctx := context.Background()

	msgs := make([]Message, 500)
	for i := 0; i < 500; i++ {
		msgs[i] = Message{
			Role:       "user",
			Content:    fmt.Sprintf("Bootstrap message %d", i),
			TokenCount: 15,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conv, _ := s.GetOrCreateConversation(ctx, fmt.Sprintf("bench:bootstrap-500:%d", i))
		convID := conv.ConversationID

		for _, m := range msgs {
			added, _ := s.AddMessage(ctx, convID, m.Role, m.Content, m.TokenCount)
			s.AppendContextMessage(ctx, convID, added.ID)
		}
	}
}
