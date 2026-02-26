package web_fetch

import (
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

type WebFetchTool struct {
	maxChars int
	proxy    string
}

func NewWebFetchTool(maxChars int) *WebFetchTool {
	if maxChars <= 0 {
		maxChars = 50000
	}
	return &WebFetchTool{
		maxChars: maxChars,
	}
}

func NewWebFetchToolWithProxy(maxChars int, proxy string) *WebFetchTool {
	if maxChars <= 0 {
		maxChars = 50000
	}
	return &WebFetchTool{
		maxChars: maxChars,
		proxy:    proxy,
	}
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

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]any) *common.ToolResult {
	urlStr, ok := args["url"].(string)
	if !ok {
		return common.ErrorResult("url is required")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return common.ErrorResult(fmt.Sprintf("invalid URL: %v", err))
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return common.ErrorResult("only http/https URLs are allowed")
	}

	if parsedURL.Host == "" {
		return common.ErrorResult("missing domain in URL")
	}

	maxChars := t.maxChars
	if mc, ok := args["maxChars"].(float64); ok {
		if int(mc) > 100 {
			maxChars = int(mc)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return common.ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("User-Agent", common.UserAgent)

	client, err := common.CreateHTTPClient(t.proxy, 60*time.Second)
	if err != nil {
		return common.ErrorResult(fmt.Sprintf("failed to create HTTP client: %v", err))
	}

	// Configure redirect handling
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("stopped after 5 redirects")
		}
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return common.ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return common.ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	contentType := resp.Header.Get("Content-Type")

	var text, extractor string

	if strings.Contains(contentType, "application/json") {
		var jsonData any
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

	result := map[string]any{
		"url":       urlStr,
		"status":    resp.StatusCode,
		"extractor": extractor,
		"truncated": truncated,
		"length":    len(text),
		"text":      text,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &common.ToolResult{
		ForLLM: fmt.Sprintf(
			"Fetched %d bytes from %s (extractor: %s, truncated: %v)",
			len(text),
			urlStr,
			extractor,
			truncated,
		),
		ForUser: string(resultJSON),
	}
}

func (t *WebFetchTool) extractText(htmlContent string) string {
	result := common.ReScript.ReplaceAllLiteralString(htmlContent, "")
	result = common.ReStyle.ReplaceAllLiteralString(result, "")
	result = common.ReTags.ReplaceAllLiteralString(result, "")

	result = strings.TrimSpace(result)

	result = common.ReWhitespace.ReplaceAllString(result, " ")
	result = common.ReBlankLines.ReplaceAllString(result, "\n\n")

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
