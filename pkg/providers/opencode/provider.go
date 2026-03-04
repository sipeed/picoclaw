// Package opencode provides an OpenAI-compatible provider for OpenCode Zen API.
// OpenCode Zen uses different endpoints depending on the model:
//   - /responses for OpenAI models (e.g., gpt-5.3-codex)
//   - /messages for Anthropic models (e.g., claude-opus-4-6)
//   - /models/{model} for Gemini models (e.g., gemini-3.1-pro)
//   - /chat/completions for OpenAI-compatible models (e.g., kimi-k2.5)
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
	ReasoningDetail        = protocoltypes.ReasoningDetail
)

type Provider struct {
	apiKey         string
	apiBase        string
	maxTokensField string
	httpClient     *http.Client
}

type Option func(*Provider)

const defaultRequestTimeout = 120 * time.Second
const defaultAPIBase = "https://opencode.ai/zen/v1"

func WithMaxTokensField(maxTokensField string) Option {
	return func(p *Provider) {
		p.maxTokensField = maxTokensField
	}
}

func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

func NewProvider(apiKey, apiBase, proxy string, opts ...Option) *Provider {
	client := &http.Client{
		Timeout: defaultRequestTimeout,
	}

	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(parsed),
			}
		}
	}

	if apiBase == "" {
		apiBase = defaultAPIBase
	}

	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: client,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	return p
}

func NewProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField string) *Provider {
	return NewProvider(apiKey, apiBase, proxy, WithMaxTokensField(maxTokensField))
}

func NewProviderWithMaxTokensFieldAndTimeout(
	apiKey, apiBase, proxy, maxTokensField string,
	requestTimeoutSeconds int,
) *Provider {
	return NewProvider(
		apiKey,
		apiBase,
		proxy,
		WithMaxTokensField(maxTokensField),
		WithRequestTimeout(time.Duration(requestTimeoutSeconds)*time.Second),
	)
}

// GetDefaultModel returns the default model for this provider.
func (p *Provider) GetDefaultModel() string {
	return "kimi-k2.5"
}

// detectEndpoint determines which endpoint to use based on the model name.
// According to OpenCode Zen docs:
//   - OpenAI models (gpt-*): /responses
//   - Anthropic models (claude-*): /messages
//   - Gemini models (gemini-*): /models/{model}
//   - Other models: /chat/completions
func detectEndpoint(model string) string {
	lowerModel := strings.ToLower(model)

	// Anthropic/Claude models use /messages endpoint
	if strings.Contains(lowerModel, "claude") {
		return "/messages"
	}

	// OpenAI GPT models use /responses endpoint
	if strings.HasPrefix(lowerModel, "gpt-") {
		return "/responses"
	}

	// Gemini models use /models/{model} endpoint
	if strings.HasPrefix(lowerModel, "gemini-") {
		return "/models/" + model
	}

	// Default to /chat/completions for all other models (kimi, etc.)
	return "/chat/completions"
}

// isOpenAIResponsesEndpoint checks if we need to use OpenAI responses format
func isOpenAIResponsesEndpoint(endpoint string) bool {
	return endpoint == "/responses"
}

// isAnthropicMessagesEndpoint checks if we need to use Anthropic messages format
func isAnthropicMessagesEndpoint(endpoint string) bool {
	return endpoint == "/messages"
}

// isGeminiEndpoint checks if we need to use Gemini models format
func isGeminiEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "/models/")
}

func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	endpoint := detectEndpoint(model)

	switch {
	case isOpenAIResponsesEndpoint(endpoint):
		return p.chatOpenAIResponses(ctx, messages, tools, model, options)
	case isAnthropicMessagesEndpoint(endpoint):
		return p.chatAnthropicMessages(ctx, messages, tools, model, options)
	case isGeminiEndpoint(endpoint):
		return p.chatGeminiModels(ctx, messages, tools, model, endpoint, options)
	default:
		return p.chatOpenAICompatible(ctx, messages, tools, model, options)
	}
}

// chatOpenAICompatible uses standard OpenAI chat completions format
func (p *Provider) chatOpenAICompatible(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	requestBody := map[string]any{
		"model":    model,
		"messages": stripSystemParts(messages),
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		fieldName := p.maxTokensField
		if fieldName == "" {
			lowerModel := strings.ToLower(model)
			if strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "o1") ||
				strings.Contains(lowerModel, "gpt-5") {
				fieldName = "max_completion_tokens"
			} else {
				fieldName = "max_tokens"
			}
		}
		requestBody[fieldName] = maxTokens
	}

	if temperature, ok := asFloat(options["temperature"]); ok {
		lowerModel := strings.ToLower(model)
		// Kimi k2 models only support temperature=1.
		if strings.Contains(lowerModel, "kimi") && strings.Contains(lowerModel, "k2") {
			requestBody["temperature"] = 1.0
		} else {
			requestBody["temperature"] = temperature
		}
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))
	}

	return parseOpenAIResponse(body)
}

