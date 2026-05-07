//go:build !bedrock

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package bedrock provides a stub implementation when built without the bedrock tag.
// To enable AWS Bedrock support, build with: go build -tags bedrock
package bedrock

import (
	"context"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	LLMResponse    = protocoltypes.LLMResponse
	Message        = protocoltypes.Message
	ToolDefinition = protocoltypes.ToolDefinition
)

// Provider is a stub that returns an error when Bedrock support is not compiled in.
type Provider struct{}

// Option is a no-op when Bedrock is not enabled.
type Option func(*providerConfig)

type providerConfig struct{}

// WithRegion is a no-op when Bedrock is not enabled.
func WithRegion(region string) Option {
	return func(c *providerConfig) {}
}

// WithProfile is a no-op when Bedrock is not enabled.
func WithProfile(profile string) Option {
	return func(c *providerConfig) {}
}

// WithBaseEndpoint is a no-op when Bedrock is not enabled.
func WithBaseEndpoint(endpoint string) Option {
	return func(c *providerConfig) {}
}

// WithRequestTimeout is a no-op when Bedrock is not enabled.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(c *providerConfig) {}
}

// NewProvider returns an error indicating Bedrock support is not compiled in.
func NewProvider(ctx context.Context, opts ...Option) (*Provider, error) {
	return nil, fmt.Errorf("bedrock provider not available: build with -tags bedrock to enable AWS Bedrock support")
}

// Chat returns an error - this should never be called since NewProvider fails.
func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	return nil, fmt.Errorf("bedrock provider not available: build with -tags bedrock to enable AWS Bedrock support")
}

// GetDefaultModel returns an empty string.
func (p *Provider) GetDefaultModel() string {
	return ""
}
