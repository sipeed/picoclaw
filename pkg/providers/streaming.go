// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import "context"

// StreamingProvider extends LLMProvider with streaming capability.
// Text tokens are delivered via onToken as they arrive from the API.
// Tool calls are accumulated internally and returned in the final *LLMResponse.
type StreamingProvider interface {
	LLMProvider
	ChatStream(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
		model string,
		options map[string]any,
		onToken func(string),
	) (*LLMResponse, error)
}
