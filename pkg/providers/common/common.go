// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package common provides shared utilities used by multiple LLM provider
// implementations (openai_compat, azure, etc.).
package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// Re-export protocol types used across providers.
type (
	ToolCall               = protocoltypes.ToolCall
	FunctionCall           = protocoltypes.FunctionCall
	LLMResponse            = protocoltypes.LLMResponse
	UsageInfo              = protocoltypes.UsageInfo
	Message                = protocoltypes.Message
	ToolDefinition         = protocoltypes.ToolDefinition
	ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
	ReasoningDetail        = protocoltypes.ReasoningDetail
)

const DefaultRequestTimeout = 120 * time.Second

// NewHTTPClient creates an *http.Client with an optional proxy and the default timeout.
func NewHTTPClient(proxy string) *http.Client {
	client := &http.Client{
		Timeout: DefaultRequestTimeout,
	}
	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err == nil {
			// Preserve http.DefaultTransport settings (TLS, HTTP/2, timeouts, etc.)
			if base, ok := http.DefaultTransport.(*http.Transport); ok {
				tr := base.Clone()
				tr.Proxy = http.ProxyURL(parsed)
				client.Transport = tr
			} else {
				// Fallback: minimal transport if DefaultTransport is not *http.Transport.
				client.Transport = &http.Transport{
					Proxy: http.ProxyURL(parsed),
				}
			}
		} else {
			log.Printf("common: invalid proxy URL %q: %v", proxy, err)
		}
	}
	return client
}

// --- Message serialization ---

// openaiMessage is the wire-format message for OpenAI-compatible APIs.
// It mirrors protocoltypes.Message but omits SystemParts, which is an
// internal field that would be unknown to third-party endpoints.
type openaiMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

// SerializeMessages converts internal Message structs to the OpenAI wire format.
//   - Strips SystemParts (unknown to third-party endpoints)
//   - Converts messages with Media to multipart content format (text + image_url parts)
//   - Preserves ToolCallID, ToolCalls, and ReasoningContent for all messages
func SerializeMessages(messages []Message) []any {
	out := make([]any, 0, len(messages))
	for _, m := range messages {
		if len(m.Media) == 0 {
			out = append(out, openaiMessage{
				Role:             m.Role,
				Content:          m.Content,
				ReasoningContent: m.ReasoningContent,
				ToolCalls:        m.ToolCalls,
				ToolCallID:       m.ToolCallID,
			})
			continue
		}

		// Multipart content format for messages with media
		parts := make([]map[string]any, 0, 1+len(m.Media))
		if m.Content != "" {
			parts = append(parts, map[string]any{
				"type": "text",
				"text": m.Content,
			})
		}
		for _, mediaURL := range m.Media {
			if strings.HasPrefix(mediaURL, "data:image/") {
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": mediaURL,
					},
				})
			}
		}

		msg := map[string]any{
			"role":    m.Role,
			"content": parts,
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			msg["tool_calls"] = m.ToolCalls
		}
		if m.ReasoningContent != "" {
			msg["reasoning_content"] = m.ReasoningContent
		}
		out = append(out, msg)
	}
	return out
}

// --- Response parsing ---

// ParseResponse parses a JSON chat completion response body into an LLMResponse.
func ParseResponse(body io.Reader) (*LLMResponse, error) {
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content          string            `json:"content"`
				ReasoningContent string            `json:"reasoning_content"`
				Reasoning        string            `json:"reasoning"`
				ReasoningDetails []ReasoningDetail `json:"reasoning_details"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function *struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
					ExtraContent *struct {
						Google *struct {
							ThoughtSignature string `json:"thought_signature"`
						} `json:"google"`
					} `json:"extra_content"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *UsageInfo `json:"usage"`
	}

	if err := json.NewDecoder(body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
		}, nil
	}

	choice := apiResponse.Choices[0]
	toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))
	for _, tc := range choice.Message.ToolCalls {
		arguments := make(map[string]any)
		name := ""

		// Extract thought_signature from Gemini/Google-specific extra content
		thoughtSignature := ""
		if tc.ExtraContent != nil && tc.ExtraContent.Google != nil {
			thoughtSignature = tc.ExtraContent.Google.ThoughtSignature
		}

		if tc.Function != nil {
			name = tc.Function.Name
			arguments = DecodeToolCallArguments(tc.Function.Arguments, name)
		}

		toolCall := ToolCall{
			ID:               tc.ID,
			Name:             name,
			Arguments:        arguments,
			ThoughtSignature: thoughtSignature,
		}

		if thoughtSignature != "" {
			toolCall.ExtraContent = &ExtraContent{
				Google: &GoogleExtra{
					ThoughtSignature: thoughtSignature,
				},
			}
		}

		toolCalls = append(toolCalls, toolCall)
	}

	return &LLMResponse{
		Content:          choice.Message.Content,
		ReasoningContent: choice.Message.ReasoningContent,
		Reasoning:        choice.Message.Reasoning,
		ReasoningDetails: choice.Message.ReasoningDetails,
		ToolCalls:        toolCalls,
		FinishReason:     normalizeFinishReason(choice.FinishReason),
		Usage:            apiResponse.Usage,
	}, nil
}

