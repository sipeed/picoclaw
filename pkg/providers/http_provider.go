// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/providers/anthropic_compat"
	"github.com/sipeed/picoclaw/pkg/providers/openai_compat"
)

// httpDelegate is the interface that both openai_compat and anthropic_compat providers implement
type httpDelegate interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any) (*LLMResponse, error)
}

type HTTPProvider struct {
	delegate httpDelegate
}

func NewHTTPProvider(apiKey, apiBase, proxy string) *HTTPProvider {
	return &HTTPProvider{
		delegate: openai_compat.NewProvider(apiKey, apiBase, proxy),
	}
}

func NewHTTPProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField string) *HTTPProvider {
	return &HTTPProvider{
		delegate: openai_compat.NewProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField),
	}
}

// NewHTTPProviderWithProtocol creates an HTTP provider with the specified protocol type.
// protocol should be "openai" for OpenAI-compatible APIs or "anthropic" for Anthropic-compatible APIs.
func NewHTTPProviderWithProtocol(apiKey, apiBase, proxy, protocol string) *HTTPProvider {
	switch protocol {
	case "anthropic":
		return &HTTPProvider{
			delegate: anthropic_compat.NewProvider(apiKey, apiBase, proxy),
		}
	default:
		return &HTTPProvider{
			delegate: openai_compat.NewProvider(apiKey, apiBase, proxy),
		}
	}
}

func (p *HTTPProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	return p.delegate.Chat(ctx, messages, tools, model, options)
}

func (p *HTTPProvider) GetDefaultModel() string {
	return ""
}
