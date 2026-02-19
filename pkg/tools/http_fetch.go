package tools

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

// HttpFetchTool performs structured HTTP requests (GET/POST/PUT/PATCH/DELETE)
// with explicit method, headers, and body. Unlike exec+curl, this is a
// first-class named tool — every call appears in Weave traces as "http_fetch"
// with full structured args (url, method, body) rather than an opaque shell
// command string. This makes API calls to internal services (e.g. the
// Handelsregister, lead discovery, or any REST endpoint) fully observable.
type HttpFetchTool struct {
	maxResponseBytes int
}

func NewHttpFetchTool(maxResponseBytes int) *HttpFetchTool {
	if maxResponseBytes <= 0 {
		maxResponseBytes = 512 * 1024 // 512 KB default
	}
	return &HttpFetchTool{maxResponseBytes: maxResponseBytes}
}

func (t *HttpFetchTool) Name() string {
	return "http_fetch"
}

func (t *HttpFetchTool) Description() string {
	return "Make an HTTP request (GET, POST, PUT, PATCH, DELETE) to any URL. " +
		"Supports JSON body and custom headers. Returns status code and response body. " +
		"Use this instead of exec+curl for all API calls — it produces structured traces."
}

func (t *HttpFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "Full URL to request (http or https)",
			},
			"method": map[string]interface{}{
				"type":        "string",
				"description": "HTTP method: GET, POST, PUT, PATCH, DELETE (default: GET)",
				"enum":        []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "Request body as a JSON string (for POST/PUT/PATCH)",
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "Optional HTTP headers as key-value pairs",
				"additionalProperties": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []string{"url"},
	}
}

func (t *HttpFetchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	rawURL, ok := args["url"].(string)
	if !ok || rawURL == "" {
		return ErrorResult("url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ErrorResult(fmt.Sprintf("invalid URL (must be http/https): %v", rawURL))
	}

	method := "GET"
	if m, ok := args["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	var bodyReader io.Reader
	bodyStr := ""
	if b, ok := args["body"].(string); ok && b != "" {
		bodyStr = b
		bodyReader = bytes.NewBufferString(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to build request: %v", err))
	}

	// Default Content-Type for requests with a body
	if bodyStr != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "picoclaw/http_fetch")

	// Apply caller-supplied headers (may override defaults)
	if hdrs, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range hdrs {
			if vs, ok := v.(string); ok {
				req.Header.Set(k, vs)
			}
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
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

	limited := io.LimitReader(resp.Body, int64(t.maxResponseBytes))
	rawBody, err := io.ReadAll(limited)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	truncated := len(rawBody) >= t.maxResponseBytes
	contentType := resp.Header.Get("Content-Type")

	// Pretty-print JSON responses for the LLM
	responseText := string(rawBody)
	if strings.Contains(contentType, "application/json") {
		var parsed interface{}
		if json.Unmarshal(rawBody, &parsed) == nil {
			if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				responseText = string(pretty)
			}
		}
	}

	result := map[string]interface{}{
		"url":          rawURL,
		"method":       method,
		"status":       resp.StatusCode,
		"content_type": contentType,
		"truncated":    truncated,
		"body":         responseText,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	isErr := resp.StatusCode >= 400
	if isErr {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("HTTP %d from %s %s\n%s", resp.StatusCode, method, rawURL, responseText),
			ForUser: string(resultJSON),
			IsError: true,
		}
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("HTTP %d from %s %s\n%s", resp.StatusCode, method, rawURL, responseText),
		ForUser: string(resultJSON),
	}
}
