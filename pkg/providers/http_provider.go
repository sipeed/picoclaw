// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"context"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers/openai_compat"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type HTTPProvider struct {
	delegate *openai_compat.Provider
}

func NewHTTPProvider(apiKey, apiBase, proxy string) *HTTPProvider {
	return &HTTPProvider{
		delegate: openai_compat.NewProvider(apiKey, apiBase, proxy),
	}
}

func NewHTTPProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField string) *HTTPProvider {
	return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(apiKey, apiBase, proxy, maxTokensField, 0)
}

func NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
	apiKey, apiBase, proxy, maxTokensField string,
	requestTimeoutSeconds int,
) *HTTPProvider {
	return &HTTPProvider{
		delegate: openai_compat.NewProvider(
			apiKey,
			apiBase,
			proxy,
			openai_compat.WithMaxTokensField(maxTokensField),
			openai_compat.WithRequestTimeout(time.Duration(requestTimeoutSeconds)*time.Second),
		),
	}
}

// NewHTTPProviderFromConfig creates an HTTPProvider from a ModelConfig,
// honoring all optional fields including stream.
func NewHTTPProviderFromConfig(cfg *config.ModelConfig, apiBase string) *HTTPProvider {
	opts := []openai_compat.Option{
		openai_compat.WithMaxTokensField(cfg.MaxTokensField),
		openai_compat.WithRequestTimeout(time.Duration(cfg.RequestTimeout) * time.Second),
	}
	if cfg.Stream != nil && *cfg.Stream {
		opts = append(opts, openai_compat.WithStream(true))
	}
	return &HTTPProvider{
		delegate: openai_compat.NewProvider(cfg.APIKey, apiBase, cfg.Proxy, opts...),
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

// CanStream returns true when SSE streaming is enabled.
func (p *HTTPProvider) CanStream() bool {
	return p.delegate.CanStream()
}

// ChatStream opens an SSE connection and returns a channel of StreamEvent.
func (p *HTTPProvider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (<-chan protocoltypes.StreamEvent, error) {
	return p.delegate.ChatStream(ctx, messages, tools, model, options)
}

func (p *HTTPProvider) GetDefaultModel() string {
	return ""
}

func (p *HTTPProvider) SupportsNativeSearch() bool {
	return p.delegate.SupportsNativeSearch()
}
