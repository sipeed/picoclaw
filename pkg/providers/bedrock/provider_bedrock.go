//go:build bedrock

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package bedrock implements the LLM provider interface for AWS Bedrock.
// It uses the Bedrock Runtime Converse API for unified access to multiple
// model families (Claude, Llama, Mistral, etc.) with tool/function calling support.
package bedrock

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/sipeed/picoclaw/pkg/providers/common"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	ToolCall               = protocoltypes.ToolCall
	FunctionCall           = protocoltypes.FunctionCall
	LLMResponse            = protocoltypes.LLMResponse
	UsageInfo              = protocoltypes.UsageInfo
	Message                = protocoltypes.Message
	ToolDefinition         = protocoltypes.ToolDefinition
	ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition
)

// Provider implements the LLM provider interface for AWS Bedrock.
type Provider struct {
	client         *bedrockruntime.Client
	region         string
	requestTimeout time.Duration
}

// Option configures the Bedrock Provider.
type Option func(*providerConfig)

type providerConfig struct {
	region         string
	profile        string
	baseEndpoint   string
	requestTimeout time.Duration
}

// WithRegion sets the AWS region for Bedrock requests.
func WithRegion(region string) Option {
	return func(c *providerConfig) {
		c.region = region
	}
}

// WithProfile sets the AWS profile to use for credentials.
func WithProfile(profile string) Option {
	return func(c *providerConfig) {
		c.profile = profile
	}
}

// WithBaseEndpoint sets a custom Bedrock endpoint URL.
// Example: https://bedrock-runtime.us-east-1.amazonaws.com
func WithBaseEndpoint(endpoint string) Option {
	return func(c *providerConfig) {
		c.baseEndpoint = endpoint
	}
}

// WithRequestTimeout sets the timeout for Bedrock API requests.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(c *providerConfig) {
		c.requestTimeout = timeout
	}
}

// NewProvider creates a new AWS Bedrock provider.
// It uses the default AWS credential chain (env vars, shared config, IAM roles, etc.).
func NewProvider(ctx context.Context, opts ...Option) (*Provider, error) {
	pc := &providerConfig{}
	for _, opt := range opts {
		opt(pc)
	}

	// Build AWS config options
	var configOpts []func(*config.LoadOptions) error

	if pc.region != "" {
		configOpts = append(configOpts, config.WithRegion(pc.region))
	}

	if pc.profile != "" {
		configOpts = append(configOpts, config.WithSharedConfigProfile(pc.profile))
	}

	// Load AWS config with automatic credential discovery
	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	// Validate region is set - required for Bedrock request signing
	if cfg.Region == "" {
		return nil, fmt.Errorf(
			"AWS region not configured: set AWS_REGION, AWS_DEFAULT_REGION, or use WithRegion option",
		)
	}

	// Build client options
	var clientOpts []func(*bedrockruntime.Options)
	if pc.baseEndpoint != "" {
		clientOpts = append(clientOpts, func(o *bedrockruntime.Options) {
			o.BaseEndpoint = aws.String(pc.baseEndpoint)
		})
	}

	client := bedrockruntime.NewFromConfig(cfg, clientOpts...)

	return &Provider{
		client:         client,
		region:         cfg.Region,
		requestTimeout: pc.requestTimeout,
	}, nil
}

