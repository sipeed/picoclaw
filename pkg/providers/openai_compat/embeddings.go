package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// EmbedQuery returns a single embedding vector for the given text.
func (p *Provider) EmbedQuery(
	ctx context.Context,
	input string,
	model string,
	dimensions int,
) ([]float64, error) {
	vectors, err := p.EmbedBatch(ctx, []string{input}, model, dimensions)
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("embedding response returned no vectors")
	}
	return vectors[0], nil
}

// EmbedBatch calls the OpenAI-compatible /embeddings endpoint and truncates
// the returned vectors locally when a smaller output width is requested.
// The request body intentionally omits dimensions so non-Matryoshka models,
// including vLLM-backed embeddings, never see an unsupported field upstream.
func (p *Provider) EmbedBatch(
	ctx context.Context,
	inputs []string,
	model string,
	dimensions int,
) ([][]float64, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}
	if len(inputs) == 0 {
		return [][]float64{}, nil
	}
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("embedding model is required")
	}
	if dimensions < 0 {
		return nil, fmt.Errorf("dimensions must be non-negative")
	}

	requestBody := map[string]any{
		"model": normalizeModel(model, p.apiBase),
		"input": inputs,
	}
	for key, value := range p.extraBody {
		if strings.EqualFold(strings.TrimSpace(key), "dimensions") {
			continue
		}
		requestBody[key] = value
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+"/embeddings", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	p.applyCustomHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	var apiResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}
	if len(apiResp.Data) != len(inputs) {
		return nil, fmt.Errorf(
			"embedding response returned %d vectors for %d inputs",
			len(apiResp.Data),
			len(inputs),
		)
	}

	results := make([][]float64, 0, len(apiResp.Data))
	for i, item := range apiResp.Data {
		vector, err := truncateEmbeddingVector(item.Embedding, dimensions)
		if err != nil {
			return nil, fmt.Errorf("embedding %d: %w", i, err)
		}
		results = append(results, vector)
	}

	return results, nil
}

func truncateEmbeddingVector(vector []float64, dimensions int) ([]float64, error) {
	if dimensions < 0 {
		return nil, fmt.Errorf("dimensions must be non-negative")
	}
	if dimensions == 0 {
		return append([]float64(nil), vector...), nil
	}
	if len(vector) < dimensions {
		return nil, fmt.Errorf(
			"embedding length %d is shorter than requested dimensions %d",
			len(vector),
			dimensions,
		)
	}

	return append([]float64(nil), vector[:dimensions]...), nil
}
