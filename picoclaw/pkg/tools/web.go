package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	userAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	userAgentHonest = "picoclaw/%s (+https://github.com/sipeed/picoclaw; AI assistant bot)"

	// HTTP client timeouts for web tool providers.
	searchTimeout     = 10 * time.Second // Brave, Tavily, DuckDuckGo
	perplexityTimeout = 30 * time.Second // Perplexity (LLM-based, slower)
	fetchTimeout      = 60 * time.Second // WebFetchTool

	defaultMaxChars = 50000
	maxRedirects    = 5
)

// Pre-compiled regexes for HTML text extraction
var (
	reScript     = regexp.MustCompile(`<script[\s\S]*?</script>`)
	reStyle      = regexp.MustCompile(`<style[\s\S]*?</style>`)
	reTags       = regexp.MustCompile(`<[^>]+>`)
	reWhitespace = regexp.MustCompile(`[^\S\n]+`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)

	// DuckDuckGo result extraction
	reDDGLink = regexp.MustCompile(
		`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`,
	)
	reDDGSnippet = regexp.MustCompile(`<a class="result__snippet[^"]*".*?>([\s\S]*?)</a>`)
)

type APIKeyPool struct {
	keys    []string
	current uint32
}

func NewAPIKeyPool(keys []string) *APIKeyPool {
	return &APIKeyPool{
		keys: keys,
	}
}

type APIKeyIterator struct {
	pool     *APIKeyPool
	startIdx uint32
	attempt  uint32
}

func (p *APIKeyPool) NewIterator() *APIKeyIterator {
	if len(p.keys) == 0 {
		return &APIKeyIterator{pool: p}
	}
	idx := atomic.AddUint32(&p.current, 1) - 1
	return &APIKeyIterator{
		pool:     p,
		startIdx: idx,
	}
}

func (it *APIKeyIterator) Next() (string, bool) {
	length := uint32(len(it.pool.keys))
	if length == 0 || it.attempt >= length {
		return "", false
	}
	key := it.pool.keys[(it.startIdx+it.attempt)%length]
	it.attempt++
	return key, true
}

type SearchProvider interface {
	Search(ctx context.Context, query string, count int, rangeCode string) (string, error)
}

func normalizeSearchRange(raw string) (string, error) {
	rangeCode := strings.ToLower(strings.TrimSpace(raw))
	switch rangeCode {
	case "", "d", "w", "m", "y":
		return rangeCode, nil
	default:
		return "", fmt.Errorf("range must be one of: d, w, m, y")
	}
}

func mapBraveFreshness(rangeCode string) string {
	switch rangeCode {
	case "d":
		return "pd"
	case "w":
		return "pw"
	case "m":
		return "pm"
	case "y":
		return "py"
	default:
		return ""
	}
}

func mapTavilyTimeRange(rangeCode string) string {
	switch rangeCode {
	case "d":
		return "day"
	case "w":
		return "week"
	case "m":
		return "month"
	case "y":
		return "year"
	default:
		return ""
	}
}

func mapPerplexityRecencyFilter(rangeCode string) string {
	switch rangeCode {
	case "d":
		return "day"
	case "w":
		return "week"
	case "m":
		return "month"
	case "y":
		return "year"
	default:
		return ""
	}
}

func mapDuckDuckGoDateFilter(rangeCode string) string {
	switch rangeCode {
	case "d":
		return "d"
	case "w":
		return "w"
	case "m":
		return "m"
	case "y":
		return "t"
	default:
		return ""
	}
}

func mapSearXNGTimeRange(rangeCode string) string {
	switch rangeCode {
	case "d":
		return "day"
	case "w":
		return "week"
	case "m":
		return "month"
	case "y":
		return "year"
	default:
		return ""
	}
}

