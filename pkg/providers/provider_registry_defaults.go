package providers

func init() {
	RegisterProvider(providerRegistration{
		Name:          "moonshot",
		ModelPrefixes: []string{"moonshot/", "moonshot-", "kimi-"},
		Creator:       moonshotCreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "openrouter",
		ModelPrefixes: []string{"openrouter/", "anthropic/", "openai/", "meta-llama/", "deepseek/", "google/"},
		Creator:       openRouterCreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "anthropic",
		Aliases:       []string{"claude"},
		ModelPrefixes: []string{"claude-", "anthropic/"},
		Creator:       claudeCreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "openai",
		Aliases:       []string{"gpt"},
		ModelPrefixes: []string{"gpt-", "o1-", "o3-", "o4-", "chatgpt-", "openai/"},
		Creator:       openAICreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "gemini",
		Aliases:       []string{"google"},
		ModelPrefixes: []string{"gemini-", "google/"},
		Creator:       geminiCreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "zhipu",
		Aliases:       []string{"glm"},
		ModelPrefixes: []string{"glm-", "zhipu/", "zai-"},
		Creator:       zhipuCreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "groq",
		ModelPrefixes: []string{"groq/"},
		Creator:       groqCreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "nvidia",
		ModelPrefixes: []string{"nvidia/"},
		Creator:       nvidiaCreator,
	})

	RegisterProvider(providerRegistration{
		Name:          "vllm",
		ModelPrefixes: []string{"vllm/"},
		Creator:       vllmCreator,
	})

	RegisterProvider(providerRegistration{
		Name:    "claude-cli",
		Aliases: []string{"claudecode", "claude-code"},
		Creator: claudeCLICreator,
	})
}
