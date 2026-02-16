package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const codexDefaultModel = "gpt-5.2"
const codexDefaultInstructions = "You are Codex, a coding assistant."

type CodexProvider struct {
	client      *openai.Client
	accountID   string
	tokenSource func() (string, string, error)
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
		client:    &client,
		accountID: accountID,
	}
}

func NewCodexProviderWithTokenSource(token, accountID string, tokenSource func() (string, string, error)) *CodexProvider {
	p := NewCodexProvider(token, accountID)
	p.tokenSource = tokenSource
	return p
}

func (p *CodexProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	var opts []option.RequestOption
	accountID := p.accountID
	resolvedModel, fallbackReason := resolveCodexModel(model)
	if fallbackReason != "" {
		logger.WarnCF("provider.codex", "Requested model is not compatible with Codex backend, using fallback", map[string]interface{}{
			"requested_model": model,
			"resolved_model":  resolvedModel,
			"reason":          fallbackReason,
		})
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
		logger.WarnCF("provider.codex", "No account id found for Codex request; backend may reject with 400", map[string]interface{}{
			"requested_model": model,
			"resolved_model":  resolvedModel,
		})
	}

	params := buildCodexParams(messages, tools, resolvedModel, options)

	stream := p.client.Responses.NewStreaming(ctx, params, opts...)
	defer stream.Close()

	var resp *responses.Response
	for stream.Next() {
		evt := stream.Current()
		if evt.Type == "response.completed" || evt.Type == "response.failed" || evt.Type == "response.incomplete" {
			evtResp := evt.Response
			if evtResp.ID != "" {
				copy := evtResp
				resp = &copy
			}
		}
	}
	err := stream.Err()
	if err != nil {
		fields := map[string]interface{}{
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
		fields := map[string]interface{}{
			"requested_model":    model,
			"resolved_model":     resolvedModel,
			"messages_count":     len(messages),
			"tools_count":        len(tools),
			"account_id_present": accountID != "",
		}
		logger.ErrorCF("provider.codex", "Codex stream ended without completed response event", fields)
		return nil, fmt.Errorf("codex API call: stream ended without completed response")
	}

	return parseCodexResponse(resp), nil
}

func (p *CodexProvider) GetDefaultModel() string {
	return codexDefaultModel
}

func resolveCodexModel(model string) (string, string) {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return codexDefaultModel, "empty model"
	}

	if strings.HasPrefix(m, "openai/") {
		m = strings.TrimPrefix(m, "openai/")
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

func buildCodexParams(messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) responses.ResponseNewParams {
	var inputItems responses.ResponseInputParam
	var instructions string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			instructions = msg.Content
		case "user":
			if msg.ToolCallID != "" {
				inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
					OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
						CallID: msg.ToolCallID,
						Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{OfString: openai.Opt(msg.Content)},
					},
				})
			} else {
				inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Role:    responses.EasyInputMessageRoleUser,
						Content: responses.EasyInputMessageContentUnionParam{OfString: openai.Opt(msg.Content)},
					},
				})
			}
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				if msg.Content != "" {
					inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
						OfMessage: &responses.EasyInputMessageParam{
							Role:    responses.EasyInputMessageRoleAssistant,
							Content: responses.EasyInputMessageContentUnionParam{OfString: openai.Opt(msg.Content)},
						},
					})
				}
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Arguments)
					inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
						OfFunctionCall: &responses.ResponseFunctionToolCallParam{
							CallID:    tc.ID,
							Name:      tc.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			} else {
				inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Role:    responses.EasyInputMessageRoleAssistant,
						Content: responses.EasyInputMessageContentUnionParam{OfString: openai.Opt(msg.Content)},
					},
				})
			}
		case "tool":
			inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
				OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
					CallID: msg.ToolCallID,
					Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{OfString: openai.Opt(msg.Content)},
				},
			})
		}
	}

	params := responses.ResponseNewParams{
		Model: model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
		Instructions: openai.Opt(instructions),
		Store:        openai.Opt(false),
	}

	if instructions != "" {
		params.Instructions = openai.Opt(instructions)
	} else {
		// ChatGPT Codex backend requires instructions to be present.
		params.Instructions = openai.Opt(defaultCodexInstructions)
	}

	if maxTokens, ok := options["max_tokens"].(int); ok {
		params.MaxOutputTokens = openai.Opt(int64(maxTokens))
	}

	if len(tools) > 0 {
		params.Tools = translateToolsForCodex(tools)
	}

	return params
}

func translateToolsForCodex(tools []ToolDefinition) []responses.ToolUnionParam {
	result := make([]responses.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		ft := responses.FunctionToolParam{
			Name:       t.Function.Name,
			Parameters: t.Function.Parameters,
			Strict:     openai.Opt(false),
		}
		if t.Function.Description != "" {
			ft.Description = openai.Opt(t.Function.Description)
		}
		result = append(result, responses.ToolUnionParam{OfFunction: &ft})
	}
	return result
}

func parseCodexResponse(resp *responses.Response) *LLMResponse {
	var content strings.Builder
	var toolCalls []ToolCall

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, c := range item.Content {
				if c.Type == "output_text" {
					content.WriteString(c.Text)
				}
			}
		case "function_call":
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(item.Arguments), &args); err != nil {
				args = map[string]interface{}{"raw": item.Arguments}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        item.CallID,
				Name:      item.Name,
				Arguments: args,
			})
		}
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	if resp.Status == "incomplete" {
		finishReason = "length"
	}

	var usage *UsageInfo
	if resp.Usage.TotalTokens > 0 {
		usage = &UsageInfo{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		}
	}

	return &LLMResponse{
		Content:      content.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}
}

func createCodexTokenSource() func() (string, string, error) {
	return func() (string, string, error) {
		cred, err := auth.GetCredential("openai")
		if err != nil {
			return "", "", fmt.Errorf("loading auth credentials: %w", err)
		}
		if cred == nil {
			return "", "", fmt.Errorf("no credentials for openai. Run: picoclaw auth login --provider openai")
		}

		if cred.AuthMethod == "oauth" && cred.NeedsRefresh() && cred.RefreshToken != "" {
			oauthCfg := auth.OpenAIOAuthConfig()
			refreshed, err := auth.RefreshAccessToken(cred, oauthCfg)
			if err != nil {
				return "", "", fmt.Errorf("refreshing token: %w", err)
			}
			if refreshed.AccountID == "" {
				refreshed.AccountID = cred.AccountID
			}
			if err := auth.SetCredential("openai", refreshed); err != nil {
				return "", "", fmt.Errorf("saving refreshed token: %w", err)
			}
			return refreshed.AccessToken, refreshed.AccountID, nil
		}

		return cred.AccessToken, cred.AccountID, nil
	}
}
