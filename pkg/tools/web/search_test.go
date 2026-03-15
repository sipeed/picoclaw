package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebTool_WebSearch_NoApiKey(t *testing.T) {
	tool, err := NewWebSearchTool(WebSearchToolOptions{BraveEnabled: true, BraveAPIKeys: nil})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tool != nil {
		t.Errorf("Expected nil tool when Brave API key is empty")
	}

	// Also nil when nothing is enabled
	tool, err = NewWebSearchTool(WebSearchToolOptions{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tool != nil {
		t.Errorf("Expected nil tool when no provider is enabled")
	}
}

func TestWebTool_WebSearch_MissingQuery(t *testing.T) {
	tool, err := NewWebSearchTool(WebSearchToolOptions{
		BraveEnabled:    true,
		BraveAPIKeys:    []string{"test-key"},
		BraveMaxResults: 5,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	ctx := context.Background()
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when query is missing")
	}
}

func TestNewWebSearchTool_PropagatesProxy(t *testing.T) {
	t.Run("perplexity", func(t *testing.T) {
		tool, err := NewWebSearchTool(WebSearchToolOptions{
			PerplexityEnabled:    true,
			PerplexityAPIKeys:    []string{"k"},
			PerplexityMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
		if err != nil {
			t.Fatalf("NewWebSearchTool() error: %v", err)
		}
		p, ok := tool.provider.(*PerplexitySearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *PerplexitySearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("brave", func(t *testing.T) {
		tool, err := NewWebSearchTool(WebSearchToolOptions{
			BraveEnabled:    true,
			BraveAPIKeys:    []string{"k"},
			BraveMaxResults: 3,
			Proxy:           "http://127.0.0.1:7890",
		})
		if err != nil {
			t.Fatalf("NewWebSearchTool() error: %v", err)
		}
		p, ok := tool.provider.(*BraveSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *BraveSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("duckduckgo", func(t *testing.T) {
		tool, err := NewWebSearchTool(WebSearchToolOptions{
			DuckDuckGoEnabled:    true,
			DuckDuckGoMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
		if err != nil {
			t.Fatalf("NewWebSearchTool() error: %v", err)
		}
		p, ok := tool.provider.(*DuckDuckGoSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *DuckDuckGoSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})
}

func TestWebTool_TavilySearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["api_key"] != "test-key" {
			t.Errorf("Expected api_key test-key, got %v", payload["api_key"])
		}
		if payload["query"] != "test query" {
			t.Errorf("Expected query 'test query', got %v", payload["query"])
		}

		response := map[string]any{
			"results": []map[string]any{
				{
					"title":   "Test Result 1",
					"url":     "https://example.com/1",
					"content": "Content for result 1",
				},
				{
					"title":   "Test Result 2",
					"url":     "https://example.com/2",
					"content": "Content for result 2",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		TavilyEnabled:    true,
		TavilyAPIKeys:    []string{"test-key"},
		TavilyBaseURL:    server.URL,
		TavilyMaxResults: 5,
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	ctx := context.Background()
	args := map[string]any{
		"query": "test query",
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForUser, "Test Result 1") ||
		!strings.Contains(result.ForUser, "https://example.com/1") {
		t.Errorf("Expected results in output, got: %s", result.ForUser)
	}

	if !strings.Contains(result.ForUser, "via Tavily") {
		t.Errorf("Expected 'via Tavily' in output, got: %s", result.ForUser)
	}
}

func TestWebTool_TavilySearch_Failover(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		apiKey := payload["api_key"].(string)

		if apiKey == "key1" {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limited"))
			return
		}

		if apiKey == "key2" {
			response := map[string]any{
				"results": []map[string]any{
					{
						"title":   "Success Result",
						"url":     "https://example.com/success",
						"content": "Success content",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		TavilyEnabled:    true,
		TavilyAPIKeys:    []string{"key1", "key2"},
		TavilyBaseURL:    server.URL,
		TavilyMaxResults: 5,
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	ctx := context.Background()
	args := map[string]any{
		"query": "test query",
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success, got Error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "Success Result") {
		t.Errorf("Expected failover to second key and success result, got: %s", result.ForUser)
	}
}

func TestWebTool_GLMSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-glm-key" {
			t.Errorf("Expected Authorization Bearer test-glm-key, got %s", r.Header.Get("Authorization"))
		}

		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["search_query"] != "test query" {
			t.Errorf("Expected search_query 'test query', got %v", payload["search_query"])
		}
		if payload["search_engine"] != "search_std" {
			t.Errorf("Expected search_engine 'search_std', got %v", payload["search_engine"])
		}

		response := map[string]any{
			"id":      "web-search-test",
			"created": 1709568000,
			"search_result": []map[string]any{
				{
					"title":        "Test GLM Result",
					"content":      "GLM search snippet",
					"link":         "https://example.com/glm",
					"media":        "Example",
					"publish_date": "2026-03-04",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		GLMSearchEnabled: true,
		GLMSearchAPIKey:  "test-glm-key",
		GLMSearchBaseURL: server.URL,
		GLMSearchEngine:  "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"query": "test query",
	})

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "Test GLM Result") {
		t.Errorf("Expected 'Test GLM Result' in output, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "https://example.com/glm") {
		t.Errorf("Expected URL in output, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "via GLM Search") {
		t.Errorf("Expected 'via GLM Search' in output, got: %s", result.ForUser)
	}
}

func TestWebTool_GLMSearch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	tool, err := NewWebSearchTool(WebSearchToolOptions{
		GLMSearchEnabled: true,
		GLMSearchAPIKey:  "bad-key",
		GLMSearchBaseURL: server.URL,
		GLMSearchEngine:  "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	result := tool.Execute(context.Background(), map[string]any{
		"query": "test query",
	})

	if !result.IsError {
		t.Errorf("Expected IsError=true for 401 response")
	}
	if !strings.Contains(result.ForLLM, "status 401") {
		t.Errorf("Expected status 401 in error, got: %s", result.ForLLM)
	}
}

func TestWebTool_GLMSearch_Priority(t *testing.T) {
	// GLM Search should only be selected when all other providers are disabled
	tool, err := NewWebSearchTool(WebSearchToolOptions{
		DuckDuckGoEnabled:    true,
		DuckDuckGoMaxResults: 5,
		GLMSearchEnabled:     true,
		GLMSearchAPIKey:      "test-key",
		GLMSearchBaseURL:     "https://example.com",
		GLMSearchEngine:      "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}

	// DuckDuckGo should win over GLM Search
	if _, ok := tool.provider.(*DuckDuckGoSearchProvider); !ok {
		t.Errorf("Expected DuckDuckGoSearchProvider when both enabled, got %T", tool.provider)
	}

	// With DuckDuckGo disabled, GLM Search should be selected
	tool2, err := NewWebSearchTool(WebSearchToolOptions{
		DuckDuckGoEnabled: false,
		GLMSearchEnabled:  true,
		GLMSearchAPIKey:   "test-key",
		GLMSearchBaseURL:  "https://example.com",
		GLMSearchEngine:   "search_std",
	})
	if err != nil {
		t.Fatalf("NewWebSearchTool() error: %v", err)
	}
	if _, ok := tool2.provider.(*GLMSearchProvider); !ok {
		t.Errorf("Expected GLMSearchProvider when only GLM enabled, got %T", tool2.provider)
	}
}
