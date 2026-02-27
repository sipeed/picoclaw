package web_search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/tools/common"
)

type SearchProvider interface {
	Search(ctx context.Context, query string, count int) (string, error)
}

type BraveSearchProvider struct {
	apiKey string
	proxy  string
}

func (p *BraveSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", p.apiKey)

	client, err := common.CreateHTTPClient(p.proxy, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		// Log error body for debugging
		fmt.Printf("Brave API Error Body: %s\n", string(body))
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	results := searchResp.Web.Results
	if len(results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s", query))
	for i, item := range results {
		if i >= count {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.URL))
		if item.Description != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Description))
		}
	}

	return strings.Join(lines, "\n"), nil
}

type TavilySearchProvider struct {
	apiKey  string
	baseURL string
	proxy   string
}

func (p *TavilySearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := p.baseURL
	if searchURL == "" {
		searchURL = "https://api.tavily.com/search"
	}

	payload := map[string]any{
		"api_key":             p.apiKey,
		"query":               query,
		"search_depth":        "advanced",
		"include_answer":      false,
		"include_images":      false,
		"include_raw_content": false,
		"max_results":         count,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", common.UserAgent)

	client, err := common.CreateHTTPClient(p.proxy, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tavily api error (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	results := searchResp.Results
	if len(results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via Tavily)", query))
	for i, item := range results {
		if i >= count {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.URL))
		if item.Content != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Content))
		}
	}

	return strings.Join(lines, "\n"), nil
}

type DuckDuckGoSearchProvider struct {
	proxy string
}

func (p *DuckDuckGoSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", common.UserAgent)

	client, err := common.CreateHTTPClient(p.proxy, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return p.extractResults(string(body), count, query)
}

func (p *DuckDuckGoSearchProvider) extractResults(html string, count int, query string) (string, error) {
	// Simple regex based extraction for DDG HTML
	// Strategy: Find all result containers or key anchors directly

	// Try finding the result links directly first, as they are the most critical
	// Pattern: <a class="result__a" href="...">Title</a>
	// The previous regex was a bit strict. Let's make it more flexible for attributes order/content
	matches := common.ReDDGLink.FindAllStringSubmatch(html, count+5)

	if len(matches) == 0 {
		return fmt.Sprintf("No results found or extraction failed. Query: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via DuckDuckGo)", query))

	// Pre-compile snippet regex to run inside the loop
	// We'll search for snippets relative to the link position or just globally if needed
	// But simple global search for snippets might mismatch order.
	// Since we only have the raw HTML string, let's just extract snippets globally and assume order matches (risky but simple for regex)
	// Or better: Let's assume the snippet follows the link in the HTML

	// A better regex approach: iterate through text and find matches in order
	// But for now, let's grab all snippets too
	snippetMatches := common.ReDDGSnippet.FindAllStringSubmatch(html, count+5)

	maxItems := min(len(matches), count)

	for i := 0; i < maxItems; i++ {
		urlStr := matches[i][1]
		title := stripTags(matches[i][2])
		title = strings.TrimSpace(title)

		// URL decoding if needed
		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				idx := strings.Index(u, "uddg=")
				if idx != -1 {
					urlStr = u[idx+5:]
				}
			}
		}

		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, title, urlStr))

		// Attempt to attach snippet if available and index aligns
		if i < len(snippetMatches) {
			snippet := stripTags(snippetMatches[i][1])
			snippet = strings.TrimSpace(snippet)
			if snippet != "" {
				lines = append(lines, fmt.Sprintf("   %s", snippet))
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

func stripTags(content string) string {
	return common.ReTags.ReplaceAllString(content, "")
}

type PerplexitySearchProvider struct {
	apiKey string
	proxy  string
}

func (p *PerplexitySearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := "https://api.perplexity.ai/chat/completions"

	payload := map[string]any{
		"model": "sonar",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a search assistant. Provide concise search results with titles, URLs, and brief descriptions in the following format:\n1. Title\n   URL\n   Description\n\nDo not add extra commentary.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Search for: %s. Provide up to %d relevant results.", query, count),
			},
		},
		"max_tokens": 1000,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("User-Agent", common.UserAgent)

	client, err := common.CreateHTTPClient(p.proxy, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Perplexity API error: %s", string(body))
	}

	var searchResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(searchResp.Choices) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	return fmt.Sprintf("Results for: %s (via Perplexity)\n%s", query, searchResp.Choices[0].Message.Content), nil
}

type WebSearchTool struct {
	provider   SearchProvider
	maxResults int
}

type WebSearchToolOptions struct {
	BraveAPIKey          string
	BraveMaxResults      int
	BraveEnabled         bool
	TavilyAPIKey         string
	TavilyBaseURL        string
	TavilyMaxResults     int
	TavilyEnabled        bool
	DuckDuckGoMaxResults int
	DuckDuckGoEnabled    bool
	PerplexityAPIKey     string
	PerplexityMaxResults int
	PerplexityEnabled    bool
	Proxy                string
}

func NewWebSearchTool(opts WebSearchToolOptions) *WebSearchTool {
	var provider SearchProvider
	maxResults := 5

	// Priority: Perplexity > Brave > Tavily > DuckDuckGo
	if opts.PerplexityEnabled && opts.PerplexityAPIKey != "" {
		provider = &PerplexitySearchProvider{apiKey: opts.PerplexityAPIKey, proxy: opts.Proxy}
		if opts.PerplexityMaxResults > 0 {
			maxResults = opts.PerplexityMaxResults
		}
	} else if opts.BraveEnabled && opts.BraveAPIKey != "" {
		provider = &BraveSearchProvider{apiKey: opts.BraveAPIKey, proxy: opts.Proxy}
		if opts.BraveMaxResults > 0 {
			maxResults = opts.BraveMaxResults
		}
	} else if opts.TavilyEnabled && opts.TavilyAPIKey != "" {
		provider = &TavilySearchProvider{
			apiKey:  opts.TavilyAPIKey,
			baseURL: opts.TavilyBaseURL,
			proxy:   opts.Proxy,
		}
		if opts.TavilyMaxResults > 0 {
			maxResults = opts.TavilyMaxResults
		}
	} else if opts.DuckDuckGoEnabled {
		provider = &DuckDuckGoSearchProvider{proxy: opts.Proxy}
		if opts.DuckDuckGoMaxResults > 0 {
			maxResults = opts.DuckDuckGoMaxResults
		}
	} else {
		return nil
	}

	return &WebSearchTool{
		provider:   provider,
		maxResults: maxResults,
	}
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

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) *common.ToolResult {
	query, ok := args["query"].(string)
	if !ok {
		return common.ErrorResult("query is required")
	}

	count := t.maxResults
	if c, ok := args["count"].(float64); ok {
		if int(c) > 0 && int(c) <= 10 {
			count = int(c)
		}
	}

	result, err := t.provider.Search(ctx, query, count)
	if err != nil {
		return common.ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	return &common.ToolResult{
		ForLLM:  result,
		ForUser: result,
	}
}
