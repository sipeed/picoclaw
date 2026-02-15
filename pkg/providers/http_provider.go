// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type HTTPProvider struct {
	apiKey      string
	apiBase     string
	httpClient  *http.Client
	keyRotator  *KeyRotator
}

func NewHTTPProvider(apiKey, apiBase, proxy string) *HTTPProvider {
	client := &http.Client{
		Timeout: 0,
	}

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	return &HTTPProvider{
		apiKey:     apiKey,
		apiBase:    apiBase,
		httpClient: client,
	}
}

func NewHTTPProviderWithKeys(keys []string, apiBase, proxy string) *HTTPProvider {
	p := NewHTTPProvider(keys[0], apiBase, proxy)
	p.keyRotator = NewKeyRotator(keys)
	return p
}

func (p *HTTPProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	ctx, span := tracing.Tracer("provider").Start(ctx, "provider.chat",
		trace.WithAttributes(
			attribute.String("model", model),
			attribute.Int("messages_count", len(messages)),
		))
	defer span.End()

	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	// Strip provider prefix from model name (e.g., moonshot/kimi-k2.5 -> kimi-k2.5)
	if idx := strings.Index(model, "/"); idx != -1 {
		prefix := model[:idx]
		if prefix == "moonshot" || prefix == "nvidia" {
			model = model[idx+1:]
		}
	}

	// Build messages for request, handling multipart content
	reqMessages := make([]interface{}, 0, len(messages))
	for _, msg := range messages {
		// If we have the raw API message (preserves thought_signature etc.), use it directly
		if len(msg.RawAPIMessage) > 0 {
			var rawMap map[string]interface{}
			if err := json.Unmarshal(msg.RawAPIMessage, &rawMap); err == nil {
				reqMessages = append(reqMessages, rawMap)
				continue
			}
		}

		m := map[string]interface{}{
			"role": msg.Role,
		}
		if len(msg.ContentParts) > 0 {
			// Multipart content (text + images)
			parts := make([]map[string]interface{}, 0, len(msg.ContentParts))
			for _, part := range msg.ContentParts {
				p := map[string]interface{}{"type": part.Type}
				if part.Type == "text" {
					p["text"] = part.Text
				} else if part.Type == "image_url" && part.ImageURL != nil {
					p["image_url"] = map[string]string{"url": part.ImageURL.URL}
				}
				parts = append(parts, p)
			}
			m["content"] = parts
		} else {
			m["content"] = msg.Content
		}
		if len(msg.ToolCalls) > 0 {
			m["tool_calls"] = msg.ToolCalls
		}
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		reqMessages = append(reqMessages, m)
	}

	requestBody := map[string]interface{}{
		"model":    model,
		"messages": reqMessages,
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := options["max_tokens"].(int); ok {
		lowerModel := strings.ToLower(model)
		if strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "o1") {
			requestBody["max_completion_tokens"] = maxTokens
		} else {
			requestBody["max_tokens"] = maxTokens
		}
	}

	if temperature, ok := options["temperature"].(float64); ok {
		lowerModel := strings.ToLower(model)
		// Kimi k2 models only support temperature=1
		if strings.Contains(lowerModel, "kimi") && strings.Contains(lowerModel, "k2") {
			requestBody["temperature"] = 1.0
		} else {
			requestBody["temperature"] = temperature
		}
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Retry loop for 429/503 with key rotation and exponential backoff
	maxRounds := 3
	keysPerRound := 1
	if p.keyRotator != nil {
		keysPerRound = p.keyRotator.Len()
	}
	maxAttempts := keysPerRound * maxRounds

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		apiKey := p.apiKey
		if p.keyRotator != nil {
			apiKey = p.keyRotator.Next()
		}
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			return p.parseResponse(body)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			lastErr = fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))

			// Backoff after exhausting all keys in a round
			if (attempt+1)%keysPerRound == 0 {
				round := (attempt + 1) / keysPerRound
				backoff := time.Duration(math.Min(float64(int(1)<<round), 30)) * time.Second
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
			}
			continue
		}

		// Non-retryable error
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	return nil, fmt.Errorf("all API keys exhausted after %d attempts: %w", maxAttempts, lastErr)
}

