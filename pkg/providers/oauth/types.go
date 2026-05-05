package oauthprovider

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	ToolCall                = protocoltypes.ToolCall
	FunctionCall            = protocoltypes.FunctionCall
	LLMResponse             = protocoltypes.LLMResponse
	UsageInfo               = protocoltypes.UsageInfo
	Message                 = protocoltypes.Message
	ToolDefinition          = protocoltypes.ToolDefinition
	ToolFunctionDefinition  = protocoltypes.ToolFunctionDefinition
	ExtraContent            = protocoltypes.ExtraContent
	GoogleExtra             = protocoltypes.GoogleExtra
	ContentBlock            = protocoltypes.ContentBlock
	CacheControl            = protocoltypes.CacheControl
	ImageGenerationRequest  = protocoltypes.ImageGenerationRequest
	GeneratedImage          = protocoltypes.GeneratedImage
	ImageGenerationResponse = protocoltypes.ImageGenerationResponse
)

type LLMProvider interface {
	Chat(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
		model string,
		options map[string]any,
	) (*LLMResponse, error)
	GetDefaultModel() string
}
