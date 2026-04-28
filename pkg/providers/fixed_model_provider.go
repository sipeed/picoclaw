package providers

import (
	"context"
	"strings"
)

type fixedModelProvider struct {
	inner LLMProvider
	model string
}

func WithDefaultModel(provider LLMProvider, model string) LLMProvider {
	model = strings.TrimSpace(model)
	if provider == nil || model == "" {
		return provider
	}
	return &fixedModelProvider{
		inner: provider,
		model: model,
	}
}

func (p *fixedModelProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	return p.inner.Chat(ctx, messages, tools, model, options)
}

func (p *fixedModelProvider) GetDefaultModel() string {
	if p == nil {
		return ""
	}
	return p.model
}

func (p *fixedModelProvider) Close() {
	if inner, ok := p.inner.(StatefulProvider); ok {
		inner.Close()
	}
}

func (p *fixedModelProvider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
	onChunk func(accumulated string),
) (*LLMResponse, error) {
	streaming, ok := p.inner.(StreamingProvider)
	if !ok {
		return p.inner.Chat(ctx, messages, tools, model, options)
	}
	return streaming.ChatStream(ctx, messages, tools, model, options, onChunk)
}

func (p *fixedModelProvider) SupportsThinking() bool {
	thinking, ok := p.inner.(ThinkingCapable)
	return ok && thinking.SupportsThinking()
}

func (p *fixedModelProvider) SupportsNativeSearch() bool {
	nativeSearch, ok := p.inner.(NativeSearchCapable)
	return ok && nativeSearch.SupportsNativeSearch()
}
