package providers

import (
	"context"
	"os"
	"testing"
)

func TestMistralProvider_Integration(t *testing.T) {
	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		t.Skip("MISTRAL_API_KEY not set")
	}

	provider := NewHTTPProvider(apiKey, "https://api.mistral.ai/v1", "")
	
	messages := []Message{
		{Role: "user", Content: "Say 'Hello from Mistral' in exactly 3 words"},
	}

	resp, err := provider.Chat(context.Background(), messages, nil, "mistral-tiny", map[string]interface{}{
		"max_tokens": 50,
	})

	if err != nil {
		t.Fatalf("Mistral chat failed: %v", err)
	}

	if resp.Content == "" {
		t.Errorf("Expected non-empty response")
	}

	t.Logf("Mistral response: %s", resp.Content)
}