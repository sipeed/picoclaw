package oauthprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/logger"
	orc "github.com/sipeed/picoclaw/pkg/providers/openai_responses_common"
)

const (
	codexDefaultModel        = "gpt-5.3-codex"
	codexDefaultInstructions = "You are Codex, a coding assistant."
)

type CodexProvider struct {
	client          *openai.Client
	accountID       string
	tokenSource     func() (string, string, error)
	enableWebSearch bool
}

const defaultCodexInstructions = "You are Codex, a coding assistant."

func NewCodexProvider(token, accountID string) *CodexProvider {
	opts := []option.RequestOption{
		option.WithBaseURL("https://chatgpt.com/backend-api/codex"),
		option.WithAPIKey(token),
		option.WithHeader("originator", "codex_cli_rs"),
		option.WithHeader("OpenAI-Beta", "responses=experimental"),
	}
	if accountID != "" {
		opts = append(opts, option.WithHeader("Chatgpt-Account-Id", accountID))
	}
	client := openai.NewClient(opts...)
	return &CodexProvider{
		client:          &client,
		accountID:       accountID,
		enableWebSearch: true,
	}
}

func NewCodexProviderWithTokenSource(
	token, accountID string, tokenSource func() (string, string, error),
) *CodexProvider {
	p := NewCodexProvider(token, accountID)
	p.tokenSource = tokenSource
	return p
}

func (p *CodexProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
) (*LLMResponse, error) {
	var opts []option.RequestOption
	accountID := p.accountID
	resolvedModel, fallbackReason := resolveCodexModel(model)
	if fallbackReason != "" {
		logger.WarnCF(
			"provider.codex",
			"Requested model is not compatible with Codex backend, using fallback",
			map[string]any{
				"requested_model": model,
				"resolved_model":  resolvedModel,
				"reason":          fallbackReason,
			},
		)
	}
	if p.tokenSource != nil {
		tok, accID, err := p.tokenSource()
		if err != nil {
			return nil, fmt.Errorf("refreshing token: %w", err)
		}
		opts = append(opts, option.WithAPIKey(tok))
		if accID != "" {
			accountID = accID
		}
	}
	if accountID != "" {
		opts = append(opts, option.WithHeader("Chatgpt-Account-Id", accountID))
	} else {
		logger.WarnCF(
			"provider.codex",
			"No account id found for Codex request; backend may reject with 400",
			map[string]any{
				"requested_model": model,
				"resolved_model":  resolvedModel,
			},
		)
	}

	// Respect tools.web.prefer_native: only inject native search when the agent
	// loop passes options["native_search"]=true, so prefer_native=false means no injection.
	useNativeSearch := p.enableWebSearch && (options["native_search"] == true)
	params := buildCodexParams(messages, tools, resolvedModel, options, useNativeSearch)

	stream := p.client.Responses.NewStreaming(ctx, params, opts...)
	defer stream.Close()

	var resp *responses.Response
	var streamText strings.Builder
	var streamToolCalls []ToolCall
	for stream.Next() {
		evt := stream.Current()
		if evt.Type == "response.output_text.done" {
			textDone := evt.AsResponseOutputTextDone()
			if textDone.Text != "" {
				streamText.Reset()
				streamText.WriteString(textDone.Text)
			}
		}
		if evt.Type == "response.output_item.done" {
			done := evt.AsResponseOutputItemDone()
			if tc, ok := codexToolCallFromOutputItem(done.Item); ok {
				streamToolCalls = append(streamToolCalls, tc)
			}
		}
		if evt.Type == "response.completed" || evt.Type == "response.failed" || evt.Type == "response.incomplete" {
			evtResp := evt.Response
			if evtResp.ID != "" {
				evtRespCopy := evtResp
				resp = &evtRespCopy
			}
		}
	}
	err := stream.Err()
	if err != nil {
		fields := map[string]any{
			"requested_model":    model,
			"resolved_model":     resolvedModel,
			"messages_count":     len(messages),
			"tools_count":        len(tools),
			"account_id_present": accountID != "",
			"error":              err.Error(),
		}
		var apiErr *openai.Error
		if errors.As(err, &apiErr) {
			fields["status_code"] = apiErr.StatusCode
			fields["api_type"] = apiErr.Type
			fields["api_code"] = apiErr.Code
			fields["api_param"] = apiErr.Param
			fields["api_message"] = apiErr.Message
			if apiErr.StatusCode == 400 {
				fields["hint"] = "verify account id header and model compatibility for codex backend"
			}
			if apiErr.Response != nil {
				fields["request_id"] = apiErr.Response.Header.Get("x-request-id")
			}
		}
		logger.ErrorCF("provider.codex", "Codex API call failed", fields)
		return nil, fmt.Errorf("codex API call: %w", err)
	}
	if resp == nil {
		fields := map[string]any{
			"requested_model":    model,
			"resolved_model":     resolvedModel,
			"messages_count":     len(messages),
			"tools_count":        len(tools),
			"account_id_present": accountID != "",
		}
		logger.ErrorCF("provider.codex", "Codex stream ended without completed response event", fields)
		return nil, fmt.Errorf("codex API call: stream ended without completed response")
	}

	parsed := orc.ParseResponseFromStruct(resp)
	if parsed.Content == "" && len(parsed.ToolCalls) == 0 && streamText.Len() > 0 {
		parsed.Content = streamText.String()
	}
	if len(parsed.ToolCalls) == 0 && len(streamToolCalls) > 0 {
		parsed.ToolCalls = streamToolCalls
		parsed.FinishReason = "tool_calls"
	}
	return parsed, nil
}

