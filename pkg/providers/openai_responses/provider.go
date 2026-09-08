// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package openairesponses implements the OpenAI Responses API over plain HTTP
// with Bearer API-key auth. It targets OpenAI-compatible gateways that expose
// a /responses endpoint (e.g. OpenCode Go) without requiring OAuth.
package openairesponses

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
	ToolCall               = protocoltypes.ToolCall
	FunctionCall           = protocoltypes.FunctionCall
	LLMResponse            = protocoltypes.LLMResponse
	UsageInfo              = protocoltypes.UsageInfo
	Message                = protocoltypes.Message
	ToolDefinition         = protocoltypes.ToolDefinition
	ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition
)

const defaultRequestTimeout = 120 * time.Second

// Provider implements the OpenAI Responses API via HTTP with an API key.
type Provider struct {
	apiKey            string
	apiBase           string
	httpClient        *http.Client
	customHeaders     map[string]string
	sessionHeaderName string // Optional per-conversation session header (e.g. x-opencode-session)
	userAgent         string
	defaultModel      string
}

// Option configures a Provider.
type Option func(*Provider)

// WithRequestTimeout sets the HTTP request timeout.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

// WithCustomHeaders injects additional headers into every HTTP request.
func WithCustomHeaders(customHeaders map[string]string) Option {
	return func(p *Provider) {
		p.customHeaders = customHeaders
	}
}

// WithSessionHeaderName enables a per-conversation session header whose value
// comes from options["session_key"]. An explicit custom_headers value wins.
func WithSessionHeaderName(headerName string) Option {
	return func(p *Provider) {
		p.sessionHeaderName = strings.TrimSpace(headerName)
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(userAgent string) Option {
	return func(p *Provider) {
		p.userAgent = userAgent
	}
}

// WithDefaultModel sets the model returned by GetDefaultModel.
func WithDefaultModel(model string) Option {
	return func(p *Provider) {
		p.defaultModel = model
	}
}

// NewProvider creates a Responses API provider. apiBase should be the full
// endpoint including the /responses path segment (e.g.
// https://opencode.ai/zen/go/v1/responses).
func NewProvider(apiKey, apiBase, userAgent string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: common.NewHTTPClient(""),
		userAgent:  userAgent,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	return p
}

// NewProviderWithTimeout creates a provider with a proxy and request timeout.
func NewProviderWithTimeout(
	apiKey, apiBase, proxy, userAgent string,
	requestTimeoutSeconds int,
	opts ...Option,
) *Provider {
	p := NewProvider(apiKey, apiBase, userAgent, opts...)
	p.httpClient = common.NewHTTPClient(proxy)
	if requestTimeoutSeconds > 0 {
		p.httpClient.Timeout = time.Duration(requestTimeoutSeconds) * time.Second
	}
	return p
}

// Chat sends a request to the Responses API endpoint.
func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("API key not configured")
	}
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	requestURL, err := url.JoinPath(p.apiBase, "responses")
	if err != nil {
		return nil, fmt.Errorf("failed to build request URL: %w", err)
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
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}
	for k, v := range p.customHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	if p.sessionHeaderName != "" {
		common.ApplySessionHeader(req.Header, options, p.sessionHeaderName)
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

// GetDefaultModel returns the configured default model (may be empty).
func (p *Provider) GetDefaultModel() string {
	return p.defaultModel
}
