package web

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
)

type SearchProvider interface {
	Search(ctx context.Context, query string, count int) (string, error)
}

type BraveSearchProvider struct {
	keyPool *APIKeyPool
	proxy   string
	client  *http.Client
}

func (p *BraveSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), count)

	var lastErr error
	iter := p.keyPool.NewIterator()

	for {
		apiKey, ok := iter.Next()
		if !ok {
			break
		}

		req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Subscription-Token", apiKey)

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
			if resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode == http.StatusUnauthorized ||
				resp.StatusCode == http.StatusForbidden ||
				resp.StatusCode >= 500 {
				continue
			}
			return "", lastErr
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

	return "", fmt.Errorf("all api keys failed, last error: %w", lastErr)
}

type TavilySearchProvider struct {
	keyPool *APIKeyPool
	baseURL string
	proxy   string
	client  *http.Client
}

func (p *TavilySearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := p.baseURL
	if searchURL == "" {
		searchURL = "https://api.tavily.com/search"
	}

	var lastErr error
	iter := p.keyPool.NewIterator()

	for {
		apiKey, ok := iter.Next()
		if !ok {
			break
		}

		payload := map[string]any{
			"api_key":             apiKey,
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
		req.Header.Set("User-Agent", userAgent)

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("tavily api error (status %d): %s", resp.StatusCode, string(body))
			if resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode == http.StatusUnauthorized ||
				resp.StatusCode == http.StatusForbidden ||
				resp.StatusCode >= 500 {
				continue
			}
			return "", lastErr
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

	return "", fmt.Errorf("all api keys failed, last error: %w", lastErr)
}

type DuckDuckGoSearchProvider struct {
	proxy  string
	client *http.Client
}

func (p *DuckDuckGoSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
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
	matches := reDDGLink.FindAllStringSubmatch(html, count+5)

	if len(matches) == 0 {
		return fmt.Sprintf("No results found or extraction failed. Query: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via DuckDuckGo)", query))

	snippetMatches := reDDGSnippet.FindAllStringSubmatch(html, count+5)

	maxItems := min(len(matches), count)

	for i := range maxItems {
		urlStr := matches[i][1]
		title := stripTags(matches[i][2])
		title = strings.TrimSpace(title)

		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				_, after, ok := strings.Cut(u, "uddg=")
				if ok {
					urlStr = after
				}
			}
		}

		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, title, urlStr))

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

type PerplexitySearchProvider struct {
	keyPool *APIKeyPool
	proxy   string
	client  *http.Client
}

func (p *PerplexitySearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := "https://api.perplexity.ai/chat/completions"

	var lastErr error
	iter := p.keyPool.NewIterator()

	for {
		apiKey, ok := iter.Next()
		if !ok {
			break
		}

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
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("User-Agent", userAgent)

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("Perplexity API error: %s", string(body))
			if resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode == http.StatusUnauthorized ||
				resp.StatusCode == http.StatusForbidden ||
				resp.StatusCode >= 500 {
				continue
			}
			return "", lastErr
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

	return "", fmt.Errorf("all api keys failed, last error: %w", lastErr)
}

type SearXNGSearchProvider struct {
	baseURL string
}

func (p *SearXNGSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=general",
		strings.TrimSuffix(p.baseURL, "/"),
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SearXNG returned status %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Engine  string  `json:"engine"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	if len(result.Results) > count {
		result.Results = result.Results[:count]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Results for: %s (via SearXNG)\n", query))
	for i, r := range result.Results {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		b.WriteString(fmt.Sprintf("   %s\n", r.URL))
		if r.Content != "" {
			b.WriteString(fmt.Sprintf("   %s\n", r.Content))
		}
	}

	return b.String(), nil
}

type GLMSearchProvider struct {
	apiKey       string
	baseURL      string
	searchEngine string
	proxy        string
	client       *http.Client
}

func (p *GLMSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := p.baseURL
	if searchURL == "" {
		searchURL = "https://open.bigmodel.cn/api/paas/v4/web_search"
	}

	payload := map[string]any{
		"search_query":  query,
		"search_engine": p.searchEngine,
		"search_intent": false,
		"count":         count,
		"content_size":  "medium",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GLM Search API error (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		SearchResult []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			Link    string `json:"link"`
		} `json:"search_result"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	results := searchResp.SearchResult
	if len(results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via GLM Search)", query))
	for i, item := range results {
		if i >= count {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.Link))
		if item.Content != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Content))
		}
	}

	return strings.Join(lines, "\n"), nil
}