// Chat sends messages to AWS Bedrock using the Converse API.
func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	// Apply request timeout if context doesn't already have a deadline.
	// Use explicit timeout if set, otherwise fall back to common default.
	effectiveTimeout := p.requestTimeout
	if effectiveTimeout <= 0 {
		effectiveTimeout = common.DefaultRequestTimeout
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, effectiveTimeout)
		defer cancel()
	}

	// Build the Converse API input
	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(model),
	}

	// Convert messages to Bedrock format
	bedrockMessages, systemPrompts := convertMessages(messages)
	input.Messages = bedrockMessages

	// Set system prompts if any
	if len(systemPrompts) > 0 {
		input.System = systemPrompts
	}

	// Set inference configuration only when options are provided
	var inferenceConfig *types.InferenceConfiguration

	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok && maxTokens > 0 {
		if inferenceConfig == nil {
			inferenceConfig = &types.InferenceConfiguration{}
		}
		// Clamp to int32 range to avoid overflow
		if maxTokens > math.MaxInt32 {
			maxTokens = math.MaxInt32
		}
		inferenceConfig.MaxTokens = aws.Int32(int32(maxTokens))
	}

	if temp, ok := common.AsFloat(options["temperature"]); ok {
		if inferenceConfig == nil {
			inferenceConfig = &types.InferenceConfiguration{}
		}
		inferenceConfig.Temperature = aws.Float32(float32(temp))
	}

	if inferenceConfig != nil {
		input.InferenceConfig = inferenceConfig
	}

	// Convert tools to Bedrock format
	// Only set ToolConfig if at least one valid tool was produced
	if len(tools) > 0 {
		toolConfig := convertTools(tools)
		if len(toolConfig.Tools) > 0 {
			input.ToolConfig = toolConfig
		}
	}

	// Call Bedrock Converse API
	output, err := p.client.Converse(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("bedrock converse: %w", err)
	}

	// Parse the response
	return parseResponse(output)
}

// GetDefaultModel returns an empty string as Bedrock models are user-configured.
func (p *Provider) GetDefaultModel() string {
	return ""
}

// Region returns the AWS region configured for this Provider.
func (p *Provider) Region() string {
	return p.region
}

// convertMessages converts internal messages to Bedrock Converse format.
// Returns the conversation messages and any system prompts separately.
// Note: Bedrock requires all tool results for a given assistant turn to be in a single
// user message with multiple ToolResultBlock content blocks. This function merges
// consecutive tool result messages accordingly.
func convertMessages(messages []Message) ([]types.Message, []types.SystemContentBlock) {
	var bedrockMessages []types.Message
	var systemPrompts []types.SystemContentBlock

	// Helper to check if a message is a tool result
	isToolResult := func(msg Message) bool {
		return (msg.Role == "tool" || (msg.Role == "user" && msg.ToolCallID != "")) && msg.ToolCallID != ""
	}

	// Helper to create a tool result content block
	makeToolResultBlock := func(msg Message) types.ContentBlock {
		return &types.ContentBlockMemberToolResult{
			Value: types.ToolResultBlock{
				ToolUseId: aws.String(msg.ToolCallID),
				Content: []types.ToolResultContentBlock{
					&types.ToolResultContentBlockMemberText{
						Value: msg.Content,
					},
				},
			},
		}
	}

	i := 0
	for i < len(messages) {
		msg := messages[i]

		switch {
		case msg.Role == "system":
			// System messages go to the System field
			systemPrompts = append(systemPrompts, &types.SystemContentBlockMemberText{
				Value: msg.Content,
			})
			i++

		case isToolResult(msg):
			// Collect all consecutive tool results into a single user message
			// Bedrock requires all tool results for a turn in one message
			var toolResultBlocks []types.ContentBlock
			for i < len(messages) && isToolResult(messages[i]) {
				toolResultBlocks = append(toolResultBlocks, makeToolResultBlock(messages[i]))
				i++
			}
			bedrockMessages = append(bedrockMessages, types.Message{
				Role:    types.ConversationRoleUser,
				Content: toolResultBlocks,
			})

		case msg.Role == "user":
			// Regular user message (no ToolCallID)
			content := buildUserContent(msg)
			bedrockMessages = append(bedrockMessages, types.Message{
				Role:    types.ConversationRoleUser,
				Content: content,
			})
			i++

		case msg.Role == "assistant":
			content := buildAssistantContent(msg)
			bedrockMessages = append(bedrockMessages, types.Message{
				Role:    types.ConversationRoleAssistant,
				Content: content,
			})
			i++

		case msg.Role == "tool" && msg.ToolCallID == "":
			// Tool message without ToolCallID - treat as regular user message
			content := buildUserContent(msg)
			bedrockMessages = append(bedrockMessages, types.Message{
				Role:    types.ConversationRoleUser,
				Content: content,
			})
			i++

		default:
			// Unknown role - skip
			i++
		}
	}

	return bedrockMessages, systemPrompts
}

