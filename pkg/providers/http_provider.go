// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type HTTPProvider struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
		lowerModel := strings.ToLower(model)
		if strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "o1") {
			requestBody["max_completion_tokens"] = maxTokens
		} else {
			requestBody["max_tokens"] = maxTokens
		}
	}

	if temperature, ok := options["temperature"].(float64); ok {
		requestBody["temperature"] = temperature
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	fullURL := p.apiBase + "/chat/completions"

	// Debug log: Log request details
	logger.DebugCF("llm", "Sending LLM request",
		map[string]interface{}{
			"url":          fullURL,
			"model":        model,
			"api_base":     p.apiBase,
			"has_api_key":  p.apiKey != "",
			"api_key_len":  len(p.apiKey),
			"message_count": len(messages),
			"tools_count":  len(tools),
			"request_body": string(jsonData),
		})

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(jsonData))
	if err != nil {
		logger.ErrorCF("llm", "Failed to create HTTP request",
			map[string]interface{}{
				"url":   fullURL,
				"error": err.Error(),
			})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		authHeader := "Bearer " + p.apiKey
		req.Header.Set("Authorization", authHeader)
		logger.DebugCF("llm", "Authorization header set",
			map[string]interface{}{
				"key_prefix": p.apiKey[:min(10, len(p.apiKey))] + "...",
			})
	}

	logger.DebugCF("llm", "Sending HTTP request",
		map[string]interface{}{
			"method":  "POST",
			"url":     fullURL,
			"headers": req.Header,
		})

	resp, err := p.httpClient.Do(req)
	if err != nil {
		logger.ErrorCF("llm", "HTTP request failed",
			map[string]interface{}{
				"url":   fullURL,
				"error": err.Error(),
			})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("llm", "Failed to read response body",
			map[string]interface{}{
				"status_code": resp.StatusCode,
				"error":       err.Error(),
			})
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Debug log: Log response details
	logger.DebugCF("llm", "Received LLM response",
		map[string]interface{}{
			"status_code":   resp.StatusCode,
			"status":        resp.Status,
			"content_length": len(body),
			"response_body": string(body),
		})

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("llm", "LLM API returned non-OK status",
			map[string]interface{}{
				"status_code":   resp.StatusCode,
				"status":        resp.Status,
				"url":           fullURL,
				"response_body": string(body),
			})
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
	return ""
}

func CreateProvider(cfg *config.Config) (LLMProvider, error) {
	model := cfg.Agents.Defaults.Model

	var apiKey, apiBase string

	lowerModel := strings.ToLower(model)

	logger.DebugCF("llm", "Creating LLM provider",
		map[string]interface{}{
			"model":       model,
			"lower_model": lowerModel,
		})

	switch {
	case strings.HasPrefix(model, "openrouter/") || strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "meta-llama/") || strings.HasPrefix(model, "deepseek/") || strings.HasPrefix(model, "google/"):
		apiKey = cfg.Providers.OpenRouter.APIKey
		if cfg.Providers.OpenRouter.APIBase != "" {
			apiBase = cfg.Providers.OpenRouter.APIBase
		} else {
			apiBase = "https://openrouter.ai/api/v1"
		}

	case (strings.Contains(lowerModel, "claude") || strings.HasPrefix(model, "anthropic/")) && cfg.Providers.Anthropic.APIKey != "":
		// Use dedicated Anthropic provider
		apiKey = cfg.Providers.Anthropic.APIKey
		apiBase = cfg.Providers.Anthropic.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}

		logger.InfoCF("llm", "Anthropic provider created successfully",
			map[string]interface{}{
				"model":       model,
				"api_base":    apiBase,
				"has_api_key": apiKey != "",
				"api_key_len": len(apiKey),
			})

		return NewAnthropicProvider(apiKey, apiBase), nil

	case (strings.Contains(lowerModel, "gpt") || strings.HasPrefix(model, "openai/")) && cfg.Providers.OpenAI.APIKey != "":
		apiKey = cfg.Providers.OpenAI.APIKey
		apiBase = cfg.Providers.OpenAI.APIBase
		if apiBase == "" {
			apiBase = "https://api.openai.com/v1"
		}

	case (strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/")) && cfg.Providers.Gemini.APIKey != "":
		apiKey = cfg.Providers.Gemini.APIKey
		apiBase = cfg.Providers.Gemini.APIBase
		if apiBase == "" {
			apiBase = "https://generativelanguage.googleapis.com/v1beta"
		}

	case (strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai")) && cfg.Providers.Zhipu.APIKey != "":
		apiKey = cfg.Providers.Zhipu.APIKey
		apiBase = cfg.Providers.Zhipu.APIBase
		if apiBase == "" {
			apiBase = "https://open.bigmodel.cn/api/paas/v4"
		}

	case (strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/")) && cfg.Providers.Groq.APIKey != "":
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
		logger.ErrorCF("llm", "No API key configured",
			map[string]interface{}{
				"model": model,
			})
		return nil, fmt.Errorf("no API key configured for provider (model: %s)", model)
	}

	if apiBase == "" {
		logger.ErrorCF("llm", "No API base configured",
			map[string]interface{}{
				"model": model,
			})
		return nil, fmt.Errorf("no API base configured for provider (model: %s)", model)
	}

	logger.InfoCF("llm", "Provider created successfully",
		map[string]interface{}{
			"model":       model,
			"api_base":    apiBase,
			"has_api_key": apiKey != "",
			"api_key_len": len(apiKey),
		})

	return NewHTTPProvider(apiKey, apiBase), nil
}