func mapGLMRecencyFilter(rangeCode string) string {
	switch rangeCode {
	case "d":
		return "oneDay"
	case "w":
		return "oneWeek"
	case "m":
		return "oneMonth"
	case "y":
		return "oneYear"
	default:
		return "noLimit"
	}
}

func mapBaiduRecencyFilter(rangeCode string) string {
	switch rangeCode {
	case "d", "w":
		// Baidu does not expose a day-level filter. Use the closest supported
		// window to keep recency bias instead of silently dropping the filter.
		return "week"
	case "m":
		return "month"
	case "y":
		return "year"
	default:
		return ""
	}
}

type BraveSearchProvider struct {
	keyPool *APIKeyPool
	proxy   string
	client  *http.Client
}

func (p *BraveSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), count)
	if freshness := mapBraveFreshness(rangeCode); freshness != "" {
		searchURL += "&freshness=" + url.QueryEscape(freshness)
	}

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

func (p *TavilySearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
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
		if timeRange := mapTavilyTimeRange(rangeCode); timeRange != "" {
			payload["time_range"] = timeRange
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

func (p *DuckDuckGoSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	if dateFilter := mapDuckDuckGoDateFilter(rangeCode); dateFilter != "" {
		searchURL += "&df=" + url.QueryEscape(dateFilter)
	}

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

func (p *DuckDuckGoSearchProvider) extractResults(
	html string,
	count int,
	query string,
) (string, error) {
	// Simple regex based extraction for DDG HTML
	// Strategy: Find all result containers or key anchors directly

	// Try finding the result links directly first, as they are the most critical
	// Pattern: <a class="result__a" href="...">Title</a>
	// The previous regex was a bit strict. Let's make it more flexible for attributes order/content
	matches := reDDGLink.FindAllStringSubmatch(html, count+5)

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
	snippetMatches := reDDGSnippet.FindAllStringSubmatch(html, count+5)

	maxItems := min(len(matches), count)

	for i := range maxItems {
		urlStr := matches[i][1]
		title := stripTags(matches[i][2])
		title = strings.TrimSpace(title)

		// URL decoding if needed
		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				_, after, ok := strings.Cut(u, "uddg=")
				if ok {
					urlStr = after
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
	return reTags.ReplaceAllString(content, "")
}

type PerplexitySearchProvider struct {
	keyPool *APIKeyPool
	proxy   string
	client  *http.Client
}

func (p *PerplexitySearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
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
					"role": "user",
					"content": fmt.Sprintf(
						"Search for: %s. Provide up to %d relevant results.",
						query,
						count,
					),
				},
			},
			"max_tokens": 1000,
		}
		if recencyFilter := mapPerplexityRecencyFilter(rangeCode); recencyFilter != "" {
			payload["search_recency_filter"] = recencyFilter
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(
			ctx,
			"POST",
			searchURL,
			strings.NewReader(string(payloadBytes)),
		)
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

		return fmt.Sprintf(
			"Results for: %s (via Perplexity)\n%s",
			query,
			searchResp.Choices[0].Message.Content,
		), nil
	}

	return "", fmt.Errorf("all api keys failed, last error: %w", lastErr)
}

type SearXNGSearchProvider struct {
	baseURL string
}

func (p *SearXNGSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=general",
		strings.TrimSuffix(p.baseURL, "/"),
		url.QueryEscape(query))
	if timeRange := mapSearXNGTimeRange(rangeCode); timeRange != "" {
		searchURL += "&time_range=" + url.QueryEscape(timeRange)
	}

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

	// Limit results to requested count
	if len(result.Results) > count {
		result.Results = result.Results[:count]
	}

	// Format results in standard PicoClaw format
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

func (p *GLMSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
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
	if recencyFilter := mapGLMRecencyFilter(rangeCode); recencyFilter != "" {
		payload["search_recency_filter"] = recencyFilter
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

type BaiduSearchProvider struct {
	apiKey  string
	baseURL string
	proxy   string
	client  *http.Client
}

func (p *BaiduSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
	searchURL := p.baseURL
	if searchURL == "" {
		searchURL = "https://qianfan.baidubce.com/v2/ai_search/web_search"
	}

	payload := map[string]any{
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": query,
			},
		},
		"search_source":        "baidu_search_v2",
		"resource_type_filter": []map[string]any{{"type": "web", "top_k": count}},
	}
	if recencyFilter := mapBaiduRecencyFilter(rangeCode); recencyFilter != "" {
		payload["search_recency_filter"] = recencyFilter
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
		return "", fmt.Errorf("baidu search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("baidu search API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		References []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"references"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.References) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	lines := []string{fmt.Sprintf("Results for: %s (via Baidu Search)", query)}
	for i, item := range result.References {
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

type WebSearchTool struct {
	provider   SearchProvider
	maxResults int
}

type WebSearchToolOptions struct {
	BraveAPIKeys          []string
	BraveMaxResults       int
	BraveEnabled          bool
	TavilyAPIKeys         []string
	TavilyBaseURL         string
	TavilyMaxResults      int
	TavilyEnabled         bool
	DuckDuckGoMaxResults  int
	DuckDuckGoEnabled     bool
	PerplexityAPIKeys     []string
	PerplexityMaxResults  int
	PerplexityEnabled     bool
	SearXNGBaseURL        string
	SearXNGMaxResults     int
	SearXNGEnabled        bool
	GLMSearchAPIKey       string
	GLMSearchBaseURL      string
	GLMSearchEngine       string
	GLMSearchMaxResults   int
	GLMSearchEnabled      bool
	BaiduSearchAPIKey     string
	BaiduSearchBaseURL    string
	BaiduSearchMaxResults int
	BaiduSearchEnabled    bool
	Proxy                 string
}

func NewWebSearchTool(opts WebSearchToolOptions) (*WebSearchTool, error) {
	var provider SearchProvider
	maxResults := 10
	// Priority: Perplexity > Brave > SearXNG > Tavily > DuckDuckGo > Baidu Search > GLM Search
	if opts.PerplexityEnabled && len(opts.PerplexityAPIKeys) > 0 {
		client, err := utils.CreateHTTPClient(opts.Proxy, perplexityTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for Perplexity: %w", err)
		}
		provider = &PerplexitySearchProvider{
			keyPool: NewAPIKeyPool(opts.PerplexityAPIKeys),
			proxy:   opts.Proxy,
			client:  client,
		}
		if opts.PerplexityMaxResults > 0 {
			maxResults = min(opts.PerplexityMaxResults, 10)
		}
	} else if opts.BraveEnabled && len(opts.BraveAPIKeys) > 0 {
		client, err := utils.CreateHTTPClient(opts.Proxy, searchTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for Brave: %w", err)
		}
		provider = &BraveSearchProvider{keyPool: NewAPIKeyPool(opts.BraveAPIKeys), proxy: opts.Proxy, client: client}
		if opts.BraveMaxResults > 0 {
			maxResults = min(opts.BraveMaxResults, 10)
		}
	} else if opts.SearXNGEnabled && opts.SearXNGBaseURL != "" {
		provider = &SearXNGSearchProvider{baseURL: opts.SearXNGBaseURL}
		if opts.SearXNGMaxResults > 0 {
			maxResults = min(opts.SearXNGMaxResults, 10)
		}
	} else if opts.TavilyEnabled && len(opts.TavilyAPIKeys) > 0 {
		client, err := utils.CreateHTTPClient(opts.Proxy, searchTimeout)
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
			maxResults = min(opts.TavilyMaxResults, 10)
		}
	} else if opts.DuckDuckGoEnabled {
		client, err := utils.CreateHTTPClient(opts.Proxy, searchTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for DuckDuckGo: %w", err)
		}
		provider = &DuckDuckGoSearchProvider{proxy: opts.Proxy, client: client}
		if opts.DuckDuckGoMaxResults > 0 {
			maxResults = min(opts.DuckDuckGoMaxResults, 10)
		}
	} else if opts.BaiduSearchEnabled && opts.BaiduSearchAPIKey != "" {
		client, err := utils.CreateHTTPClient(opts.Proxy, perplexityTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client for Baidu Search: %w", err)
		}
		provider = &BaiduSearchProvider{
			apiKey:  opts.BaiduSearchAPIKey,
			baseURL: opts.BaiduSearchBaseURL,
			proxy:   opts.Proxy,
			client:  client,
		}
		if opts.BaiduSearchMaxResults > 0 {
			maxResults = min(opts.BaiduSearchMaxResults, 10)
		}
	} else if opts.GLMSearchEnabled && opts.GLMSearchAPIKey != "" {
		client, err := utils.CreateHTTPClient(opts.Proxy, searchTimeout)
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
			maxResults = min(opts.GLMSearchMaxResults, 10)
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
	return "Search the web for current information. Supports query, count, and an optional temporal range filter. Returns titles, URLs, and snippets from search results."
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
				"description": "Number of results (default: 10, max: 10)",
				"minimum":     1.0,
				"maximum":     10.0,
			},
			"range": map[string]any{
				"type":        "string",
				"description": "Optional time filter: d (day), w (week), m (month), y (year)",
				"enum":        []string{"d", "w", "m", "y"},
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return ErrorResult("query is required")
	}
	query = strings.TrimSpace(query)

	count64, err := getInt64Arg(args, "count", int64(t.maxResults))
	if err != nil {
		return ErrorResult(err.Error())
	}
	count := t.maxResults
	if count64 > 0 && count64 <= 10 {
		count = int(count64)
	}

	rangeCode, err := normalizeSearchRange("")
	if err != nil {
		return ErrorResult(err.Error())
	}
	if rawRange, exists := args["range"]; exists {
		rangeStr, ok := rawRange.(string)
		if !ok {
			return ErrorResult("range must be a string")
		}
		rangeCode, err = normalizeSearchRange(rangeStr)
		if err != nil {
			return ErrorResult(err.Error())
		}
	}

	result, err := t.provider.Search(ctx, query, count, rangeCode)
	if err != nil {
		return ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	return &ToolResult{
		ForLLM:  result,
		ForUser: result,
	}
}

type WebFetchTool struct {
	maxChars        int
	proxy           string
	client          *http.Client
	format          string
	fetchLimitBytes int64
	whitelist       *privateHostWhitelist
}

type privateHostWhitelist struct {
	exact map[string]struct{}
	cidrs []*net.IPNet
}

func NewWebFetchTool(maxChars int, format string, fetchLimitBytes int64) (*WebFetchTool, error) {
	// createHTTPClient cannot fail with an empty proxy string.
	return NewWebFetchToolWithConfig(maxChars, "", format, fetchLimitBytes, nil)
}

// allowPrivateWebFetchHosts controls whether loopback/private hosts are allowed.
// This is false in normal runtime to reduce SSRF exposure, and tests can override it temporarily.
var allowPrivateWebFetchHosts atomic.Bool

func NewWebFetchToolWithProxy(
	maxChars int,
	proxy string,
	format string,
	fetchLimitBytes int64,
	privateHostWhitelist []string,
) (*WebFetchTool, error) {
	return NewWebFetchToolWithConfig(maxChars, proxy, format, fetchLimitBytes, privateHostWhitelist)
}

func NewWebFetchToolWithConfig(
	maxChars int,
	proxy string,
	format string,
	fetchLimitBytes int64,
	privateHostWhitelist []string,
) (*WebFetchTool, error) {
	if maxChars <= 0 {
		maxChars = defaultMaxChars
	}
	whitelist, err := newPrivateHostWhitelist(privateHostWhitelist)
	if err != nil {
		return nil, fmt.Errorf("failed to parse web fetch private host whitelist: %w", err)
	}
	client, err := utils.CreateHTTPClient(proxy, fetchTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for web fetch: %w", err)
	}
	if transport, ok := client.Transport.(*http.Transport); ok {
		dialer := &net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		transport.DialContext = newSafeDialContext(dialer, whitelist)
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("stopped after %d redirects", maxRedirects)
		}
		if isObviousPrivateHost(req.URL.Hostname(), whitelist) {
			return fmt.Errorf("redirect target is private or local network host")
		}
		return nil
	}
	if fetchLimitBytes <= 0 {
		fetchLimitBytes = 10 * 1024 * 1024 // Security Fallback
	}
	return &WebFetchTool{
		maxChars:        maxChars,
		proxy:           proxy,
		client:          client,
		format:          format,
		fetchLimitBytes: fetchLimitBytes,
		whitelist:       whitelist,
	}, nil
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch a URL and extract readable content (HTML to text). Use this to get weather info, news, articles, or any web content."
}

func (t *WebFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "URL to fetch",
			},
			"maxChars": map[string]any{
				"type":        "integer",
				"description": "Maximum characters to extract",
				"minimum":     100.0,
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	urlStr, ok := args["url"].(string)
	if !ok {
		return ErrorResult("url is required")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid URL: %v", err))
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrorResult("only http/https URLs are allowed")
	}

	if parsedURL.Host == "" {
		return ErrorResult("missing domain in URL")
	}

	// Lightweight pre-flight: block obvious localhost/literal-IP without DNS resolution.
	// The real SSRF guard is newSafeDialContext at connect time.
	hostname := parsedURL.Hostname()
	if isObviousPrivateHost(hostname, t.whitelist) {
		return ErrorResult("fetching private or local network hosts is not allowed")
	}

	maxChars := t.maxChars
	if mc, ok := args["maxChars"].(float64); ok {
		if int(mc) > 100 {
			maxChars = int(mc)
		}
	}

	doFetch := func(ua string) (*http.Response, []byte, error) {
		req, reqErr := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if reqErr != nil {
			return nil, nil, fmt.Errorf("failed to create request: %w", reqErr)
		}
		req.Header.Set("User-Agent", ua)
		resp, doErr := t.client.Do(req)
		if doErr != nil {
			return nil, nil, fmt.Errorf("request failed: %w", doErr)
		}
		resp.Body = http.MaxBytesReader(nil, resp.Body, t.fetchLimitBytes)

		b, readErr := io.ReadAll(resp.Body)
		return resp, b, readErr
	}

	resp, body, err := doFetch(userAgent)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return ErrorResult(
				fmt.Sprintf(
					"failed to read response: size exceeded %d bytes limit",
					t.fetchLimitBytes,
				),
			)
		}
		return ErrorResult(err.Error())
	}

	// Cloudflare (and similar WAFs) signal bot challenges with 403 + cf-mitigated: challenge.
	// Retry once with an honest User-Agent that identifies picoclaw, which some
	// operators explicitly allow-list for AI assistants.
	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("Cf-Mitigated") == "challenge" {
		logger.DebugCF("tool", "Cloudflare challenge detected, retrying with honest User-Agent",
			map[string]any{"url": urlStr})
		honestUA := fmt.Sprintf(userAgentHonest, config.Version)
		resp2, body2, err2 := doFetch(honestUA)
		if resp2 != nil && resp2.Body != nil {
			defer resp2.Body.Close()
		}

		if err2 == nil {
			resp, body = resp2, body2
		} else {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err2, &maxBytesErr) {
				return ErrorResult(
					fmt.Sprintf("failed to read response: size exceeded %d bytes limit", t.fetchLimitBytes),
				)
			}
			return ErrorResult(err2.Error())
		}
	}

	bodyStr := string(body)
	contentType := resp.Header.Get("Content-Type")

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// The most common error here is "mime: no media type" if the header is empty.
		logger.WarnCF("tool", "Failed to parse Content-Type", map[string]any{
			"raw_header": contentType,
			"error":      err.Error(),
		})

		// security fallback
		mediaType = "application/octet-stream"
	}

	charset, hasCharset := params["charset"]
	if hasCharset {
		// If the charset is not utf-8, we might have to convert the bodyStr
		// before passing it to the HTML/Markdown parser
		if strings.ToLower(charset) != "utf-8" {
			logger.WarnCF(
				"tool",
				"Note: the content is not in UTF-8",
				map[string]any{"charset": charset},
			)
		}
	}

	var text, extractor string

	switch {
	case mediaType == "application/json":
		var jsonData any
		if err := json.Unmarshal(body, &jsonData); err != nil {
			text = bodyStr
			extractor = "raw"
			break
		}

		formatted, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			text = bodyStr
			extractor = "raw"
			break
		}

		text = string(formatted)
		extractor = "json"

	case mediaType == "text/html" || looksLikeHTML(bodyStr):
		switch strings.ToLower(t.format) {
		case "markdown":
			var err error
			text, err = utils.HtmlToMarkdown(bodyStr)
			if err != nil {
				return ErrorResult(fmt.Sprintf("failed to HTML to markdown: %v", err))
			}
			extractor = "markdown"

		default:
			text = t.extractText(bodyStr)
			extractor = "text"
		}

	default:
		text = bodyStr
		extractor = "raw"
	}

	truncated := len(text) > maxChars
	if truncated {
		text = text[:maxChars] + "\n[Content truncated due to size limit]"
	}

	result := map[string]any{
		"url":       urlStr,
		"status":    resp.StatusCode,
		"extractor": extractor,
		"truncated": truncated,
		"length":    len(text),
		"text":      text,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &ToolResult{
		ForLLM: string(resultJSON),
		ForUser: fmt.Sprintf(
			"Fetched %d bytes from %s (extractor: %s, truncated: %v)",
			len(text),
			urlStr,
			extractor,
			truncated,
		),
	}
}

