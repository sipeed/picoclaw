package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

const (
	// qwenOAuthBaseURL is the Qwen Portal API endpoint (OpenAI-compatible).
	// This is the same endpoint used by OpenClaw for Qwen OAuth authentication.
	// Reference: https://github.com/openclaw/openclaw/tree/main/extensions/qwen-portal-auth
	// Available models: https://chat.qwen.ai
	//   - coder-model: Qwen Coder (code generation and understanding)
	//   - vision-model: Qwen Vision (image understanding)
	qwenOAuthBaseURL      = "https://portal.qwen.ai/v1"
	qwenOAuthDefaultModel = "coder-model"
)

// QwenOAuthProvider implements LLMProvider using Qwen Portal API
// authenticated via Qwen OAuth (QR-code scan). It uses the OpenAI-compatible
// /chat/completions endpoint so no extra SDK is required.
type QwenOAuthProvider struct {
	tokenSource func() (string, error)
	apiBase     string
	httpClient  *http.Client
}

// NewQwenOAuthProvider creates a QwenOAuthProvider that reads credentials from
// the auth store and refreshes them transparently.
func NewQwenOAuthProvider() *QwenOAuthProvider {
	return &QwenOAuthProvider{
		tokenSource: auth.CreateQwenTokenSource(),
		apiBase:     strings.TrimRight(qwenOAuthBaseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// NewQwenOAuthProviderWithTokenSource creates a QwenOAuthProvider with a
// custom token-source (useful for testing).
func NewQwenOAuthProviderWithTokenSource(
	tokenSource func() (string, error),
	apiBase string,
) *QwenOAuthProvider {
	base := qwenOAuthBaseURL
	if apiBase != "" {
		base = strings.TrimRight(apiBase, "/")
	}
	return &QwenOAuthProvider{
		tokenSource: tokenSource,
		apiBase:     base,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GetDefaultModel returns the default Qwen model.
func (p *QwenOAuthProvider) GetDefaultModel() string {
	return qwenOAuthDefaultModel
}

// Chat implements LLMProvider.Chat using the Qwen Portal OpenAI-compatible API.
func (p *QwenOAuthProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	token, err := p.tokenSource()
	if err != nil {
		return nil, fmt.Errorf("qwen oauth: %w", err)
	}

	if model == "" || model == "qwen-oauth" {
		model = qwenOAuthDefaultModel
	}
	// Strip protocol prefixes.
	model = strings.TrimPrefix(model, "qwen-oauth/")
	model = strings.TrimPrefix(model, "qwen/")

	logger.DebugCF("provider.qwen_oauth", "Starting chat", map[string]any{
		"model": model,
	})

	reqBody, err := p.buildRequestBody(messages, tools, model, options)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	apiURL := p.apiBase + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("qwen request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("qwen OAuth token rejected (%d) – run: picoclaw auth login --provider qwen",
			resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("qwen rate limit exceeded (429)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qwen API error (%d): %s", resp.StatusCode, string(body))
	}

	return p.parseResponse(body)
}

// buildRequestBody serialises the chat request as an OpenAI-compatible JSON body.
func (p *QwenOAuthProvider) buildRequestBody(
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) ([]byte, error) {
	body := map[string]any{
		"model":    model,
		"messages": convertMessagesForQwen(messages),
		"stream":   false,
	}

	// Forward supported options.
	if v, ok := options["temperature"]; ok {
		body["temperature"] = v
	}
	if v, ok := options["max_tokens"]; ok {
		body["max_tokens"] = v
	}
	if v, ok := options["top_p"]; ok {
		body["top_p"] = v
	}

	if len(tools) > 0 {
		body["tools"] = convertToolsForQwen(tools)
		body["tool_choice"] = "auto"
	}

	return json.Marshal(body)
}

// parseResponse parses an OpenAI-compatible chat/completions response.
func (p *QwenOAuthProvider) parseResponse(body []byte) (*LLMResponse, error) {
	var raw struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing qwen response: %w (body: %.300s)", err, string(body))
	}

	if raw.Error != nil && raw.Error.Message != "" {
		return nil, fmt.Errorf("qwen API error [%s]: %s", raw.Error.Code, raw.Error.Message)
	}

	if len(raw.Choices) == 0 {
		return nil, fmt.Errorf("qwen returned no choices (body: %.300s)", string(body))
	}

	choice := raw.Choices[0]
	llmResp := &LLMResponse{
		Content: choice.Message.Content,
		Usage: &protocoltypes.UsageInfo{
			PromptTokens:     raw.Usage.PromptTokens,
			CompletionTokens: raw.Usage.CompletionTokens,
			TotalTokens:      raw.Usage.TotalTokens,
		},
	}

	for _, tc := range choice.Message.ToolCalls {
		llmResp.ToolCalls = append(llmResp.ToolCalls, protocoltypes.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: &protocoltypes.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return llmResp, nil
}

// convertMessagesForQwen converts internal Message slice to OpenAI-compatible format.
func convertMessagesForQwen(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		entry := map[string]any{
			"role":    m.Role,
			"content": m.Content,
		}
		if m.ToolCallID != "" {
			entry["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			entry["tool_calls"] = tcs
		}
		out = append(out, entry)
	}
	return out
}

// convertToolsForQwen converts internal ToolDefinition slice to OpenAI-compatible format.
func convertToolsForQwen(tools []ToolDefinition) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Function.Name,
				"description": t.Function.Description,
				"parameters":  t.Function.Parameters,
			},
		})
	}
	return out
}

// createQwenOAuthProvider creates a QwenOAuthProvider using stored credentials.
func createQwenOAuthProvider() (LLMProvider, error) {
	cred, err := getCredential("qwen")
	if err != nil {
		return nil, fmt.Errorf("loading qwen credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for qwen. Run: picoclaw auth login --provider qwen")
	}
	// Always use the standard DashScope URL - it's defined in the provider constant
	return NewQwenOAuthProvider(), nil
}
