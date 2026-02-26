package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
	miniMaxOpen   = "[TOOL_CALL]"
	miniMaxCloseA = "</minimax:tool_call>"
	miniMaxCloseB = "[/minimax:tool_call]"
	invokeOpen    = "<invoke name=\""
	invokeClose   = "</invoke>"
	paramOpen     = "<parameter name=\""
	paramClose    = "</parameter>"

	maxReasoningBlocks  = 10
	maxMiniMaxToolCalls = 20
	maxParameters       = 50
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
)

type Provider struct {
	apiKey         string
	apiBase        string
	maxTokensField string // Field name for max tokens (e.g., "max_completion_tokens" for o1/glm models)
	httpClient     *http.Client
}

type Option func(*Provider)

const defaultRequestTimeout = 120 * time.Second

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
		} else {
			log.Printf("openai_compat: invalid proxy URL %q: %v", proxy, err)
		}
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

	model = normalizeModel(model, p.apiBase)

	requestBody := map[string]any{
		"model":    model,
		"messages": stripSystemParts(messages),
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		// Use configured maxTokensField if specified, otherwise fallback to model-based detection
		fieldName := p.maxTokensField
		if fieldName == "" {
			// Fallback: detect from model name for backward compatibility
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

	// Prompt caching: pass a stable cache key so OpenAI can bucket requests
	// with the same key and reuse prefix KV cache across calls.
	// The key is typically the agent ID — stable per agent, shared across requests.
	// See: https://platform.openai.com/docs/guides/prompt-caching
	// Prompt caching is only supported by OpenAI-native endpoints.
	// Gemini and other providers reject unknown fields, so skip for non-OpenAI APIs.
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		if !strings.Contains(p.apiBase, "generativelanguage.googleapis.com") {
			requestBody["prompt_cache_key"] = cacheKey
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

	return parseResponse(body)
}

func parseResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
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
	content := choice.Message.Content
	reasoningContent := choice.Message.ReasoningContent
	toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))

	// --- 1. Extract reasoning blocks from content if present ---
	// This handles models that put reasoning in the main content block.
	// We handle multiple blocks and join them, with a safety limit.
	for i := 0; i < maxReasoningBlocks; i++ {
		start := strings.Index(content, thinkOpenTag)
		if start == -1 {
			break
		}
		// Search for the closing tag starting after the opening tag
		endRel := strings.Index(content[start:], thinkCloseTag)
		if endRel == -1 {
			// Malformed or cut off; leave it to avoid data loss
			break
		}
		end := start + endRel
		extracted := strings.TrimSpace(content[start+len(thinkOpenTag) : end])
		if reasoningContent == "" {
			reasoningContent = extracted
		} else if extracted != "" {
			reasoningContent += "\n\n" + extracted
		}
		content = strings.TrimSpace(content[:start] + content[end+len(thinkCloseTag):])
	}

	// --- 2. Extract MiniMax-style [TOOL_CALL] XML if present ---
	for miniMaxIdx := 0; miniMaxIdx < maxMiniMaxToolCalls; miniMaxIdx++ {
		tagStart := strings.Index(content, miniMaxOpen)
		if tagStart == -1 {
			break
		}

		// Search for the earliest valid closing tag after the opening tag
		angleIdx := strings.Index(content[tagStart:], miniMaxCloseA)
		bracketIdx := strings.Index(content[tagStart:], miniMaxCloseB)

		tagEndIdx := -1
		tagLen := 0

		if angleIdx != -1 && (bracketIdx == -1 || angleIdx < bracketIdx) {
			tagEndIdx = tagStart + angleIdx
			tagLen = len(miniMaxCloseA)
		} else if bracketIdx != -1 {
			tagEndIdx = tagStart + bracketIdx
			tagLen = len(miniMaxCloseB)
		}

		// If no closing tag is found, the string is malformed or cut off. Break to avoid infinite loop.
		if tagEndIdx == -1 {
			break
		}

		// Calculate indices for XML content extraction
		xmlBodyStart := tagStart + len(miniMaxOpen)
		if xmlBodyStart > tagEndIdx {
			break
		}
		xmlPart := content[xmlBodyStart:tagEndIdx]

		// Very basic XML-ish parsing for MiniMax format
		// Extract name: <invoke name="([^"]+)">
		nameStart := strings.Index(xmlPart, invokeOpen)
		if nameStart != -1 {
			nameStart += len(invokeOpen)
			nameEnd := strings.Index(xmlPart[nameStart:], "\"")
			invokeEnd := strings.Index(xmlPart, invokeClose)

			if nameEnd != -1 && invokeEnd != -1 {
				toolName := xmlPart[nameStart : nameStart+nameEnd]
				args := make(map[string]any)

				// Extract parameters: <parameter name="([^"]+)">([^<]*)</parameter>
				paramsPart := xmlPart[nameStart+nameEnd:]
				malformedParams := false
				for pCount := 0; pCount < maxParameters; pCount++ {
					pStart := strings.Index(paramsPart, paramOpen)
					if pStart == -1 {
						break
					}
					pStart += len(paramOpen)
					
					// Bounds check for parameter name
					if pStart >= len(paramsPart) {
						malformedParams = true
						break
					}
					pNameEnd := strings.Index(paramsPart[pStart:], "\"")
					if pNameEnd == -1 {
						malformedParams = true
						break
					}
					pName := paramsPart[pStart : pStart+pNameEnd]
					
					// Search for value start ">"
					valMarkerIdx := strings.Index(paramsPart[pStart+pNameEnd:], ">")
					if valMarkerIdx == -1 {
						malformedParams = true
						break
					}
					valueStart := pStart + pNameEnd + valMarkerIdx + 1
					if valueStart > len(paramsPart) {
						malformedParams = true
						break
					}

					// Search for closing </parameter>
					valueEndMarkerIdx := strings.Index(paramsPart[valueStart:], paramClose)
					if valueEndMarkerIdx == -1 {
						malformedParams = true
						break
					}
					valueEnd := valueStart + valueEndMarkerIdx
					
					// Extract and decode entities
					value := html.UnescapeString(paramsPart[valueStart:valueEnd])
					args[pName] = value

					// Advance to the next parameter
					nextParamStart := valueEnd + len(paramClose)
					if nextParamStart > len(paramsPart) {
						paramsPart = ""
						break
					}
					paramsPart = paramsPart[nextParamStart:]
				}

				if !malformedParams {
					toolCalls = append(toolCalls, ToolCall{
						ID:        fmt.Sprintf("minimax-%d-%d", time.Now().UnixNano(), miniMaxIdx),
						Name:      toolName,
						Arguments: args,
					})
				}
			}
		}

		content = strings.TrimSpace(content[:tagStart] + content[tagEndIdx+tagLen:])

		// Override FinishReason so the loop agent knows to execute it
		if choice.FinishReason != "tool_calls" {
			choice.FinishReason = "tool_calls"
		}
	}

	// --- 3. Process standard OpenAI JSON tool calls ---
	for _, tc := range choice.Message.ToolCalls {
		arguments := make(map[string]any)
		name := ""

		// Extract thought_signature from Gemini/Google-specific extra content
		thoughtSignature := ""
		if tc.ExtraContent != nil && tc.ExtraContent.Google != nil {
			thoughtSignature = tc.ExtraContent.Google.ThoughtSignature
		}

		if tc.Function != nil {
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					log.Printf("openai_compat: failed to decode tool call arguments for %q: %v", name, err)
					arguments["raw"] = tc.Function.Arguments
				}
			}
		}

		// Build ToolCall with ExtraContent for Gemini 3 thought_signature persistence
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
		Content:          content,
		ReasoningContent: reasoningContent,
		ToolCalls:        toolCalls,
		FinishReason:     choice.FinishReason,
		Usage:            apiResponse.Usage,
	}, nil
}

// openaiMessage is the wire-format message for OpenAI-compatible APIs.
// It mirrors protocoltypes.Message but omits SystemParts, which is an
// internal field that would be unknown to third-party endpoints.
type openaiMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// stripSystemParts converts []Message to []openaiMessage, dropping the
// SystemParts field so it doesn't leak into the JSON payload sent to
// OpenAI-compatible APIs (some strict endpoints reject unknown fields).
func stripSystemParts(messages []Message) []openaiMessage {
	out := make([]openaiMessage, len(messages))
	for i, m := range messages {
		out[i] = openaiMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}
	return out
}

func normalizeModel(model, apiBase string) string {
	idx := strings.Index(model, "/")
	if idx == -1 {
		return model
	}

	if strings.Contains(strings.ToLower(apiBase), "openrouter.ai") {
		return model
	}

	prefix := strings.ToLower(model[:idx])
	switch prefix {
	case "moonshot", "nvidia", "groq", "ollama", "deepseek", "google", "openrouter", "zhipu", "mistral":
		return model[idx+1:]
	default:
		return model
	}
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
