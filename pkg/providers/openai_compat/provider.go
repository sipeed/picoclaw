package openai_compat

import (
	"bufio"
	"bytes"
	"context"
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

type Provider struct {
	apiKey         string
	apiBase        string
	maxTokensField string // Field name for max tokens (e.g., "max_completion_tokens" for o1/glm models)
	httpClient     *http.Client
}

type Option func(*Provider)

const defaultRequestTimeout = 120 * time.Second

func WithMaxTokensField(maxTokensField string) Option {
	return func(p *Provider) {
		p.maxTokensField = maxTokensField
	}
}

func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

func NewProvider(apiKey, apiBase, proxy string, opts ...Option) *Provider {
	client := &http.Client{
		Timeout: defaultRequestTimeout,
	}

	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(parsed),
			}
		} else {
			log.Printf("openai_compat: invalid proxy URL %q: %v", proxy, err)
		}
	}

	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: client,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	return p
}

func NewProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField string) *Provider {
	return NewProvider(apiKey, apiBase, proxy, WithMaxTokensField(maxTokensField))
}

func NewProviderWithMaxTokensFieldAndTimeout(
	apiKey, apiBase, proxy, maxTokensField string,
	requestTimeoutSeconds int,
) *Provider {
	return NewProvider(
		apiKey,
		apiBase,
		proxy,
		WithMaxTokensField(maxTokensField),
		WithRequestTimeout(time.Duration(requestTimeoutSeconds)*time.Second),
	)
}

func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	model = normalizeModel(model, p.apiBase)

	requestBody := map[string]any{
		"model":    model,
		"messages": serializeMessagesForProvider(messages, p.apiBase, model),
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		// Use configured maxTokensField if specified, otherwise fallback to model-based detection
		fieldName := p.maxTokensField
		if fieldName == "" {
			// Fallback: detect from model name for backward compatibility
			lowerModel := strings.ToLower(model)
			if strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "o1") ||
				strings.Contains(lowerModel, "gpt-5") {
				fieldName = "max_completion_tokens"
			} else {
				fieldName = "max_tokens"
			}
		}
		requestBody[fieldName] = maxTokens
	}

	if temperature, ok := asFloat(options["temperature"]); ok {
		lowerModel := strings.ToLower(model)
		// Kimi k2 models only support temperature=1.
		if strings.Contains(lowerModel, "kimi") && strings.Contains(lowerModel, "k2") {
			requestBody["temperature"] = 1.0
		} else {
			requestBody["temperature"] = temperature
		}
	}
	if strings.Contains(strings.ToLower(model), "glm-4.6v") {
		if level, _ := options["thinking_level"].(string); strings.TrimSpace(level) != "" && strings.ToLower(level) != "off" {
			requestBody["thinking"] = map[string]any{"type": "enabled"}
		}
	}

	// Prompt caching: pass a stable cache key so OpenAI can bucket requests
	// with the same key and reuse prefix KV cache across calls.
	// The key is typically the agent ID — stable per agent, shared across requests.
	// See: https://platform.openai.com/docs/guides/prompt-caching
	// Prompt caching is only supported by OpenAI-native endpoints.
	// Gemini and other providers reject unknown fields, so skip for non-OpenAI APIs.
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		if !strings.Contains(p.apiBase, "generativelanguage.googleapis.com") {
			requestBody["prompt_cache_key"] = cacheKey
		}
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	requestCtx := requestErrorContext{
		APIBase: p.apiBase,
		Model:   model,
		Path:    req.URL.Path,
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	// Non-200: read a prefix to tell HTML error page apart from JSON error body.
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 256))
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response: %w", readErr)
		}
		if looksLikeHTML(body, contentType) {
			return nil, wrapHTMLResponseError(resp.StatusCode, body, contentType, requestCtx)
		}
		return nil, formatAPIError(resp.StatusCode, body, requestCtx)
	}

	// Peek without consuming so the full stream reaches the JSON decoder.
	reader := bufio.NewReader(resp.Body)
	prefix, err := reader.Peek(256) // io.EOF/ErrBufferFull are normal; only real errors abort
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return nil, fmt.Errorf("failed to inspect response: %w", err)
	}
	if looksLikeHTML(prefix, contentType) {
		return nil, wrapHTMLResponseError(resp.StatusCode, prefix, contentType, requestCtx)
	}

	out, err := parseResponse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return out, nil
}

type requestErrorContext struct {
	APIBase string
	Model   string
	Path    string
}

func wrapHTMLResponseError(statusCode int, body []byte, contentType string, requestCtx requestErrorContext) error {
	respPreview := responsePreview(body, 128)
	return fmt.Errorf(
		"API request failed: %s returned HTML instead of JSON (content-type: %s); check api_base or proxy configuration.\n  Status: %d\n  API Base: %s\n  Model:   %s\n  Path:    %s\n  Body:    %s",
		requestCtx.APIBase,
		contentType,
		statusCode,
		requestCtx.APIBase,
		requestCtx.Model,
		requestCtx.Path,
		respPreview,
	)
}

