package providers

import (
	"sort"
	"strings"
)

// ModelProviderOptions returns the canonical provider catalog exposed to the Web UI.
func ModelProviderOptions() []ModelProviderOption {
	options := make([]ModelProviderOption, 0, len(modelProviderOptionsByName))
	for _, option := range modelProviderOptionsByName {
		options = append(options, option)
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].ID < options[j].ID
	})
	return options
}

// IsSupportedModelProvider reports whether provider resolves to a provider ID
// returned by ModelProviderOptions.
func IsSupportedModelProvider(provider string) bool {
	_, ok := modelProviderOptionForName(provider)
	return ok
}

// IsModelProviderFetchable reports whether provider supports upstream /models
// listing through the launcher fetch endpoint.
func IsModelProviderFetchable(provider string) bool {
	option, ok := modelProviderOptionForName(provider)
	return ok && option.SupportsFetch
}

// IsCreatableModelProvider reports whether provider can be selected for a new
// model entry from the Web UI.
func IsCreatableModelProvider(provider string) bool {
	option, ok := modelProviderOptionForName(provider)
	return ok && option.CreateAllowed
}

// IsDefaultModelProvider reports whether provider can be used as the default
// chat model. Some providers such as ASR-only entries are intentionally
// exposed in model_list management but cannot drive the gateway default model.
func IsDefaultModelProvider(provider string) bool {
	option, ok := modelProviderOptionForName(provider)
	return ok && option.DefaultModelAllowed
}

// SplitModelProviderAndID separates a legacy "provider/model" string into its
// effective provider and canonical model ID. Unknown prefixes are treated as
// part of the model ID and fall back to defaultProvider.
func SplitModelProviderAndID(model, defaultProvider string) (provider, modelID string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", ""
	}

	provider, modelID = splitKnownProviderModel(model)
	if provider != "" || modelID != "" {
		return provider, modelID
	}

	return NormalizeProvider(defaultProvider), model
}

func splitKnownProviderModel(model string) (provider, modelID string) {
	provider, modelID, found := strings.Cut(strings.TrimSpace(model), "/")
	if !found {
		return "", ""
	}
	provider = strings.TrimSpace(provider)
	modelID = strings.TrimSpace(modelID)
	if provider == "" {
		return "", modelID
	}
	if !IsSupportedModelProvider(provider) {
		return "", ""
	}
	return NormalizeProvider(provider), modelID
}
