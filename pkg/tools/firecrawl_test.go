package tools

import (
	"context"
	"os"
	"testing"
)

func TestFirecrawlTool_Integration(t *testing.T) {
	apiKey := os.Getenv("FIRECRAWL_API_KEY")
	if apiKey == "" {
		t.Skip("FIRECRAWL_API_KEY not set")
	}

	tool := NewFirecrawlTool(apiKey, "")
	
	result := tool.Execute(context.Background(), map[string]interface{}{
		"url": "https://example.com",
		"formats": []string{"markdown"},
	})

	if result.IsError {
		t.Errorf("Firecrawl execution failed: %s", result.ForLLM)
	}

	t.Logf("Firecrawl result:\n%s", result.ForUser)
}