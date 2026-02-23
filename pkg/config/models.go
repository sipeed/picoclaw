package config

type ModelInfo struct {
	Name        string
	ID          string
	Description string
	ContextSize int
	InputCost   float64
	OutputCost  float64
}

type ProviderModels struct {
	Provider   string
	Models     []ModelInfo
	AuthMethod string
}

var ProviderModelsList = []ProviderModels{
	{
		Provider:   "openai",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "gpt-5.2", Name: "GPT-5.2", ContextSize: 128000},
			{ID: "gpt-5.2-chat-20251211", Name: "GPT-5.2 Chat", ContextSize: 128000},
			{ID: "gpt-5.2-pro-20251211", Name: "GPT-5.2 Pro", ContextSize: 128000},
			{ID: "gpt-5.2-codex-20260114", Name: "GPT-5.2 Codex", ContextSize: 128000},
			{ID: "gpt-5.1-codex-max-20251204", Name: "GPT-5.1 Codex Max", ContextSize: 128000},
			{ID: "gpt-5.1-20251113", Name: "GPT-5.1", ContextSize: 128000},
			{ID: "gpt-5-pro-2025-10-06", Name: "GPT-5 Pro", ContextSize: 128000},
			{ID: "gpt-5-2025-08-07", Name: "GPT-5", ContextSize: 128000},
			{ID: "gpt-4o-2024-11-20", Name: "GPT-4o (Nov 2024)", ContextSize: 128000},
			{ID: "gpt-4o", Name: "GPT-4o", ContextSize: 128000},
		},
	},
	{
		Provider:   "anthropic",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "claude-4.6-sonnet-20260217", Name: "Claude 4.6 Sonnet", ContextSize: 200000},
			{ID: "claude-4.6-opus-20260205", Name: "Claude 4.6 Opus", ContextSize: 200000},
			{ID: "claude-4.5-opus-20251124", Name: "Claude 4.5 Opus", ContextSize: 200000},
			{ID: "claude-4.5-sonnet-20250929", Name: "Claude 4.5 Sonnet", ContextSize: 200000},
			{ID: "claude-4.5-haiku-20251001", Name: "Claude 4.5 Haiku", ContextSize: 200000},
			{ID: "claude-4.1-opus-20250805", Name: "Claude 4.1 Opus", ContextSize: 200000},
			{ID: "claude-4-sonnet-20250522", Name: "Claude 4 Sonnet", ContextSize: 200000},
			{ID: "claude-4-opus-20250522", Name: "Claude 4 Opus", ContextSize: 200000},
			{ID: "claude-3-7-sonnet-20250219", Name: "Claude 3.7 Sonnet", ContextSize: 200000},
			{ID: "claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", ContextSize: 200000},
		},
	},
	{
		Provider:   "antigravity",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", ContextSize: 1000000},
			{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", ContextSize: 200000},
			{ID: "gemini-2.5-pro-preview-06-05", Name: "Gemini 2.5 Pro (Preview)", ContextSize: 200000},
			{ID: "gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash Lite", ContextSize: 1000000},
			{
				ID: "gemini-2.5-flash-lite-preview-09-2025", Name: "Gemini 2.5 Flash Lite (Sep 2025)",
				ContextSize: 1000000,
			},
			{ID: "gemini-2.5-flash-image", Name: "Gemini 2.5 Flash (Image)", ContextSize: 1000000},
			{ID: "gemini-2.0-flash-001", Name: "Gemini 2.0 Flash", ContextSize: 1000000},
			{ID: "gemini-2.0-flash-lite-001", Name: "Gemini 2.0 Flash Lite", ContextSize: 1000000},
			{ID: "gemma-3-27b-it", Name: "Gemma 3 27B", ContextSize: 128000},
			{ID: "gemma-3-12b-it", Name: "Gemma 3 12B", ContextSize: 128000},
		},
	},
	{
		Provider:   "gemini",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", ContextSize: 1000000},
			{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", ContextSize: 200000},
			{ID: "gemini-2.5-pro-preview-06-05", Name: "Gemini 2.5 Pro (Preview)", ContextSize: 200000},
			{ID: "gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash Lite", ContextSize: 1000000},
			{ID: "gemini-2.0-flash-001", Name: "Gemini 2.0 Flash", ContextSize: 1000000},
			{ID: "gemini-2.0-flash-lite-001", Name: "Gemini 2.0 Flash Lite", ContextSize: 1000000},
			{ID: "gemma-3-27b-it", Name: "Gemma 3 27B", ContextSize: 128000},
			{ID: "gemma-3-12b-it", Name: "Gemma 3 12B", ContextSize: 128000},
			{ID: "gemma-2-27b-it", Name: "Gemma 2 27B", ContextSize: 128000},
			{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", ContextSize: 200000},
		},
	},
	{
		Provider:   "deepseek",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "deepseek-v3.2-speciale-20251201", Name: "DeepSeek V3.2 Speciale", ContextSize: 64000},
			{ID: "deepseek-v3.2-20251201", Name: "DeepSeek V3.2", ContextSize: 64000},
			{ID: "deepseek-v3.1-terminus", Name: "DeepSeek V3.1 Terminus", ContextSize: 64000},
			{ID: "deepseek-chat-v3.1", Name: "DeepSeek V3.1", ContextSize: 64000},
			{ID: "deepseek-r1-0528", Name: "DeepSeek R1 (May 2025)", ContextSize: 64000},
			{ID: "deepseek-r1", Name: "DeepSeek R1", ContextSize: 64000},
			{ID: "deepseek-r1-distill-qwen-32b", Name: "DeepSeek R1 Distill Qwen 32B", ContextSize: 32000},
			{ID: "deepseek-r1-distill-llama-70b", Name: "DeepSeek R1 Distill Llama 70B", ContextSize: 128000},
			{ID: "deepseek-chat-v3-0324", Name: "DeepSeek V3 (Mar 2024)", ContextSize: 64000},
			{ID: "deepseek-chat", Name: "DeepSeek Chat", ContextSize: 64000},
		},
	},
	{
		Provider:   "moonshot",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "kimi-k2.5-0127", Name: "Kimi k2.5", ContextSize: 128000},
			{ID: "kimi-k2-thinking-20251106", Name: "Kimi k2 Thinking", ContextSize: 128000},
			{ID: "kimi-k2-0905", Name: "Kimi k2 (Sep 2024)", ContextSize: 128000},
			{ID: "kimi-k2", Name: "Kimi k2", ContextSize: 128000},
		},
	},
	{
		Provider:   "qwen",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "qwen3-next-80b-a3b-instruct-2509", Name: "Qwen 3 Next 80B", ContextSize: 128000},
			{ID: "qwen3-next-80b-a3b-thinking-2509", Name: "Qwen 3 Next 80B Thinking", ContextSize: 128000},
			{ID: "qwen3-max-thinking-20260123", Name: "Qwen 3 Max Thinking", ContextSize: 128000},
			{ID: "qwen-plus-2025-07-28", Name: "Qwen Plus (Jul 2025)", ContextSize: 32000},
			{ID: "qwen3-235b-a22b-07-25", Name: "Qwen 3 235B (Jul 2025)", ContextSize: 128000},
			{ID: "qwen3-coder-480b-a35b-07-25", Name: "Qwen 3 Coder 480B", ContextSize: 128000},
			{ID: "qwen3-max", Name: "Qwen 3 Max", ContextSize: 128000},
			{ID: "qwen3-235b-a22b-thinking-2507", Name: "Qwen 3 235B Thinking", ContextSize: 128000},
			{ID: "qwen-plus-2025-01-25", Name: "Qwen Plus", ContextSize: 32000},
			{ID: "qwen-max-2025-01-25", Name: "Qwen Max", ContextSize: 32000},
		},
	},
	{
		Provider:   "mistral",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "mistral-large-2512", Name: "Mistral Large", ContextSize: 128000},
			{ID: "devstral-2512", Name: "Devstral", ContextSize: 128000},
			{ID: "codestral-2508", Name: "Codestral", ContextSize: 256000},
			{ID: "mistral-saba-2502", Name: "Mistral Saba", ContextSize: 128000},
			{ID: "mistral-small-3.2-24b-instruct-2506", Name: "Mistral Small 3.2", ContextSize: 24000},
			{ID: "mistral-small-creative-20251216", Name: "Mistral Small Creative", ContextSize: 24000},
			{ID: "mistral-large-2411", Name: "Mistral Large (Nov 2024)", ContextSize: 128000},
			{ID: "ministral-14b-2512", Name: "Ministral 14B", ContextSize: 128000},
			{ID: "ministral-8b-2512", Name: "Ministral 8B", ContextSize: 128000},
			{ID: "pixtral-large-2411", Name: "Pixtral Large", ContextSize: 128000},
		},
	},
	{
		Provider:   "ollama",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "llama3.3", Name: "Llama 3.3", ContextSize: 128000},
			{ID: "llama3.1", Name: "Llama 3.1", ContextSize: 128000},
			{ID: "qwen2.5", Name: "Qwen 2.5", ContextSize: 32768},
			{ID: "phi4", Name: "Phi 4", ContextSize: 8192},
			{ID: "mistral", Name: "Mistral", ContextSize: 8192},
			{ID: "codellama", Name: "Code Llama", ContextSize: 16384},
			{ID: "llama3", Name: "Llama 3", ContextSize: 8192},
			{ID: "gemma2", Name: "Gemma 2", ContextSize: 8192},
			{ID: "granite3.1", Name: "Granite 3.1", ContextSize: 128000},
			{ID: "deepseek-llm", Name: "DeepSeek LLM", ContextSize: 16384},
		},
	},
	{
		Provider:   "groq",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B", ContextSize: 8192},
			{ID: "llama-3.2-90b-versatile", Name: "Llama 3.2 90B", ContextSize: 8192},
			{ID: "mixtral-8x22b-32768", Name: "Mixtral 8x22B", ContextSize: 32768},
			{ID: "deepseek-r1-distill-llama-70b", Name: "DeepSeek R1 Distill Llama 70B", ContextSize: 8192},
			{ID: "llama-3.1-70b-versatile", Name: "Llama 3.1 70B", ContextSize: 8192},
			{ID: "gemma2-9b-it", Name: "Gemma 2 9B", ContextSize: 8192},
			{ID: "qwen-2.5-72b-instruct", Name: "Qwen 2.5 72B", ContextSize: 8192},
			{ID: "llama3-70b-8192", Name: "Llama 3 70B", ContextSize: 8192},
			{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B", ContextSize: 8192},
			{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", ContextSize: 32768},
		},
	},
	{
		Provider:   "openrouter",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "openai/gpt-4o", Name: "GPT-4o (OpenRouter)", ContextSize: 128000},
			{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini (OpenRouter)", ContextSize: 128000},
			{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet (OpenRouter)", ContextSize: 200000},
			{ID: "google/gemini-2.5-flash", Name: "Gemini 2.5 Flash (OpenRouter)", ContextSize: 1000000},
			{ID: "deepseek/deepseek-r1", Name: "DeepSeek R1 (OpenRouter)", ContextSize: 64000},
			{ID: "meta-llama/llama-3.3-70b-instruct", Name: "Llama 3.3 70B (OpenRouter)", ContextSize: 128000},
			{ID: "mistralai/mistral-large", Name: "Mistral Large (OpenRouter)", ContextSize: 128000},
			{ID: "qwen/qwen-2.5-72b-instruct", Name: "Qwen 2.5 72B (OpenRouter)", ContextSize: 32000},
			{ID: "x-ai/grok-3-beta", Name: "Grok 3 Beta (OpenRouter)", ContextSize: 131072},
			{ID: "cohere/command-a", Name: "Command A (OpenRouter)", ContextSize: 128000},
		},
	},
	{
		Provider:   "nvidia",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "nvidia/llama-3.1-nemotron-70b-instruct", Name: "Nemotron 70B", ContextSize: 128000},
			{ID: "nvidia/llama-3.1-nemotron-70b-instruct-mini", Name: "Nemotron 70B Mini", ContextSize: 128000},
			{ID: "nvidia/llama-3.3-70b-instruct", Name: "Llama 3.3 70B", ContextSize: 128000},
		},
	},
	{
		Provider:   "vllm",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "qwen-2.5-72b-instruct", Name: "Qwen 2.5 72B", ContextSize: 32768},
			{ID: "llama-3.1-8b-instruct", Name: "Llama 3.1 8B", ContextSize: 8192},
		},
	},
	{
		Provider:   "cerebras",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "llama-3.3-70b", Name: "Llama 3.3 70B", ContextSize: 8192},
			{ID: "llama-3.1-8b", Name: "Llama 3.1 8B", ContextSize: 8192},
		},
	},
	{
		Provider:   "zhipu",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "glm-4-plus", Name: "GLM-4 Plus", ContextSize: 128000},
			{ID: "glm-4-long", Name: "GLM-4 Long", ContextSize: 128000},
			{ID: "glm-4", Name: "GLM-4", ContextSize: 128000},
			{ID: "glm-4-flash", Name: "GLM-4 Flash", ContextSize: 128000},
			{ID: "glm-3-turbo", Name: "GLM-3 Turbo", ContextSize: 6000},
		},
	},
	{
		Provider:   "github_copilot",
		AuthMethod: "oauth",
		Models: []ModelInfo{
			{ID: "copilot", Name: "GitHub Copilot", ContextSize: 128000},
		},
	},
	{
		Provider:   "codex",
		AuthMethod: "oauth",
		Models: []ModelInfo{
			{ID: "codex-cli", Name: "OpenAI Codex CLI", ContextSize: 200000},
		},
	},
	{
		Provider:   "volcengine",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "doubao-pro-32k", Name: "Doubao Pro 32K", ContextSize: 32000},
			{ID: "doubao-lite-32k", Name: "Doubao Lite 32K", ContextSize: 32000},
		},
	},
	{
		Provider:   "shengsuanyun",
		AuthMethod: "api_key",
		Models: []ModelInfo{
			{ID: "yi-large", Name: "Yi Large", ContextSize: 16000},
			{ID: "yi-medium", Name: "Yi Medium", ContextSize: 32000},
		},
	},
}

func GetModelsForProvider(provider string) []ModelInfo {
	for _, pm := range ProviderModelsList {
		if pm.Provider == provider {
			return pm.Models
		}
	}
	return nil
}

func GetProviderAuthMethod(provider string) string {
	for _, pm := range ProviderModelsList {
		if pm.Provider == provider {
			return pm.AuthMethod
		}
	}
	return "api_key"
}
