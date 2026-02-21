package anthropicprovider

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	oauthBetaHeader         = "oauth-2025-04-20"
	interleavedThinkingBeta = "interleaved-thinking-2025-05-14"
	userAgent               = "claude-cli/2.1.2 (external, cli)"
	appName                 = "PicoClaw"
	mcpToolPrefix           = "mcp_"
)

// OAuthMiddlewareConfig configures the OAuth request middleware.
type OAuthMiddlewareConfig struct {
	TokenSource     func() (string, error)
	SanitizePrompts bool
	RenameTools     bool
}

// NewOAuthMiddleware returns SDK request options that transform every outgoing
// request into a Claude-Code-compatible OAuth request.
func NewOAuthMiddleware(cfg OAuthMiddlewareConfig) []option.RequestOption {
	return []option.RequestOption{
		option.WithMiddleware(func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
			token, err := cfg.TokenSource()
			if err != nil {
				return nil, err
			}

			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Del("X-Api-Key")
			req.Header.Del("x-api-key")
			req.Header.Set("User-Agent", userAgent)

			existingBeta := req.Header.Get("anthropic-beta")
			betaValues := []string{oauthBetaHeader, interleavedThinkingBeta}
			if existingBeta != "" {
				betaValues = append(betaValues, existingBeta)
			}
			req.Header.Set("anthropic-beta", strings.Join(betaValues, ","))

			if strings.Contains(req.URL.Path, "/v1/messages") {
				q := req.URL.Query()
				q.Set("beta", "true")
				req.URL.RawQuery = q.Encode()
			}

			if req.Body != nil && req.Method == "POST" && (cfg.RenameTools || cfg.SanitizePrompts) {
				bodyBytes, readErr := io.ReadAll(req.Body)
				req.Body.Close()
				if readErr == nil && len(bodyBytes) > 0 {
					bodyBytes = transformRequestBody(bodyBytes, cfg.RenameTools, cfg.SanitizePrompts)
					req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					req.ContentLength = int64(len(bodyBytes))
				}
			}

			resp, err := next(req)
			if err != nil {
				return resp, err
			}

			if cfg.RenameTools && resp != nil && resp.Body != nil {
				resp.Body = newToolNameStripper(resp.Body)
			}

			return resp, err
		}),
	}
}

func transformRequestBody(body []byte, renameTools, sanitizePrompts bool) []byte {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return body
	}

	modified := false

	if renameTools {
		modified = prefixToolNames(parsed) || modified
	}

	if sanitizePrompts {
		modified = sanitizeSystemPrompts(parsed) || modified
	}

	if !modified {
		return body
	}

	result, err := json.Marshal(parsed)
	if err != nil {
		return body
	}
	return result
}

func prefixToolNames(parsed map[string]interface{}) bool {
	modified := false

	if tools, ok := parsed["tools"].([]interface{}); ok {
		for _, t := range tools {
			if tool, ok := t.(map[string]interface{}); ok {
				if name, ok := tool["name"].(string); ok && !strings.HasPrefix(name, mcpToolPrefix) {
					tool["name"] = mcpToolPrefix + name
					modified = true
				}
			}
		}
	}

	if messages, ok := parsed["messages"].([]interface{}); ok {
		for _, m := range messages {
			if msg, ok := m.(map[string]interface{}); ok {
				if content, ok := msg["content"].([]interface{}); ok {
					for _, c := range content {
						if block, ok := c.(map[string]interface{}); ok {
							if block["type"] == "tool_use" {
								if name, ok := block["name"].(string); ok && !strings.HasPrefix(name, mcpToolPrefix) {
									block["name"] = mcpToolPrefix + name
									modified = true
								}
							}
							if block["type"] == "tool_result" {
								if name, ok := block["name"].(string); ok && !strings.HasPrefix(name, mcpToolPrefix) {
									block["name"] = mcpToolPrefix + name
									modified = true
								}
							}
						}
					}
				}
			}
		}
	}

	return modified
}

func sanitizeSystemPrompts(parsed map[string]interface{}) bool {
	modified := false

	if system, ok := parsed["system"].([]interface{}); ok {
		for _, s := range system {
			if block, ok := s.(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					newText := sanitizeText(text)
					if newText != text {
						block["text"] = newText
						modified = true
					}
				}
			}
		}
	}

	if system, ok := parsed["system"].(string); ok {
		newSystem := sanitizeText(system)
		if newSystem != system {
			parsed["system"] = newSystem
			modified = true
		}
	}

	return modified
}

func sanitizeText(text string) string {
	text = strings.ReplaceAll(text, "PicoClaw", "Claude Code")
	text = strings.ReplaceAll(text, "picoclaw", "Claude")
	text = strings.ReplaceAll(text, "Picoclaw", "Claude Code")
	return text
}

type toolNameStripper struct {
	source    io.ReadCloser
	buffer    bytes.Buffer
	remainder []byte
}

func newToolNameStripper(source io.ReadCloser) io.ReadCloser {
	return &toolNameStripper{source: source}
}

func (s *toolNameStripper) Read(p []byte) (int, error) {
	if s.buffer.Len() > 0 {
		return s.buffer.Read(p)
	}

	n, err := s.source.Read(p)
	if n > 0 {
		data := string(p[:n])
		data = stripMCPPrefix(data)
		copy(p, []byte(data))
		n = len(data)
	}
	return n, err
}

func (s *toolNameStripper) Close() error {
	return s.source.Close()
}

func stripMCPPrefix(data string) string {
	data = strings.ReplaceAll(data, `"name":"mcp_`, `"name":"`)
	data = strings.ReplaceAll(data, `"name": "mcp_`, `"name": "`)
	return data
}
