package openai_compat

import (
	"bufio"
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
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
)

type Provider struct {
	apiKey         string
	apiBase        string
	endpointPath   string // API path appended to apiBase (default: "/chat/completions")
	maxTokensField string // Field name for max tokens (e.g., "max_completion_tokens" for o1/glm models)
	stream         bool   // Use SSE streaming internally (accumulates into a single LLMResponse)
	httpClient     *http.Client
}

// Options configures optional behaviour for the provider.
type Options struct {
	EndpointPath   string // API path appended to apiBase (default: "/chat/completions")
	MaxTokensField string // Field name for max tokens parameter
	Stream         bool   // Use SSE streaming internally
}

func NewProvider(apiKey, apiBase, proxy string) *Provider {
	return NewProviderWithMaxTokensField(apiKey, apiBase, proxy, "")
}

func NewProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField string) *Provider {
	return NewProviderWithOptions(apiKey, apiBase, proxy, Options{
		MaxTokensField: maxTokensField,
	})
}

func NewProviderWithOptions(apiKey, apiBase, proxy string, opts Options) *Provider {
	timeout := 120 * time.Second
	if opts.Stream {
		timeout = 5 * time.Minute
	}
	client := &http.Client{
		Timeout: timeout,
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

	endpointPath := opts.EndpointPath
	if endpointPath == "" {
		endpointPath = "/chat/completions"
	}

	return &Provider{
		apiKey:         apiKey,
		apiBase:        strings.TrimRight(apiBase, "/"),
		endpointPath:   endpointPath,
		maxTokensField: opts.MaxTokensField,
		stream:         opts.Stream,
		httpClient:     client,
	}
}

// streamBufferSize is the channel buffer size for ChatStream events.
const streamBufferSize = 32

// buildHTTPRequest constructs a ready-to-send *http.Request for the chat API.
func (p *Provider) buildHTTPRequest(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
	stream bool,
) (*http.Request, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	model = normalizeModel(model, p.apiBase)

	requestBody := map[string]any{
		"model":    model,
		"messages": messages,
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

	if stream {
		requestBody["stream"] = true
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+p.endpointPath, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
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
	// When streaming is enabled, delegate to ChatStream + AccumulateStream
	// so that the SSE→channel path is always exercised.
	if p.stream {
		ch, err := p.ChatStream(ctx, messages, tools, model, options)
		if err != nil {
			return nil, err
		}
		return AccumulateStream(ch)
	}

	req, err := p.buildHTTPRequest(ctx, messages, tools, model, options, false)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return parseResponse(body)
}

// CanStream returns true when this provider is configured for SSE streaming.
func (p *Provider) CanStream() bool {
	return p.stream
}

// ChatStream opens an SSE connection and returns a channel of StreamEvent.
// The channel is closed when the stream ends or an error occurs.
// Cancelling ctx will abort the HTTP request and close the channel.
func (p *Provider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (<-chan protocoltypes.StreamEvent, error) {
	req, err := p.buildHTTPRequest(ctx, messages, tools, model, options, true)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))
	}

	ch := make(chan protocoltypes.StreamEvent, streamBufferSize)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		readSSEIntoChannel(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// readSSEIntoChannel reads SSE lines from r and sends StreamEvent values on ch.
// It returns when the stream ends, an error occurs, or ctx is cancelled.
func readSSEIntoChannel(ctx context.Context, r io.Reader, ch chan<- protocoltypes.StreamEvent) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		// Check for context cancellation between lines.
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		ev := protocoltypes.StreamEvent{}

		if chunk.Usage != nil {
			ev.Usage = chunk.Usage
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			ev.ContentDelta = choice.Delta.Content
			if choice.FinishReason != "" {
				ev.FinishReason = choice.FinishReason
			}
			for _, tc := range choice.Delta.ToolCalls {
				delta := protocoltypes.StreamToolCallDelta{
					Index: tc.Index,
					ID:    tc.ID,
				}
				if tc.Function != nil {
					delta.Name = tc.Function.Name
					delta.ArgumentsDelta = tc.Function.Arguments
				}
				ev.ToolCallDeltas = append(ev.ToolCallDeltas, delta)
			}
		}

		select {
		case ch <- ev:
		case <-ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case ch <- protocoltypes.StreamEvent{Err: fmt.Errorf("reading stream: %w", err)}:
		case <-ctx.Done():
		}
	}
}

// AccumulateStream drains a StreamEvent channel and returns a complete LLMResponse.
func AccumulateStream(ch <-chan protocoltypes.StreamEvent) (*LLMResponse, error) {
	var content strings.Builder
	var toolCalls []streamToolCallAcc
	var finishReason string
	var usage *UsageInfo

	for ev := range ch {
		if ev.Err != nil {
			return nil, ev.Err
		}
		if ev.ContentDelta != "" {
			content.WriteString(ev.ContentDelta)
		}
		if ev.FinishReason != "" {
			finishReason = ev.FinishReason
		}
		if ev.Usage != nil {
			usage = ev.Usage
		}
		for _, tc := range ev.ToolCallDeltas {
			for len(toolCalls) <= tc.Index {
				toolCalls = append(toolCalls, streamToolCallAcc{})
			}
			if tc.ID != "" {
				toolCalls[tc.Index].ID = tc.ID
			}
			if tc.Name != "" {
				toolCalls[tc.Index].Name = tc.Name
			}
			toolCalls[tc.Index].Arguments.WriteString(tc.ArgumentsDelta)
		}
	}

	result := &LLMResponse{
		Content:      content.String(),
		FinishReason: finishReason,
		Usage:        usage,
	}

	for _, tc := range toolCalls {
		arguments := make(map[string]any)
		argStr := tc.Arguments.String()
		if argStr != "" {
			if err := json.Unmarshal([]byte(argStr), &arguments); err != nil {
				log.Printf("openai_compat: failed to decode streamed tool call arguments for %q: %v", tc.Name, err)
				arguments["raw"] = argStr
			}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: arguments,
		})
	}

	return result, nil
}

