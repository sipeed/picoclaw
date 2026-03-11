package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"

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

var vendorPrefixes = map[string]struct{}{
	"litellm":    {},
	"moonshot":   {},
	"nvidia":     {},
	"groq":       {},
	"ollama":     {},
	"deepseek":   {},
	"google":     {},
	"openrouter": {},
	"zhipu":      {},
	"mistral":    {},
	"vivgrid":    {},
	"minimax":    {},
}

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

func WithRetry(maxRetries int, minWait, maxWait time.Duration) Option {
	return func(p *Provider) {
		if rc, ok := p.httpClient.Transport.(*retryablehttp.RoundTripper); ok {
			if maxRetries >= 0 {
				rc.Client.RetryMax = maxRetries
			}
			if minWait > 0 {
				rc.Client.RetryWaitMin = minWait
			}
			if maxWait > 0 {
				rc.Client.RetryWaitMax = maxWait
			}
		}
	}
}

func NewProvider(apiKey, apiBase, proxy string, opts ...Option) *Provider {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 30 * time.Second
	retryClient.Backoff = retryablehttp.LinearJitterBackoff
	retryClient.Logger = nil

	transport := &http.Transport{}
	if proxy != "" {
		if parsed, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		} else {
			log.Printf("openai_compat: invalid proxy URL %q: %v", proxy, err)
		}
	}

	retryClient.HTTPClient.Transport = transport
	retryClient.HTTPClient.Timeout = defaultRequestTimeout

	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: retryClient.StandardClient(),
	}

	p.httpClient.Timeout = defaultRequestTimeout

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	if p.maxTokensField == "" {
		p.maxTokensField = "max_tokens"
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
		requestBody["tools"] = buildToolsList(tools, nativeSearch)
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok {
		requestBody[p.maxTokensField] = maxTokens
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

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// continue processing response
	}

	// Non-200: read a prefix to tell HTML error page apart from JSON error body.
	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	return common.ReadAndParseResponse(resp, p.apiBase)
}

func normalizeModel(model, apiBase string) string {
	if strings.Contains(strings.ToLower(apiBase), "openrouter.ai") {
		return model
	}

	before, after, ok := strings.Cut(model, "/")
	if !ok {
		return model
	}

	if _, exists := vendorPrefixes[strings.ToLower(before)]; exists {
		return after
	}

	return model
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
