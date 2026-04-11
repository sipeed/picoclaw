// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// buildModelWithProtocol constructs a model string with protocol prefix.
// If the model already contains a "/" (indicating it has a protocol prefix), it is returned as-is.
// Otherwise, the protocol prefix is added.
func buildModelWithProtocol(protocol, model string) string {
	if strings.Contains(model, "/") {
		// Model already has a protocol prefix, return as-is
		return model
	}
	return protocol + "/" + model
}

// loadConfigV1 loads a version 1 config (current schema)
func loadConfig(data []byte) (*Config, error) {
	cfg := DefaultConfig()

	// Pre-scan the JSON to check how many model_list entries the user provided.
	// Go's JSON decoder reuses existing slice backing-array elements rather than
	// zero-initializing them, so fields absent from the user's JSON (e.g. api_base)
	// would silently inherit values from the DefaultConfig template at the same
	// index position. We only reset cfg.ModelList when the user actually provides
	// entries; when count is 0 we keep DefaultConfig's built-in list as fallback.
	var tmp Config
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}
	if len(tmp.ModelList) > 0 {
		cfg.ModelList = nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func mergeAPIKeys(apiKey string, apiKeys []string) []string {
	seen := make(map[string]struct{})
	var all []string

	if k := strings.TrimSpace(apiKey); k != "" {
		if _, exists := seen[k]; !exists {
			seen[k] = struct{}{}
			all = append(all, k)
		}
	}

	for _, k := range apiKeys {
		if trimmed := strings.TrimSpace(k); trimmed != "" {
			if _, exists := seen[trimmed]; !exists {
				seen[trimmed] = struct{}{}
				all = append(all, trimmed)
			}
		}
	}

	return all
}

func compareInt(v any, expected int) bool {
	switch val := v.(type) {
	case int:
		return val == expected
	case float64:
		return val == float64(expected)
	case nil:
		return expected == 0
	default:
		return false
	}
}

// migrateV0ToV1 converts a V0 (legacy, no version field) config JSON to V1 format:
//  1. Migrates legacy providers to model_list
//  2. Migrates agents.defaults.model → agents.defaults.model_name
//  3. Sets version to 1
func migrateV0ToV1(m map[string]any) error {
	if !compareInt(m["version"], 0) {
		return fmt.Errorf("migrateV0ToV1: expected version 0, got %v", m["version"])
	}

	// Migrate agents.defaults.model → agents.defaults.model_name
	if agents, ok := m["agents"].(map[string]any); ok {
		if defaults, ok := agents["defaults"].(map[string]any); ok {
			if model, hasModel := defaults["model"]; hasModel {
				if _, hasModelName := defaults["model_name"]; !hasModelName {
					defaults["model_name"] = model
				}
				delete(defaults, "model")
			}
		}
	}

	// Migrate legacy providers to model_list if no model_list exists
	if _, hasModelList := m["model_list"]; !hasModelList {
		if providers, hasProviders := m["providers"]; hasProviders {
			if provMap, ok := providers.(map[string]any); ok && !isProvidersMapEmpty(provMap) {
				// Extract user's provider and model from agents.defaults
				userProvider := ""
				userModel := ""
				if agents, ok := m["agents"].(map[string]any); ok {
					if defaults, ok := agents["defaults"].(map[string]any); ok {
						if v, ok := defaults["provider"].(string); ok {
							userProvider = v
						}
						// Check both model_name (new) and model (old) fields
						if v, ok := defaults["model_name"].(string); ok && v != "" {
							userModel = v
						} else if v, ok := defaults["model"].(string); ok && v != "" {
							userModel = v
						}
					}
				}

				modelListRaw := v0ProvidersMapToModelList(provMap, userProvider, userModel)
				if len(modelListRaw) > 0 {
					m["model_list"] = modelListRaw
				}
			}
		}
	}

	// Convert model_list api_key → api_keys
	if modelList, ok := m["model_list"].([]any); ok {
		for _, model := range modelList {
			if mVal, ok := model.(map[string]any); ok {
				if ss := toUniqueStrings(mVal["api_key"], mVal["api_keys"]); len(ss) > 0 {
					mVal["api_keys"] = ss
					delete(mVal, "api_key")
				}
			}
		}
	}

	m["version"] = 1

	return nil
}

func toUniqueStrings(s any, ss any) []string {
	set := make(map[string]struct{})

	// process s
	if str, ok := s.(string); ok && str != "" {
		set[str] = struct{}{}
	}

	// process ss as []any (JSON arrays)
	if slice, ok := ss.([]any); ok {
		for _, item := range slice {
			if str, ok := item.(string); ok && str != "" {
				set[str] = struct{}{}
			}
		}
	}

	// process ss as []string
	if slice, ok := ss.([]string); ok {
		for _, item := range slice {
			if item != "" {
				set[item] = struct{}{}
			}
		}
	}

	// map to slice
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}

	return result
}

