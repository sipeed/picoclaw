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

	"github.com/sipeed/picoclaw/pkg/providers/common"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	LLMResponse    = protocoltypes.LLMResponse
	Message        = protocoltypes.Message
	ToolDefinition = protocoltypes.ToolDefinition
)

const (
	// azureAPIVersion is the Azure OpenAI API version used for all requests.
	azureAPIVersion       = "2024-10-21"
	defaultRequestTimeout = common.DefaultRequestTimeout
)

// Provider implements the LLM provider interface for Azure OpenAI endpoints.
// It handles Azure-specific authentication (api-key header), URL construction
// (deployment-based), and request body formatting (max_completion_tokens, no model field).
type Provider struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
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

// NewProvider creates a new Azure OpenAI provider.
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

// NewProviderWithTimeout creates a new Azure OpenAI provider with a custom request timeout in seconds.
func NewProviderWithTimeout(apiKey, apiBase, proxy string, requestTimeoutSeconds int) *Provider {
	return NewProvider(
		apiKey, apiBase, proxy,
		WithRequestTimeout(time.Duration(requestTimeoutSeconds)*time.Second),
	)
}

// Chat sends a chat completion request to the Azure OpenAI endpoint.
// The model parameter is used as the Azure deployment name in the URL.
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

	// model is the deployment name for Azure OpenAI
	deployment := model

	// Build Azure-specific URL safely using url.JoinPath and query encoding
	// to prevent path traversal or query injection via deployment names.
	base, err := url.JoinPath(p.apiBase, "openai/deployments", deployment, "chat/completions")
	if err != nil {
		return nil, fmt.Errorf("failed to build Azure request URL: %w", err)
	}
	requestURL := base + "?api-version=" + azureAPIVersion

	// Build request body — no "model" field (Azure infers from deployment URL)
	requestBody := map[string]any{
		"messages": common.SerializeMessages(messages),
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	// Azure OpenAI always uses max_completion_tokens
	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok {
		requestBody["max_completion_tokens"] = maxTokens
	}

	if temperature, ok := common.AsFloat(options["temperature"]); ok {
		requestBody["temperature"] = temperature
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Azure uses api-key header instead of Authorization: Bearer
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Api-Key", p.apiKey)
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

// GetDefaultModel returns an empty string as Azure deployments are user-configured.
func (p *Provider) GetDefaultModel() string {
	return ""
}
