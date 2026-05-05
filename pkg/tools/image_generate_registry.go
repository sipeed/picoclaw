package tools

import "strings"

const (
	defaultImageGenerationProvider = "openai-codex"
	defaultImageGenerationModel    = "gpt-image-2"
)

var imageGenerationProviderFactories = map[string]imageGenerationProviderFactory{
	defaultImageGenerationProvider: func() imageGenerationProvider {
		return newOpenAICodexImageGenerationProvider()
	},
}

type imageGenerationModelSpec struct {
	Provider string
	Model    string
}

func parseImageGenerationModel(model string) imageGenerationModelSpec {
	model = strings.TrimSpace(model)
	if model == "" {
		return imageGenerationModelSpec{
			Provider: defaultImageGenerationProvider,
			Model:    defaultImageGenerationModel,
		}
	}
	provider, modelName, ok := strings.Cut(model, "/")
	if !ok || strings.TrimSpace(provider) == "" || strings.TrimSpace(modelName) == "" {
		return imageGenerationModelSpec{
			Provider: defaultImageGenerationProvider,
			Model:    model,
		}
	}
	provider = strings.TrimSpace(provider)
	modelName = strings.TrimSpace(modelName)
	if provider == "openai" {
		provider = defaultImageGenerationProvider
	}
	return imageGenerationModelSpec{
		Provider: provider,
		Model:    modelName,
	}
}
