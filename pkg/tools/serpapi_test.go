package tools

import (
	"context"
	"os"
	"testing"
)

func TestSerpAPITool_Integration(t *testing.T) {
	apiKey := os.Getenv("SERPAPI_API_KEY")
	if apiKey == "" {
		t.Skip("SERPAPI_API_KEY not set")
	}

	tool := NewSerpAPITool(apiKey, 5)
	
	result := tool.Execute(context.Background(), map[string]interface{}{
		"query":   "golang programming",
		"engine":  "google",
		"num":     float64(3),
	})

	if result.IsError {
		t.Errorf("SerpAPI execution failed: %s", result.ForLLM)
	}

	t.Logf("SerpAPI result:\n%s", result.ForUser)
}