func looksLikeHTML(body string) bool {
	if body == "" {
		return false
	}

	lower := strings.ToLower(body)

	return strings.HasPrefix(body, "<!doctype") ||
		strings.HasPrefix(lower, "<html")
}

func (t *WebFetchTool) extractText(htmlContent string) string {
	result := reScript.ReplaceAllLiteralString(htmlContent, "")
	result = reStyle.ReplaceAllLiteralString(result, "")
	result = reTags.ReplaceAllLiteralString(result, "")

	result = strings.TrimSpace(result)

	result = reWhitespace.ReplaceAllString(result, " ")
	result = reBlankLines.ReplaceAllString(result, "\n\n")

	lines := strings.Split(result, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// newSafeDialContext re-resolves DNS at connect time to mitigate DNS rebinding (TOCTOU)
// where a hostname resolves to a public IP during pre-flight but a private IP at connect time.
func newSafeDialContext(
	dialer *net.Dialer,
	whitelist *privateHostWhitelist,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if allowPrivateWebFetchHosts.Load() {
			return dialer.DialContext(ctx, network, address)
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid target address %q: %w", address, err)
		}
		if host == "" {
			return nil, fmt.Errorf("empty target host")
		}

		if ip := net.ParseIP(host); ip != nil {
			if shouldBlockPrivateIP(ip, whitelist) {
				return nil, fmt.Errorf("blocked private or local target: %s", host)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}

		ipAddrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", host, err)
		}

		attempted := 0
		var lastErr error
		for _, ipAddr := range ipAddrs {
			if shouldBlockPrivateIP(ipAddr.IP, whitelist) {
				continue
			}
			attempted++
			conn, err := dialer.DialContext(
				ctx,
				network,
				net.JoinHostPort(ipAddr.IP.String(), port),
			)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}

		if attempted == 0 {
			return nil, fmt.Errorf(
				"all resolved addresses for %s are private, restricted, or not whitelisted",
				host,
			)
		}
		if lastErr != nil {
			return nil, fmt.Errorf(
				"failed connecting to public addresses for %s: %w",
				host,
				lastErr,
			)
		}
		return nil, fmt.Errorf("failed connecting to public addresses for %s", host)
	}
}

func newPrivateHostWhitelist(entries []string) (*privateHostWhitelist, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	whitelist := &privateHostWhitelist{
		exact: make(map[string]struct{}),
		cidrs: make([]*net.IPNet, 0, len(entries)),
	}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			whitelist.exact[normalizeWhitelistIP(ip).String()] = struct{}{}
			continue
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid entry %q: expected IP or CIDR", entry)
		}
		whitelist.cidrs = append(whitelist.cidrs, network)
	}

	if len(whitelist.exact) == 0 && len(whitelist.cidrs) == 0 {
		return nil, nil
	}
	return whitelist, nil
}

