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
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

type HTTPProvider struct {
	apiKey     string
	apiBase    string
	httpClient *http.Client
}

func NewHTTPProvider(apiKey, apiBase, proxy string) *HTTPProvider {
	client := &http.Client{
		Timeout: 120 * time.Second,
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
		apiBase:    strings.TrimRight(apiBase, "/"),
		httpClient: client,
	}
}

func (p *HTTPProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
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

	// Determine the endpoint and request format
	// Mistral /v1/conversations uses "inputs" and "completion_args"
	useConversations := strings.Contains(p.apiBase, "/conversations")

	var requestBody map[string]interface{}
	if useConversations {
		// Mistral conversations API: filter out system messages and put them in instructions
		// Also convert tool messages to user messages
		var filteredInputs []Message
		var systemContent string

		for _, msg := range messages {
			if msg.Role == "system" {
				// Collect system content
				if systemContent != "" {
					systemContent += "\n\n"
				}
				systemContent += msg.Content
			} else if msg.Role == "tool" {
				// Convert tool results to user messages for Mistral
				// Format: "The result of tool_name is: <result>"
				toolName := ""
				if len(msg.ToolCalls) > 0 {
					toolName = msg.ToolCalls[0].Name
				}
				if toolName != "" {
					filteredInputs = append(filteredInputs, Message{
						Role:    "user",
						Content: "The result of " + toolName + " is: " + msg.Content,
					})
				} else {
					filteredInputs = append(filteredInputs, Message{
						Role:    "user",
						Content: msg.Content,
					})
				}
			} else {
				// Only keep user and assistant messages
				filteredInputs = append(filteredInputs, msg)
			}
		}

		// Convert filteredInputs to []map[string]interface{} for JSON
		inputsForJSON := make([]map[string]interface{}, len(filteredInputs))
		for i, msg := range filteredInputs {
			inputsForJSON[i] = map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			}
		}

		// Mistral conversations API format
		requestBody = map[string]interface{}{
			"model":  model,
			"inputs": inputsForJSON,
		}

		// Add instructions from system message
		if systemContent != "" {
			requestBody["instructions"] = systemContent
		}

		// Add completion_args for Mistral conversations API
		completionArgs := map[string]interface{}{}
		if maxTokens, ok := options["max_tokens"].(int); ok {
			completionArgs["max_tokens"] = maxTokens
		}
		if temperature, ok := options["temperature"].(float64); ok {
			completionArgs["temperature"] = temperature
		}
		if topP, ok := options["top_p"].(float64); ok {
			completionArgs["top_p"] = topP
		}
		if len(completionArgs) > 0 {
			requestBody["completion_args"] = completionArgs
		}
	} else {
		// Standard OpenAI-compatible format
		requestBody = map[string]interface{}{
			"model":    model,
			"messages": messages,
		}
	}

	// Add tools for Mistral conversations API
	// For Mistral, use built-in web_search instead of custom tools
	if useConversations && len(tools) > 0 {
		// Convert tools to Mistral format
		mistralTools := convertToolsForMistral(tools)
		if len(mistralTools) > 0 {
			requestBody["tools"] = mistralTools
		}
	}

	// Add tools for non-conversations API only
	if len(tools) > 0 && !useConversations {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	// Only add top-level params for non-conversations API
	if !useConversations {
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
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Determine the endpoint - use /chat/completions unless already using /conversations
	endpoint := "/chat/completions"
	if strings.Contains(p.apiBase, "/conversations") {
		endpoint = ""
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+endpoint, bytes.NewReader(jsonData))
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
		return nil, fmt.Errorf("API request failed:\n  URL: %s\n  Status: %d\n  Body:   %s", p.apiBase+endpoint, resp.StatusCode, string(body))
	}

	return p.parseResponse(body)
}

