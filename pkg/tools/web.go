package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	userAgent = "Mozilla/5.0 (compatible; picoclaw/1.0)"
)

// --- Ollama Search Tool ---

type OllamaSearchTool struct {
	apiKey     string
	maxResults int
}

func NewOllamaSearchTool(apiKey string, maxResults int) *OllamaSearchTool {
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}
	return &OllamaSearchTool{
		apiKey:     apiKey,
		maxResults: maxResults,
	}
}

func (t *OllamaSearchTool) Name() string {
	return "web_search"
}

func (t *OllamaSearchTool) Description() string {
	return "Search the web for current information using Ollama. Returns titles, URLs, and snippets."
}

func (t *OllamaSearchTool) Parameters() map[string]interface{} {
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

func (t *OllamaSearchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
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

	requestBody := map[string]interface{}{
		"query":       query,
		"max_results": count,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal request: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://ollama.com/api/web_search", bytes.NewReader(jsonData))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	} else {
		return &ToolResult{
			ForLLM:  "Error: OLLAMA_API_KEY not configured",
			ForUser: "Error: OLLAMA_API_KEY not configured",
			IsError: false,
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Error: Ollama API returned %d: %s", resp.StatusCode, string(body)),
			ForUser: fmt.Sprintf("Error: Ollama API returned %d", resp.StatusCode),
			IsError: false,
		}
	}

	var searchResp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
	}

	if len(searchResp.Results) == 0 {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("No results for: %s", query),
			ForUser: fmt.Sprintf("No results for: %s", query),
			IsError: false,
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s", query))
	for i, item := range searchResp.Results {
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.URL))
		if item.Content != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Content))
		}
	}

	return &ToolResult{
		ForLLM:  strings.Join(lines, "\n"),
		ForUser: strings.Join(lines, "\n"),
		IsError: false,
	}
}

// --- Ollama Fetch Tool ---

type OllamaFetchTool struct {
	apiKey   string
	maxChars int
}

func NewOllamaFetchTool(apiKey string, maxChars int) *OllamaFetchTool {
	if maxChars <= 0 {
		maxChars = 50000
	}
	return &OllamaFetchTool{
		apiKey:   apiKey,
		maxChars: maxChars,
	}
}

func (t *OllamaFetchTool) Name() string {
	return "web_fetch"
}

func (t *OllamaFetchTool) Description() string {
	return "Fetch a URL and extract readable content using Ollama's Web Fetch API."
}

func (t *OllamaFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
		},
		"required": []string{"url"},
	}
}

func (t *OllamaFetchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	urlStr, ok := args["url"].(string)
	if !ok {
		return ErrorResult("url is required")
	}

	requestBody := map[string]interface{}{
		"url": urlStr,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal request: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://ollama.com/api/web_fetch", bytes.NewReader(jsonData))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	} else {
		return &ToolResult{
			ForLLM:  "Error: OLLAMA_API_KEY not configured",
			ForUser: "Error: OLLAMA_API_KEY not configured",
			IsError: false,
		}
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Error: Ollama API returned %d: %s", resp.StatusCode, string(body)),
			ForUser: fmt.Sprintf("Error: Ollama API returned %d", resp.StatusCode),
			IsError: false,
		}
	}

	var fetchResp struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Links   []string `json:"links"`
	}

	if err := json.Unmarshal(body, &fetchResp); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
	}

	text := fetchResp.Content
	if len(text) > t.maxChars {
		text = text[:t.maxChars]
	}

	result := map[string]interface{}{
		"url":       urlStr,
		"title":     fetchResp.Title,
		"status":    resp.StatusCode,
		"extractor": "ollama",
		"truncated": len(fetchResp.Content) > t.maxChars,
		"length":    len(text),
		"text":      text,
		"links":     fetchResp.Links,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &ToolResult{
		ForLLM:  string(resultJSON),
		ForUser: string(resultJSON),
		IsError: false,
	}
}

// --- Original Brave Search Tool ---

type WebSearchTool struct {
	apiKey     string
	maxResults int
}

func NewWebSearchTool(apiKey string, maxResults int) *WebSearchTool {
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}
	return &WebSearchTool{
		apiKey:     apiKey,
		maxResults: maxResults,
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search_brave"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for current information using Brave Search. Returns titles, URLs, and snippets."
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
	if t.apiKey == "" {
		return &ToolResult{
			ForLLM:  "Error: BRAVE_API_KEY not configured",
			ForUser: "Error: BRAVE_API_KEY not configured",
			IsError: false,
		}
	}

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

	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", t.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
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
		return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
	}

	results := searchResp.Web.Results
	if len(results) == 0 {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("No results for: %s", query),
			ForUser: fmt.Sprintf("No results for: %s", query),
			IsError: false,
		}
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

	return &ToolResult{
		ForLLM:  strings.Join(lines, "\n"),
		ForUser: strings.Join(lines, "\n"),
		IsError: false,
	}
}

// --- Original Web Fetch Tool ---

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
	return "web_fetch_raw"
}

func (t *WebFetchTool) Description() string {
	return "Fetch a URL and extract readable content (HTML to text) directly. Use if Ollama fetch fails."
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
		ForLLM:  string(resultJSON),
		ForUser: string(resultJSON),
		IsError: false,
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
