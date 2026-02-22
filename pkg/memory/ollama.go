// ABOUTME: Ollama connectivity for embedding generation.
// ABOUTME: Provides health checks and constants for the Ollama embedding service.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	DefaultOllamaURL      = "http://localhost:11434"
	DefaultEmbeddingModel = "nomic-embed-text"
)

// ollamaTagsResponse is the response from GET /api/tags.
type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name string `json:"name"`
}

// CheckOllamaAvailable checks whether Ollama is reachable and has the
// requested embedding model installed. Returns nil on success.
func CheckOllamaAvailable(ctx context.Context, ollamaURL, model string) error {
	if ollamaURL == "" {
		ollamaURL = DefaultOllamaURL
	}
	if model == "" {
		model = DefaultEmbeddingModel
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama unreachable at %s: %w", ollamaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return fmt.Errorf("decoding tags response: %w", err)
	}

	for _, m := range tags.Models {
		// Ollama model names can include ":latest" suffix
		if m.Name == model || m.Name == model+":latest" {
			return nil
		}
	}

	return fmt.Errorf("model %q not found in ollama (have %d models); run: ollama pull %s",
		model, len(tags.Models), model)
}