// chatOpenAIResponses uses OpenAI responses API format (/responses endpoint)
func (p *Provider) chatOpenAIResponses(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	// Convert messages to input format for responses API
	var inputMessages []map[string]any
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			inputMessages = append(inputMessages, map[string]any{
				"role":    "user",
				"content": msg.Content,
			})
		case "assistant":
			assistantMsg := map[string]any{
				"role":    "assistant",
				"content": msg.Content,
			}
			if len(msg.ToolCalls) > 0 {
				var toolCalls []map[string]any
				for _, tc := range msg.ToolCalls {
					toolCall := map[string]any{
						"id":   tc.ID,
						"type": "function",
						"name": tc.Name,
					}
					if len(tc.Arguments) > 0 {
						argsJSON, _ := json.Marshal(tc.Arguments)
						toolCall["arguments"] = string(argsJSON)
					}
					toolCalls = append(toolCalls, toolCall)
				}
				assistantMsg["tool_calls"] = toolCalls
			}
			inputMessages = append(inputMessages, assistantMsg)
		case "tool":
			inputMessages = append(inputMessages, map[string]any{
				"role":         "tool",
				"tool_call_id": msg.ToolCallID,
				"content":      msg.Content,
			})
		}
	}

	requestBody := map[string]any{
		"model": model,
		"input": inputMessages,
	}

	if len(tools) > 0 {
		var toolDefs []map[string]any
		for _, tool := range tools {
			toolDef := map[string]any{
				"type": "function",
				"name": tool.Function.Name,
			}
			if tool.Function.Description != "" {
				toolDef["description"] = tool.Function.Description
			}
			if len(tool.Function.Parameters) > 0 {
				toolDef["parameters"] = tool.Function.Parameters
			}
			toolDefs = append(toolDefs, toolDef)
		}
		requestBody["tools"] = toolDefs
	}

	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		requestBody["max_output_tokens"] = maxTokens
	}

	if temperature, ok := asFloat(options["temperature"]); ok {
		requestBody["temperature"] = temperature
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/responses", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))
	}

	return parseResponsesAPIResponse(body)
}

// chatAnthropicMessages uses Anthropic messages API format (/messages endpoint)
func (p *Provider) chatAnthropicMessages(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	var system string
	var anthropicMessages []map[string]any

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			if len(msg.SystemParts) > 0 {
				for _, part := range msg.SystemParts {
					system += part.Text
				}
			} else {
				system += msg.Content
			}
		case "user":
			if msg.ToolCallID != "" {
				// Tool result
				anthropicMessages = append(anthropicMessages, map[string]any{
					"role": "user",
					"content": []map[string]any{
						{
							"type":        "tool_result",
							"tool_use_id": msg.ToolCallID,
							"content":     msg.Content,
						},
					},
				})
			} else {
				anthropicMessages = append(anthropicMessages, map[string]any{
					"role": "user",
					"content": []map[string]any{
						{
							"type": "text",
							"text": msg.Content,
						},
					},
				})
			}
		case "assistant":
			content := []map[string]any{}
			if msg.Content != "" {
				content = append(content, map[string]any{
					"type": "text",
					"text": msg.Content,
				})
			}
			for _, tc := range msg.ToolCalls {
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": tc.Arguments,
				})
			}
			anthropicMessages = append(anthropicMessages, map[string]any{
				"role":    "assistant",
				"content": content,
			})
		case "tool":
			// Tool result - same format as user message with tool_use_id
			anthropicMessages = append(anthropicMessages, map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type":        "tool_result",
						"tool_use_id": msg.ToolCallID,
						"content":     msg.Content,
					},
				},
			})
		}
	}

	requestBody := map[string]any{
		"model":    model,
		"messages": anthropicMessages,
	}

	if system != "" {
		requestBody["system"] = system
	}

	maxTokens := 4096
	if mt, ok := options["max_tokens"].(int); ok {
		maxTokens = mt
	}
	requestBody["max_tokens"] = maxTokens

	if len(tools) > 0 {
		var toolDefs []map[string]any
		for _, tool := range tools {
			toolDef := map[string]any{
				"name": tool.Function.Name,
				"input_schema": map[string]any{
					"type":       "object",
					"properties": tool.Function.Parameters["properties"],
				},
			}
			if tool.Function.Description != "" {
				toolDef["description"] = tool.Function.Description
			}
			if required, ok := tool.Function.Parameters["required"].([]any); ok {
				reqFields := make([]string, 0, len(required))
				for _, r := range required {
					if s, ok := r.(string); ok {
						reqFields = append(reqFields, s)
					}
				}
				toolDef["input_schema"].(map[string]any)["required"] = reqFields
			}
			toolDefs = append(toolDefs, toolDef)
		}
		requestBody["tools"] = toolDefs
	}

	if temperature, ok := asFloat(options["temperature"]); ok {
		requestBody["temperature"] = temperature
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))
	}

	return parseAnthropicResponse(body)
}

