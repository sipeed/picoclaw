// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/providers/openai_compat"
)

type HTTPProvider struct {
	delegate *openai_compat.Provider
}

func NewHTTPProvider(apiKey, apiBase, proxy string) *HTTPProvider {
	return &HTTPProvider{
		delegate: openai_compat.NewProvider(apiKey, apiBase, proxy),
	}
}

func (p *HTTPProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	resp, err := p.delegate.Chat(ctx, messages, tools, model, options)
	if err != nil {
		return nil, err
	}
	// If provider returned no structured tool_calls but Content has XML
	// tool call blocks (e.g. minimax), parse them as a fallback.
	if len(resp.ToolCalls) == 0 {
		if xmlCalls := extractXMLToolCalls(resp.Content); len(xmlCalls) > 0 {
			resp.ToolCalls = xmlCalls
		}
	}
	// Strip XML tool call artifacts from Content regardless.
	resp.Content = stripXMLToolCalls(resp.Content)
	return resp, nil
}

func (p *HTTPProvider) GetDefaultModel() string {
	return ""
}
