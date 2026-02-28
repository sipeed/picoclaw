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
	"sync"
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
	endpointPath   string // API path appended to apiBase (default: "/chat/completions")
	maxTokensField string // Field name for max tokens (e.g., "max_completion_tokens" for o1/glm models)
	stream         bool   // Use SSE streaming internally (accumulates into a single LLMResponse)
	httpClient     *http.Client

	// Rate limiting: minimum interval between consecutive API requests.
	// Shared across all goroutines using this provider instance.
	mu            sync.Mutex
	lastRequestAt time.Time
	minInterval   time.Duration
}

// Option is a functional option for configuring a Provider.
type Option func(*Provider)

const defaultRequestTimeout = 120 * time.Second

// WithMaxTokensField sets the field name for max tokens (e.g., "max_completion_tokens").
func WithMaxTokensField(maxTokensField string) Option {
	return func(p *Provider) {
		p.maxTokensField = maxTokensField
	}
}

// WithRequestTimeout overrides the HTTP client timeout.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

// WithStream enables SSE streaming mode.
func WithStream(stream bool) Option {
	return func(p *Provider) {
		p.stream = stream
		if stream && p.httpClient.Timeout == defaultRequestTimeout {
			p.httpClient.Timeout = 5 * time.Minute
		}
	}
}

// WithMinInterval sets the minimum interval between consecutive API requests.
// This prevents rate limit errors when many subagents share the same provider.
func WithMinInterval(d time.Duration) Option {
	return func(p *Provider) {
		p.minInterval = d
	}
}

// WithEndpointPath sets the API path appended to apiBase (default: "/chat/completions").
func WithEndpointPath(path string) Option {
	return func(p *Provider) {
		if path != "" {
			p.endpointPath = path
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
		apiKey:       apiKey,
		apiBase:      strings.TrimRight(apiBase, "/"),
		endpointPath: "/chat/completions",
		httpClient:   client,
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

	if stream {
		requestBody["stream"] = true
	}

	// Prompt caching: pass a stable cache key so OpenAI can bucket requests
	// with the same key and reuse prefix KV cache across calls.
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		if !strings.Contains(p.apiBase, "generativelanguage.googleapis.com") {
			requestBody["prompt_cache_key"] = cacheKey
		}
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

// waitForInterval enforces the minimum interval between consecutive API requests.
// It sleeps if needed, then records the current time as the last request time.
func (p *Provider) waitForInterval() {
	if p.minInterval <= 0 {
		return
	}
	p.mu.Lock()
	if !p.lastRequestAt.IsZero() {
		elapsed := time.Since(p.lastRequestAt)
		if wait := p.minInterval - elapsed; wait > 0 {
			p.mu.Unlock()
			time.Sleep(wait)
			p.mu.Lock()
		}
	}
	p.lastRequestAt = time.Now()
	p.mu.Unlock()
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

	p.waitForInterval()

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
// Canceling ctx will abort the HTTP request and close the channel.
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

	p.waitForInterval()

	resp, err := p.httpClient.Do(req) //nolint:bodyclose // closed in goroutine or error path below
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
// It returns when the stream ends, an error occurs, or ctx is canceled.
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
			ev.ReasoningDelta = choice.Delta.ReasoningContent
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
	var reasoning strings.Builder
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
		if ev.ReasoningDelta != "" {
			reasoning.WriteString(ev.ReasoningDelta)
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
		Reasoning:    reasoning.String(),
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
		Content:          choice.Message.Content,
		ReasoningContent: choice.Message.ReasoningContent,
		Reasoning:        choice.Message.Reasoning,
		ReasoningDetails: choice.Message.ReasoningDetails,
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
	case "openai",
		"moonshot",
		"nvidia",
		"groq",
		"ollama",
		"deepseek",
		"google",
		"openrouter",
		"zhipu",
		"minimax",
		"mistral":
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
	Content          string          `json:"content"`
	ReasoningContent string          `json:"reasoning_content"`
	ToolCalls        []streamDeltaTC `json:"tool_calls"`
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
