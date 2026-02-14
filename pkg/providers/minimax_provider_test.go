package providers

import (
	"context"
	"os"
	"testing"
)

func TestMiniMaxProvider_Chat(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping MiniMax integration test: MINIMAX_API_KEY not set")
	}

	apiBase := "https://api.minimax.io/v1"
	provider := MiniMaxProvider(apiKey, apiBase)

	resp, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "Hi"}}, nil, "M2-her", nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content == "" {
		t.Errorf("Expected non-empty content")
	}

	t.Logf("Response: %s", resp.Content)
}
