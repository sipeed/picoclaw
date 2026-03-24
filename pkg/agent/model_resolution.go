package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func ensureProtocol(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if strings.Contains(model, "/") {
		return model
	}
	return "openai/" + model
}

func resolveFromModelList(cfg *config.Config, raw string) (string, bool) {
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
