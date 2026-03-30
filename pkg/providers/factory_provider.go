// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	anthropicmessages "github.com/sipeed/picoclaw/pkg/providers/anthropic_messages"
	"github.com/sipeed/picoclaw/pkg/providers/azure"
	"github.com/sipeed/picoclaw/pkg/providers/bedrock"
)

type protocolMeta struct {
	defaultAPIBase     string
	emptyAPIKeyAllowed bool
}

var protocolMetaByName = map[string]protocolMeta{
	"openai":                   {defaultAPIBase: "https://api.openai.com/v1"},
	"openrouter":               {defaultAPIBase: "https://openrouter.ai/api/v1"},
	"litellm":                  {defaultAPIBase: "http://localhost:4000/v1"},
	"lmstudio":                 {defaultAPIBase: "http://localhost:1234/v1", emptyAPIKeyAllowed: true},
	"novita":                   {defaultAPIBase: "https://api.novita.ai/openai"},
	"groq":                     {defaultAPIBase: "https://api.groq.com/openai/v1"},
	"zhipu":                    {defaultAPIBase: "https://open.bigmodel.cn/api/paas/v4"},
	"gemini":                   {defaultAPIBase: "https://generativelanguage.googleapis.com/v1beta"},
	"nvidia":                   {defaultAPIBase: "https://integrate.api.nvidia.com/v1"},
	"ollama":                   {defaultAPIBase: "http://localhost:11434/v1", emptyAPIKeyAllowed: true},
	"moonshot":                 {defaultAPIBase: "https://api.moonshot.cn/v1"},
	"shengsuanyun":             {defaultAPIBase: "https://router.shengsuanyun.com/api/v1"},
	"deepseek":                 {defaultAPIBase: "https://api.deepseek.com/v1"},
	"cerebras":                 {defaultAPIBase: "https://api.cerebras.ai/v1"},
	"vivgrid":                  {defaultAPIBase: "https://api.vivgrid.com/v1"},
	"volcengine":               {defaultAPIBase: "https://ark.cn-beijing.volces.com/api/v3"},
	"qwen":                     {defaultAPIBase: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
	"qwen-intl":                {defaultAPIBase: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
	"qwen-international":       {defaultAPIBase: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
	"dashscope-intl":           {defaultAPIBase: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
	"qwen-us":                  {defaultAPIBase: "https://dashscope-us.aliyuncs.com/compatible-mode/v1"},
	"dashscope-us":             {defaultAPIBase: "https://dashscope-us.aliyuncs.com/compatible-mode/v1"},
	"coding-plan":              {defaultAPIBase: "https://coding-intl.dashscope.aliyuncs.com/v1"},
	"alibaba-coding":           {defaultAPIBase: "https://coding-intl.dashscope.aliyuncs.com/v1"},
	"qwen-coding":              {defaultAPIBase: "https://coding-intl.dashscope.aliyuncs.com/v1"},
	"coding-plan-anthropic":    {defaultAPIBase: "https://coding-intl.dashscope.aliyuncs.com/apps/anthropic"},
	"alibaba-coding-anthropic": {defaultAPIBase: "https://coding-intl.dashscope.aliyuncs.com/apps/anthropic"},
	"vllm":                     {defaultAPIBase: "http://localhost:8000/v1", emptyAPIKeyAllowed: true},
	"mistral":                  {defaultAPIBase: "https://api.mistral.ai/v1"},
	"avian":                    {defaultAPIBase: "https://api.avian.io/v1"},
	"minimax":                  {defaultAPIBase: "https://api.minimaxi.com/v1"},
	"longcat":                  {defaultAPIBase: "https://api.longcat.chat/openai"},
	"modelscope":               {defaultAPIBase: "https://api-inference.modelscope.cn/v1"},
	"mimo":                     {defaultAPIBase: "https://api.xiaomimimo.com/v1"},
}

// createClaudeAuthProvider creates a Claude provider using OAuth credentials from auth store.
func createClaudeAuthProvider() (LLMProvider, error) {
	cred, err := getCredential("anthropic")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for anthropic. Run: picoclaw auth login --provider anthropic")
	}
	return NewClaudeProviderWithTokenSource(cred.AccessToken, createClaudeTokenSource()), nil
}

// createCodexAuthProvider creates a Codex provider using OAuth credentials from auth store.
func createCodexAuthProvider() (LLMProvider, error) {
	cred, err := getCredential("openai")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for openai. Run: picoclaw auth login --provider openai")
	}
	return NewCodexProviderWithTokenSource(cred.AccessToken, cred.AccountID, createCodexTokenSource()), nil
}

// ExtractProtocol extracts the protocol prefix and model identifier from a model string.
// If no prefix is specified, it defaults to "openai".
// Examples:
//   - "openai/gpt-4o" -> ("openai", "gpt-4o")
//   - "anthropic/claude-sonnet-4.6" -> ("anthropic", "claude-sonnet-4.6")
//   - "gpt-4o" -> ("openai", "gpt-4o")  // default protocol
func ExtractProtocol(model string) (protocol, modelID string) {
	model = strings.TrimSpace(model)
	protocol, modelID, found := strings.Cut(model, "/")
	if !found {
		return "openai", model
	}
	return protocol, modelID
}

// CreateProviderFromConfig creates a provider based on the ModelConfig.
// It uses the protocol prefix in the Model field to determine which provider to create.
// Supported protocol families include OpenAI-compatible prefixes (e.g., openai, openrouter, groq, gemini),
// Azure OpenAI, Amazon Bedrock, Anthropic (including messages), and various CLI/compatibility shims.
// See the switch on protocol in this function for the authoritative list.
// Returns the provider, the model ID (without protocol prefix), and any error.
func CreateProviderFromConfig(cfg *config.ModelConfig) (LLMProvider, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("config is nil")
	}

	if cfg.Model == "" {
		return nil, "", fmt.Errorf("model is required")
	}

	protocol, modelID := ExtractProtocol(cfg.Model)

	switch protocol {
	case "openai":
		// OpenAI with OAuth/token auth (Codex-style)
		if cfg.AuthMethod == "oauth" || cfg.AuthMethod == "token" {
			provider, err := createCodexAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		// OpenAI with API key
		if cfg.APIKey() == "" && cfg.APIBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = getDefaultAPIBase(protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey(),
			apiBase,
			cfg.Proxy,
			cfg.MaxTokensField,
			cfg.RequestTimeout,
			cfg.ExtraBody,
		), modelID, nil

	case "azure", "azure-openai":
		// Azure OpenAI uses deployment-based URLs, api-key header auth,
		// and always sends max_completion_tokens.
		if cfg.APIKey() == "" {
			return nil, "", fmt.Errorf("api_key is required for azure protocol")
		}
		if cfg.APIBase == "" {
			return nil, "", fmt.Errorf(
				"api_base is required for azure protocol (e.g., https://your-resource.openai.azure.com)",
			)
		}
		return azure.NewProviderWithTimeout(
			cfg.APIKey(),
			cfg.APIBase,
			cfg.Proxy,
			cfg.RequestTimeout,
		), modelID, nil

	case "bedrock":
		// AWS Bedrock uses AWS SDK credentials (env vars, profiles, IAM roles, etc.)
		// api_base can be:
		//   - A full endpoint URL: https://bedrock-runtime.us-east-1.amazonaws.com
		//   - A region name: us-east-1 (AWS SDK resolves endpoint automatically)
		var opts []bedrock.Option
		if cfg.APIBase != "" {
			if !strings.Contains(cfg.APIBase, "://") {
				// Treat as region: let AWS SDK resolve the correct endpoint
				// (supports all AWS partitions: aws, aws-cn, aws-us-gov, etc.)
				opts = append(opts, bedrock.WithRegion(cfg.APIBase))
			} else {
				// Full endpoint URL provided (for custom endpoints or testing)
				opts = append(opts, bedrock.WithBaseEndpoint(cfg.APIBase))
			}
		}
		// Use a separate timeout for AWS config loading (credential resolution can block)
		initTimeout := 30 * time.Second
		if cfg.RequestTimeout > 0 {
			reqTimeout := time.Duration(cfg.RequestTimeout) * time.Second
			// Set request timeout for API calls
			opts = append(opts, bedrock.WithRequestTimeout(reqTimeout))
			// Ensure init timeout is at least as large as request timeout
			if reqTimeout > initTimeout {
				initTimeout = reqTimeout
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), initTimeout)
		defer cancel()
		// Note: AWS_PROFILE env var is automatically used by AWS SDK
		provider, err := bedrock.NewProvider(ctx, opts...)
		if err != nil {
			return nil, "", fmt.Errorf("creating bedrock provider: %w", err)
		}
		return provider, modelID, nil

	case "litellm", "lmstudio", "openrouter", "groq", "zhipu", "gemini", "nvidia",
		"ollama", "moonshot", "shengsuanyun", "deepseek", "cerebras",
		"vivgrid", "volcengine", "vllm", "qwen", "qwen-intl", "qwen-international", "dashscope-intl",
		"qwen-us", "dashscope-us", "mistral", "avian", "longcat", "modelscope", "novita",
		"coding-plan", "alibaba-coding", "qwen-coding", "mimo":
		// All other OpenAI-compatible HTTP providers
		if cfg.APIKey() == "" && cfg.APIBase == "" && !isEmptyAPIKeyAllowed(protocol) {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = getDefaultAPIBase(protocol)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey(),
			apiBase,
			cfg.Proxy,
			cfg.MaxTokensField,
			cfg.RequestTimeout,
			cfg.ExtraBody,
		), modelID, nil

	case "minimax":
		// Minimax requires reasoning_split: true in the request body
		if cfg.APIKey() == "" && cfg.APIBase == "" {
			return nil, "", fmt.Errorf("api_key or api_base is required for HTTP-based protocol %q", protocol)
		}
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = getDefaultAPIBase(protocol)
		}
		extraBody := cfg.ExtraBody
		if extraBody == nil {
			extraBody = make(map[string]any)
		}
		if _, ok := extraBody["reasoning_split"]; !ok {
			extraBody["reasoning_split"] = true
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey(),
			apiBase,
			cfg.Proxy,
			cfg.MaxTokensField,
			cfg.RequestTimeout,
			extraBody,
		), modelID, nil

	case "anthropic":
		if cfg.AuthMethod == "oauth" || cfg.AuthMethod == "token" {
			// Use OAuth credentials from auth store
			provider, err := createClaudeAuthProvider()
			if err != nil {
				return nil, "", err
			}
			return provider, modelID, nil
		}
		// Use API key with HTTP API
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		if cfg.APIKey() == "" {
			return nil, "", fmt.Errorf("api_key is required for anthropic protocol (model: %s)", cfg.Model)
		}
		return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
			cfg.APIKey(),
			apiBase,
			cfg.Proxy,
			cfg.MaxTokensField,
			cfg.RequestTimeout,
			cfg.ExtraBody,
		), modelID, nil

	case "anthropic-messages":
		// Anthropic Messages API with native format (HTTP-based, no SDK)
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.anthropic.com/v1"
		}
		if cfg.APIKey() == "" {
			return nil, "", fmt.Errorf("api_key is required for anthropic-messages protocol (model: %s)", cfg.Model)
		}
		return anthropicmessages.NewProviderWithTimeout(
			cfg.APIKey(),
			apiBase,
			cfg.RequestTimeout,
		), modelID, nil

	case "coding-plan-anthropic", "alibaba-coding-anthropic":
		// Alibaba Coding Plan with Anthropic-compatible API
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = getDefaultAPIBase(protocol)
		}
		if cfg.APIKey() == "" {
			return nil, "", fmt.Errorf("api_key is required for %q protocol (model: %s)", protocol, cfg.Model)
		}
		return anthropicmessages.NewProviderWithTimeout(
			cfg.APIKey(),
			apiBase,
			cfg.RequestTimeout,
		), modelID, nil

	case "antigravity":
		return NewAntigravityProvider(), modelID, nil

	case "claude-cli", "claudecli":
		workspace := cfg.Workspace
		if workspace == "" {
			workspace = "."
		}
		return NewClaudeCliProvider(workspace), modelID, nil

	case "codex-cli", "codexcli":
		workspace := cfg.Workspace
		if workspace == "" {
			workspace = "."
		}
		return NewCodexCliProvider(workspace), modelID, nil

	case "github-copilot", "copilot":
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "localhost:4321"
		}
		connectMode := cfg.ConnectMode
		if connectMode == "" {
			connectMode = "grpc"
		}
		provider, err := NewGitHubCopilotProvider(apiBase, connectMode, modelID)
		if err != nil {
			return nil, "", err
		}
		return provider, modelID, nil

	default:
		return nil, "", fmt.Errorf("unknown protocol %q in model %q", protocol, cfg.Model)
	}
}

func isEmptyAPIKeyAllowed(protocol string) bool {
	meta, ok := protocolMetaByName[protocol]
	return ok && meta.emptyAPIKeyAllowed
}

// IsEmptyAPIKeyAllowedForProtocol reports whether a protocol allows requests
// without api_key when using its default local endpoint.
func IsEmptyAPIKeyAllowedForProtocol(protocol string) bool {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	return isEmptyAPIKeyAllowed(protocol)
}

// DefaultAPIBaseForProtocol returns the configured default API base for a protocol.
// It returns empty string if the protocol has no default base.
func DefaultAPIBaseForProtocol(protocol string) string {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	return getDefaultAPIBase(protocol)
}

// getDefaultAPIBase returns the default API base URL for a given protocol.
func getDefaultAPIBase(protocol string) string {
	meta, ok := protocolMetaByName[protocol]
	if !ok {
		return ""
	}
	return meta.defaultAPIBase
}
