package agent

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func buildModelListResolver(cfg *config.Config) func(raw string) (string, bool) {
	ensureProtocol := func(model string) string {
		model = strings.TrimSpace(model)
		if model == "" {
			return ""
		}
		if strings.Contains(model, "/") {
			return model
		}
		return "openai/" + model
	}

	return func(raw string) (string, bool) {
		raw = strings.TrimSpace(raw)
		if raw == "" || cfg == nil {
			return "", false
		}

		if mc, err := cfg.GetModelConfig(raw); err == nil && mc != nil && strings.TrimSpace(mc.Model) != "" {
			return ensureProtocol(mc.Model), true
		}

		for i := range cfg.ModelList {
			fullModel := strings.TrimSpace(cfg.ModelList[i].Model)
			if fullModel == "" {
				continue
			}
			if fullModel == raw {
				return ensureProtocol(fullModel), true
			}
			_, modelID := providers.ExtractProtocol(fullModel)
			if modelID == raw {
				return ensureProtocol(fullModel), true
			}
		}

		return "", false
	}
}

func resolveModelCandidates(
	cfg *config.Config,
	defaultProvider string,
	primary string,
	fallbacks []string,
) []providers.FallbackCandidate {
	return providers.ResolveCandidatesWithLookup(
		providers.ModelConfig{
			Primary:   primary,
			Fallbacks: fallbacks,
		},
		defaultProvider,
		buildModelListResolver(cfg),
	)
}

func resolvedCandidateModel(candidates []providers.FallbackCandidate, fallback string) string {
	if len(candidates) > 0 && strings.TrimSpace(candidates[0].Model) != "" {
		return candidates[0].Model
	}
	return fallback
}

func resolvedCandidateProvider(candidates []providers.FallbackCandidate, fallback string) string {
	if len(candidates) > 0 && strings.TrimSpace(candidates[0].Provider) != "" {
		return candidates[0].Provider
	}
	return fallback
}

func resolvedModelConfig(cfg *config.Config, modelName, workspace string) (*config.ModelConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	modelCfg, err := cfg.GetModelConfig(strings.TrimSpace(modelName))
	if err != nil {
		return nil, err
	}

	clone := *modelCfg
	if clone.Workspace == "" {
		clone.Workspace = workspace
	}

	return &clone, nil
}