// chatGeminiModels uses Gemini models API format (/models/{model} endpoint)
func (p *Provider) chatGeminiModels(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	endpoint string,
	options map[string]any,
) (*LLMResponse, error) {
	// Build contents from messages (Gemini format)
	var contents []map[string]any
	var systemParts []string
	// Track tool call ID to function name mapping for proper functionResponse
	toolCallIDToName := make(map[string]string)

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			if len(msg.SystemParts) > 0 {
				for _, part := range msg.SystemParts {
					systemParts = append(systemParts, part.Text)
				}
			} else {
				systemParts = append(systemParts, msg.Content)
			}
		case "user":
			contents = append(contents, map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"text": msg.Content},
				},
			})
		case "assistant":
			parts := []map[string]any{}
			if msg.Content != "" {
				parts = append(parts, map[string]any{"text": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				// Track the mapping from tool call ID to function name
				toolCallIDToName[tc.ID] = tc.Name
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": tc.Name,
						"args": tc.Arguments,
					},
				})
			}
			contents = append(contents, map[string]any{
				"role":  "model",
				"parts": parts,
			})
		case "tool":
			// Tool result - use mapped function name, not ToolCallID
			funcName := toolCallIDToName[msg.ToolCallID]
			if funcName == "" {
				// Fallback: if no mapping found, this shouldn't happen in normal flow
				// but we need to handle it gracefully
				funcName = "unknown_function"
			}
			contents = append(contents, map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{
						"functionResponse": map[string]any{
							"name": funcName,
							"response": map[string]any{
								"result": msg.Content,
							},
						},
					},
				},
			})
		}
	}

	requestBody := map[string]any{
		"contents": contents,
	}

	// Add system instruction if present
	if len(systemParts) > 0 {
		requestBody["systemInstruction"] = map[string]any{
			"parts": []map[string]any{
				{"text": strings.Join(systemParts, "\n")},
			},
		}
	}

	// Add tools if present
	if len(tools) > 0 {
		var toolDecls []map[string]any
		for _, tool := range tools {
			toolDecl := map[string]any{
				"name": tool.Function.Name,
				"parameters": map[string]any{
					"type":       "object",
					"properties": tool.Function.Parameters["properties"],
				},
			}
			if tool.Function.Description != "" {
				toolDecl["description"] = tool.Function.Description
			}
			if required, ok := tool.Function.Parameters["required"].([]any); ok {
				reqFields := make([]string, 0, len(required))
				for _, r := range required {
					if s, ok := r.(string); ok {
						reqFields = append(reqFields, s)
					}
				}
				toolDecl["parameters"].(map[string]any)["required"] = reqFields
			}
			toolDecls = append(toolDecls, map[string]any{
				"functionDeclarations": []map[string]any{toolDecl},
			})
		}
		requestBody["tools"] = toolDecls
	}

	// Add generation config
	genConfig := map[string]any{}
	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		genConfig["maxOutputTokens"] = maxTokens
	}
	if temperature, ok := asFloat(options["temperature"]); ok {
		genConfig["temperature"] = temperature
	}
	if len(genConfig) > 0 {
		requestBody["generationConfig"] = genConfig
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use the full endpoint path including /models/{model}
	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))
	}

	return parseGeminiResponse(body)
}

func parseGeminiResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata *struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	if len(apiResponse.Candidates) == 0 {
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
		}, nil
	}

	candidate := apiResponse.Candidates[0]
	var content strings.Builder
	var toolCalls []ToolCall

	for i, part := range candidate.Content.Parts {
		if part.Text != "" {
			content.WriteString(part.Text)
		}
		if part.FunctionCall != nil {
			toolCalls = append(toolCalls, ToolCall{
				ID:        fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, i),
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
		}
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	if candidate.FinishReason == "MAX_TOKENS" {
		finishReason = "length"
	}

	var usage *UsageInfo
	if apiResponse.UsageMetadata != nil {
		usage = &UsageInfo{
			PromptTokens:     apiResponse.UsageMetadata.PromptTokenCount,
			CompletionTokens: apiResponse.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      apiResponse.UsageMetadata.TotalTokenCount,
		}
	}

	return &LLMResponse{
		Content:      content.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

func parseOpenAIResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content          string            `json:"content"`
				ReasoningContent string            `json:"reasoning_content"`
				Reasoning        string            `json:"reasoning"`
				ReasoningDetails []ReasoningDetail `json:"reasoning_details"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function *struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
					ExtraContent *struct {
						Google *struct {
							ThoughtSignature string `json:"thought_signature"`
						} `json:"google"`
					} `json:"extra_content"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *UsageInfo `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
		}, nil
	}

	choice := apiResponse.Choices[0]
	toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))
	for _, tc := range choice.Message.ToolCalls {
		arguments := make(map[string]any)
		name := ""

		thoughtSignature := ""
		if tc.ExtraContent != nil && tc.ExtraContent.Google != nil {
			thoughtSignature = tc.ExtraContent.Google.ThoughtSignature
		}

		if tc.Function != nil {
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					arguments["raw"] = tc.Function.Arguments
				}
			}
		}

		toolCall := ToolCall{
			ID:               tc.ID,
			Name:             name,
			Arguments:        arguments,
			ThoughtSignature: thoughtSignature,
		}

		if thoughtSignature != "" {
			toolCall.ExtraContent = &ExtraContent{
				Google: &GoogleExtra{
					ThoughtSignature: thoughtSignature,
				},
			}
		}

		toolCalls = append(toolCalls, toolCall)
	}

	return &LLMResponse{
		Content:          choice.Message.Content,
		ReasoningContent: choice.Message.ReasoningContent,
		Reasoning:        choice.Message.Reasoning,
		ReasoningDetails: choice.Message.ReasoningDetails,
		ToolCalls:        toolCalls,
		FinishReason:     choice.FinishReason,
		Usage:            apiResponse.Usage,
	}, nil
}

func parseResponsesAPIResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Output []struct {
			Type string `json:"type"`
			// For message type
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			// For tool call type
			ID        string `json:"id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal responses API response: %w", err)
	}

	var content strings.Builder
	var toolCalls []ToolCall

	for _, output := range apiResponse.Output {
		switch output.Type {
		case "message":
			for _, c := range output.Content {
				if c.Type == "text" {
					content.WriteString(c.Text)
				}
			}
		case "tool_call":
			arguments := make(map[string]any)
			if output.Arguments != "" {
				if err := json.Unmarshal([]byte(output.Arguments), &arguments); err != nil {
					arguments["raw"] = output.Arguments
				}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        output.ID,
				Name:      output.Name,
				Arguments: arguments,
			})
		}
	}

	var usage *UsageInfo
	if apiResponse.Usage != nil {
		usage = &UsageInfo{
			PromptTokens:     apiResponse.Usage.InputTokens,
			CompletionTokens: apiResponse.Usage.OutputTokens,
			TotalTokens:      apiResponse.Usage.TotalTokens,
		}
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	return &LLMResponse{
		Content:      content.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

func parseAnthropicResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal anthropic response: %w", err)
	}

	var content strings.Builder
	var toolCalls []ToolCall

	for _, block := range apiResponse.Content {
		switch block.Type {
		case "text":
			content.WriteString(block.Text)
		case "tool_use":
			var args map[string]any
			if err := json.Unmarshal(block.Input, &args); err != nil {
				args = map[string]any{"raw": string(block.Input)}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: args,
			})
		}
	}

	finishReason := "stop"
	switch apiResponse.StopReason {
	case "tool_use":
		finishReason = "tool_calls"
	case "max_tokens":
		finishReason = "length"
	}

	return &LLMResponse{
		Content:      content.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage: &UsageInfo{
			PromptTokens:     apiResponse.Usage.InputTokens,
			CompletionTokens: apiResponse.Usage.OutputTokens,
			TotalTokens:      apiResponse.Usage.InputTokens + apiResponse.Usage.OutputTokens,
		},
	}, nil
}

// openaiMessage is the wire-format message for OpenAI-compatible APIs.
type openaiMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

func stripSystemParts(messages []Message) []openaiMessage {
	out := make([]openaiMessage, len(messages))
	for i, m := range messages {
		out[i] = openaiMessage{
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
			ToolCalls:        m.ToolCalls,
			ToolCallID:       m.ToolCallID,
		}
	}
	return out
}

func asInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

func asFloat(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
