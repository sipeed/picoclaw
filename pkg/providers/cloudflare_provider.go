// CloudflareProvider is an LLM provider for Cloudflare AI Gateway.
//
// Cloudflare AI Gateway provides a unified OpenAI-compatible endpoint that
// proxies requests to multiple upstream AI providers (OpenAI, Anthropic,
// Google, Workers AI, etc.) through a single API.
//
// Endpoint format:
//
//	https://gateway.ai.cloudflare.com/v1/{account_id}/{gateway_id}/compat/chat/completions
//
// Authentication modes:
//   - Unified Billing: only cf_token required (Cloudflare pays the upstream provider)
//   - BYOK (Bring Your Own Key): api_key for the upstream provider + optional cf_token
//     for authenticated gateways
//
// Model format in the request body: "{provider}/{model}"
//
//	e.g., "openai/gpt-5.2", "anthropic/claude-sonnet-4.6",
//	      "workers-ai/@cf/openai/gpt-oss-120b"
//
// References:
//   - https://developers.cloudflare.com/ai-gateway/usage/chat-completion/
//   - https://developers.cloudflare.com/ai-gateway/features/unified-billing/
//   - https://developers.cloudflare.com/ai-gateway/configuration/authentication/
//   - https://developers.cloudflare.com/workers-ai/models/

package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/openai_compat"
)

type CloudflareProvider struct {
	delegate *openai_compat.Provider
}

// CfAIGAuthHeader is the HTTP header name used by Cloudflare AI Gateway
// for gateway-level authentication.
const CfAIGAuthHeader = "cf-aig-authorization"

// NewCloudflareProvider creates a new Cloudflare AI Gateway provider.
//
// Parameters:
//   - apiKey: upstream provider API key (for BYOK mode), sent as Authorization header.
//     Leave empty for Unified Billing mode.
//   - apiBase: the AI Gateway compat endpoint URL, e.g.,
//     "https://gateway.ai.cloudflare.com/v1/{account_id}/{gateway_id}/compat"
//   - cfToken: Cloudflare API token for gateway authentication, sent as
//     cf-aig-authorization header. Required for Unified Billing; optional for BYOK
//     (only needed if the gateway has authentication enabled).
//   - proxy: optional HTTP proxy URL.
//   - maxTokensField: optional field name override for max tokens.
//   - requestTimeoutSeconds: request timeout in seconds (0 for default).
func NewCloudflareProvider(
	apiKey, apiBase, cfToken, proxy, maxTokensField string,
	requestTimeoutSeconds int,
) (*CloudflareProvider, error) {
	if apiBase == "" {
		return nil, fmt.Errorf("api_base is required for Cloudflare AI Gateway")
	}

	// Build extra headers for Cloudflare AI Gateway authentication
	extraHeaders := make(map[string]string)
	if cfToken != "" {
		extraHeaders[CfAIGAuthHeader] = "Bearer " + cfToken
	}

	opts := []openai_compat.Option{
		openai_compat.WithMaxTokensField(maxTokensField),
		openai_compat.WithExtraHeaders(extraHeaders),
	}
	if requestTimeoutSeconds > 0 {
		opts = append(opts, openai_compat.WithRequestTimeout(
			time.Duration(requestTimeoutSeconds)*time.Second,
		))
	}

	delegate := openai_compat.NewProvider(apiKey, apiBase, proxy, opts...)

	return &CloudflareProvider{delegate: delegate}, nil
}

func (p *CloudflareProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	return p.delegate.Chat(ctx, messages, tools, model, options)
}

func (p *CloudflareProvider) GetDefaultModel() string {
	return ""
}