// migrateV1ToV2 converts a V1 config JSON to V2 format:
//  1. Migrates legacy "mention_only" to "group_trigger.mention_only"
//  2. Infers "enabled" field for models
//  3. Sets version to 2
func migrateV1ToV2(m map[string]any) error {
	if !compareInt(m["version"], 1) {
		return fmt.Errorf("migrateV1ToV2: expected version 1, got %#v", m["version"])
	}

	// Migrate channels: move "mention_only" to "group_trigger.mention_only"
	if channels, ok := m["channels"]; ok {
		if chMap, ok := channels.(map[string]any); ok {
			for _, ch := range chMap {
				if chVal, ok := ch.(map[string]any); ok {
					if mentionOnly, hasMention := chVal["mention_only"]; hasMention {
						delete(chVal, "mention_only")
						if gt, hasGT := chVal["group_trigger"].(map[string]any); hasGT {
							gt["mention_only"] = mentionOnly
						} else {
							chVal["group_trigger"] = map[string]any{"mention_only": mentionOnly}
						}
					}
				}
			}
		}
	}

	// Infer "enabled" field for models matching configV1.migrateModelEnabled behavior
	if modelList, ok := m["model_list"].([]any); ok {
		// Convert api_key → api_keys for each model
		for _, model := range modelList {
			if mVal, ok := model.(map[string]any); ok {
				if ss := toUniqueStrings(mVal["api_key"], mVal["api_keys"]); len(ss) > 0 {
					mVal["api_keys"] = ss
					delete(mVal, "api_key")
				}
			}
		}

		// Infer enabled status
		for _, model := range modelList {
			if mVal, ok := model.(map[string]any); ok {
				// Skip if explicitly set
				if _, hasEnabled := mVal["enabled"]; hasEnabled {
					continue
				}
				// Models with API keys are considered enabled
				if apiKeys, hasAPIKeys := mVal["api_keys"]; hasAPIKeys {
					// Check for []any or []string
					hasKeys := false
					if keys, ok := apiKeys.([]any); ok {
						hasKeys = len(keys) > 0
					} else if keys, ok := apiKeys.([]string); ok {
						hasKeys = len(keys) > 0
					}
					if hasKeys {
						mVal["enabled"] = true
						continue
					}
				}
				// The reserved "local-model" entry is considered enabled
				if mVal["model_name"] == "local-model" {
					mVal["enabled"] = true
				}
				logger.Infof("model: %v", mVal)
			}
		}
	} else {
		logger.Warnf("model_list is not a slice: %#v", m["model_list"])
	}

	m["version"] = 2

	return nil
}

// migrateV2ToV3 converts a V2 config JSON to V3 format:
//  1. Renames "channels" key to "channel_list"
//  2. Converts flat-format channel entries to nested format (wrapping
//     channel-specific fields in "settings")
//  3. Sets version to 3
func migrateV2ToV3(m map[string]any) error {
	if !compareInt(m["version"], 2) {
		return fmt.Errorf("migrateV2ToV3: expected version 2, got %v", m["version"])
	}

	// Rename channels → channel_list
	if channels, ok := m["channels"]; ok {
		delete(m, "channels")

		// Convert each channel from flat to nested format
		if chMap, ok := channels.(map[string]any); ok {
			for k, ch := range chMap {
				if chVal, ok := ch.(map[string]any); ok {
					chVal["type"] = k
					// If already has "settings" key, leave as-is
					if _, hasSettings := chVal["settings"]; hasSettings {
						continue
					}

					// Migrate Onebot "group_trigger_prefix" → "group_trigger.prefixes"
					if gtp, hasGTP := chVal["group_trigger_prefix"]; hasGTP {
						if gt, hasGT := chVal["group_trigger"].(map[string]any); hasGT {
							if _, hasPrefixes := gt["prefixes"]; !hasPrefixes {
								gt["prefixes"] = gtp
							}
						} else {
							chVal["group_trigger"] = map[string]any{"prefixes": gtp}
						}
						delete(chVal, "group_trigger_prefix")
					}

					// Separate channel-specific fields into "settings"
					settings := make(map[string]any)
					for fieldKey, v := range chVal {
						if _, exists := BaseFieldNames[fieldKey]; !exists {
							settings[fieldKey] = v
							delete(chVal, fieldKey)
						}
					}
					if len(settings) > 0 {
						chVal["settings"] = settings
					}
				}
			}
		}

		m["channel_list"] = channels
	}

	m["version"] = CurrentVersion

	return nil
}