// buildUserContent builds Bedrock content blocks for a user message.
func buildUserContent(msg Message) []types.ContentBlock {
	var content []types.ContentBlock

	// Add text content
	if msg.Content != "" {
		content = append(content, &types.ContentBlockMemberText{
			Value: msg.Content,
		})
	}

	// Add images from Media field
	for _, mediaURL := range msg.Media {
		if strings.HasPrefix(mediaURL, "data:image/") {
			// Parse data URL: data:image/jpeg;base64,<data>
			parts := strings.SplitN(mediaURL, ",", 2)
			if len(parts) != 2 {
				continue
			}

			// Extract media type from "data:image/jpeg;base64"
			mediaType := ""
			header := parts[0]
			if idx := strings.Index(header, "/"); idx != -1 {
				end := strings.Index(header[idx:], ";")
				if end == -1 {
					end = len(header) - idx
				}
				mediaType = header[idx+1 : idx+end]
			}

			// Verify this is base64 encoded
			if !strings.Contains(header, ";base64") {
				continue // Skip non-base64 encoded data
			}

			// Map media type to Bedrock format
			var format types.ImageFormat
			switch mediaType {
			case "jpeg", "jpg":
				format = types.ImageFormatJpeg
			case "png":
				format = types.ImageFormatPng
			case "gif":
				format = types.ImageFormatGif
			case "webp":
				format = types.ImageFormatWebp
			default:
				continue // Skip unsupported formats
			}

			// Check size before decoding to prevent excessive memory allocation
			// Bedrock has a ~20MB request limit; cap decoded images at 10MB
			const maxImageSize = 10 * 1024 * 1024
			decodedLen := base64.StdEncoding.DecodedLen(len(parts[1]))
			if decodedLen > maxImageSize {
				log.Printf("bedrock: skipping image exceeding size limit (%d bytes > %d)", decodedLen, maxImageSize)
				continue
			}

			// Decode base64 data
			imageData, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				log.Printf("bedrock: failed to decode base64 image data: %v", err)
				continue
			}

			content = append(content, &types.ContentBlockMemberImage{
				Value: types.ImageBlock{
					Format: format,
					Source: &types.ImageSourceMemberBytes{
						Value: imageData,
					},
				},
			})
		}
	}

	// Bedrock requires at least one content block; add empty text if needed
	if len(content) == 0 {
		content = append(content, &types.ContentBlockMemberText{Value: ""})
	}

	return content
}

// buildAssistantContent builds Bedrock content blocks for an assistant message.
func buildAssistantContent(msg Message) []types.ContentBlock {
	var content []types.ContentBlock

	// Add text content if present
	if msg.Content != "" {
		content = append(content, &types.ContentBlockMemberText{
			Value: msg.Content,
		})
	}

	// Add tool use blocks
	for _, tc := range msg.ToolCalls {
		// Validate tool call ID - Bedrock requires non-empty ToolUseId
		if strings.TrimSpace(tc.ID) == "" {
			log.Printf("bedrock: skipping tool call with empty ID (name: %q)", tc.Name)
			continue
		}

		// Resolve tool name: prefer tc.Name, fallback to tc.Function.Name
		// (tc.Name/tc.Arguments are json:"-" and may be empty when from JSON)
		toolName := tc.Name
		if toolName == "" && tc.Function != nil {
			toolName = tc.Function.Name
		}
		if strings.TrimSpace(toolName) == "" {
			continue
		}

		// Resolve arguments: prefer tc.Arguments, fallback to parsing tc.Function.Arguments
		args := tc.Arguments
		if args == nil && tc.Function != nil && tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				log.Printf("bedrock: failed to parse Function.Arguments for tool %q: %v", toolName, err)
				args = map[string]any{}
			}
		}
		if args == nil {
			args = map[string]any{}
		}

		// Convert arguments to a Bedrock document using NewLazyDocument
		inputDoc := document.NewLazyDocument(args)

		content = append(content, &types.ContentBlockMemberToolUse{
			Value: types.ToolUseBlock{
				ToolUseId: aws.String(tc.ID),
				Name:      aws.String(toolName),
				Input:     inputDoc,
			},
		})
	}

	// Bedrock requires at least one content block; add empty text if needed
	if len(content) == 0 {
		content = append(content, &types.ContentBlockMemberText{Value: ""})
	}

	return content
}

