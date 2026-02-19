package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type SearchProvider interface {
	Search(ctx context.Context, query string, count int) (string, error)
}

type BraveSearchProvider struct {
	apiKey string
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

	client := &http.Client{Timeout: 10 * time.Second}
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

type DuckDuckGoSearchProvider struct{}

func (p *DuckDuckGoSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 10 * time.Second}
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
	reLink := regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	matches := reLink.FindAllStringSubmatch(html, count+5)

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
	reSnippet := regexp.MustCompile(`<a class="result__snippet[^"]*".*?>([\s\S]*?)</a>`)
	snippetMatches := reSnippet.FindAllStringSubmatch(html, count+5)

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
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(content, "")
}

type OllamaSearchProvider struct {
	baseURL    string
	apiKey     string
	queryParam string // Parameter name for query (e.g., "query", "q")
}

func (p *OllamaSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	// Call Ollama REST API for web search
	requestBody := map[string]interface{}{
		p.queryParam: query,
	}

	bodyJSON, _ := json.Marshal(requestBody)

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to Ollama failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("tool", "Ollama web search request failed",
			map[string]interface{}{
				"provider":    "ollama",
				"status_code": resp.StatusCode,
				"endpoint":    p.baseURL,
				"response":    strings.TrimSpace(string(body)),
			})
		return "", fmt.Errorf("Ollama API error: %s", string(body))
	}

	// Return the raw JSON string for the agent/tool to process
	return string(body), nil
}

type PerplexitySearchProvider struct {
	apiKey string
}

