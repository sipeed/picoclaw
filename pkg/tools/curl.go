package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	curlDefaultTimeout  = 30 * time.Second
	curlDefaultMaxBytes = int64(1 << 20)
	curlMaxRedirects    = 5
)

type CurlTool struct {
	allowedDomains []string
	client         *http.Client
	maxBytes       int64
}

type CurlToolOptions struct {
	AllowedDomains []string
	Proxy          string
	TimeoutSeconds int
	MaxBytes       int64
}

func NewCurlTool(opts CurlToolOptions) (*CurlTool, error) {
	timeout := curlDefaultTimeout
	if opts.TimeoutSeconds > 0 {
		timeout = time.Duration(opts.TimeoutSeconds) * time.Second
	}

	maxBytes := curlDefaultMaxBytes
	if opts.MaxBytes > 0 {
		maxBytes = opts.MaxBytes
	}

	client, err := utils.CreateHTTPClient(opts.Proxy, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for curl tool: %w", err)
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= curlMaxRedirects {
			return fmt.Errorf("stopped after %d redirects", curlMaxRedirects)
		}
		if !isDomainAllowed(req.URL.Hostname(), opts.AllowedDomains) {
			return fmt.Errorf("redirect to disallowed domain %q", req.URL.Hostname())
		}
		return nil
	}

	return &CurlTool{
		allowedDomains: normalizeDomains(opts.AllowedDomains),
		client:         client,
		maxBytes:       maxBytes,
	}, nil
}

func (t *CurlTool) Name() string {
	return "curl"
}

func (t *CurlTool) Description() string {
	return "Make HTTP requests to external APIs. Use url (required), method (GET/POST/PUT/DELETE/etc), headers (optional map), body (optional string), and timeout (optional seconds). Only allowed domains can be accessed."
}

func (t *CurlTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "URL to request (http/https only, must match allowed domains)",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)",
				"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Optional HTTP headers as key-value pairs",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Optional request body (for POST, PUT, PATCH)",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Optional timeout in seconds (default: 30, max: 120)",
				"minimum":     1.0,
				"maximum":     120.0,
			},
		},
		"required": []string{"url"},
	}
}

func (t *CurlTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	urlStr, ok := args["url"].(string)
	if !ok || strings.TrimSpace(urlStr) == "" {
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

	if !isDomainAllowed(parsedURL.Hostname(), t.allowedDomains) {
		return ErrorResult(fmt.Sprintf("domain %q is not in the allowed domains list", parsedURL.Hostname()))
	}

	method := "GET"
	if m, ok := args["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	var bodyReader io.Reader
	if bodyStr, ok := args["body"].(string); ok && bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("User-Agent", fmt.Sprintf("picoclaw/curl (+https://github.com/sipeed/picoclaw)"))

	if headers, ok := args["headers"].(map[string]any); ok {
		for k, v := range headers {
			if vs, ok := v.(string); ok {
				req.Header.Set(k, vs)
			}
		}
	}

	timeout := curlDefaultTimeout
	if tSec, ok := args["timeout"].(float64); ok && tSec > 0 {
		timeout = time.Duration(tSec) * time.Second
		if timeout > 120*time.Second {
			timeout = 120 * time.Second
		}
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req = req.WithContext(ctxWithTimeout)

	resp, err := t.client.Do(req)
	if err != nil {
		if ctxWithTimeout.Err() == context.DeadlineExceeded {
			return ErrorResult(fmt.Sprintf("request timed out after %v", timeout))
		}
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, t.maxBytes))
	if err != nil {
		if err == io.ErrUnexpectedEOF || strings.Contains(err.Error(), "http: request body too large") {
			return ErrorResult(fmt.Sprintf("response body exceeded %d bytes limit", t.maxBytes))
		}
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	var headersOut map[string]string
	if resp.Header != nil {
		headersOut = make(map[string]string)
		for k, v := range resp.Header {
			if len(v) > 0 {
				headersOut[k] = v[0]
			}
		}
	}

	result := map[string]any{
		"status":      resp.StatusCode,
		"status_text": resp.Status,
		"headers":     headersOut,
		"body":        string(body),
		"url":         urlStr,
		"method":      method,
		"truncated":   int64(len(body)) >= t.maxBytes,
	}

	return &ToolResult{
		ForLLM:  formatCurlResult(result),
		ForUser: fmt.Sprintf("HTTP %d from %s %s", resp.StatusCode, method, urlStr),
	}
}

func formatCurlResult(result map[string]any) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Status: %v\n", result["status"]))
	b.WriteString(fmt.Sprintf("URL: %v\n", result["url"]))
	b.WriteString(fmt.Sprintf("Method: %v\n", result["method"]))

	if headers, ok := result["headers"].(map[string]string); ok && len(headers) > 0 {
		b.WriteString("Headers:\n")
		for k, v := range headers {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	if truncated, ok := result["truncated"].(bool); ok && truncated {
		b.WriteString("\n[Response body truncated due to size limit]\n")
	}

	if body, ok := result["body"].(string); ok {
		b.WriteString("\nBody:\n")
		b.WriteString(body)
	}

	return b.String()
}

func isDomainAllowed(hostname string, allowedDomains []string) bool {
	if len(allowedDomains) == 0 {
		return true
	}
	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))
	for _, domain := range allowedDomains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			return true
		}
	}
	return false
}

func normalizeDomains(domains []string) []string {
	result := make([]string, 0, len(domains))
	seen := make(map[string]struct{})
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		d = strings.TrimPrefix(d, "http://")
		d = strings.TrimPrefix(d, "https://")
		d = strings.TrimSuffix(d, "/")
		if d == "" {
			continue
		}
		if _, exists := seen[d]; !exists {
			seen[d] = struct{}{}
			result = append(result, d)
		}
	}
	return result
}