func formatAPIError(statusCode int, body []byte, requestCtx requestErrorContext) error {
	if details, ok := extractStructuredAPIError(body); ok {
		if details.Code != "" {
			return fmt.Errorf(
				"API request failed:\n  Status: %d\n  API Base: %s\n  Model:   %s\n  Path:    %s\n  Message: %s\n  Code:    %s\n  Body:    %s",
				statusCode,
				requestCtx.APIBase,
				requestCtx.Model,
				requestCtx.Path,
				details.Message,
				details.Code,
				responsePreview(body, 128),
			)
		}
		return fmt.Errorf(
			"API request failed:\n  Status: %d\n  API Base: %s\n  Model:   %s\n  Path:    %s\n  Message: %s\n  Body:    %s",
			statusCode,
			requestCtx.APIBase,
			requestCtx.Model,
			requestCtx.Path,
			details.Message,
			responsePreview(body, 128),
		)
	}
	return fmt.Errorf(
		"API request failed:\n  Status: %d\n  API Base: %s\n  Model:   %s\n  Path:    %s\n  Body:    %s",
		statusCode,
		requestCtx.APIBase,
		requestCtx.Model,
		requestCtx.Path,
		responsePreview(body, 128),
	)
}

type structuredAPIError struct {
	Message string
	Code    string
}

func extractStructuredAPIError(body []byte) (structuredAPIError, bool) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return structuredAPIError{}, false
	}

	var payload map[string]any
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		return structuredAPIError{}, false
	}

	details := structuredAPIError{}
	if msg := stringifyErrorField(payload["message"]); msg != "" {
		details.Message = msg
	}
	if code := stringifyErrorField(payload["code"]); code != "" {
		details.Code = code
	}

	if errField, ok := payload["error"]; ok {
		switch e := errField.(type) {
		case string:
			if details.Message == "" {
				details.Message = e
			}
		case map[string]any:
			if details.Message == "" {
				details.Message = stringifyErrorField(e["message"])
			}
			if details.Code == "" {
				details.Code = stringifyErrorField(e["code"])
			}
			if details.Message == "" {
				details.Message = stringifyErrorField(e["type"])
			}
		}
	}

	if details.Message == "" {
		return structuredAPIError{}, false
	}
	return details, true
}

func stringifyErrorField(v any) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case json.Number:
		return val.String()
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", val))
	case float32:
		return strings.TrimSpace(fmt.Sprintf("%.0f", val))
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case int32:
		return fmt.Sprintf("%d", val)
	default:
		return ""
	}
}

func looksLikeHTML(body []byte, contentType string) bool {
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

func responsePreview(body []byte, maxLen int) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return "<empty>"
	}
	if len(trimmed) <= maxLen {
		return string(trimmed)
	}
	return string(trimmed[:maxLen]) + "..."
}

func parseResponse(body io.Reader) (*LLMResponse, error) {
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
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
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
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					log.Printf("openai_compat: failed to decode tool call arguments for %q: %v", name, err)
					arguments["raw"] = tc.Function.Arguments
				}
			}
		}

		// Build ToolCall with ExtraContent for Gemini 3 thought_signature persistence
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
		FinishReason:     choice.FinishReason,
		Usage:            apiResponse.Usage,
	}, nil
}

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

// serializeMessages converts internal Message structs to the OpenAI wire format.
// - Strips SystemParts (unknown to third-party endpoints)
// - Converts messages with Media to multipart content format (text + image_url parts)
// - Preserves ToolCallID, ToolCalls, and ReasoningContent for all messages
func serializeMessages(messages []Message) []any {
	return serializeMessagesForProvider(messages, "", "")
}

func serializeMessagesForProvider(messages []Message, apiBase, model string) []any {
	useObjectImageURL := usesObjectImageURL(apiBase, model)
	preferImageFirst := usesImageFirstOrdering(apiBase, model)
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

		parts := make([]map[string]any, 0, 1+len(m.Media))
		appendImageParts := func() {
			for _, mediaURL := range m.Media {
				if !strings.HasPrefix(mediaURL, "data:image/") && !strings.HasPrefix(mediaURL, "http://") && !strings.HasPrefix(mediaURL, "https://") {
					continue
				}
				part := map[string]any{"type": "image_url"}
				if useObjectImageURL {
					part["image_url"] = map[string]any{"url": mediaURL}
				} else {
					part["image_url"] = mediaURL
				}
				parts = append(parts, part)
			}
		}
		appendTextPart := func() {
			if m.Content != "" {
				parts = append(parts, map[string]any{"type": "text", "text": m.Content})
			}
		}
		if preferImageFirst {
			appendImageParts()
			appendTextPart()
		} else {
			appendTextPart()
			appendImageParts()
		}

		msg := map[string]any{"role": m.Role, "content": parts}
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

func usesImageFirstOrdering(apiBase, model string) bool {
	base := strings.ToLower(strings.TrimSpace(apiBase))
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(base, "bigmodel.cn") || strings.Contains(base, "zhipu") || strings.Contains(m, "glm-4.6v")
}

func usesObjectImageURL(apiBase, model string) bool {
	base := strings.ToLower(strings.TrimSpace(apiBase))
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(base, "bigmodel.cn") || strings.Contains(base, "zhipu") || strings.Contains(m, "glm")
}

func normalizeModel(model, apiBase string) string {
	before, after, ok := strings.Cut(model, "/")
	if !ok {
		return model
	}

	if strings.Contains(strings.ToLower(apiBase), "openrouter.ai") {
		return model
	}

	prefix := strings.ToLower(before)
	switch prefix {
	case "litellm", "moonshot", "nvidia", "groq", "ollama", "deepseek", "google",
		"openrouter", "zhipu", "mistral", "vivgrid", "minimax":
		return after
	default:
		return model
	}
}

func asInt(v any) (int, bool) {
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

func asFloat(v any) (float64, bool) {
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
