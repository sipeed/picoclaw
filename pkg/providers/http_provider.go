// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

type HTTPProvider struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

func NewHTTPProvider(apiKey, apiBase string) *HTTPProvider {
	return &HTTPProvider{
		apiKey:  apiKey,
		apiBase: apiBase,
		httpClient: &http.Client{
			Timeout: 0,
		},
	}
}

func (p *HTTPProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := options["max_tokens"].(int); ok {
		requestBody["max_tokens"] = maxTokens
	}

	if temperature, ok := options["temperature"].(float64); ok {
		requestBody["temperature"] = temperature
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
		authHeader := "Bearer " + p.apiKey
		req.Header.Set("Authorization", authHeader)
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
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	return p.parseResponse(body)
}

func (p *HTTPProvider) parseResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function *struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
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
		arguments := make(map[string]interface{})
		name := ""

		// Handle OpenAI format with nested function object
		if tc.Type == "function" && tc.Function != nil {
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					arguments["raw"] = tc.Function.Arguments
				}
			}
		} else if tc.Function != nil {
			// Legacy format without type field
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					arguments["raw"] = tc.Function.Arguments
				}
			}
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      name,
			Arguments: arguments,
		})
	}

	return &LLMResponse{
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: choice.FinishReason,
		Usage:        apiResponse.Usage,
	}, nil
}

func (p *HTTPProvider) GetDefaultModel() string {
	return "gpt-3.5-turbo"
}

// ChatStream performs a streaming chat request with real-time token generation
func (p *HTTPProvider) ChatStream(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}, callback func(chunk string)) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true, // Enable streaming
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := options["max_tokens"].(int); ok {
		requestBody["max_tokens"] = maxTokens
	}

	if temperature, ok := options["temperature"].(float64); ok {
		requestBody["temperature"] = temperature
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
		authHeader := "Bearer " + p.apiKey
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	// Parse streaming response
	return p.parseStreamingResponse(resp.Body, callback)
}

func (p *HTTPProvider) parseStreamingResponse(body io.Reader, callback func(chunk string)) (*LLMResponse, error) {
	scanner := bufio.NewScanner(body)
	response := &LLMResponse{
		Content:   "",
		ToolCalls: []ToolCall{},
	}

	var contentBuilder strings.Builder
	toolCallsMap := make(map[int]*ToolCall)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// SSE format: "data: {json}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]

		// Handle content streaming
		if choice.Delta.Content != "" {
			contentBuilder.WriteString(choice.Delta.Content)
			if callback != nil {
				callback(choice.Delta.Content)
			}
		}

		// Handle tool calls
		for _, tc := range choice.Delta.ToolCalls {
			if _, exists := toolCallsMap[tc.Index]; !exists {
				toolCallsMap[tc.Index] = &ToolCall{
					ID:        tc.ID,
					Type:      tc.Type,
					Name:      tc.Function.Name,
					Arguments: make(map[string]interface{}),
				}
			}

			// Accumulate function arguments
			if tc.Function.Arguments != "" {
				existing := toolCallsMap[tc.Index]
				// Parse and merge arguments incrementally
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
					for k, v := range args {
						existing.Arguments[k] = v
					}
				}
			}
		}

		// Set finish reason
		if choice.FinishReason != "" {
			response.FinishReason = choice.FinishReason
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	response.Content = contentBuilder.String()

	// Convert tool calls map to slice
	for _, tc := range toolCallsMap {
		response.ToolCalls = append(response.ToolCalls, *tc)
	}

	return response, nil
}

func CreateProvider(cfg *config.Config) (LLMProvider, error) {
	model := cfg.Agents.Defaults.Model

	var apiKey, apiBase string

	lowerModel := strings.ToLower(model)

	switch {
	case strings.HasPrefix(model, "openrouter/") || strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "meta-llama/") || strings.HasPrefix(model, "deepseek/") || strings.HasPrefix(model, "google/"):
		apiKey = cfg.Providers.OpenRouter.APIKey
		if cfg.Providers.OpenRouter.APIBase != "" {
			apiBase = cfg.Providers.OpenRouter.APIBase
		} else {
			apiBase = "https://openrouter.ai/api/v1"
		}

	case strings.Contains(lowerModel, "claude") || strings.HasPrefix(model, "anthropic/"):
		apiKey = cfg.Providers.Anthropic.APIKey
		apiBase = cfg.Providers.Anthropic.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}

	case strings.Contains(lowerModel, "gpt") || strings.HasPrefix(model, "openai/"):
		apiKey = cfg.Providers.OpenAI.APIKey
		apiBase = cfg.Providers.OpenAI.APIBase
		if apiBase == "" {
			apiBase = "https://api.openai.com/v1"
		}

	case strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/"):
		apiKey = cfg.Providers.Gemini.APIKey
		apiBase = cfg.Providers.Gemini.APIBase
		if apiBase == "" {
			apiBase = "https://generativelanguage.googleapis.com/v1beta"
		}

	case strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai"):
		apiKey = cfg.Providers.Zhipu.APIKey
		apiBase = cfg.Providers.Zhipu.APIBase
		if apiBase == "" {
			apiBase = "https://open.bigmodel.cn/api/paas/v4"
		}

	case strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/"):
		apiKey = cfg.Providers.Groq.APIKey
		apiBase = cfg.Providers.Groq.APIBase
		if apiBase == "" {
			apiBase = "https://api.groq.com/openai/v1"
		}

	case cfg.Providers.VLLM.APIBase != "":
		apiKey = cfg.Providers.VLLM.APIKey
		apiBase = cfg.Providers.VLLM.APIBase

	default:
		if cfg.Providers.OpenRouter.APIKey != "" {
			apiKey = cfg.Providers.OpenRouter.APIKey
			if cfg.Providers.OpenRouter.APIBase != "" {
				apiBase = cfg.Providers.OpenRouter.APIBase
			} else {
				apiBase = "https://openrouter.ai/api/v1"
			}
		} else {
			return nil, fmt.Errorf("no API key configured for model: %s", model)
		}
	}

	if apiKey == "" && !strings.HasPrefix(model, "bedrock/") {
		return nil, fmt.Errorf("no API key configured for provider (model: %s)", model)
	}

	if apiBase == "" {
		return nil, fmt.Errorf("no API base configured for provider (model: %s)", model)
	}

	return NewHTTPProvider(apiKey, apiBase), nil
}
