// Package utils provides shared, reusable algorithms.
// This file implements a generic BM25 search engine.
//
// Usage:
//
//	type MyDoc struct { ID string; Body string }
//
//	corpus := []MyDoc{...}
//	engine := bm25.New(corpus, func(d MyDoc) string {
//	    return d.ID + " " + d.Body
//	})
//	results := engine.Search("my query", 5)
package utils

import (
	"math"
	"sort"
	"strings"
)

// ── Tuning defaults ───────────────────────────────────────────────────────────

const (
	// DefaultBM25K1 is the term-frequency saturation factor (typical range 1.2–2.0).
	// Higher values give more weight to repeated terms.
	DefaultBM25K1 = 1.2

	// DefaultBM25B is the document-length normalization factor (0 = none, 1 = full).
	DefaultBM25B = 0.75
)

// BM25Engine is a query-time BM25 search engine over a generic corpus.
// T is the document type; the caller supplies a TextFunc that extracts the
// searchable text from each document.
//
// The engine is stateless between queries: no caching, no invalidation logic.
// All indexing work is performed inside Search() on every call, making it
// safe to use on corpora that change frequently.
type BM25Engine[T any] struct {
	corpus   []T
	textFunc func(T) string
	k1       float64
	b        float64
}

// BM25Option is a functional option to configure a BM25Engine.
type BM25Option func(*bm25Config)

type bm25Config struct {
	k1 float64
	b  float64
}

// WithK1 overrides the term-frequency saturation constant (default 1.2).
func WithK1(k1 float64) BM25Option {
	return func(c *bm25Config) { c.k1 = k1 }
}

// WithB overrides the document-length normalization factor (default 0.75).
func WithB(b float64) BM25Option {
	return func(c *bm25Config) { c.b = b }
}

// NewBM25Engine creates a BM25Engine for the given corpus.
//
//   - corpus   : slice of documents of any type T.
//   - textFunc : function that returns the searchable text for a document.
//   - opts     : optional tuning (WithK1, WithB).
//
// The corpus slice is referenced, not copied. Callers must not mutate it
// concurrently with Search().
func NewBM25Engine[T any](corpus []T, textFunc func(T) string, opts ...BM25Option) *BM25Engine[T] {
	cfg := bm25Config{k1: DefaultBM25K1, b: DefaultBM25B}
	for _, o := range opts {
		o(&cfg)
	}
	return &BM25Engine[T]{
		corpus:   corpus,
		textFunc: textFunc,
		k1:       cfg.k1,
		b:        cfg.b,
	}
}

// BM25Result is a single ranked result from a Search call.
type BM25Result[T any] struct {
	Document T
	Score    float32
}

