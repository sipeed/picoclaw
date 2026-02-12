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

	"github.com/sipeed/picoclaw/pkg/logger"
)

// AnthropicProvider implements the Anthropic Messages API
type AnthropicProvider struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

func NewAnthropicProvider(apiKey, apiBase string) *AnthropicProvider {
	if apiBase == "" {
		apiBase = "https://api.anthropic.com/v1"
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		apiBase: apiBase,
		httpClient: &http.Client{
			Timeout: 0,
		},
	}
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	// Convert messages to Anthropic format
	anthropicMessages := make([]map[string]interface{}, 0, len(messages))
	var systemPrompt string

	// Track tool_use IDs from the previous assistant message
	validToolUseIDs := make(map[string]bool)

	for i, msg := range messages {
		if msg.Role == "system" {
			// Anthropic uses separate system parameter
			systemPrompt = msg.Content
			continue
		}

		anthMsg := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}

		// Handle tool results - only include if there's a corresponding tool_use
		if msg.Role == "tool" || msg.ToolCallID != "" {
			// Check if this tool_call_id was in the previous assistant message
			if !validToolUseIDs[msg.ToolCallID] {
				logger.WarnCF("llm", "Skipping tool result without corresponding tool_use",
					map[string]interface{}{
						"tool_call_id": msg.ToolCallID,
						"message_idx":  i,
					})
				continue // Skip orphaned tool results
			}

			anthMsg["role"] = "user"
			anthMsg["content"] = []map[string]interface{}{
				{
					"type":         "tool_result",
					"tool_use_id":  msg.ToolCallID,
					"content":      msg.Content,
				},
			}
		}

		// Handle assistant messages with tool calls
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// Clear previous valid IDs and record new ones
			validToolUseIDs = make(map[string]bool)

			contentBlocks := make([]map[string]interface{}, 0, len(msg.ToolCalls)+1)

			// Add text content if present
			if msg.Content != "" {
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type": "text",
					"text": msg.Content,
				})
			}

			// Add tool use blocks
			for _, tc := range msg.ToolCalls {
				// Extract name from either tc.Name or tc.Function.Name
				name := tc.Name
				if name == "" && tc.Function != nil {
					name = tc.Function.Name
				}

				// Extract arguments - they might be in tc.Arguments (map) or tc.Function.Arguments (JSON string)
				var input map[string]interface{}
				if len(tc.Arguments) > 0 {
					input = tc.Arguments
				} else if tc.Function != nil && tc.Function.Arguments != "" {
					// Parse JSON string to map
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
						logger.ErrorCF("llm", "Failed to parse tool arguments",
							map[string]interface{}{
								"id":    tc.ID,
								"name":  name,
								"error": err.Error(),
								"raw":   tc.Function.Arguments,
							})
						continue
					}
				} else {
					input = make(map[string]interface{})
				}

				logger.DebugCF("llm", "Converting tool call to Anthropic format",
					map[string]interface{}{
						"id":    tc.ID,
						"name":  name,
						"type":  tc.Type,
						"input": input,
					})

				if name == "" {
					logger.ErrorCF("llm", "Tool call has no name",
						map[string]interface{}{
							"id":            tc.ID,
							"type":          tc.Type,
							"has_function":  tc.Function != nil,
							"tc_name":       tc.Name,
						})
					continue // Skip this tool call
				}

				// Record this as a valid tool_use ID
				validToolUseIDs[tc.ID] = true

				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  name,
					"input": input,
				})
			}

			anthMsg["content"] = contentBlocks
		} else if msg.Role == "assistant" {
			// Assistant message without tool calls - clear valid IDs
			validToolUseIDs = make(map[string]bool)
		}

		anthropicMessages = append(anthropicMessages, anthMsg)
	}

	// Build request body
	requestBody := map[string]interface{}{
		"model":    model,
		"messages": anthropicMessages,
	}

	if systemPrompt != "" {
		requestBody["system"] = systemPrompt
	}

	// Add tools if present
	if len(tools) > 0 {
		anthropicTools := make([]map[string]interface{}, 0, len(tools))
		for _, tool := range tools {
			anthropicTools = append(anthropicTools, map[string]interface{}{
				"name":         tool.Function.Name,
				"description":  tool.Function.Description,
				"input_schema": tool.Function.Parameters,
			})
		}
		requestBody["tools"] = anthropicTools
	}

	// Add max_tokens (required by Anthropic)
	if maxTokens, ok := options["max_tokens"].(int); ok {
		requestBody["max_tokens"] = maxTokens
	} else {
		requestBody["max_tokens"] = 8192 // Default
	}

	// Add temperature if specified
	if temperature, ok := options["temperature"].(float64); ok {
		requestBody["temperature"] = temperature
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	fullURL := p.apiBase + "/messages"

	// Debug log: Log request details
	logger.DebugCF("llm", "Sending Anthropic API request",
		map[string]interface{}{
			"url":           fullURL,
			"model":         model,
			"api_base":      p.apiBase,
			"has_api_key":   p.apiKey != "",
			"api_key_len":   len(p.apiKey),
			"message_count": len(anthropicMessages),
			"tools_count":   len(tools),
			"request_body":  string(jsonData),
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

	// Set Anthropic-specific headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	logger.DebugCF("llm", "Sending HTTP request",
		map[string]interface{}{
			"method": "POST",
			"url":    fullURL,
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
	logger.DebugCF("llm", "Received Anthropic API response",
		map[string]interface{}{
			"status_code":    resp.StatusCode,
			"status":         resp.Status,
			"content_length": len(body),
			"response_body":  string(body),
		})

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("llm", "Anthropic API returned non-OK status",
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

func (p *AnthropicProvider) parseResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text,omitempty"`
			ID    string `json:"id,omitempty"`
			Name  string `json:"name,omitempty"`
			Input map[string]interface{} `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract text content and tool calls
	var textContent string
	toolCalls := make([]ToolCall, 0)

	for _, content := range apiResponse.Content {
		switch content.Type {
		case "text":
			textContent += content.Text
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: content.Input,
			})
		}
	}

	return &LLMResponse{
		Content:      textContent,
		ToolCalls:    toolCalls,
		FinishReason: apiResponse.StopReason,
		Usage: &UsageInfo{
			PromptTokens:     apiResponse.Usage.InputTokens,
			CompletionTokens: apiResponse.Usage.OutputTokens,
			TotalTokens:      apiResponse.Usage.InputTokens + apiResponse.Usage.OutputTokens,
		},
	}, nil
}

func (p *AnthropicProvider) GetDefaultModel() string {
	return "claude-sonnet-4-5"
}
