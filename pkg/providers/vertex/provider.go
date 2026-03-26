// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package vertex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/common"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	LLMResponse    = protocoltypes.LLMResponse
	Message        = protocoltypes.Message
	ToolDefinition = protocoltypes.ToolDefinition
	ToolCall       = protocoltypes.ToolCall
	FunctionCall   = protocoltypes.FunctionCall
)

// Provider implements the LLM provider interface for Google Vertex AI.
// It uses the standard Vertex AI REST API for Gemini models.
type Provider struct {
	apiKey     string
	apiBase    string // If provided, overrides the default construction
	projectID  string
	region     string
	httpClient *http.Client
}

// Option configures the Vertex Provider.
type Option func(*Provider)

// WithRequestTimeout sets the HTTP request timeout.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

// NewProvider creates a new Vertex AI provider.
func NewProvider(apiKey, apiBase, proxy, projectID, region string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		projectID:  projectID,
		region:     region,
		httpClient: common.NewHTTPClient(proxy),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	return p
}

// buildURL constructs the Vertex AI REST endpoint URL.
func (p *Provider) buildURL(model string, action string) string {
	if action == "" {
		action = "generateContent"
	}

	var baseURL string
	if p.apiBase != "" {
		if strings.Contains(p.apiBase, "generateContent") {
			baseURL = p.apiBase
		} else {
			baseURL = fmt.Sprintf("%s/%s:%s", p.apiBase, model, action)
		}
	} else {
		region := p.region
		if region == "" {
			region = "us-central1"
		}
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s", region, p.projectID, region, model, action)
	}

	// Only append ?key= for custom apiBase endpoints
	if p.apiBase != "" && p.apiKey != "" && !strings.Contains(baseURL, "key=") {
		if strings.Contains(baseURL, "?") {
			baseURL = fmt.Sprintf("%s&key=%s", baseURL, p.apiKey)
		} else {
			baseURL = fmt.Sprintf("%s?key=%s", baseURL, p.apiKey)
		}
	}

	if action == "streamGenerateContent" && !strings.Contains(baseURL, "alt=sse") {
		if strings.Contains(baseURL, "?") {
			baseURL = fmt.Sprintf("%s&alt=sse", baseURL)
		} else {
			baseURL = fmt.Sprintf("%s?alt=sse", baseURL)
		}
	}

	return baseURL
}

// parseMediaData converts base64 media data into the Vertex AI inlineData format.
// It tries to detect mime type from the data URI scheme if present.
func parseMediaData(mediaData string) map[string]any {
	mimeType := "image/jpeg"
	data := mediaData

	if strings.HasPrefix(mediaData, "data:") {
		idx := strings.Index(mediaData, ";base64,")
		if idx != -1 {
			mimeType = mediaData[5:idx]
			data = mediaData[idx+8:]
		}
	}

	return map[string]any{
		"inlineData": map[string]any{
			"mimeType": mimeType,
			"data":     data,
		},
	}
}

// buildRequestBody formats the standard messages and tools into the Vertex AI (Gemini) REST payload format.
func (p *Provider) buildRequestBody(
	messages []Message,
	tools []ToolDefinition,
	options map[string]any,
) (map[string]any, error) {
	req := make(map[string]any)

	var contents []map[string]any
	var systemInstruction *map[string]any

	var currentContent map[string]any

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemInstruction = &map[string]any{
				"role": "system",
				"parts": []map[string]any{
					{"text": msg.Content},
				},
			}
		case "user":
			if currentContent != nil && currentContent["role"] == "user" {
				// We need to group consecutive user messages (like tool responses)
			} else {
				if currentContent != nil {
					contents = append(contents, currentContent)
				}
				currentContent = map[string]any{
					"role":  "user",
					"parts": []map[string]any{},
				}
			}

			parts := currentContent["parts"].([]map[string]any)

			if msg.ToolCallID != "" {
				// Tool response
				parts = append(parts, map[string]any{
					"functionResponse": map[string]any{
						"name": msg.ToolCallID,
						"response": map[string]any{
							"result": msg.Content,
						},
					},
				})
			} else {
				if msg.Content != "" {
					parts = append(parts, map[string]any{"text": msg.Content})
				}
				for _, media := range msg.Media {
					parts = append(parts, parseMediaData(media))
				}
			}
			currentContent["parts"] = parts

		case "assistant":
			if currentContent != nil {
				contents = append(contents, currentContent)
			}
			currentContent = map[string]any{
				"role":  "model",
				"parts": []map[string]any{},
			}

			parts := currentContent["parts"].([]map[string]any)

			if msg.Content != "" {
				parts = append(parts, map[string]any{"text": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": tc.Name,
						"args": tc.Arguments,
					},
				})
			}
			currentContent["parts"] = parts

		case "tool":
			if currentContent != nil && currentContent["role"] == "user" {
				// Group tool response
			} else {
				if currentContent != nil {
					contents = append(contents, currentContent)
				}
				currentContent = map[string]any{
					"role":  "user",
					"parts": []map[string]any{},
				}
			}
			parts := currentContent["parts"].([]map[string]any)

			// Try to handle tool responses that might be strings instead of objects
			// if they are just basic strings. But Gemini API expects an object.
			responseObj := map[string]any{"result": msg.Content}

			parts = append(parts, map[string]any{
				"functionResponse": map[string]any{
					"name":     msg.ToolCallID,
					"response": responseObj,
				},
			})
			currentContent["parts"] = parts
		}
	}

	if currentContent != nil {
		contents = append(contents, currentContent)
	}

	req["contents"] = contents
	if systemInstruction != nil {
		req["systemInstruction"] = *systemInstruction
	}

	if len(tools) > 0 {
		var funcDecls []map[string]any
		for _, t := range tools {
			if t.Type != "function" {
				continue
			}
			decl := map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
			}
			if t.Function.Parameters != nil {
				decl["parameters"] = t.Function.Parameters
			}
			funcDecls = append(funcDecls, decl)
		}
		if len(funcDecls) > 0 {
			req["tools"] = []map[string]any{
				{
					"functionDeclarations": funcDecls,
				},
			}
		}
	}

	generationConfig := make(map[string]any)
	if val, ok := options["max_tokens"]; ok {
		if maxTokens, ok := common.AsInt(val); ok {
			generationConfig["maxOutputTokens"] = maxTokens
		}
	}
	if temp, ok := common.AsFloat(options["temperature"]); ok {
		generationConfig["temperature"] = temp
	}
	if len(generationConfig) > 0 {
		req["generationConfig"] = generationConfig
	}

	return req, nil
}

