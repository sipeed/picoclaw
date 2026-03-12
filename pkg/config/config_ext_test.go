package config

import (
	"encoding/json"
	"testing"
)

func TestAgentDefaults_PlanModel_StringParse(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7",
				"plan_model": "anthropic/claude-sonnet-4-6",
				"plan_model_fallbacks": ["openai/gpt-4o"],
				"max_tokens": 8192,
				"max_tool_iterations": 20
			}
		}
	}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(jsonData), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cfg.Agents.Defaults.PlanModel != "anthropic/claude-sonnet-4-6" {
		t.Errorf("PlanModel = %q, want 'anthropic/claude-sonnet-4-6'", cfg.Agents.Defaults.PlanModel)
	}
	if len(cfg.Agents.Defaults.PlanModelFallbacks) != 1 ||
		cfg.Agents.Defaults.PlanModelFallbacks[0] != "openai/gpt-4o" {
		t.Errorf("PlanModelFallbacks = %v, want [openai/gpt-4o]", cfg.Agents.Defaults.PlanModelFallbacks)
	}
}

func TestAgentConfig_PlanModel_ObjectParse(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7",
				"max_tokens": 8192,
				"max_tool_iterations": 20
			},
			"list": [
				{
					"id": "main",
					"plan_model": "anthropic/claude-sonnet-4-6"
				},
				{
					"id": "advanced",
					"plan_model": {
						"primary": "anthropic/claude-opus-4",
						"fallbacks": ["anthropic/claude-sonnet-4-6"]
					}
				}
			]
		}
	}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(jsonData), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(cfg.Agents.List) != 2 {
		t.Fatalf("agents.list len = %d, want 2", len(cfg.Agents.List))
	}

	main := cfg.Agents.List[0]
	if main.PlanModel == nil || main.PlanModel.Primary != "anthropic/claude-sonnet-4-6" {
		t.Errorf("main.PlanModel = %+v, want primary 'anthropic/claude-sonnet-4-6'", main.PlanModel)
	}

	adv := cfg.Agents.List[1]
	if adv.PlanModel == nil || adv.PlanModel.Primary != "anthropic/claude-opus-4" {
		t.Errorf("advanced.PlanModel = %+v, want primary 'anthropic/claude-opus-4'", adv.PlanModel)
	}
	if len(adv.PlanModel.Fallbacks) != 1 || adv.PlanModel.Fallbacks[0] != "anthropic/claude-sonnet-4-6" {
		t.Errorf("advanced.PlanModel.Fallbacks = %v", adv.PlanModel.Fallbacks)
	}
}

func TestAgentConfig_PlanModel_OverridesDefaults(t *testing.T) {
	jsonData := `{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model": "glm-4.7",
				"plan_model": "default-plan-model",
				"plan_model_fallbacks": ["default-fallback"],
				"max_tokens": 8192,
				"max_tool_iterations": 20
			},
			"list": [
				{
					"id": "custom",
					"plan_model": {
						"primary": "custom-plan-model",
						"fallbacks": ["custom-fallback"]
					}
				}
			]
		}
	}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(jsonData), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	custom := cfg.Agents.List[0]
	if custom.PlanModel == nil || custom.PlanModel.Primary != "custom-plan-model" {
		t.Errorf("custom.PlanModel.Primary = %v, want 'custom-plan-model'", custom.PlanModel)
	}
	if len(custom.PlanModel.Fallbacks) != 1 || custom.PlanModel.Fallbacks[0] != "custom-fallback" {
		t.Errorf("custom.PlanModel.Fallbacks = %v, want [custom-fallback]", custom.PlanModel.Fallbacks)
	}

	if cfg.Agents.Defaults.PlanModel != "default-plan-model" {
		t.Errorf("defaults.PlanModel = %q, want 'default-plan-model'", cfg.Agents.Defaults.PlanModel)
	}
}