// normalizeFinishReason normalizes finish_reason values across providers.
// Converts "length" to "truncated" for consistent handling.
func normalizeFinishReason(reason string) string {
	if reason == "length" {
		return "truncated"
	}
	return reason
}

// DecodeToolCallArguments decodes a tool call's arguments from raw JSON.
func DecodeToolCallArguments(raw json.RawMessage, name string) map[string]any {
	arguments := make(map[string]any)
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return arguments
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		log.Printf("common: failed to decode tool call arguments payload for %q: %v", name, err)
		arguments["raw"] = string(raw)
		return arguments
	}

	switch v := decoded.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return arguments
		}
		if err := json.Unmarshal([]byte(v), &arguments); err != nil {
			log.Printf("common: failed to decode tool call arguments for %q: %v", name, err)
			arguments["raw"] = v
		}
		return arguments
	case map[string]any:
		return v
	default:
		log.Printf("common: unsupported tool call arguments type for %q: %T", name, decoded)
		arguments["raw"] = string(raw)
		return arguments
	}
}

// --- HTTP response helpers ---

// HandleErrorResponse reads a non-200 response body and returns an appropriate error.
func HandleErrorResponse(resp *http.Response, apiBase string) error {
	contentType := resp.Header.Get("Content-Type")
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 256))
	if readErr != nil {
		return fmt.Errorf("failed to read response: %w", readErr)
	}
	if LooksLikeHTML(body, contentType) {
		return WrapHTMLResponseError(resp.StatusCode, body, contentType, apiBase)
	}
	return fmt.Errorf(
		"API request failed:\n  Status: %d\n  Body:   %s",
		resp.StatusCode,
		ResponsePreview(body, 128),
	)
}

// ReadAndParseResponse peeks at the response body to detect HTML errors,
// then parses the JSON response into an LLMResponse.
func ReadAndParseResponse(resp *http.Response, apiBase string) (*LLMResponse, error) {
	contentType := resp.Header.Get("Content-Type")
	reader := bufio.NewReader(resp.Body)
	prefix, err := reader.Peek(256)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return nil, fmt.Errorf("failed to inspect response: %w", err)
	}
	if LooksLikeHTML(prefix, contentType) {
		return nil, WrapHTMLResponseError(resp.StatusCode, prefix, contentType, apiBase)
	}
	out, err := ParseResponse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return out, nil
}

// LooksLikeHTML checks if the response body appears to be HTML.
func LooksLikeHTML(body []byte, contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml+xml") {
		return true
	}
	prefix := bytes.ToLower(leadingTrimmedPrefix(body, 128))
	return bytes.HasPrefix(prefix, []byte("<!doctype html")) ||
		bytes.HasPrefix(prefix, []byte("<html")) ||
		bytes.HasPrefix(prefix, []byte("<head")) ||
		bytes.HasPrefix(prefix, []byte("<body"))
}

// WrapHTMLResponseError creates a descriptive error for HTML responses.
func WrapHTMLResponseError(statusCode int, body []byte, contentType, apiBase string) error {
	respPreview := ResponsePreview(body, 128)
	return fmt.Errorf(
		"API request failed: %s returned HTML instead of JSON (content-type: %s); check api_base or proxy configuration.\n  Status: %d\n  Body:   %s",
		apiBase,
		contentType,
		statusCode,
		respPreview,
	)
}

// ResponsePreview returns a truncated preview of response body for error messages.
func ResponsePreview(body []byte, maxLen int) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return "<empty>"
	}
	if len(trimmed) <= maxLen {
		return string(trimmed)
	}
	return string(trimmed[:maxLen]) + "..."
}

func leadingTrimmedPrefix(body []byte, maxLen int) []byte {
	i := 0
	for i < len(body) {
		switch body[i] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			i++
		default:
			end := i + maxLen
			if end > len(body) {
				end = len(body)
			}
			return body[i:end]
		}
	}
	return nil
}

// --- Numeric helpers ---

// AsInt converts various numeric types to int.
func AsInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

// AsFloat converts various numeric types to float64.
func AsFloat(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