func (p *HTTPProvider) parseResponse(body []byte) (*LLMResponse, error) {
	// First pass: capture raw message JSON to preserve API-specific fields
	// (e.g. thought_signature for Gemini thinking models)
	var rawResponse struct {
		Choices []struct {
			Message      json.RawMessage `json:"message"`
			FinishReason string          `json:"finish_reason"`
		} `json:"choices"`
	}
	json.Unmarshal(body, &rawResponse)

	var rawMsg json.RawMessage
	if len(rawResponse.Choices) > 0 {
		rawMsg = rawResponse.Choices[0].Message
	}

	// Second pass: structured parse
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID           string                 `json:"id"`
					Type         string                 `json:"type"`
					Function     *struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
					ExtraContent map[string]interface{} `json:"extra_content,omitempty"`
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
		arguments := make(map[string]interface{})
		name := ""

		// Handle OpenAI format with nested function object
		if tc.Type == "function" && tc.Function != nil {
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					arguments["raw"] = tc.Function.Arguments
				}
			}
		} else if tc.Function != nil {
			// Legacy format without type field
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					arguments["raw"] = tc.Function.Arguments
				}
			}
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:           tc.ID,
			Name:         name,
			Arguments:    arguments,
			ExtraContent: tc.ExtraContent,
		})
	}

	return &LLMResponse{
		Content:             choice.Message.Content,
		ToolCalls:           toolCalls,
		FinishReason:        choice.FinishReason,
		Usage:               apiResponse.Usage,
		RawAssistantMessage: rawMsg,
	}, nil
}

func (p *HTTPProvider) GetDefaultModel() string {
	return ""
}

func createClaudeAuthProvider() (LLMProvider, error) {
	cred, err := auth.GetCredential("anthropic")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for anthropic. Run: picoclaw auth login --provider anthropic")
	}
	return NewClaudeProviderWithTokenSource(cred.AccessToken, createClaudeTokenSource()), nil
}

func createCodexAuthProvider() (LLMProvider, error) {
	cred, err := auth.GetCredential("openai")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for openai. Run: picoclaw auth login --provider openai")
	}
	return NewCodexProviderWithTokenSource(cred.AccessToken, cred.AccountID, createCodexTokenSource()), nil
}

