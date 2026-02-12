package vecstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// Embedder generates embedding vectors from text.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// HTTPEmbedder calls an OpenAI-compatible /v1/embeddings endpoint.
type HTTPEmbedder struct {
	apiBase string
	apiKey  string
	model   string
	client  *http.Client
}

// NewHTTPEmbedder creates an embedder targeting an OpenAI-compatible API.
func NewHTTPEmbedder(apiBase, apiKey, model string) *HTTPEmbedder {
	return &HTTPEmbedder{
		apiBase: apiBase,
		apiKey:  apiKey,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Embed sends all texts in one batch request and returns their embeddings.
// Retries up to 3 times with exponential backoff on transient errors.
func (e *HTTPEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(embeddingRequest{
		Input: texts,
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	url := e.apiBase + "/embeddings"

	const maxRetries = 3
	var lastErr error
	for attempt := range maxRetries {
		result, err := e.doRequest(ctx, url, body, len(texts))
		if err == nil {
			return result, nil
		}
		lastErr = err
		// Exponential backoff: 500ms, 2s, 8s
		backoff := time.Duration(math.Pow(4, float64(attempt))) * 500 * time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, fmt.Errorf("embedding failed after %d retries: %w", maxRetries, lastErr)
}

func (e *HTTPEmbedder) doRequest(ctx context.Context, url string, body []byte, n int) ([][]float32, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API %d: %s", resp.StatusCode, string(respBody))
	}

	var result embeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("embedding API error: %s", result.Error.Message)
	}

	// Order by index
	embeddings := make([][]float32, n)
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}