func codexToolCallFromOutputItem(item responses.ResponseOutputItemUnion) (ToolCall, bool) {
	if item.Type != "function_call" {
		return ToolCall{}, false
	}

	call := item.AsFunctionCall()
	if call.Name == "" {
		return ToolCall{}, false
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		args = map[string]any{"raw": call.Arguments}
	}

	id := call.CallID
	if id == "" {
		id = call.ID
	}

	return ToolCall{
		ID:        id,
		Name:      call.Name,
		Arguments: args,
		Function: &FunctionCall{
			Name:      call.Name,
			Arguments: call.Arguments,
		},
	}, true
}

func (p *CodexProvider) GetDefaultModel() string {
	return codexDefaultModel
}

func (p *CodexProvider) SupportsNativeSearch() bool {
	return p.enableWebSearch
}

func (p *CodexProvider) SupportsThinking() bool {
	return true
}

func resolveCodexModel(model string) (string, string) {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return codexDefaultModel, "empty model"
	}

	if after, ok := strings.CutPrefix(m, "openai/"); ok {
		m = after
	} else if strings.Contains(m, "/") {
		return codexDefaultModel, "non-openai model namespace"
	}

	unsupportedPrefixes := []string{
		"glm",
		"claude",
		"anthropic",
		"gemini",
		"google",
		"moonshot",
		"kimi",
		"qwen",
		"deepseek",
		"llama",
		"meta-llama",
		"mistral",
		"grok",
		"xai",
		"zhipu",
	}
	for _, prefix := range unsupportedPrefixes {
		if strings.HasPrefix(m, prefix) {
			return codexDefaultModel, "unsupported model prefix"
		}
	}

	if strings.HasPrefix(m, "gpt-") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") {
		return m, ""
	}

	return codexDefaultModel, "unsupported model family"
}

func buildCodexParams(
	messages []Message, tools []ToolDefinition, model string, options map[string]any, enableWebSearch bool,
) responses.ResponseNewParams {
	inputItems, instructions := orc.TranslateMessages(messages)

	params := responses.ResponseNewParams{
		Model: model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
		Store: openai.Opt(false),
		Reasoning: shared.ReasoningParam{
			Effort: codexReasoningEffort(options["thinking_level"]),
		},
	}

	if instructions != "" {
		params.Instructions = openai.Opt(instructions)
	} else {
		// ChatGPT Codex backend requires instructions to be present.
		params.Instructions = openai.Opt(defaultCodexInstructions)
	}

	// Prompt caching: pass a stable cache key so OpenAI can bucket requests
	// and reuse prefix KV cache across calls with the same key.
	// See: https://platform.openai.com/docs/guides/prompt-caching
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		params.PromptCacheKey = openai.Opt(cacheKey)
	}

	if len(tools) > 0 || enableWebSearch {
		params.Tools = orc.TranslateTools(tools, enableWebSearch)
	}

	return params
}

func codexReasoningEffort(raw any) shared.ReasoningEffort {
	level, _ := raw.(string)
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "low":
		return shared.ReasoningEffortLow
	case "medium", "adaptive":
		return shared.ReasoningEffortMedium
	case "high":
		return shared.ReasoningEffortHigh
	case "xhigh", "max":
		return shared.ReasoningEffortXhigh
	default:
		return shared.ReasoningEffortNone
	}
}

func CreateCodexTokenSource() func() (string, string, error) {
	return func() (string, string, error) {
		return auth.GetOpenAIToken()
	}
}