// Search ranks the corpus against query and returns the top-k results.
// Returns an empty slice (not nil) when there are no matches.
//
// Complexity: O(N×L) for indexing + O(|Q|×avgPostingLen) for scoring,
// where N = corpus size, L = average document length, Q = query terms.
// Top-k extraction uses a fixed-size min-heap: O(candidates × log k).
func (e *BM25Engine[T]) Search(query string, topK int) []BM25Result[T] {
	if topK <= 0 {
		return []BM25Result[T]{}
	}

	queryTerms := bm25Tokenize(query)
	if len(queryTerms) == 0 {
		return []BM25Result[T]{}
	}

	N := len(e.corpus)
	if N == 0 {
		return []BM25Result[T]{}
	}

	// Step 1: build per-document tf + raw doc lengths
	type docEntry struct {
		tf     map[string]uint32
		rawLen int
	}

	entries := make([]docEntry, N)
	df := make(map[string]int, 64)
	totalLen := 0

	for i, doc := range e.corpus {
		tokens := bm25Tokenize(e.textFunc(doc))
		totalLen += len(tokens)

		tf := make(map[string]uint32, len(tokens))
		for _, t := range tokens {
			tf[t]++
		}
		// df: each term counts once per document (iterate the map, keys are unique)
		for t := range tf {
			df[t]++
		}

		entries[i] = docEntry{tf: tf, rawLen: len(tokens)}
	}

	avgDocLen := float64(totalLen) / float64(N)

	// Step 2: pre-compute IDF and per-doc length normalization
	// IDF (Robertson smoothing): log( (N - df(t) + 0.5) / (df(t) + 0.5) + 1 )
	idf := make(map[string]float32, len(df))
	for term, freq := range df {
		idf[term] = float32(math.Log(
			(float64(N)-float64(freq)+0.5)/(float64(freq)+0.5) + 1,
		))
	}

	// docLenNorm[i] = k1 * (1 - b + b * |doc_i| / avgDocLen)
	// Stored as float32 — sufficient precision for ranking.
	docLenNorm := make([]float32, N)
	for i, entry := range entries {
		docLenNorm[i] = float32(e.k1 * (1 - e.b + e.b*float64(entry.rawLen)/avgDocLen))
	}

	// Step 3: build inverted index (posting lists)
	// Iterate the tf map directly — map keys are already unique, no seen-set needed.
	posting := make(map[string][]int32, len(df))
	for i, entry := range entries {
		for term := range entry.tf {
			posting[term] = append(posting[term], int32(i))
		}
	}

	// Step 4: score via posting lists
	// Deduplicate query terms to avoid double-weighting the same term.
	unique := bm25Dedupe(queryTerms)

	scores := make(map[int32]float32)
	for _, term := range unique {
		termIDF, ok := idf[term]
		if !ok {
			continue // term not in vocabulary → zero contribution
		}
		for _, docID := range posting[term] {
			freq := float32(entries[docID].tf[term])
			// TF_norm = freq * (k1+1) / (freq + docLenNorm)
			tfNorm := freq * float32(e.k1+1) / (freq + docLenNorm[docID])
			scores[docID] += termIDF * tfNorm
		}
	}

	if len(scores) == 0 {
		return []BM25Result[T]{}
	}

	// Step 5: top-K via fixed-size min-heap
	heap := make([]bm25ScoredDoc, 0, topK)

	for docID, sc := range scores {
		switch {
		case len(heap) < topK:
			heap = append(heap, bm25ScoredDoc{docID: docID, score: sc})
			if len(heap) == topK {
				bm25MinHeapify(heap)
			}
		case sc > heap[0].score:
			heap[0] = bm25ScoredDoc{docID: docID, score: sc}
			bm25SiftDown(heap, 0)
		}
	}

	sort.Slice(heap, func(i, j int) bool { return heap[i].score > heap[j].score })

	out := make([]BM25Result[T], len(heap))
	for i, h := range heap {
		out[i] = BM25Result[T]{
			Document: e.corpus[h.docID],
			Score:    h.score,
		}
	}
	return out
}

// bm25Tokenize splits s into lowercase tokens, stripping edge punctuation.
func bm25Tokenize(s string) []string {
	raw := strings.Fields(strings.ToLower(s))
	out := raw[:0] // reuse backing array to avoid extra allocation
	for _, t := range raw {
		t = strings.Trim(t, ".,;:!?\"'()/\\-_")
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// bm25Dedupe returns a new slice with duplicate tokens removed,
// preserving first-occurrence order.
func bm25Dedupe(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

type bm25ScoredDoc struct {
	docID int32
	score float32
}

// bm25MinHeapify builds a min-heap in-place using Floyd's algorithm: O(k).
func bm25MinHeapify(h []bm25ScoredDoc) {
	for i := len(h)/2 - 1; i >= 0; i-- {
		bm25SiftDown(h, i)
	}
}

// bm25SiftDown restores the min-heap property starting at node i: O(log k).
func bm25SiftDown(h []bm25ScoredDoc, i int) {
	n := len(h)
	for {
		smallest := i
		l, r := 2*i+1, 2*i+2
		if l < n && h[l].score < h[smallest].score {
			smallest = l
		}
		if r < n && h[r].score < h[smallest].score {
			smallest = r
		}
		if smallest == i {
			break
		}
		h[i], h[smallest] = h[smallest], h[i]
		i = smallest
	}
}
