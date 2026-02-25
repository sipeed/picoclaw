package anthropic_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
)

type Provider struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

func NewProvider(apiKey, apiBase, proxy string) *Provider {
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(parsed),
			}
		} else {
			log.Printf("anthropic_compat: invalid proxy URL %q: %v", proxy, err)
		}
	}

	return &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: client,
	}
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
	// logger.InfoC("LLM-Chat", p.apiBase)

	requestBody, err := buildRequestBody(messages, tools, model, options)
	if err != nil {
		return nil, err
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

	return parseResponse(body)
}

func buildRequestBody(
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (map[string]any, error) {
	var system []string
	var anthropicMessages []map[string]any

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			system = append(system, msg.Content)
		case "user":
			msgMap := map[string]any{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": msg.Content},
				},
			}
			// Handle tool result messages
			if msg.ToolCallID != "" {
				msgMap["content"] = []map[string]any{
					{"type": "tool_result", "tool_use_id": msg.ToolCallID, "content": msg.Content},
				}
			}
			anthropicMessages = append(anthropicMessages, msgMap)
		case "assistant":
			content := make([]map[string]any, 0)
			if msg.Content != "" {
				content = append(content, map[string]any{"type": "text", "text": msg.Content})
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
			anthropicMessages = append(anthropicMessages, map[string]any{
				"role": "user",
				"content": []map[string]any{
					{"type": "tool_result", "tool_use_id": msg.ToolCallID, "content": msg.Content},
				},
			})
		}
	}

	requestBody := map[string]any{
		"model":    model,
		"messages": anthropicMessages,
	}

	if len(system) > 0 {
		requestBody["system"] = system
	}

	if len(tools) > 0 {
		requestBody["tools"] = translateTools(tools)
	}

	maxTokens := 4096
	if mt, ok := options["max_tokens"].(int); ok {
		maxTokens = mt
	}
	requestBody["max_tokens"] = maxTokens

	if temperature, ok := options["temperature"].(float64); ok {
		requestBody["temperature"] = temperature
	}

	return requestBody, nil
}

func translateTools(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		tool := map[string]any{
			"name":         t.Function.Name,
			"input_schema": t.Function.Parameters,
		}
		if desc := t.Function.Description; desc != "" {
			tool["description"] = desc
		}
		result = append(result, tool)
	}
	return result
}

func parseResponse(body []byte) (*LLMResponse, error) {
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
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var content string
	var toolCalls []ToolCall

	for _, block := range apiResponse.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			var args map[string]any
			if err := json.Unmarshal(block.Input, &args); err != nil {
				log.Printf("anthropic_compat: failed to decode tool call input for %q: %v", block.Name, err)
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
	case "end_turn", "stop":
		finishReason = "stop"
	}

	return &LLMResponse{
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage: &UsageInfo{
			PromptTokens:     apiResponse.Usage.InputTokens,
			CompletionTokens: apiResponse.Usage.OutputTokens,
			TotalTokens:      apiResponse.Usage.InputTokens + apiResponse.Usage.OutputTokens,
		},
	}, nil
}
