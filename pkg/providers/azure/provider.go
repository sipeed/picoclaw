package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	"github.com/sipeed/picoclaw/pkg/providers/common"
	orc "github.com/sipeed/picoclaw/pkg/providers/openai_responses_common"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	LLMResponse    = protocoltypes.LLMResponse
	Message        = protocoltypes.Message
	ToolDefinition = protocoltypes.ToolDefinition
)

const (
	defaultRequestTimeout = common.DefaultRequestTimeout
	responsesAPIPath      = "openai/v1/responses"
)

// Provider implements the LLM provider interface for Azure OpenAI endpoints.
// It handles Azure-specific authentication (Bearer token), URL construction
// (Responses API), and request/response formatting.
type Provider struct {
	apiKey      string
	apiBase     string
	httpClient  *http.Client
	userAgent   string
	tokenSource func(ctx context.Context) (string, error)
}

// Option configures the Azure Provider.
type Option func(*Provider)

// WithRequestTimeout sets the HTTP request timeout.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

// WithUserAgent sets the User-Agent header for requests.
func WithUserAgent(userAgent string) Option {
	return func(p *Provider) {
		p.userAgent = userAgent
	}
}

// WithTokenSource sets a callback that returns a bearer token per request.
// When set, it takes precedence over the static api key.
func WithTokenSource(ts func(ctx context.Context) (string, error)) Option {
	return func(p *Provider) {
		p.tokenSource = ts
	}
}

// NewProvider creates a new Azure OpenAI provider.
func NewProvider(apiKey, apiBase, proxy, userAgent string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		userAgent:  userAgent,
		httpClient: common.NewHTTPClient(proxy),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	return p
}

// NewProviderWithTimeout creates a new Azure OpenAI provider with a custom request timeout in seconds.
func NewProviderWithTimeout(apiKey, apiBase, proxy, userAgent string, requestTimeoutSeconds int) *Provider {
	return NewProvider(
		apiKey, apiBase, proxy, userAgent,
		WithRequestTimeout(time.Duration(requestTimeoutSeconds)*time.Second),
	)
}

// NewProviderWithTokenSource creates a new Azure OpenAI provider that obtains its
// bearer token from the supplied callback on every request. Used for Entra ID auth
// where tokens are short-lived and refreshed by the underlying credential.
func NewProviderWithTokenSource(
	apiBase, proxy, userAgent string,
	tokenSource func(ctx context.Context) (string, error),
	opts ...Option,
) *Provider {
	p := &Provider{
		apiBase:     strings.TrimRight(apiBase, "/"),
		userAgent:   userAgent,
		httpClient:  common.NewHTTPClient(proxy),
		tokenSource: tokenSource,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	return p
}

// Chat sends a request to the Azure OpenAI Responses API endpoint.
// The model parameter is passed in the request body.
func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("Azure API base not configured")
	}

	requestURL, err := url.JoinPath(p.apiBase, responsesAPIPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build Azure request URL: %w", err)
	}

	input, instructions := orc.TranslateMessages(messages)

	requestBody := responses.ResponseNewParams{
		Model: model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
		Store: openai.Opt(false),
	}

	if instructions != "" {
		requestBody.Instructions = openai.Opt(instructions)
	}

	if len(tools) > 0 {
		enableWebSearch, _ := options["native_search"].(bool)
		requestBody.Tools = orc.TranslateTools(tools, enableWebSearch)
		requestBody.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto),
		}
	}

	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok {
		requestBody.MaxOutputTokens = openai.Opt(int64(maxTokens))
	}

	if temperature, ok := common.AsFloat(options["temperature"]); ok {
		requestBody.Temperature = openai.Opt(temperature)
	}

	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		requestBody.PromptCacheKey = openai.Opt(cacheKey)
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	switch {
	case p.tokenSource != nil:
		tok, tokErr := p.tokenSource(ctx)
		if tokErr != nil {
			return nil, fmt.Errorf("acquiring azure identity token: %w", tokErr)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	case p.apiKey != "":
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	return orc.ParseResponseBody(resp.Body)
}

// GetDefaultModel returns an empty string as Azure deployments are user-configured.
func (p *Provider) GetDefaultModel() string {
	return ""
}
