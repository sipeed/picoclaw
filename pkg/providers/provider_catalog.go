package providers

import (
	"sort"
	"strings"
)

// ModelProviderOption describes a canonical provider entry exposed to the Web UI.
type ModelProviderOption struct {
	ID                  string `json:"id"`
	DefaultAPIBase      string `json:"default_api_base"`
	EmptyAPIKeyAllowed  bool   `json:"empty_api_key_allowed"`
	CreateAllowed       bool   `json:"create_allowed"`
	DefaultModelAllowed bool   `json:"default_model_allowed"`
	DefaultAuthMethod   string `json:"default_auth_method,omitempty"`
	AuthMethodLocked    bool   `json:"auth_method_locked,omitempty"`
}

type attachedModelProviderMeta struct {
	protocolMeta
	createAllowed       bool
	defaultModelAllowed bool
	defaultAuthMethod   string
	authMethodLocked    bool
}

// attachedModelProviderMetaByName augments protocolMetaByName for provider
// families that are implemented in CreateProviderFromConfig but intentionally
// kept out of the core HTTP metadata map because they have special auth/runtime
// semantics.
var attachedModelProviderMetaByName = map[string]attachedModelProviderMeta{
	"azure": {createAllowed: true, defaultModelAllowed: true},
	"anthropic": {
		protocolMeta:        protocolMeta{defaultAPIBase: "https://api.anthropic.com/v1"},
		createAllowed:       true,
		defaultModelAllowed: true,
	},
	"anthropic-messages": {
		protocolMeta:        protocolMeta{defaultAPIBase: "https://api.anthropic.com/v1"},
		createAllowed:       true,
		defaultModelAllowed: true,
	},
	"bedrock": {createAllowed: true, defaultModelAllowed: true},
	"antigravity": {
		createAllowed:       true,
		defaultModelAllowed: true,
		defaultAuthMethod:   "oauth",
		authMethodLocked:    true,
	},
	"claude-cli": {createAllowed: true, defaultModelAllowed: true},
	"codex-cli":  {createAllowed: true, defaultModelAllowed: true},
	"github-copilot": {
		protocolMeta:        protocolMeta{defaultAPIBase: "localhost:4321"},
		createAllowed:       true,
		defaultModelAllowed: true,
	},
	// ElevenLabs is intentionally exposed only as an ASR-capable provider. It
	// belongs in the shared model catalog because ASR is configured via
	// model_list, but it must not be selectable as the default chat model.
	"elevenlabs": {
		protocolMeta:        protocolMeta{defaultAPIBase: "https://api.elevenlabs.io"},
		createAllowed:       true,
		defaultModelAllowed: false,
	},
}

// ModelProviderOptions returns the canonical provider catalog exposed to the Web UI.
func ModelProviderOptions() []ModelProviderOption {
	optionsByID := make(map[string]ModelProviderOption, len(protocolMetaByName)+len(attachedModelProviderMetaByName))
	for provider := range protocolMetaByName {
		if NormalizeProvider(provider) != provider {
			continue
		}
		optionsByID[provider] = ModelProviderOption{
			ID:                  provider,
			DefaultAPIBase:      DefaultAPIBaseForProtocol(provider),
			EmptyAPIKeyAllowed:  IsEmptyAPIKeyAllowedForProtocol(provider),
			CreateAllowed:       true,
			DefaultModelAllowed: true,
		}
	}
	for provider, meta := range attachedModelProviderMetaByName {
		if NormalizeProvider(provider) != provider {
			continue
		}
		optionsByID[provider] = ModelProviderOption{
			ID:                  provider,
			DefaultAPIBase:      meta.defaultAPIBase,
			EmptyAPIKeyAllowed:  meta.emptyAPIKeyAllowed,
			CreateAllowed:       meta.createAllowed,
			DefaultModelAllowed: meta.defaultModelAllowed,
			DefaultAuthMethod:   meta.defaultAuthMethod,
			AuthMethodLocked:    meta.authMethodLocked,
		}
	}

	options := make([]ModelProviderOption, 0, len(optionsByID))
	for _, option := range optionsByID {
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
	normalized := NormalizeProvider(provider)
	if normalized == "" {
		return false
	}
	if _, ok := protocolMetaByName[normalized]; ok {
		return true
	}
	_, ok := attachedModelProviderMetaByName[normalized]
	return ok
}

// IsCreatableModelProvider reports whether provider can be selected for a new
// model entry from the Web UI.
func IsCreatableModelProvider(provider string) bool {
	normalized := NormalizeProvider(provider)
	if normalized == "" {
		return false
	}
	if _, ok := protocolMetaByName[normalized]; ok {
		return true
	}
	meta, ok := attachedModelProviderMetaByName[normalized]
	return ok && meta.createAllowed
}

// IsDefaultModelProvider reports whether provider can be used as the default
// chat model. Some providers such as ASR-only entries are intentionally
// exposed in model_list management but cannot drive the gateway default model.
func IsDefaultModelProvider(provider string) bool {
	normalized := NormalizeProvider(provider)
	if normalized == "" {
		return false
	}
	if _, ok := protocolMetaByName[normalized]; ok {
		return true
	}
	meta, ok := attachedModelProviderMetaByName[normalized]
	return ok && meta.defaultModelAllowed
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