func (p *HTTPProvider) parseResponse(body []byte) (*LLMResponse, error) {
	// First, try to parse as conversations API response format
	// The /v1/conversations API returns {"outputs": [{"type": "message.output", "content": [...]}]}
	// When tools are used, there can be multiple outputs (tool.execution + message.output)
	var convResponse struct {
		Outputs []struct {
			Type      string      `json:"type"`
			Content   interface{} `json:"content"`   // Can be string, array of {type, text}, or nil
			Name      string      `json:"name"`      // For tool.execution
			Arguments string      `json:"arguments"` // For tool.execution
		} `json:"outputs"`
		Usage *UsageInfo `json:"usage"`
	}

	if err := json.Unmarshal(body, &convResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if it's a conversations API response
	if len(convResponse.Outputs) > 0 {
		// Parse as conversations API format
		toolCalls := []ToolCall{}
		content := ""

		for _, output := range convResponse.Outputs {
			if output.Type == "tool.execution" {
				// This is a tool call
				args := make(map[string]interface{})
				if output.Arguments != "" {
					json.Unmarshal([]byte(output.Arguments), &args)
				}
				toolCalls = append(toolCalls, ToolCall{
					ID:        output.Name, // Use tool name as ID
					Name:      output.Name,
					Arguments: args,
				})
			} else if output.Type == "message.output" || output.Type == "message" {
				// This is the actual message content
				switch c := output.Content.(type) {
				case string:
					content = c
				case []interface{}:
					// Array of {type, text} objects
					for _, item := range c {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if text, ok := itemMap["text"].(string); ok {
								content += text
							}
						}
					}
				}
			}
		}

		return &LLMResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			FinishReason: "stop",
			Usage:        convResponse.Usage,
		}, nil
	}

	// Fallback: try standard OpenAI format
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
			ID:        tc.ID,
			Name:      name,
			Arguments: arguments,
		})
	}

	return &LLMResponse{
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: choice.FinishReason,
		Usage:        apiResponse.Usage,
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

	lowerModel := strings.ToLower(model)

	// First, try to use explicitly configured provider
	if providerName != "" {
		switch providerName {
		case "groq":
			if cfg.Providers.Groq.APIKey != "" {
				apiKey = cfg.Providers.Groq.APIKey
				apiBase = cfg.Providers.Groq.APIBase
				if apiBase == "" {
					apiBase = "https://api.groq.com/openai/v1"
				}
			}
		case "openai", "gpt":
			if cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != "" {
				if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
					return createCodexAuthProvider()
				}
				apiKey = cfg.Providers.OpenAI.APIKey
				apiBase = cfg.Providers.OpenAI.APIBase
				if apiBase == "" {
					apiBase = "https://api.openai.com/v1"
				}
			}
		case "anthropic", "claude":
			if cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != "" {
				if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
					return createClaudeAuthProvider()
				}
				apiKey = cfg.Providers.Anthropic.APIKey
				apiBase = cfg.Providers.Anthropic.APIBase
				if apiBase == "" {
					apiBase = "https://api.anthropic.com/v1"
				}
			}
		case "openrouter":
			if cfg.Providers.OpenRouter.APIKey != "" {
				apiKey = cfg.Providers.OpenRouter.APIKey
				if cfg.Providers.OpenRouter.APIBase != "" {
					apiBase = cfg.Providers.OpenRouter.APIBase
				} else {
					apiBase = "https://openrouter.ai/api/v1"
				}
			}
		case "zhipu", "glm":
			if cfg.Providers.Zhipu.APIKey != "" {
				apiKey = cfg.Providers.Zhipu.APIKey
				apiBase = cfg.Providers.Zhipu.APIBase
				if apiBase == "" {
					apiBase = "https://open.bigmodel.cn/api/paas/v4"
				}
			}
		case "gemini", "google":
			if cfg.Providers.Gemini.APIKey != "" {
				apiKey = cfg.Providers.Gemini.APIKey
				apiBase = cfg.Providers.Gemini.APIBase
				if apiBase == "" {
					apiBase = "https://generativelanguage.googleapis.com/v1beta"
				}
			}
		case "vllm":
			if cfg.Providers.VLLM.APIBase != "" {
				apiKey = cfg.Providers.VLLM.APIKey
				apiBase = cfg.Providers.VLLM.APIBase
			}
		case "shengsuanyun":
			if cfg.Providers.ShengSuanYun.APIKey != "" {
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
		case "deepseek":
			if cfg.Providers.DeepSeek.APIKey != "" {
				apiKey = cfg.Providers.DeepSeek.APIKey
				apiBase = cfg.Providers.DeepSeek.APIBase
				if apiBase == "" {
					apiBase = "https://api.deepseek.com/v1"
				}
				if model != "deepseek-chat" && model != "deepseek-reasoner" {
					model = "deepseek-chat"
				}
			}
		case "github_copilot", "copilot":
			if cfg.Providers.GitHubCopilot.APIBase != "" {
				apiBase = cfg.Providers.GitHubCopilot.APIBase
			} else {
				apiBase = "localhost:4321"
			}
			return NewGitHubCopilotProvider(apiBase, cfg.Providers.GitHubCopilot.ConnectMode, model)
		case "mistral":
			if cfg.Providers.Mistral.APIKey != "" {
				apiKey = cfg.Providers.Mistral.APIKey
				apiBase = cfg.Providers.Mistral.APIBase
				proxy = cfg.Providers.Mistral.Proxy
				// Mistral /v1/conversations endpoint for better rate limits
				if apiBase == "" {
					apiBase = "https://api.mistral.ai/v1/conversations"
				}
			}

		}

	}

	// Fallback: detect provider from model name
	if apiKey == "" && apiBase == "" {
		switch {
		case (strings.Contains(lowerModel, "kimi") || strings.Contains(lowerModel, "moonshot") || strings.HasPrefix(model, "moonshot/")) && cfg.Providers.Moonshot.APIKey != "":
			apiKey = cfg.Providers.Moonshot.APIKey
			apiBase = cfg.Providers.Moonshot.APIBase
			proxy = cfg.Providers.Moonshot.Proxy
			if apiBase == "" {
				apiBase = "https://api.moonshot.cn/v1"
			}

		case strings.HasPrefix(model, "openrouter/") || strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "meta-llama/") || strings.HasPrefix(model, "deepseek/") || strings.HasPrefix(model, "google/") || strings.HasPrefix(model, "mistral/"):
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
			apiKey = cfg.Providers.OpenAI.APIKey
			apiBase = cfg.Providers.OpenAI.APIBase
			proxy = cfg.Providers.OpenAI.Proxy
			if apiBase == "" {
				apiBase = "https://api.openai.com/v1"
			}

		case (strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/")) && cfg.Providers.Gemini.APIKey != "":
			apiKey = cfg.Providers.Gemini.APIKey
			apiBase = cfg.Providers.Gemini.APIBase
			proxy = cfg.Providers.Gemini.Proxy
			if apiBase == "" {
				apiBase = "https://generativelanguage.googleapis.com/v1beta"
			}

		case (strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai")) && cfg.Providers.Zhipu.APIKey != "":
			apiKey = cfg.Providers.Zhipu.APIKey
			apiBase = cfg.Providers.Zhipu.APIBase
			proxy = cfg.Providers.Zhipu.Proxy
			if apiBase == "" {
				apiBase = "https://open.bigmodel.cn/api/paas/v4"
			}

		case (strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/")) && cfg.Providers.Groq.APIKey != "":
			apiKey = cfg.Providers.Groq.APIKey
			apiBase = cfg.Providers.Groq.APIBase
			proxy = cfg.Providers.Groq.Proxy
			if apiBase == "" {
				apiBase = "https://api.groq.com/openai/v1"
			}

		case (strings.Contains(lowerModel, "nvidia") || strings.HasPrefix(model, "nvidia/")) && cfg.Providers.Nvidia.APIKey != "":
			apiKey = cfg.Providers.Nvidia.APIKey
			apiBase = cfg.Providers.Nvidia.APIBase
			proxy = cfg.Providers.Nvidia.Proxy
			if apiBase == "" {
				apiBase = "https://integrate.api.nvidia.com/v1"
			}

		case (strings.Contains(lowerModel, "mistral") || strings.HasPrefix(model, "mistral/")) && cfg.Providers.Mistral.APIKey != "":
			apiKey = cfg.Providers.Mistral.APIKey
			apiBase = cfg.Providers.Mistral.APIBase
			proxy = cfg.Providers.Mistral.Proxy
			if apiBase == "" {
				apiBase = "https://api.mistral.ai/v1"
			}

		case cfg.Providers.VLLM.APIBase != "":
			apiKey = cfg.Providers.VLLM.APIKey
			apiBase = cfg.Providers.VLLM.APIBase
			proxy = cfg.Providers.VLLM.Proxy

		default:
			if cfg.Providers.OpenRouter.APIKey != "" {
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

	if apiKey == "" && !strings.HasPrefix(model, "bedrock/") {
		return nil, fmt.Errorf("no API key configured for provider (model: %s)", model)
	}

	if apiBase == "" {
		return nil, fmt.Errorf("no API base configured for provider (model: %s)", model)
	}

	return NewHTTPProvider(apiKey, apiBase, proxy), nil
}

// convertToolsForMistral converts PicoClaw tools to Mistral format
// For Mistral conversations API, we use built-in web_search instead of custom tools
func convertToolsForMistral(tools []ToolDefinition) []map[string]interface{} {
	mistralTools := []map[string]interface{}{}

	for _, tool := range tools {
		if tool.Type == "function" {
			// Check if this is a web_search tool - use Mistral's built-in instead
			if tool.Function.Name == "web_search" || tool.Function.Name == "search" {
				// Use Mistral's built-in web_search
				mistralTools = append(mistralTools, map[string]interface{}{
					"type": "web_search",
				})
			} else {
				// Keep custom function in Mistral format
				mistralTools = append(mistralTools, map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        tool.Function.Name,
						"description": tool.Function.Description,
						"parameters":  tool.Function.Parameters,
					},
				})
			}
		}
	}

	return mistralTools
}
