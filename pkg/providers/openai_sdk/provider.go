package openai_sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

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

const (
	defaultModel          = "gpt-4o"
	defaultRequestTimeout = 120 * time.Second
)

type Provider struct {
	apiBase    string
	httpClient *http.Client
	client     *openai.Client
}

type Option func(*Provider)

func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

func NewProvider(apiKey, apiBase, proxy string, opts ...Option) *Provider {
	httpClient := &http.Client{Timeout: defaultRequestTimeout}
	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err == nil {
			httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(parsed)}
		} else {
			log.Printf("openai_sdk: invalid proxy URL %q: %v", proxy, err)
		}
	}

	p := &Provider{
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: httpClient,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	reqOpts := []option.RequestOption{
		option.WithBaseURL(p.apiBase),
		option.WithHTTPClient(p.httpClient),
	}
	if apiKey != "" {
		reqOpts = append(reqOpts, option.WithAPIKey(apiKey))
	}
	client := openai.NewClient(reqOpts...)
	p.client = &client
	return p
}

func (p *Provider) GetDefaultModel() string {
	return defaultModel
}

func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if strings.TrimSpace(p.apiBase) == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	params := openai.ChatCompletionNewParams{
		Model:    normalizeModel(model),
		Messages: buildChatMessages(messages),
	}

	if len(tools) > 0 {
		params.Tools = buildChatTools(tools)
		params.ToolChoice.OfAuto = openai.String(string(openai.ChatCompletionToolChoiceOptionAutoAuto))
	}
	applyOptions(&params, options)

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		var apiErr *openai.Error
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf(
				"OpenAI API request failed (status=%d): %s",
				apiErr.StatusCode,
				strings.TrimSpace(apiErr.Message),
			)
		}
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no choices")
	}

	choice := resp.Choices[0]
	return &LLMResponse{
		Content:      choice.Message.Content,
		ToolCalls:    parseChoiceToolCalls(choice.Message.ToolCalls),
		FinishReason: choice.FinishReason,
		Usage:        mapUsage(resp.Usage),
	}, nil
}

func normalizeModel(model string) string {
	trimmed := strings.TrimSpace(model)
	if strings.HasPrefix(strings.ToLower(trimmed), "openai/") {
		return trimmed[len("openai/"):]
	}
	return trimmed
}

func buildChatMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			out = append(out, openai.SystemMessage(msg.Content))
		case "assistant":
			out = append(out, buildAssistantMessage(msg))
		case "tool":
			out = append(out, openai.ToolMessage(msg.Content, msg.ToolCallID))
		case "user":
			fallthrough
		default:
			out = append(out, openai.UserMessage(msg.Content))
		}
	}
	return out
}

func buildAssistantMessage(msg Message) openai.ChatCompletionMessageParamUnion {
	assistant := openai.ChatCompletionAssistantMessageParam{}
	if msg.Content != "" {
		assistant.Content.OfString = openai.String(msg.Content)
	}
	if len(msg.ToolCalls) > 0 {
		assistant.ToolCalls = make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			name := tc.Name
			if name == "" && tc.Function != nil {
				name = tc.Function.Name
			}
			if name == "" {
				continue
			}
			args := "{}"
			if len(tc.Arguments) > 0 {
				if b, err := json.Marshal(tc.Arguments); err == nil {
					args = string(b)
				}
			}
			assistant.ToolCalls = append(assistant.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      name,
						Arguments: args,
					},
				},
			})
		}
	}
	return openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant}
}

func buildChatTools(tools []ToolDefinition) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		if tool.Function.Name == "" {
			continue
		}
		fn := shared.FunctionDefinitionParam{
			Name:        tool.Function.Name,
			Description: openai.String(tool.Function.Description),
			Parameters:  shared.FunctionParameters(tool.Function.Parameters),
		}
		out = append(out, openai.ChatCompletionFunctionTool(fn))
	}
	return out
}

func parseChoiceToolCalls(calls []openai.ChatCompletionMessageToolCallUnion) []ToolCall {
	if len(calls) == 0 {
		return nil
	}

	result := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		switch v := call.AsAny().(type) {
		case openai.ChatCompletionMessageFunctionToolCall:
			args := map[string]any{}
			if strings.TrimSpace(v.Function.Arguments) != "" {
				if err := json.Unmarshal([]byte(v.Function.Arguments), &args); err != nil {
					log.Printf("openai_sdk: failed to decode tool call arguments for %q: %v", v.Function.Name, err)
				}
			}
			result = append(result, ToolCall{
				ID:   v.ID,
				Type: "function",
				Function: &FunctionCall{
					Name:      v.Function.Name,
					Arguments: v.Function.Arguments,
				},
				Name:      v.Function.Name,
				Arguments: args,
			})
		}
	}
	return result
}

func applyOptions(
	params *openai.ChatCompletionNewParams,
	options map[string]any,
) {
	if params == nil || options == nil {
		return
	}
	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		params.MaxCompletionTokens = openai.Opt(int64(maxTokens))
	}
	if temp, ok := asFloat(options["temperature"]); ok {
		params.Temperature = openai.Opt(temp)
	}
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && strings.TrimSpace(cacheKey) != "" {
		params.PromptCacheKey = openai.String(cacheKey)
	}
}

func mapUsage(usage openai.CompletionUsage) *UsageInfo {
	if usage.TotalTokens == 0 && usage.PromptTokens == 0 && usage.CompletionTokens == 0 {
		return nil
	}
	return &UsageInfo{
		PromptTokens:     int(usage.PromptTokens),
		CompletionTokens: int(usage.CompletionTokens),
		TotalTokens:      int(usage.TotalTokens),
	}
}

func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int8:
		return int(x), true
	case int16:
		return int(x), true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	default:
		return 0, false
	}
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}