// convertTools converts tool definitions to Bedrock format.
func convertTools(tools []ToolDefinition) *types.ToolConfiguration {
	bedrockTools := make([]types.Tool, 0, len(tools))

	for _, tool := range tools {
		// Skip tools with empty names
		if strings.TrimSpace(tool.Function.Name) == "" {
			continue
		}

		// Ensure parameters is not nil - default to minimal object schema
		params := tool.Function.Parameters
		if params == nil {
			params = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}

		// Convert parameters schema to a Bedrock document
		inputSchema := document.NewLazyDocument(params)

		bedrockTools = append(bedrockTools, &types.ToolMemberToolSpec{
			Value: types.ToolSpecification{
				Name:        aws.String(tool.Function.Name),
				Description: aws.String(tool.Function.Description),
				InputSchema: &types.ToolInputSchemaMemberJson{
					Value: inputSchema,
				},
			},
		})
	}

	return &types.ToolConfiguration{
		Tools: bedrockTools,
	}
}

// parseResponse converts Bedrock Converse output to LLMResponse.
func parseResponse(output *bedrockruntime.ConverseOutput) (*LLMResponse, error) {
	var content strings.Builder
	toolCalls := make([]ToolCall, 0)

	// Process output content blocks
	if output.Output != nil {
		if msgOutput, ok := output.Output.(*types.ConverseOutputMemberMessage); ok {
			for _, block := range msgOutput.Value.Content {
				switch b := block.(type) {
				case *types.ContentBlockMemberText:
					content.WriteString(b.Value)

				case *types.ContentBlockMemberToolUse:
					// Unmarshal the document interface to a map
					args := make(map[string]any)
					if b.Value.Input != nil {
						if err := b.Value.Input.UnmarshalSmithyDocument(&args); err != nil {
							log.Printf("bedrock: failed to unmarshal tool input for tool %q (id %q): %v",
								aws.ToString(b.Value.Name),
								aws.ToString(b.Value.ToolUseId),
								err,
							)
							args = make(map[string]any)
						}
					}

					// Serialize arguments to JSON string for FunctionCall
					argsJSON, err := json.Marshal(args)
					if err != nil {
						log.Printf("bedrock: failed to marshal tool arguments for tool %q (id %q): %v",
							aws.ToString(b.Value.Name),
							aws.ToString(b.Value.ToolUseId),
							err,
						)
						argsJSON = []byte("{}")
					}

					toolCalls = append(toolCalls, ToolCall{
						ID:        aws.ToString(b.Value.ToolUseId),
						Name:      aws.ToString(b.Value.Name),
						Arguments: args,
						Function: &FunctionCall{
							Name:      aws.ToString(b.Value.Name),
							Arguments: string(argsJSON),
						},
					})
				}
			}
		}
	}

	// Map stop reason
	finishReason := "stop"
	switch output.StopReason {
	case types.StopReasonToolUse:
		finishReason = "tool_calls"
	case types.StopReasonMaxTokens:
		finishReason = "length"
	case types.StopReasonEndTurn:
		finishReason = "stop"
	case types.StopReasonStopSequence:
		finishReason = "stop"
	case types.StopReasonContentFiltered:
		finishReason = "content_filter"
	}

	// Build usage info
	var usage *UsageInfo
	if output.Usage != nil {
		usage = &UsageInfo{
			PromptTokens:     int(aws.ToInt32(output.Usage.InputTokens)),
			CompletionTokens: int(aws.ToInt32(output.Usage.OutputTokens)),
			TotalTokens:      int(aws.ToInt32(output.Usage.InputTokens)) + int(aws.ToInt32(output.Usage.OutputTokens)),
		}
	}

	return &LLMResponse{
		Content:      content.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}