func (w *privateHostWhitelist) Contains(ip net.IP) bool {
	if w == nil || ip == nil {
		return false
	}

	normalized := normalizeWhitelistIP(ip)
	if _, ok := w.exact[normalized.String()]; ok {
		return true
	}
	for _, network := range w.cidrs {
		if network.Contains(normalized) {
			return true
		}
	}
	return false
}

func normalizeWhitelistIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4
	}
	return ip
}

func shouldBlockPrivateIP(ip net.IP, whitelist *privateHostWhitelist) bool {
	return isPrivateOrRestrictedIP(ip) && !whitelist.Contains(ip)
}

// isObviousPrivateHost performs a lightweight, no-DNS check for obviously private hosts.
// It catches localhost, literal private IPs, and empty hosts. It does NOT resolve DNS —
// the real SSRF guard is newSafeDialContext which checks IPs at connect time.
func isObviousPrivateHost(host string, whitelist *privateHostWhitelist) bool {
	if allowPrivateWebFetchHosts.Load() {
		return false
	}

	h := strings.ToLower(strings.TrimSpace(host))
	h = strings.TrimSuffix(h, ".")
	if h == "" {
		return true
	}

	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}

	if ip := net.ParseIP(h); ip != nil {
		return shouldBlockPrivateIP(ip, whitelist)
	}

	return false
}

