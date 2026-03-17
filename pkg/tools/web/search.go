package web

import (
	"context"
	"fmt"
	"jane/pkg/tools"
)

type WebSearchTool struct {
	provider   SearchProvider
	maxResults int
}

type WebSearchToolOptions struct {
	BraveAPIKeys         []string
	BraveMaxResults      int
	BraveEnabled         bool
	TavilyAPIKeys        []string
	TavilyBaseURL        string
	TavilyMaxResults     int
	TavilyEnabled        bool
	DuckDuckGoMaxResults int
	DuckDuckGoEnabled    bool
	PerplexityAPIKeys    []string
	PerplexityMaxResults int
	PerplexityEnabled    bool
	SearXNGBaseURL       string
	SearXNGMaxResults    int
	SearXNGEnabled       bool
	GLMSearchAPIKey      string
	GLMSearchBaseURL     string
	GLMSearchEngine      string
	GLMSearchMaxResults  int
	GLMSearchEnabled     bool
	Proxy                string
}

func NewWebSearchTool(opts WebSearchToolOptions) (*WebSearchTool, error) {
	var provider SearchProvider
	maxResults := 5
	// Priority: Perplexity > Brave > SearXNG > Tavily > DuckDuckGo > GLM Search
	if opts.PerplexityEnabled && len(opts.PerplexityAPIKeys) > 0 {
		client, err := createHTTPClient(opts.Proxy, perplexityTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for Perplexity: %w", err)
		}
		provider = &PerplexitySearchProvider{
			keyPool: NewAPIKeyPool(opts.PerplexityAPIKeys),
			proxy:   opts.Proxy,
			client:  client,
		}
		if opts.PerplexityMaxResults > 0 {
			maxResults = opts.PerplexityMaxResults
		}
	} else if opts.BraveEnabled && len(opts.BraveAPIKeys) > 0 {
		client, err := createHTTPClient(opts.Proxy, searchTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for Brave: %w", err)
		}
		provider = &BraveSearchProvider{keyPool: NewAPIKeyPool(opts.BraveAPIKeys), proxy: opts.Proxy, client: client}
		if opts.BraveMaxResults > 0 {
			maxResults = opts.BraveMaxResults
		}
	} else if opts.SearXNGEnabled && opts.SearXNGBaseURL != "" {
		provider = &SearXNGSearchProvider{baseURL: opts.SearXNGBaseURL}
		if opts.SearXNGMaxResults > 0 {
			maxResults = opts.SearXNGMaxResults
		}
	} else if opts.TavilyEnabled && len(opts.TavilyAPIKeys) > 0 {
		client, err := createHTTPClient(opts.Proxy, searchTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for Tavily: %w", err)
		}
		provider = &TavilySearchProvider{
			keyPool: NewAPIKeyPool(opts.TavilyAPIKeys),
			baseURL: opts.TavilyBaseURL,
			proxy:   opts.Proxy,
			client:  client,
		}
		if opts.TavilyMaxResults > 0 {
			maxResults = opts.TavilyMaxResults
		}
	} else if opts.DuckDuckGoEnabled {
		client, err := createHTTPClient(opts.Proxy, searchTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for DuckDuckGo: %w", err)
		}
		provider = &DuckDuckGoSearchProvider{proxy: opts.Proxy, client: client}
		if opts.DuckDuckGoMaxResults > 0 {
			maxResults = opts.DuckDuckGoMaxResults
		}
	} else if opts.GLMSearchEnabled && opts.GLMSearchAPIKey != "" {
		client, err := createHTTPClient(opts.Proxy, searchTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for GLM Search: %w", err)
		}
		searchEngine := opts.GLMSearchEngine
		if searchEngine == "" {
			searchEngine = "search_std"
		}
		provider = &GLMSearchProvider{
			apiKey:       opts.GLMSearchAPIKey,
			baseURL:      opts.GLMSearchBaseURL,
			searchEngine: searchEngine,
			proxy:        opts.Proxy,
			client:       client,
		}
		if opts.GLMSearchMaxResults > 0 {
			maxResults = opts.GLMSearchMaxResults
		}
	} else {
		return nil, nil
	}

	return &WebSearchTool{
		provider:   provider,
		maxResults: maxResults,
	}, nil
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for current information. Returns titles, URLs, and snippets from search results."
}

func (t *WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
			"count": map[string]any{
				"type":        "integer",
				"description": "Number of results (1-10)",
				"minimum":     1.0,
				"maximum":     10.0,
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	query, ok := args["query"].(string)
	if !ok {
		return tools.ErrorResult("query is required")
	}

	count := t.maxResults
	if c, ok := args["count"].(float64); ok {
		if int(c) > 0 && int(c) <= 10 {
			count = int(c)
		}
	}

	result, err := t.provider.Search(ctx, query, count)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	return &tools.ToolResult{
		ForLLM:  result,
		ForUser: result,
	}
}
