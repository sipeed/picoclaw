package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/sipeed/picoclaw/pkg/auth"
)

type ClaudeProvider struct {
	client      *anthropic.Client
	tokenSource func() (string, error)
}

func NewClaudeProvider(token string) *ClaudeProvider {
	client := anthropic.NewClient(
		option.WithAuthToken(token),
		option.WithBaseURL("https://api.anthropic.com"),
	)
	return &ClaudeProvider{client: &client}
}

func NewClaudeProviderWithTokenSource(token string, tokenSource func() (string, error)) *ClaudeProvider {
	p := NewClaudeProvider(token)
	p.tokenSource = tokenSource
	return p
}

func (p *ClaudeProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	var opts []option.RequestOption
	if p.tokenSource != nil {
		tok, err := p.tokenSource()
		if err != nil {
			return nil, fmt.Errorf("refreshing token: %w", err)
		}
		opts = append(opts, option.WithAuthToken(tok))
	}

	params, err := buildClaudeParams(messages, tools, model, options)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Messages.New(ctx, params, opts...)
	if err != nil {
		return nil, fmt.Errorf("claude API call: %w", err)
	}

	return parseClaudeResponse(resp), nil
}

func (p *ClaudeProvider) GetDefaultModel() string {
	return "claude-sonnet-4-5-20250929"
}

// buildContentBlocks converts Content (interface{}) to Claude content blocks.
// Handles both string content and multipart content (text + images).
func buildContentBlocks(content interface{}) []anthropic.ContentBlockParamUnion {
	if content == nil {
		return []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock("")}
	}

	// Try string content first
	if s, ok := content.(string); ok {
		return []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(s)}
	}

	// Try multipart content ([]interface{})
	if parts, ok := content.([]interface{}); ok {
		var blocks []anthropic.ContentBlockParamUnion

		for _, part := range parts {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}

			partType, _ := partMap["type"].(string)
			switch partType {
			case "text":
				if text, ok := partMap["text"].(string); ok {
					blocks = append(blocks, anthropic.NewTextBlock(text))
				}
			case "image_url":
				if imageURL, ok := partMap["image_url"].(map[string]interface{}); ok {
					if url, ok := imageURL["url"].(string); ok {
						// Parse data URL: data:image/jpeg;base64,<data>
						if strings.HasPrefix(url, "data:") {
							parts := strings.SplitN(url, ",", 2)
							if len(parts) == 2 {
								// Extract media type from data URL
								mediaType := anthropic.Base64ImageSourceMediaTypeImageJPEG // default
								if strings.Contains(parts[0], ";") {
									mediaTypePart := strings.Split(parts[0], ";")[0]
									if strings.HasPrefix(mediaTypePart, "data:") {
										mimeType := mediaTypePart[5:]
										switch mimeType {
										case "image/png":
											mediaType = anthropic.Base64ImageSourceMediaTypeImagePNG
										case "image/gif":
											mediaType = anthropic.Base64ImageSourceMediaTypeImageGIF
										case "image/webp":
											mediaType = anthropic.Base64ImageSourceMediaTypeImageWebP
										default:
											mediaType = anthropic.Base64ImageSourceMediaTypeImageJPEG
										}
									}
								}
								imageSource := anthropic.Base64ImageSourceParam{
									Data:      parts[1],
									MediaType: mediaType,
								}
								blocks = append(blocks, anthropic.NewImageBlock(imageSource))
							}
						}
					}
				}
			}
		}

		if len(blocks) > 0 {
			return blocks
		}
	}

	// Fallback to empty text block
	return []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock("")}
}

func buildClaudeParams(messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (anthropic.MessageNewParams, error) {
	var system []anthropic.TextBlockParam
	var anthropicMessages []anthropic.MessageParam

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemText := ContentToString(msg.Content)
			system = append(system, anthropic.TextBlockParam{Text: systemText})
		case "user":
			if msg.ToolCallID != "" {
				// Tool result
				resultText := ContentToString(msg.Content)
				anthropicMessages = append(anthropicMessages,
					anthropic.NewUserMessage(anthropic.NewToolResultBlock(msg.ToolCallID, resultText, false)),
				)
			} else {
				// Regular user message (may include images)
				blocks := buildContentBlocks(msg.Content)
				anthropicMessages = append(anthropicMessages,
					anthropic.NewUserMessage(blocks...),
				)
			}
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				var blocks []anthropic.ContentBlockParamUnion
				contentText := ContentToString(msg.Content)
				if contentText != "" {
					blocks = append(blocks, anthropic.NewTextBlock(contentText))
				}
				for _, tc := range msg.ToolCalls {
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Arguments, tc.Name))
				}
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(blocks...))
			} else {
				contentText := ContentToString(msg.Content)
				anthropicMessages = append(anthropicMessages,
					anthropic.NewAssistantMessage(anthropic.NewTextBlock(contentText)),
				)
			}
		case "tool":
			resultText := ContentToString(msg.Content)
			anthropicMessages = append(anthropicMessages,
				anthropic.NewUserMessage(anthropic.NewToolResultBlock(msg.ToolCallID, resultText, false)),
			)
		}
	}

	maxTokens := int64(4096)
	if mt, ok := options["max_tokens"].(int); ok {
		maxTokens = int64(mt)
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		Messages:  anthropicMessages,
		MaxTokens: maxTokens,
	}

	if len(system) > 0 {
		params.System = system
	}

	if temp, ok := options["temperature"].(float64); ok {
		params.Temperature = anthropic.Float(temp)
	}

	if len(tools) > 0 {
		params.Tools = translateToolsForClaude(tools)
	}

	return params, nil
}

func translateToolsForClaude(tools []ToolDefinition) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		tool := anthropic.ToolParam{
			Name: t.Function.Name,
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: t.Function.Parameters["properties"],
			},
		}
		if desc := t.Function.Description; desc != "" {
			tool.Description = anthropic.String(desc)
		}
		if req, ok := t.Function.Parameters["required"].([]interface{}); ok {
			required := make([]string, 0, len(req))
			for _, r := range req {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
			tool.InputSchema.Required = required
		}
		result = append(result, anthropic.ToolUnionParam{OfTool: &tool})
	}
	return result
}

func parseClaudeResponse(resp *anthropic.Message) *LLMResponse {
	var content string
	var toolCalls []ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			tb := block.AsText()
			content += tb.Text
		case "tool_use":
			tu := block.AsToolUse()
			var args map[string]interface{}
			if err := json.Unmarshal(tu.Input, &args); err != nil {
				args = map[string]interface{}{"raw": string(tu.Input)}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        tu.ID,
				Name:      tu.Name,
				Arguments: args,
			})
		}
	}

	finishReason := "stop"
	switch resp.StopReason {
	case anthropic.StopReasonToolUse:
		finishReason = "tool_calls"
	case anthropic.StopReasonMaxTokens:
		finishReason = "length"
	case anthropic.StopReasonEndTurn:
		finishReason = "stop"
	}

	return &LLMResponse{
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage: &UsageInfo{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}

func createClaudeTokenSource() func() (string, error) {
	return func() (string, error) {
		cred, err := auth.GetCredential("anthropic")
		if err != nil {
			return "", fmt.Errorf("loading auth credentials: %w", err)
		}
		if cred == nil {
			return "", fmt.Errorf("no credentials for anthropic. Run: picoclaw auth login --provider anthropic")
		}
		return cred.AccessToken, nil
	}
}
