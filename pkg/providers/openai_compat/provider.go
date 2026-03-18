package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/common"
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

const defaultRequestTimeout = common.DefaultRequestTimeout

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
	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: common.NewHTTPClient(proxy),
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
		"messages": common.SerializeMessages(messages),
	}

	// When fallback uses a different provider (e.g. DeepSeek), that provider must not inject web_search_preview.
	nativeSearch, _ := options["native_search"].(bool)
	nativeSearch = nativeSearch && isNativeSearchHost(p.apiBase)
	if len(tools) > 0 || nativeSearch {
		// Build tool list with native search handling, then apply strict mode sanitization
		toolsList := buildToolsList(tools, nativeSearch)
		requestBody["tools"] = p.finalizeTools(toolsList, options)
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok {
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

	if temperature, ok := common.AsFloat(options["temperature"]); ok {
		lowerModel := strings.ToLower(model)
		// Kimi k2 models only support temperature=1.
		if strings.Contains(lowerModel, "kimi") && strings.Contains(lowerModel, "k2") {
			requestBody["temperature"] = 1.0
		} else {
			requestBody["temperature"] = temperature
		}
	}

	// Prompt caching: pass a stable cache key so OpenAI can bucket requests
	// with the same key and reuse prefix KV cache across calls.
	// The key is typically the agent ID — stable per agent, shared across requests.
	// See: https://platform.openai.com/docs/guides/prompt-caching
	// Prompt caching is only supported by OpenAI-native endpoints.
	// Non-OpenAI providers (Mistral, Gemini, DeepSeek, etc.) reject unknown
	// fields with 422 errors, so only include it for OpenAI APIs.
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		if supportsPromptCacheKey(p.apiBase) {
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

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	return common.ReadAndParseResponse(resp, p.apiBase)
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

func buildToolsList(tools []ToolDefinition, nativeSearch bool) []any {
	result := make([]any, 0, len(tools)+1)
	for _, t := range tools {
		if nativeSearch && strings.EqualFold(t.Function.Name, "web_search") {
			continue
		}
		result = append(result, t)
	}
	if nativeSearch {
		result = append(result, map[string]any{"type": "web_search_preview"})
	}
	return result
}

func (p *Provider) SupportsNativeSearch() bool {
	return isNativeSearchHost(p.apiBase)
}

func isNativeSearchHost(apiBase string) bool {
	u, err := url.Parse(apiBase)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "api.openai.com" || strings.HasSuffix(host, ".openai.azure.com")
}

// supportsPromptCacheKey reports whether the given API base is known to
// support the prompt_cache_key request field. Currently only OpenAI's own
// API and Azure OpenAI support this. All other OpenAI-compatible providers
// (Mistral, Gemini, DeepSeek, Groq, etc.) reject unknown fields with 422 errors.
func supportsPromptCacheKey(apiBase string) bool {
	u, err := url.Parse(apiBase)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "api.openai.com" || strings.HasSuffix(host, ".openai.azure.com")
}

// supportsStrictMode reports whether the given API base is known to
// support the 'strict' flag in tool definitions (Structured Outputs).
// Non-OpenAI compatible providers often reject this flag with 400 errors.
func supportsStrictMode(apiBase string) bool {
	// For now, mirror prompt caching check: only native OpenAI is trusted.
	return supportsPromptCacheKey(apiBase)
}

// finalizeTools handles OpenAI Strict Mode compatibility.
// Strips 'strict' flag for non-OpenAI providers to prevent 400 errors.
func (p *Provider) finalizeTools(tools []any, options map[string]any) any {
	forceStrict, hasForce := options["strict_mode"].(bool)
	isNative := supportsStrictMode(p.apiBase)

	// Native providers: pass through unless user forces a mode
	if isNative && !hasForce {
		return tools
	}

	// Decide whether to use strict mode
	useStrict := isNative
	if hasForce {
		if forceStrict && !isNative {
			slog.Warn("openai_compat: strict_mode=true ignored for non-OpenAI provider", "api_base", p.apiBase)
			useStrict = false
		} else {
			useStrict = forceStrict
		}
	}

	// Build tool list with appropriate strict mode setting
	out := make([]any, 0, len(tools))
	for _, t := range tools {
		var toolMap map[string]any
		switch v := t.(type) {
		case ToolDefinition:
			toolMap = map[string]any{
				"type": v.Type,
				"function": map[string]any{
					"name":        v.Function.Name,
					"description": v.Function.Description,
					"parameters":  v.Function.Parameters,
				},
			}
		case map[string]any:
			toolMap = v
		default:
			out = append(out, t)
			continue
		}

		// Add strict flag only to function tools
		if useStrict && toolMap["type"] == "function" {
			if fn, ok := toolMap["function"].(map[string]any); ok {
				fn["strict"] = true
			}
		}
		out = append(out, toolMap)
	}
	return out
}