func (p *PerplexitySearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := "https://api.perplexity.ai/chat/completions"

	payload := map[string]interface{}{
		"model": "sonar",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a search assistant. Provide concise search results with titles, URLs, and brief descriptions in the following format:\n1. Title\n   URL\n   Description\n\nDo not add extra commentary."},
			{"role": "user", "content": fmt.Sprintf("Search for: %s. Provide up to %d relevant results.", query, count)},
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
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
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
	fallback   SearchProvider
	maxResults int
}

// WebSearchToolOptions defines configuration for web search providers.
// This is the unified API for all search providers.
//
// If Provider is set, it's considered enabled. Use empty Provider to disable.
// BaseURL is generic and works for any provider (e.g., custom Ollama instances).
// Model is not needed for web search - results are translated by the main LLM.
//
// Usage example:
//
//	opts := []WebSearchToolOptions{
//	    {Provider: "brave", APIKey: "key", MaxResults: 5},
//	    {Provider: "ollama", BaseURL: "http://localhost:11434", MaxResults: 5},
//	    {Provider: "perplexity", APIKey: "key", MaxResults: 5},
//	    {Provider: "duckduckgo", MaxResults: 5},
//	}
//	tool := NewWebSearchTool(opts...)
type WebSearchToolOptions struct {
	Provider   string // "brave", "ollama", "perplexity", "duckduckgo"
	APIKey     string // For Brave API
	BaseURL    string // For custom providers (e.g., Ollama)
	MaxResults int    // Default: 5
	Mode       string // "GET" or "POST" (reserved for future use)
	Param      string // Query param name: "q", "query", etc. (reserved for future use)
}

func NewWebSearchTool(opts ...WebSearchToolOptions) *WebSearchTool {
	// Priority order: Brave > Ollama > Perplexity > DuckDuckGo
	priorityOrder := []string{"brave", "ollama", "perplexity", "duckduckgo"}
	optMap := make(map[string]WebSearchToolOptions)

	// Build map of enabled providers
	for _, opt := range opts {
		if opt.Provider != "" {
			optMap[opt.Provider] = opt
		}
	}

	var selectedOpt *WebSearchToolOptions
	var provider SearchProvider

	// Try providers in priority order
	for _, providerName := range priorityOrder {
		opt, exists := optMap[providerName]
		if !exists {
			continue
		}

		selectedOpt = &opt
		var err error
		provider, err = createProvider(&opt)
		if err == nil && provider != nil {
			break
		}
	}

	if provider == nil {
		return nil
	}

	var fallbackProvider SearchProvider
	if selectedOpt != nil && selectedOpt.Provider != "duckduckgo" {
		if ddgOpt, exists := optMap["duckduckgo"]; exists {
			if p, err := createProvider(&ddgOpt); err == nil && p != nil {
				fallbackProvider = p
			}
		}
	}

	maxResults := 5
	if selectedOpt != nil && selectedOpt.MaxResults > 0 {
		maxResults = selectedOpt.MaxResults
	}

	return &WebSearchTool{
		provider:   provider,
		fallback:   fallbackProvider,
		maxResults: maxResults,
	}
}

// createProvider creates a SearchProvider based on configuration
// Provider is considered enabled if non-empty
func createProvider(opt *WebSearchToolOptions) (SearchProvider, error) {
	if opt == nil || opt.Provider == "" {
		return nil, fmt.Errorf("provider not set")
	}

	switch opt.Provider {
	case "brave":
		if opt.APIKey == "" {
			return nil, fmt.Errorf("Brave API key required")
		}
		return &BraveSearchProvider{apiKey: opt.APIKey}, nil

	case "ollama":
		if opt.BaseURL == "" {
			return nil, fmt.Errorf("Ollama BaseURL required")
		}
		queryParam := opt.Param
		if queryParam == "" {
			queryParam = "query" // Default query parameter
		}
		return &OllamaSearchProvider{
			baseURL:    opt.BaseURL,
			apiKey:     opt.APIKey, // Bearer token for API authentication
			queryParam: queryParam,
		}, nil

	case "duckduckgo":
		return &DuckDuckGoSearchProvider{}, nil

	case "perplexity":
		if opt.APIKey == "" {
			return nil, fmt.Errorf("Perplexity API key required")
		}
		return &PerplexitySearchProvider{apiKey: opt.APIKey}, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", opt.Provider)
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for current information. Returns titles, URLs, and snippets from search results."
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query",
			},
			"count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results (1-10)",
				"minimum":     1.0,
				"maximum":     10.0,
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	query, ok := args["query"].(string)
	if !ok {
		return ErrorResult("query is required")
	}

	count := t.maxResults
	if c, ok := args["count"].(float64); ok {
		if int(c) > 0 && int(c) <= 10 {
			count = int(c)
		}
	}

	result, err := t.provider.Search(ctx, query, count)
	if err != nil {
		if t.fallback != nil {
			logger.WarnCF("tool", "Primary web search failed, attempting fallback",
				map[string]interface{}{
					"primary_error": err.Error(),
					"fallback":      "duckduckgo",
					"query":         query,
				})

			fallbackResult, fallbackErr := t.fallback.Search(ctx, query, count)
			if fallbackErr == nil {
				logger.InfoCF("tool", "Fallback web search succeeded",
					map[string]interface{}{
						"provider": "duckduckgo",
						"query":    query,
					})

				return &ToolResult{
					ForLLM:  fmt.Sprintf("Primary search provider failed: %v\nFallback provider (duckduckgo) result:\n%s", err, fallbackResult),
					ForUser: fallbackResult,
				}
			}

			logger.ErrorCF("tool", "Fallback web search failed",
				map[string]interface{}{
					"provider": "duckduckgo",
					"error":    fallbackErr.Error(),
					"query":    query,
				})
		}
		return ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	// If Ollama, synthesize a readable answer for the user
	if _, ok := t.provider.(*OllamaSearchProvider); ok {
		var parsed struct {
			Results []struct {
				Title   string `json:"title"`
				URL     string `json:"url"`
				Content string `json:"content"`
			} `json:"results"`
		}
		if err := json.Unmarshal([]byte(result), &parsed); err == nil && len(parsed.Results) > 0 {
			// Synthesize a readable answer
			var summary []string
			summary = append(summary, fmt.Sprintf("Web search summary for: %s", query))
			maxItems := count
			if len(parsed.Results) < count {
				maxItems = len(parsed.Results)
			}
			for i := 0; i < maxItems; i++ {
				r := parsed.Results[i]
				summary = append(summary, fmt.Sprintf("%d. %s\n   %s", i+1, r.Title, r.URL))
				if r.Content != "" {
					summary = append(summary, fmt.Sprintf("   %s", r.Content))
				}
			}
			return &ToolResult{
				ForLLM:  result, // raw for LLM
				ForUser: strings.Join(summary, "\n"),
			}
		}
	}

	// Default: return as-is
	return &ToolResult{
		ForLLM:  result,
		ForUser: result,
	}
}

type WebFetchTool struct {
	maxChars int
}

func NewWebFetchTool(maxChars int) *WebFetchTool {
	if maxChars <= 0 {
		maxChars = 50000
	}
	return &WebFetchTool{
		maxChars: maxChars,
	}
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch a URL and extract readable content (HTML to text). Use this to get weather info, news, articles, or any web content."
}

func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
			"maxChars": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum characters to extract",
				"minimum":     100.0,
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
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

	maxChars := t.maxChars
	if mc, ok := args["maxChars"].(float64); ok {
		if int(mc) > 100 {
			maxChars = int(mc)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			TLSHandshakeTimeout: 15 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	contentType := resp.Header.Get("Content-Type")

	var text, extractor string

	if strings.Contains(contentType, "application/json") {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			formatted, _ := json.MarshalIndent(jsonData, "", "  ")
			text = string(formatted)
			extractor = "json"
		} else {
			text = string(body)
			extractor = "raw"
		}
	} else if strings.Contains(contentType, "text/html") || len(body) > 0 &&
		(strings.HasPrefix(string(body), "<!DOCTYPE") || strings.HasPrefix(strings.ToLower(string(body)), "<html")) {
		text = t.extractText(string(body))
		extractor = "text"
	} else {
		text = string(body)
		extractor = "raw"
	}

	truncated := len(text) > maxChars
	if truncated {
		text = text[:maxChars]
	}

	result := map[string]interface{}{
		"url":       urlStr,
		"status":    resp.StatusCode,
		"extractor": extractor,
		"truncated": truncated,
		"length":    len(text),
		"text":      text,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Fetched %d bytes from %s (extractor: %s, truncated: %v)", len(text), urlStr, extractor, truncated),
		ForUser: string(resultJSON),
	}
}

func (t *WebFetchTool) extractText(htmlContent string) string {
	re := regexp.MustCompile(`<script[\s\S]*?</script>`)
	result := re.ReplaceAllLiteralString(htmlContent, "")
	re = regexp.MustCompile(`<style[\s\S]*?</style>`)
	result = re.ReplaceAllLiteralString(result, "")
	re = regexp.MustCompile(`<[^>]+>`)
	result = re.ReplaceAllLiteralString(result, "")

	result = strings.TrimSpace(result)

	re = regexp.MustCompile(`\s+`)
	result = re.ReplaceAllLiteralString(result, " ")

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
