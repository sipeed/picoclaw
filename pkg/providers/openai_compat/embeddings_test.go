package openai_compat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderEmbedBatch_OmitsDimensionsAndTruncatesLocally(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("path = %s, want /embeddings", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		response := map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{1, 2, 3, 4}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewProvider(
		"test-key",
		server.URL,
		"",
		WithExtraBody(map[string]any{
			"dimensions":      8,
			"encoding_format": "float",
			"user":            "picoclaw",
		}),
	)
	vectors, err := provider.EmbedBatch(t.Context(), []string{"hello"}, "gemma-2b-embeddings", 2)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}
	if len(vectors) != 1 {
		t.Fatalf("len(vectors) = %d, want 1", len(vectors))
	}
	if len(vectors[0]) != 2 || vectors[0][0] != 1 || vectors[0][1] != 2 {
		t.Fatalf("vectors[0] = %#v, want [1 2]", vectors[0])
	}

	if _, ok := requestBody["dimensions"]; ok {
		t.Fatalf("request body unexpectedly included dimensions: %#v", requestBody)
	}
	if got := requestBody["encoding_format"]; got != "float" {
		t.Fatalf("encoding_format = %#v, want %q", got, "float")
	}
	if got := requestBody["user"]; got != "picoclaw" {
		t.Fatalf("user = %#v, want %q", got, "picoclaw")
	}
	if got := requestBody["model"]; got != "gemma-2b-embeddings" {
		t.Fatalf("model = %#v, want %q", got, "gemma-2b-embeddings")
	}
	input, ok := requestBody["input"].([]any)
	if !ok || len(input) != 1 || input[0] != "hello" {
		t.Fatalf("input = %#v, want [hello]", requestBody["input"])
	}
}

func TestProviderEmbedQuery_ReturnsErrorWhenVectorIsTooShort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{1, 2}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewProvider("test-key", server.URL, "")
	_, err := provider.EmbedQuery(t.Context(), "hello", "gemma-2b-embeddings", 3)
	if err == nil {
		t.Fatal("EmbedQuery() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "shorter than requested dimensions") {
		t.Fatalf("EmbedQuery() error = %q, want short-vector error", err)
	}
}
