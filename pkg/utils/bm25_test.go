package utils

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// testDoc is a generic structure for use in tests.
type testDoc struct {
	ID   int
	Text string
}

func extractText(d testDoc) string {
	return d.Text
}

func TestBM25Search_EdgeCases(t *testing.T) {
	corpus := []testDoc{
		{1, "hello world"},
		{2, "foo bar"},
	}
	engine := NewBM25Engine(corpus, extractText)

	tests := []struct {
		name  string
		query string
		topK  int
	}{
		{"Zero topK", "hello", 0},
		{"Negative topK", "hello", -1},
		{"Empty query", "", 5},
		{"Query with only punctuation", "...,,,!!!", 5},
		{"No matches found", "golang", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := engine.Search(tt.query, tt.topK)
			if len(results) != 0 {
				t.Errorf("expected 0 results, got %d", len(results))
			}
			// Check that it never returns nil, but an empty slice
			if results == nil {
				t.Errorf("expected empty slice, got nil")
			}
		})
	}
}

func TestBM25Search_EmptyCorpus(t *testing.T) {
	engine := NewBM25Engine([]testDoc{}, extractText)
	results := engine.Search("hello", 5)
	if len(results) != 0 || results == nil {
		t.Errorf("expected empty slice from empty corpus, got %v", results)
	}
}

func TestBM25Search_RankingLogic(t *testing.T) {
	corpus := []testDoc{
		{1, "the quick brown fox jumps over the lazy dog"},
		{2, "quick fox"},
		{3, "quick quick quick fox"}, // High Term Frequency (TF)
		{4, "completely irrelevant document here"},
	}
	engine := NewBM25Engine(corpus, extractText)

	t.Run("Term Frequency (TF) boosts score", func(t *testing.T) {
		results := engine.Search("quick", 5)
		if len(results) < 3 {
			t.Fatalf("expected at least 3 results, got %d", len(results))
		}
		// Doc 3 has the word "quick" repeated 3 times, it should beat Doc 2
		if results[0].Document.ID != 3 {
			t.Errorf("expected doc 3 to rank first due to high TF, got doc %d", results[0].Document.ID)
		}
	})

	t.Run("Document Length penalty", func(t *testing.T) {
		results := engine.Search("fox", 5)
		if len(results) < 3 {
			t.Fatalf("expected at least 3 results, got %d", len(results))
		}
		// Doc 2 ("quick fox") is much shorter than Doc 1 ("the quick brown fox..."),
		// so, with equal Term Frequency for the word "fox" (1 time), Doc 2 wins.
		if results[0].Document.ID != 2 {
			t.Errorf("expected doc 2 to rank first due to shorter length, got doc %d", results[0].Document.ID)
		}
	})

	t.Run("TopK limits results", func(t *testing.T) {
		results := engine.Search("quick", 2)
		if len(results) != 2 {
			t.Errorf("expected exactly 2 results, got %d", len(results))
		}
	})
}

func TestBM25Tokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello World", []string{"hello", "world"}},
		{"  spaces   everywhere  ", []string{"spaces", "everywhere"}},
		{"punctuation... test!!!", []string{"punctuation", "test"}},
		{"(parentheses) and-hyphens", []string{"parentheses", "and-hyphens"}}, // hyphens trimmed from edges
		{"internal-hyphen is kept", []string{"internal-hyphen", "is", "kept"}},
		{".,;?!", []string{}}, // Becomes empty after trim
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := bm25Tokenize(tt.input)
			if len(got) == 0 && len(tt.expected) == 0 {
				return // Both empty
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("bm25Tokenize(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBM25Dedupe(t *testing.T) {
	input := []string{"apple", "banana", "apple", "orange", "banana"}
	expected := []string{"apple", "banana", "orange"}

	got := bm25Dedupe(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("bm25Dedupe() = %v, want %v", got, expected)
	}
}

func TestBM25Options(t *testing.T) {
	corpus := []testDoc{{1, "test"}}

	engine := NewBM25Engine(
		corpus,
		extractText,
		WithK1(2.5),
		WithB(0.9),
	)

	if engine.k1 != 2.5 {
		t.Errorf("expected k1 to be 2.5, got %v", engine.k1)
	}
	if engine.b != 0.9 {
		t.Errorf("expected b to be 0.9, got %v", engine.b)
	}
}

func TestBM25Search_SortingStability(t *testing.T) {
	// Ensure that sorting by heap returns in correct descending order
	corpus := []testDoc{
		{1, "golang is good"},
		{2, "golang golang"},
		{3, "golang golang golang"},
		{4, "golang golang golang golang"},
	}
	engine := NewBM25Engine(corpus, extractText)
	results := engine.Search("golang", 10)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Score should be strictly decreasing
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted correctly: result %d score (%v) > result %d score (%v)",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func BenchmarkBM25Search_ReusedIndex(b *testing.B) {
	corpus := benchmarkBM25Corpus(2000)
	engine := NewBM25Engine(corpus, extractText)
	query := "hardware gpio i2c sensor controller latency"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := engine.Search(query, 10)
		if len(results) == 0 {
			b.Fatal("expected non-empty results")
		}
	}
}

func BenchmarkBM25Search_RebuildEachTime(b *testing.B) {
	corpus := benchmarkBM25Corpus(2000)
	query := "hardware gpio i2c sensor controller latency"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine := NewBM25Engine(corpus, extractText)
		results := engine.Search(query, 10)
		if len(results) == 0 {
			b.Fatal("expected non-empty results")
		}
	}
}

func benchmarkBM25Corpus(size int) []testDoc {
	corpus := make([]testDoc, size)
	topics := []string{
		"hardware gpio pwm adc sensor controller latency throughput",
		"telegram markdown parser message escape formatting bot command",
		"jsonl memory session history storage append compact recovery",
		"openai provider routing agent tool search registry hidden tools",
		"i2c spi uart serial device bus address transfer clock",
	}

	for i := range corpus {
		topic := topics[i%len(topics)]
		corpus[i] = testDoc{
			ID: i,
			Text: fmt.Sprintf(
				"doc %d %s repeated repeated %s variant-%d %s",
				i,
				topic,
				topic,
				i%17,
				strings.Repeat("token ", (i%7)+1),
			),
		}
	}

	return corpus
}