func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiBase == "" && p.projectID == "" {
		return nil, fmt.Errorf("Vertex AI requires either an api_base or a project_id")
	}

	requestBody, err := p.buildRequestBody(messages, tools, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build request body: %w", err)
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	requestURL := p.buildURL(model, "generateContent")

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" && !strings.Contains(requestURL, "key=") {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, "vertex")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return p.parseResponse(bodyBytes)
}

func (p *Provider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
	onChunk func(accumulated string),
) (*LLMResponse, error) {
	if p.apiBase == "" && p.projectID == "" {
		return nil, fmt.Errorf("Vertex AI requires either an api_base or a project_id")
	}

	requestBody, err := p.buildRequestBody(messages, tools, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build request body: %w", err)
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	requestURL := p.buildURL(model, "streamGenerateContent")

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" && !strings.Contains(requestURL, "key=") {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, "vertex")
	}

	var accumulatedText string
	var allToolCalls []ToolCall
	var finalResponse *LLMResponse

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimPrefix(line, "data: ")
		} else if line == "[" || line == "]" || line == "," {
			continue
		}

		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text         string `json:"text"`
						FunctionCall *struct {
							Name string         `json:"name"`
							Args map[string]any `json:"args"`
						} `json:"functionCall,omitempty"`
					} `json:"parts"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
			UsageMetadata *struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) > 0 {
			candidate := chunk.Candidates[0]
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					accumulatedText += part.Text
					if onChunk != nil {
						onChunk(accumulatedText)
					}
				}
				if part.FunctionCall != nil {
					argsJSON, _ := json.Marshal(part.FunctionCall.Args)
					toolCall := ToolCall{
						ID:        fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, time.Now().UnixNano()),
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
						Function: &FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsJSON),
						},
					}
					allToolCalls = append(allToolCalls, toolCall)
				}
			}

			if candidate.FinishReason != "" && finalResponse == nil {
				finishReason := candidate.FinishReason
				if finishReason == "STOP" {
					finishReason = "stop"
				} else if len(allToolCalls) > 0 {
					finishReason = "tool_calls"
				}

				finalResponse = &LLMResponse{
					Content:      accumulatedText,
					ToolCalls:    allToolCalls,
					FinishReason: finishReason,
				}
			}
		}

		if chunk.UsageMetadata != nil {
			if finalResponse == nil {
				finalResponse = &LLMResponse{
					Content:   accumulatedText,
					ToolCalls: allToolCalls,
				}
			}
			finalResponse.Usage = &protocoltypes.UsageInfo{
				PromptTokens:     chunk.UsageMetadata.PromptTokenCount,
				CompletionTokens: chunk.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      chunk.UsageMetadata.TotalTokenCount,
			}
		}
	}

	if finalResponse == nil {
		finishReason := "stop"
		if len(allToolCalls) > 0 {
			finishReason = "tool_calls"
		}
		finalResponse = &LLMResponse{
			Content:      accumulatedText,
			ToolCalls:    allToolCalls,
			FinishReason: finishReason,
		}
	}

	return finalResponse, nil
}

func (p *Provider) parseResponse(body []byte) (*LLMResponse, error) {
	var vResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &vResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(vResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := vResp.Candidates[0]

	var content string
	var toolCalls []ToolCall

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			content += part.Text
		}
		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			toolCalls = append(toolCalls, ToolCall{
				ID:        fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, time.Now().UnixNano()),
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
				Function: &FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	finishReason := candidate.FinishReason
	if finishReason == "STOP" {
		finishReason = "stop"
	} else if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	return &LLMResponse{
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage: &protocoltypes.UsageInfo{
			PromptTokens:     vResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: vResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      vResp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func (p *Provider) GetDefaultModel() string {
	return "gemini-1.5-pro-preview-0409"
}
