package openai_compat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
	ReasoningDetail        = protocoltypes.ReasoningDetail
)

type Provider struct {
	apiKey          string
	apiBase         string
	maxTokensField  string // Field name for max tokens (e.g., "max_completion_tokens" for o1/glm models)
	httpClient      *http.Client
	extraBody       map[string]any // Additional fields to inject into request body
	preferResponses bool           // Prefer /responses for OpenAI-native configs.
}

type Option func(*Provider)

const defaultRequestTimeout = common.DefaultRequestTimeout

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

func WithResponsesPreferred() Option {
	return func(p *Provider) {
		p.preferResponses = true
	}
}

func WithExtraBody(extraBody map[string]any) Option {
	return func(p *Provider) {
		p.extraBody = extraBody
	}
}

func NewProvider(apiKey, apiBase, proxy string, opts ...Option) *Provider {
	p := &Provider{
		apiKey:     apiKey,
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: common.NewHTTPClient(proxy),
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

// buildRequestBody constructs the common request body for Chat and ChatStream.
func (p *Provider) buildRequestBody(
	messages []Message, tools []ToolDefinition, model string, options map[string]any,
) map[string]any {
	requestBody := map[string]any{
		"model":    model,
		"messages": common.SerializeMessages(messages),
	}

	// When fallback uses a different provider (e.g. DeepSeek), that provider must not inject web_search_preview.
	nativeSearch, _ := options["native_search"].(bool)
	nativeSearch = nativeSearch && isNativeSearchHost(p.apiBase)
	if len(tools) > 0 || nativeSearch {
		requestBody["tools"] = buildToolsList(tools, nativeSearch)
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok {
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

	if temperature, ok := requestTemperature(model, options); ok {
		requestBody["temperature"] = temperature
	}

	// Prompt caching: pass a stable cache key so OpenAI can bucket requests
	// with the same key and reuse prefix KV cache across calls.
	// Prompt caching is only supported by OpenAI-native endpoints.
	// Non-OpenAI providers reject unknown fields with 422 errors.
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		if supportsPromptCacheKey(p.apiBase) {
			requestBody["prompt_cache_key"] = cacheKey
		}
	}

	// Merge extra body fields configured per-provider/model.
	// These are injected last so they take precedence over defaults.
	for k, v := range p.extraBody {
		requestBody[k] = v
	}

	return requestBody
}

func requestTemperature(model string, options map[string]any) (float64, bool) {
	temperature, ok := common.AsFloat(options["temperature"])
	if !ok {
		return 0, false
	}

	lowerModel := strings.ToLower(model)
	if strings.Contains(lowerModel, "kimi") && strings.Contains(lowerModel, "k2") {
		return 1.0, true
	}

	return temperature, true
}

func shouldPreferResponses(rawModel, normalizedModel string, preferOpenAIModels bool) bool {
	rawModel = strings.ToLower(strings.TrimSpace(rawModel))
	normalizedModel = strings.ToLower(strings.TrimSpace(normalizedModel))

	return preferOpenAIModels ||
		strings.HasPrefix(rawModel, "gpt-5") ||
		strings.HasPrefix(normalizedModel, "gpt-5")
}

func hasReasoningContentHistory(messages []Message) bool {
	for _, message := range messages {
		if strings.TrimSpace(message.ReasoningContent) != "" {
			return true
		}
	}

	return false
}

func (p *Provider) buildResponsesRequestBody(
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (map[string]any, error) {
	input, err := buildResponsesInput(messages)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"model": model,
		"input": input,
	}

	nativeSearch, _ := options["native_search"].(bool)
	nativeSearch = nativeSearch && isNativeSearchHost(p.apiBase)
	responseTools := buildResponsesToolsList(tools, nativeSearch)
	if len(responseTools) > 0 {
		requestBody["tools"] = responseTools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok {
		requestBody["max_output_tokens"] = maxTokens
	}

	if temperature, ok := requestTemperature(model, options); ok {
		requestBody["temperature"] = temperature
	}

	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		if supportsPromptCacheKey(p.apiBase) {
			requestBody["prompt_cache_key"] = cacheKey
		}
	}

	for k, v := range p.extraBody {
		requestBody[k] = v
	}

	return requestBody, nil
}

func buildResponsesInput(messages []Message) ([]any, error) {
	input := make([]any, 0, len(messages))

	for _, m := range messages {
		switch m.Role {
		case "system", "user":
			input = append(input, map[string]any{
				"type":    "message",
				"role":    m.Role,
				"content": serializeResponsesMessageContent(m),
			})
		case "assistant":
			if strings.TrimSpace(m.Content) != "" || strings.TrimSpace(m.ReasoningContent) != "" || len(m.Media) > 0 || len(m.ToolCalls) == 0 {
				input = append(input, map[string]any{
					"type":    "message",
					"role":    m.Role,
					"content": serializeResponsesMessageContent(m),
				})
			}

			for _, tc := range m.ToolCalls {
				name, args, ok := resolveResponseToolCall(tc)
				if !ok {
					log.Printf("openai_compat: skipping invalid assistant tool call in responses history: id=%q", tc.ID)
					continue
				}
				input = append(input, map[string]any{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      name,
					"arguments": args,
				})
			}
		case "tool":
			if strings.TrimSpace(m.ToolCallID) == "" {
				return nil, fmt.Errorf("tool message missing tool_call_id")
			}
			input = append(input, map[string]any{
				"type":    "function_call_output",
				"call_id": m.ToolCallID,
				"output":  m.Content,
			})
		default:
			return nil, fmt.Errorf("unsupported message role: %s", m.Role)
		}
	}

	return input, nil
}

func serializeResponsesMessageContent(m Message) any {
	effectiveText := m.Content
	if effectiveText == "" {
		effectiveText = m.ReasoningContent
	}

	if len(m.Media) == 0 {
		return effectiveText
	}

	parts := make([]map[string]any, 0, 1+len(m.Media))
	if effectiveText != "" {
		parts = append(parts, map[string]any{
			"type": "input_text",
			"text": effectiveText,
		})
	}

	for _, mediaURL := range m.Media {
		if strings.HasPrefix(mediaURL, "data:image/") {
			parts = append(parts, map[string]any{
				"type":      "input_image",
				"image_url": mediaURL,
			})
		}
	}

	if len(parts) == 0 {
		return effectiveText
	}

	return parts
}

func buildResponsesToolsList(tools []ToolDefinition, nativeSearch bool) []any {
	result := make([]any, 0, len(tools)+1)
	for _, tool := range tools {
		if nativeSearch && strings.EqualFold(tool.Function.Name, "web_search") {
			continue
		}
		if tool.Type != "" && tool.Type != "function" {
			continue
		}

		entry := map[string]any{
			"type":       "function",
			"name":       tool.Function.Name,
			"parameters": tool.Function.Parameters,
		}
		if entry["parameters"] == nil {
			entry["parameters"] = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		if tool.Function.Description != "" {
			entry["description"] = tool.Function.Description
		}

		result = append(result, entry)
	}

	if nativeSearch {
		result = append(result, map[string]any{"type": "web_search_preview"})
	}

	return result
}

func resolveResponseToolCall(tc ToolCall) (name string, arguments string, ok bool) {
	name = tc.Name
	if name == "" && tc.Function != nil {
		name = tc.Function.Name
	}
	if name == "" {
		return "", "", false
	}

	if len(tc.Arguments) > 0 {
		argsJSON, err := json.Marshal(tc.Arguments)
		if err != nil {
			return "", "", false
		}
		return name, string(argsJSON), true
	}

	if tc.Function != nil && tc.Function.Arguments != "" {
		return name, tc.Function.Arguments, true
	}

	return name, "{}", true
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

	normalizedModel := normalizeModel(model, p.apiBase)
	if shouldPreferResponses(model, normalizedModel, p.preferResponses) && !hasReasoningContentHistory(messages) {
		out, err := p.chatResponses(ctx, messages, tools, normalizedModel, options)
		if err == nil {
			return out, nil
		}
		if ctx.Err() != nil {
			return nil, err
		}
		log.Printf("openai_compat: /responses failed for %q, falling back to /chat/completions: %v", normalizedModel, err)

		fallbackOut, fallbackErr := p.chatCompletions(ctx, messages, tools, normalizedModel, options)
		if fallbackErr != nil {
			return nil, fmt.Errorf("responses request failed; fallback chat/completions failed: %w", errors.Join(err, fallbackErr))
		}
		return fallbackOut, nil
	}

	return p.chatCompletions(ctx, messages, tools, normalizedModel, options)
}

func (p *Provider) chatCompletions(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	requestBody := p.buildRequestBody(messages, tools, model, options)
	return p.doRequest(ctx, "/chat/completions", requestBody, nil)
}

func (p *Provider) chatResponses(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	requestBody, err := p.buildResponsesRequestBody(messages, tools, model, options)
	if err != nil {
		return nil, err
	}
	return p.doRequest(ctx, "/responses", requestBody, parseResponsesResponse)
}

func (p *Provider) doRequest(
	ctx context.Context,
	path string,
	requestBody map[string]any,
	parse func(io.Reader) (*LLMResponse, error),
) (*LLMResponse, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+path, bytes.NewReader(jsonData))
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

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	if parse == nil {
		return common.ReadAndParseResponse(resp, p.apiBase)
	}

	contentType := resp.Header.Get("Content-Type")
	reader := bufio.NewReader(resp.Body)
	prefix, err := reader.Peek(256)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return nil, fmt.Errorf("failed to inspect response: %w", err)
	}
	if common.LooksLikeHTML(prefix, contentType) {
		return nil, common.WrapHTMLResponseError(resp.StatusCode, prefix, contentType, p.apiBase)
	}

	out, err := parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return out, nil
}

// ChatStream implements streaming via OpenAI-compatible SSE (stream: true).
// onChunk receives the accumulated text so far on each text delta.
func (p *Provider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
	onChunk func(accumulated string),
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	normalizedModel := normalizeModel(model, p.apiBase)
	requestBody := p.buildRequestBody(messages, tools, normalizedModel, options)
	requestBody["stream"] = true

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Use a client without Timeout for streaming — the http.Client.Timeout covers
	// the entire request lifecycle including body reads, which would kill long streams.
	// Context cancellation still provides the safety net.
	streamClient := &http.Client{Transport: p.httpClient.Transport}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	return parseStreamResponse(ctx, resp.Body, onChunk)
}

// parseStreamResponse parses an OpenAI-compatible SSE stream.
func parseStreamResponse(
	ctx context.Context,
	reader io.Reader,
	onChunk func(accumulated string),
) (*LLMResponse, error) {
	var textContent strings.Builder
	var finishReason string
	var usage *UsageInfo

	// Tool call assembly: OpenAI streams tool calls as incremental deltas
	type toolAccum struct {
		id       string
		name     string
		argsJSON strings.Builder
	}
	activeTools := map[int]*toolAccum{}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 1MB initial, 10MB max
	for scanner.Scan() {
		// Check for context cancellation between chunks
		if err := ctx.Err(); err != nil {
			return nil, err
		}

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
						Function *struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *UsageInfo `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]

		// Accumulate text content
		if choice.Delta.Content != "" {
			textContent.WriteString(choice.Delta.Content)
			if onChunk != nil {
				onChunk(textContent.String())
			}
		}

		// Accumulate tool call deltas
		for _, tc := range choice.Delta.ToolCalls {
			acc, ok := activeTools[tc.Index]
			if !ok {
				acc = &toolAccum{}
				activeTools[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Function != nil {
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					acc.argsJSON.WriteString(tc.Function.Arguments)
				}
			}
		}

		if choice.FinishReason != nil {
			finishReason = *choice.FinishReason
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("streaming read error: %w", err)
	}

	// Assemble tool calls from accumulated deltas
	var toolCalls []ToolCall
	for i := 0; i < len(activeTools); i++ {
		acc, ok := activeTools[i]
		if !ok {
			continue
		}
		args := make(map[string]any)
		raw := acc.argsJSON.String()
		if raw != "" {
			if err := json.Unmarshal([]byte(raw), &args); err != nil {
				log.Printf("openai_compat stream: failed to decode tool call arguments for %q: %v", acc.name, err)
				args["raw"] = raw
			}
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:        acc.id,
			Name:      acc.name,
			Arguments: args,
		})
	}

	if finishReason == "" {
		finishReason = "stop"
	}

	return &LLMResponse{
		Content:      textContent.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

func parseResponsesResponse(body io.Reader) (*LLMResponse, error) {
	var apiResponse struct {
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
		Output []struct {
			ID        string `json:"id"`
			Type      string `json:"type"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
			Summary   []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"summary"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	status := strings.TrimSpace(apiResponse.Status)
	switch status {
	case "completed", "incomplete":
		if len(apiResponse.Output) == 0 {
			return nil, errors.New("openai responses returned terminal status with empty output")
		}
	case "failed":
		if apiResponse.Error != nil {
			if msg := strings.TrimSpace(apiResponse.Error.Message); msg != "" {
				return nil, errors.New(msg)
			}
		}
		return nil, errors.New("openai responses request failed")
	default:
		return nil, fmt.Errorf("openai responses returned unexpected or non-terminal status: %q", status)
	}

	var content strings.Builder
	var reasoning strings.Builder
	var reasoningContent strings.Builder
	reasoningDetails := make([]ReasoningDetail, 0)
	toolCalls := make([]ToolCall, 0)
	for _, item := range apiResponse.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Text != "" {
					content.WriteString(part.Text)
					continue
				}
				if part.Refusal != "" {
					content.WriteString(part.Refusal)
				}
			}
		case "reasoning":
			for _, part := range item.Summary {
				if part.Text == "" {
					continue
				}
				if reasoning.Len() > 0 {
					reasoning.WriteString("\n")
				}
				reasoning.WriteString(part.Text)
				reasoningDetails = append(reasoningDetails, ReasoningDetail{
					Format: "text",
					Index:  len(reasoningDetails),
					Type:   part.Type,
					Text:   part.Text,
				})
			}
			for _, part := range item.Content {
				if part.Text == "" {
					continue
				}
				if reasoningContent.Len() > 0 {
					reasoningContent.WriteString("\n")
				}
				reasoningContent.WriteString(part.Text)
				reasoningDetails = append(reasoningDetails, ReasoningDetail{
					Format: "text",
					Index:  len(reasoningDetails),
					Type:   part.Type,
					Text:   part.Text,
				})
			}
		case "function_call":
			arguments := make(map[string]any)
			if item.Arguments != "" {
				if err := json.Unmarshal([]byte(item.Arguments), &arguments); err != nil {
					log.Printf("openai_compat: failed to decode responses tool call arguments for %q: %v", item.Name, err)
					arguments["raw"] = item.Arguments
				}
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:        firstNonEmpty(item.CallID, item.ID),
				Name:      item.Name,
				Arguments: arguments,
			})
		}
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	} else if status == "incomplete" {
		finishReason = "length"
		if apiResponse.IncompleteDetails != nil && apiResponse.IncompleteDetails.Reason != "" && apiResponse.IncompleteDetails.Reason != "max_output_tokens" {
			finishReason = apiResponse.IncompleteDetails.Reason
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

	return &LLMResponse{
		Content:          content.String(),
		ReasoningContent: reasoningContent.String(),
		Reasoning:        reasoning.String(),
		ReasoningDetails: reasoningDetails,
		ToolCalls:        toolCalls,
		FinishReason:     finishReason,
		Usage:            usage,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func normalizeModel(model, apiBase string) string {
	before, after, ok := strings.Cut(model, "/")
	if !ok {
		return model
	}

	if strings.Contains(strings.ToLower(apiBase), "openrouter.ai") {
		return model
	}

	prefix := strings.ToLower(before)
	switch prefix {
	case "litellm", "moonshot", "nvidia", "groq", "ollama", "deepseek", "google",
		"openrouter", "zhipu", "mistral", "vivgrid", "minimax", "novita":
		return after
	default:
		return model
	}
}

func buildToolsList(tools []ToolDefinition, nativeSearch bool) []any {
	result := make([]any, 0, len(tools)+1)
	for _, t := range tools {
		if nativeSearch && strings.EqualFold(t.Function.Name, "web_search") {
			continue
		}
		result = append(result, t)
	}
	if nativeSearch {
		result = append(result, map[string]any{"type": "web_search_preview"})
	}
	return result
}

func (p *Provider) SupportsNativeSearch() bool {
	return isNativeSearchHost(p.apiBase)
}

func isNativeSearchHost(apiBase string) bool {
	u, err := url.Parse(apiBase)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "api.openai.com" || strings.HasSuffix(host, ".openai.azure.com")
}

// supportsPromptCacheKey reports whether the given API base is known to
// support the prompt_cache_key request field. Currently only OpenAI's own
// API and Azure OpenAI support this. All other OpenAI-compatible providers
// (Mistral, Gemini, DeepSeek, Groq, etc.) reject unknown fields with 422 errors.
func supportsPromptCacheKey(apiBase string) bool {
	u, err := url.Parse(apiBase)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "api.openai.com" || strings.HasSuffix(host, ".openai.azure.com")
}