func loadConfigMap(path string) (map[string]any, error) {
	var m1, m2 map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m1, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	if err = json.Unmarshal(data, &m1); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	secPath := securityPath(path)
	data, err = os.ReadFile(secPath)
	if err != nil {
		if os.IsNotExist(err) {
			return m1, nil
		}
		return nil, fmt.Errorf("failed to read security config: %w", err)
	}
	if err = yaml.Unmarshal(data, &m2); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}
	if m2["web"] != nil || m2["skills"] != nil {
		m3 := make(map[string]any)
		if m2["web"] != nil {
			m3["web"] = m2["web"]
			delete(m2, "web")
		}
		if m2["skills"] != nil {
			m3["skills"] = m2["skills"]
			delete(m2, "skills")
			if m, ok := m3["skills"].(map[string]any); ok {
				if m["clawhub"] != nil {
					m["registries"] = map[string]any{"clawhub": m["clawhub"]}
					delete(m, "clawhub")
				}
			}
		}
		m2["tools"] = m3
	}

	// Handle model_list merging specially: m1 has array format, m2 has map format
	if mainML, hasMainML := m1["model_list"]; hasMainML {
		if secML, hasSecML := m2["model_list"]; hasSecML {
			if secMap, ok := secML.(map[string]any); ok {
				// JSON unmarshals arrays as []any, convert to []map[string]any
				var mainArr []any
				if rawArr, ok := mainML.([]any); ok {
					mainArr = make([]any, 0, len(rawArr))
					for _, item := range rawArr {
						if mVal, ok := item.(map[string]any); ok {
							mainArr = append(mainArr, mVal)
						}
					}
				}
				if len(mainArr) > 0 {
					// Merge array-style with map-style in-place
					err = mergeModelListsWithMap(mainArr, secMap)
					if err != nil {
						logger.Errorf("mergeModelListsWithMap error: %v", err)
						return nil, err
					}
					m1["model_list"] = mainArr
				}
			}
		}
	}
	// Remove model_list from m2 so mergeMap doesn't override the array with map
	delete(m2, "model_list")

	m := mergeMap(m1, m2)
	return m, nil
}

// mergeModelListsWithMap merges array-style model_list with map-style security model_list.
// It generates indexed keys from model_name (like toNameIndex) and uses them
// to look up security entries, falling back to ModelName if the indexed key doesn't exist.
func mergeModelListsWithMap(mainML []any, secML map[string]any) error {
	// Build indexed keys like toNameIndex does
	indexedKeys := make(map[string]int)
	countMap := make(map[string]int)
	for i, m := range mainML {
		if mVal, ok := m.(map[string]any); ok {
			if name, hasName := mVal["model_name"]; hasName {
				nameStr := name.(string)
				index := countMap[nameStr]
				indexedKeys[fmt.Sprintf("%s:%d", nameStr, index)] = i
				if _, ok := indexedKeys[nameStr]; !ok {
					indexedKeys[nameStr] = i
				}
				countMap[nameStr]++
			} else {
				return fmt.Errorf("model_name is required: %#v", mVal)
			}
		}
	}

	for k, v := range secML {
		if i, ok := indexedKeys[k]; ok {
			if vv, ok := v.(map[string]any); ok {
				if mVal, ok := mainML[i].(map[string]any); ok {
					mVal["api_keys"] = vv["api_keys"]
				}
			}
		} else {
			logger.Warnf("model_name not found in main config: %s", k)
		}
		delete(secML, k)
	}

	return nil
}
