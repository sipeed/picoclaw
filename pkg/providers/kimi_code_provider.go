// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"context"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/openai_compat"
)

// KimiCodeProvider implements LLMProvider for Kimi For Coding API.
// It uses OpenAI-compatible format with custom User-Agent headers
// to identify as a Coding Agent.
type KimiCodeProvider struct {
	delegate *openai_compat.Provider
}

// KimiCodeDefaultBaseURL is the default API base URL for Kimi For Coding.
const KimiCodeDefaultBaseURL = "https://api.kimi.com/coding/v1"

// NewKimiCodeProvider creates a new Kimi For Coding provider with default settings.
// It automatically sets the User-Agent to identify as a Coding Agent.
func NewKimiCodeProvider(apiKey, apiBase, proxy string) *KimiCodeProvider {
	return NewKimiCodeProviderWithTimeout(apiKey, apiBase, proxy, 0)
}

// NewKimiCodeProviderWithTimeout creates a Kimi For Coding provider with custom request timeout.
func NewKimiCodeProviderWithTimeout(apiKey, apiBase, proxy string, timeoutSeconds int) *KimiCodeProvider {
	// Use default base URL if not provided
	base := apiBase
	if base == "" {
		base = KimiCodeDefaultBaseURL
	}

	// Kimi For Coding API requires specific User-Agent headers to identify as an approved Coding Agent.
	// Currently, Kimi only allows access from officially recognized agents like Kimi CLI, Claude Code, Roo Code, etc.
	// See: https://www.kimi.com/code/docs/en/
	//
	// FIXME: This is a temporary workaround using Roo Code's User-Agent to access the API.
	// We have opened an issue to request official support for PicoClaw as a recognized Coding Agent.
	// Once approved by Kimi, this should be changed to use PicoClaw's own User-Agent.
	//
	// WARNING: This workaround may violate Kimi's Terms of Service. Use at your own risk.
	// If you have concerns, please consider using the official Kimi CLI instead.
	//
	// Related:
	// - Roo Code: https://github.com/Roo-Code/Roo-Code (Apache 2.0 License)
	// - Kimi For Coding: https://www.kimi.com/code
	headers := map[string]string{
		"User-Agent": "RooCode/3.0.0", // Borrowing Roo Code's identity temporarily
		"X-App":      "cli",
	}

	timeout := 120 * time.Second
	if timeoutSeconds > 0 {
		timeout = time.Duration(timeoutSeconds) * time.Second
	}

	return &KimiCodeProvider{
		delegate: openai_compat.NewProvider(
			apiKey,
			base,
			proxy,
			openai_compat.WithHeaders(headers),
			openai_compat.WithRequestTimeout(timeout),
		),
	}
}

// Chat sends messages to Kimi For Coding API and returns the response.
func (p *KimiCodeProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	return p.delegate.Chat(ctx, messages, tools, model, options)
}

// GetDefaultModel returns the default model for this provider.
func (p *KimiCodeProvider) GetDefaultModel() string {
	return "kimi-for-coding"
}
