// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package openai_compat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatStream implements providers.StreamingProvider on *Provider.
// Text tokens are delivered via onToken as they arrive. Tool calls are
// accumulated across chunks and returned in the final *LLMResponse.
func (p *Provider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
	onToken func(string),
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}
	model = normalizeModel(model, p.apiBase)

	requestBody := map[string]any{
		"model":    model,
		"messages": serializeMessages(messages),
		"stream":   true,
	}
	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}
	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		fieldName := p.maxTokensField
		if fieldName == "" {
			lm := strings.ToLower(model)
			if strings.Contains(lm, "glm") || strings.Contains(lm, "o1") || strings.Contains(lm, "gpt-5") {
				fieldName = "max_completion_tokens"
			} else {
				fieldName = "max_tokens"
			}
		}
		requestBody[fieldName] = maxTokens
	}
	if temperature, ok := asFloat(options["temperature"]); ok {
		lm := strings.ToLower(model)
		if strings.Contains(lm, "kimi") && strings.Contains(lm, "k2") {
			requestBody["temperature"] = 1.0
		} else {
			requestBody["temperature"] = temperature
		}
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Use a client without a read timeout — context handles cancellation.
	streamClient := &http.Client{Transport: p.httpClient.Transport}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("API request failed: status=%d body=%s", resp.StatusCode, responsePreview(body, 128))
	}

	return parseStreamResponse(resp.Body, onToken)
}

type toolCallChunkBuilder struct {
	id       string
	callType string
	name     string
	args     strings.Builder
}

func parseStreamResponse(body io.Reader, onToken func(string)) (*LLMResponse, error) {
	var contentBuf strings.Builder
	var finishReason string
	builders := map[int]*toolCallChunkBuilder{}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
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
						Function *struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]
		if choice.FinishReason != nil {
			finishReason = *choice.FinishReason
		}
		if choice.Delta.Content != "" {
			contentBuf.WriteString(choice.Delta.Content)
			if onToken != nil {
				onToken(choice.Delta.Content)
			}
		}
		for _, tc := range choice.Delta.ToolCalls {
			b, ok := builders[tc.Index]
			if !ok {
				b = &toolCallChunkBuilder{}
				builders[tc.Index] = b
			}
			if tc.ID != "" {
				b.id = tc.ID
			}
			if tc.Type != "" {
				b.callType = tc.Type
			}
			if tc.Function != nil {
				if tc.Function.Name != "" {
					b.name = tc.Function.Name
				}
				b.args.WriteString(tc.Function.Arguments)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream read error: %w", err)
	}

	toolCalls := make([]ToolCall, 0, len(builders))
	for i := 0; i < len(builders); i++ {
		b, ok := builders[i]
		if !ok {
			break
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:   b.id,
			Type: b.callType,
			Name: b.name,
			Function: &FunctionCall{
				Name:      b.name,
				Arguments: b.args.String(),
			},
		})
	}

	return &LLMResponse{
		Content:      contentBuf.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}, nil
}