// isPrivateOrRestrictedIP returns true for IPs that should never be reached via web_fetch:
// RFC 1918, loopback, link-local (incl. cloud metadata 169.254.x.x), carrier-grade NAT,
// IPv6 unique-local (fc00::/7), 6to4 (2002::/16), and Teredo (2001:0000::/32).
func isPrivateOrRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		// IPv4 private, loopback, link-local, and carrier-grade NAT ranges.
		if ip4[0] == 10 ||
			ip4[0] == 127 ||
			ip4[0] == 0 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			(ip4[0] == 169 && ip4[1] == 254) ||
			(ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127) {
			return true
		}
		return false
	}

	if len(ip) == net.IPv6len {
		// IPv6 unique local addresses (fc00::/7)
		if (ip[0] & 0xfe) == 0xfc {
			return true
		}
		// 6to4 addresses (2002::/16): check the embedded IPv4 at bytes [2:6].
		if ip[0] == 0x20 && ip[1] == 0x02 {
			embedded := net.IPv4(ip[2], ip[3], ip[4], ip[5])
			return isPrivateOrRestrictedIP(embedded)
		}
		// Teredo (2001:0000::/32): client IPv4 is at bytes [12:16], XOR-inverted.
		if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x00 && ip[3] == 0x00 {
			client := net.IPv4(ip[12]^0xff, ip[13]^0xff, ip[14]^0xff, ip[15]^0xff)
			return isPrivateOrRestrictedIP(client)
		}
	}

	return false
}