func parseResponse(body []byte) (*LLMResponse, error) {
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
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: choice.FinishReason,
		Usage:        apiResponse.Usage,
	}, nil
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
	case "moonshot", "nvidia", "groq", "ollama", "deepseek", "google", "openrouter", "zhipu", "minimax", "mistral":
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

// --- SSE streaming support ---

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
	Usage   *UsageInfo     `json:"usage"`
}

type streamChoice struct {
	Delta        streamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type streamDelta struct {
	Content   string          `json:"content"`
	ToolCalls []streamDeltaTC `json:"tool_calls"`
}

type streamDeltaTC struct {
	Index    int                  `json:"index"`
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function *streamDeltaFunction `json:"function"`
}

type streamDeltaFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type streamToolCallAcc struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// parseStreamResponse reads an SSE (text/event-stream) response and
// accumulates it into a single LLMResponse.
func parseStreamResponse(r io.Reader) (*LLMResponse, error) {
	scanner := bufio.NewScanner(r)
	// Allow up to 1 MB per SSE line to handle large argument deltas.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var content strings.Builder
	var toolCalls []streamToolCallAcc
	var finishReason string
	var usage *UsageInfo

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		if len(chunk.Choices) == 0 {
			if chunk.Usage != nil {
				usage = chunk.Usage
			}
			continue
		}

		choice := chunk.Choices[0]
		if choice.Delta.Content != "" {
			content.WriteString(choice.Delta.Content)
		}
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}

		// Accumulate streaming tool calls by index.
		for _, tc := range choice.Delta.ToolCalls {
			for len(toolCalls) <= tc.Index {
				toolCalls = append(toolCalls, streamToolCallAcc{})
			}
			if tc.ID != "" {
				toolCalls[tc.Index].ID = tc.ID
			}
			if tc.Function != nil {
				if tc.Function.Name != "" {
					toolCalls[tc.Index].Name = tc.Function.Name
				}
				toolCalls[tc.Index].Arguments.WriteString(tc.Function.Arguments)
			}
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}

	result := &LLMResponse{
		Content:      content.String(),
		FinishReason: finishReason,
		Usage:        usage,
	}

	for _, tc := range toolCalls {
		arguments := make(map[string]any)
		argStr := tc.Arguments.String()
		if argStr != "" {
			if err := json.Unmarshal([]byte(argStr), &arguments); err != nil {
				log.Printf("openai_compat: failed to decode streamed tool call arguments for %q: %v", tc.Name, err)
				arguments["raw"] = argStr
			}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: arguments,
		})
	}

	return result, nil
}