func CreateProvider(cfg *config.Config) (LLMProvider, error) {
	model := cfg.Agents.Defaults.Model
	providerName := strings.ToLower(cfg.Agents.Defaults.Provider)

	var apiKey, apiBase, proxy string
	var providerCfg *config.ProviderConfig

	lowerModel := strings.ToLower(model)

	// First, try to use explicitly configured provider
	if providerName != "" {
		switch providerName {
		case "groq":
			if cfg.Providers.Groq.APIKey != "" || len(cfg.Providers.Groq.APIKeys) > 0 {
				providerCfg = &cfg.Providers.Groq
				apiKey = cfg.Providers.Groq.APIKey
				apiBase = cfg.Providers.Groq.APIBase
				if apiBase == "" {
					apiBase = "https://api.groq.com/openai/v1"
				}
			}
		case "openai", "gpt":
			if cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != "" || len(cfg.Providers.OpenAI.APIKeys) > 0 {
				if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
					return createCodexAuthProvider()
				}
				providerCfg = &cfg.Providers.OpenAI
				apiKey = cfg.Providers.OpenAI.APIKey
				apiBase = cfg.Providers.OpenAI.APIBase
				if apiBase == "" {
					apiBase = "https://api.openai.com/v1"
				}
			}
		case "anthropic", "claude":
			if cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != "" || len(cfg.Providers.Anthropic.APIKeys) > 0 {
				if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
					return createClaudeAuthProvider()
				}
				providerCfg = &cfg.Providers.Anthropic
				apiKey = cfg.Providers.Anthropic.APIKey
				apiBase = cfg.Providers.Anthropic.APIBase
				if apiBase == "" {
					apiBase = "https://api.anthropic.com/v1"
				}
			}
		case "openrouter":
			if cfg.Providers.OpenRouter.APIKey != "" || len(cfg.Providers.OpenRouter.APIKeys) > 0 {
				providerCfg = &cfg.Providers.OpenRouter
				apiKey = cfg.Providers.OpenRouter.APIKey
				if cfg.Providers.OpenRouter.APIBase != "" {
					apiBase = cfg.Providers.OpenRouter.APIBase
				} else {
					apiBase = "https://openrouter.ai/api/v1"
				}
			}
		case "zhipu", "glm":
			if cfg.Providers.Zhipu.APIKey != "" || len(cfg.Providers.Zhipu.APIKeys) > 0 {
				providerCfg = &cfg.Providers.Zhipu
				apiKey = cfg.Providers.Zhipu.APIKey
				apiBase = cfg.Providers.Zhipu.APIBase
				if apiBase == "" {
					apiBase = "https://open.bigmodel.cn/api/paas/v4"
				}
			}
		case "gemini", "google":
			if cfg.Providers.Gemini.APIKey != "" || len(cfg.Providers.Gemini.APIKeys) > 0 {
				providerCfg = &cfg.Providers.Gemini
				apiKey = cfg.Providers.Gemini.APIKey
				apiBase = cfg.Providers.Gemini.APIBase
				if apiBase == "" {
					apiBase = "https://generativelanguage.googleapis.com/v1beta/openai"
				}
			}
		case "vllm":
			if cfg.Providers.VLLM.APIBase != "" {
				providerCfg = &cfg.Providers.VLLM
				apiKey = cfg.Providers.VLLM.APIKey
				apiBase = cfg.Providers.VLLM.APIBase
			}
		case "shengsuanyun":
			if cfg.Providers.ShengSuanYun.APIKey != "" || len(cfg.Providers.ShengSuanYun.APIKeys) > 0 {
				providerCfg = &cfg.Providers.ShengSuanYun
				apiKey = cfg.Providers.ShengSuanYun.APIKey
				apiBase = cfg.Providers.ShengSuanYun.APIBase
				if apiBase == "" {
					apiBase = "https://router.shengsuanyun.com/api/v1"
				}
			}
		case "claude-cli", "claudecode", "claude-code":
			workspace := cfg.Agents.Defaults.Workspace
			if workspace == "" {
				workspace = "."
			}
			return NewClaudeCliProvider(workspace), nil
		}
	}

	// Fallback: detect provider from model name
	if apiKey == "" && apiBase == "" && providerCfg == nil {
		switch {
		case (strings.Contains(lowerModel, "kimi") || strings.Contains(lowerModel, "moonshot") || strings.HasPrefix(model, "moonshot/")) && cfg.Providers.Moonshot.APIKey != "":
			providerCfg = &cfg.Providers.Moonshot
			apiKey = cfg.Providers.Moonshot.APIKey
			apiBase = cfg.Providers.Moonshot.APIBase
			proxy = cfg.Providers.Moonshot.Proxy
			if apiBase == "" {
				apiBase = "https://api.moonshot.cn/v1"
			}

		case strings.HasPrefix(model, "openrouter/") || strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "meta-llama/") || strings.HasPrefix(model, "deepseek/") || strings.HasPrefix(model, "google/"):
			providerCfg = &cfg.Providers.OpenRouter
			apiKey = cfg.Providers.OpenRouter.APIKey
			proxy = cfg.Providers.OpenRouter.Proxy
			if cfg.Providers.OpenRouter.APIBase != "" {
				apiBase = cfg.Providers.OpenRouter.APIBase
			} else {
				apiBase = "https://openrouter.ai/api/v1"
			}

		case (strings.Contains(lowerModel, "claude") || strings.HasPrefix(model, "anthropic/")) && (cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != ""):
			if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
				return createClaudeAuthProvider()
			}
			providerCfg = &cfg.Providers.Anthropic
			apiKey = cfg.Providers.Anthropic.APIKey
			apiBase = cfg.Providers.Anthropic.APIBase
			proxy = cfg.Providers.Anthropic.Proxy
			if apiBase == "" {
				apiBase = "https://api.anthropic.com/v1"
			}

		case (strings.Contains(lowerModel, "gpt") || strings.HasPrefix(model, "openai/")) && (cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != ""):
			if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
				return createCodexAuthProvider()
			}
			providerCfg = &cfg.Providers.OpenAI
			apiKey = cfg.Providers.OpenAI.APIKey
			apiBase = cfg.Providers.OpenAI.APIBase
			proxy = cfg.Providers.OpenAI.Proxy
			if apiBase == "" {
				apiBase = "https://api.openai.com/v1"
			}

		case (strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/")) && cfg.Providers.Gemini.APIKey != "":
			providerCfg = &cfg.Providers.Gemini
			apiKey = cfg.Providers.Gemini.APIKey
			apiBase = cfg.Providers.Gemini.APIBase
			proxy = cfg.Providers.Gemini.Proxy
			if apiBase == "" {
				apiBase = "https://generativelanguage.googleapis.com/v1beta/openai"
			}

		case (strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai")) && cfg.Providers.Zhipu.APIKey != "":
			providerCfg = &cfg.Providers.Zhipu
			apiKey = cfg.Providers.Zhipu.APIKey
			apiBase = cfg.Providers.Zhipu.APIBase
			proxy = cfg.Providers.Zhipu.Proxy
			if apiBase == "" {
				apiBase = "https://open.bigmodel.cn/api/paas/v4"
			}

		case (strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/")) && cfg.Providers.Groq.APIKey != "":
			providerCfg = &cfg.Providers.Groq
			apiKey = cfg.Providers.Groq.APIKey
			apiBase = cfg.Providers.Groq.APIBase
			proxy = cfg.Providers.Groq.Proxy
			if apiBase == "" {
				apiBase = "https://api.groq.com/openai/v1"
			}

		case (strings.Contains(lowerModel, "nvidia") || strings.HasPrefix(model, "nvidia/")) && cfg.Providers.Nvidia.APIKey != "":
			providerCfg = &cfg.Providers.Nvidia
			apiKey = cfg.Providers.Nvidia.APIKey
			apiBase = cfg.Providers.Nvidia.APIBase
			proxy = cfg.Providers.Nvidia.Proxy
			if apiBase == "" {
				apiBase = "https://integrate.api.nvidia.com/v1"
			}

		case cfg.Providers.VLLM.APIBase != "":
			providerCfg = &cfg.Providers.VLLM
			apiKey = cfg.Providers.VLLM.APIKey
			apiBase = cfg.Providers.VLLM.APIBase
			proxy = cfg.Providers.VLLM.Proxy

		default:
			if cfg.Providers.OpenRouter.APIKey != "" {
				providerCfg = &cfg.Providers.OpenRouter
				apiKey = cfg.Providers.OpenRouter.APIKey
				proxy = cfg.Providers.OpenRouter.Proxy
				if cfg.Providers.OpenRouter.APIBase != "" {
					apiBase = cfg.Providers.OpenRouter.APIBase
				} else {
					apiBase = "https://openrouter.ai/api/v1"
				}
			} else {
				return nil, fmt.Errorf("no API key configured for model: %s", model)
			}
		}
	}

	// Use key rotation if api_keys is configured
	if providerCfg != nil && len(providerCfg.APIKeys) > 0 {
		return NewHTTPProviderWithKeys(providerCfg.APIKeys, apiBase, proxy), nil
	}

	if apiKey == "" && !strings.HasPrefix(model, "bedrock/") {
		return nil, fmt.Errorf("no API key configured for provider (model: %s)", model)
	}

	if apiBase == "" {
		return nil, fmt.Errorf("no API base configured for provider (model: %s)", model)
	}

	return NewHTTPProvider(apiKey, apiBase, proxy), nil
}
