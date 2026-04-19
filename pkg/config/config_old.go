// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import "strings"

// isProvidersMapEmpty checks if a providers map has any non-empty provider configurations.
func isProvidersMapEmpty(providers map[string]any) bool {
	for _, prov := range providers {
		if provMap, ok := prov.(map[string]any); ok {
			if apiKey, ok := provMap["api_key"]; ok && apiKey != "" {
				return false
			}
			if apiBase, ok := provMap["api_base"]; ok && apiBase != "" {
				return false
			}
			if connectMode, ok := provMap["connect_mode"]; ok && connectMode != "" {
				return false
			}
			if authMethod, ok := provMap["auth_method"]; ok && authMethod != "" {
				return false
			}
		}
	}
	return true
}

// v0ProvidersMapToModelList converts a V0 providers map to a model_list slice.
func v0ProvidersMapToModelList(providers map[string]any, userProvider, userModel string) []any {
	// providerMigration defines migration rules for a provider
	type providerMigration struct {
		jsonKeys  []string
		protocol  string
		defModel  string
		extractFn func(prov map[string]any) map[string]any
	}

	migrations := []providerMigration{
		{
			jsonKeys: []string{"openai", "gpt"},
			protocol: "openai",
			defModel: "openai/gpt-5.4",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				if v, ok := prov["auth_method"]; ok && v != "" {
					entry["auth_method"] = v
				}
				if v, ok := prov["web_search"]; ok && v != false {
					entry["web_search"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"anthropic", "claude"},
			protocol: "anthropic",
			defModel: "anthropic/claude-sonnet-4.6",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				if v, ok := prov["auth_method"]; ok && v != "" {
					entry["auth_method"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"litellm"},
			protocol: "litellm",
			defModel: "litellm/auto",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"openrouter"},
			protocol: "openrouter",
			defModel: "openrouter/auto",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"groq"},
			protocol: "groq",
			defModel: "groq/llama-3.1-70b-versatile",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"zhipu", "glm"},
			protocol: "zhipu",
			defModel: "zhipu/glm-4",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"vllm"},
			protocol: "vllm",
			defModel: "vllm/auto",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"gemini", "google"},
			protocol: "gemini",
			defModel: "gemini/gemini-pro",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"nvidia"},
			protocol: "nvidia",
			defModel: "nvidia/meta/llama-3.1-8b-instruct",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"ollama"},
			protocol: "ollama",
			defModel: "ollama/llama3",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"moonshot", "kimi"},
			protocol: "moonshot",
			defModel: "moonshot/kimi",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"shengsuanyun"},
			protocol: "shengsuanyun",
			defModel: "shengsuanyun/auto",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"deepseek"},
			protocol: "deepseek",
			defModel: "deepseek/deepseek-chat",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"cerebras"},
			protocol: "cerebras",
			defModel: "cerebras/llama-3.3-70b",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"vivgrid"},
			protocol: "vivgrid",
			defModel: "vivgrid/auto",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"volcengine", "doubao"},
			protocol: "volcengine",
			defModel: "volcengine/doubao-pro",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"github_copilot", "copilot"},
			protocol: "github-copilot",
			defModel: "github-copilot/gpt-5.4",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["connect_mode"]; ok && v != "" {
					entry["connect_mode"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"antigravity"},
			protocol: "antigravity",
			defModel: "antigravity/gemini-2.0-flash",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["auth_method"]; ok && v != "" {
					entry["auth_method"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"qwen", "tongyi"},
			protocol: "qwen",
			defModel: "qwen/qwen-max",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"mistral"},
			protocol: "mistral",
			defModel: "mistral/mistral-small-latest",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"avian"},
			protocol: "avian",
			defModel: "avian/deepseek/deepseek-v3.2",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"minimax"},
			protocol: "minimax",
			defModel: "minimax/minimax",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"longcat"},
			protocol: "longcat",
			defModel: "longcat/LongCat-Flash-Thinking",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"modelscope"},
			protocol: "modelscope",
			defModel: "modelscope/Qwen/Qwen3-235B-A22B-Instruct-2507",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
		{
			jsonKeys: []string{"novita"},
			protocol: "novita",
			defModel: "novita/auto",
			extractFn: func(prov map[string]any) map[string]any {
				entry := make(map[string]any)
				if v, ok := prov["api_key"]; ok && v != "" {
					entry["api_key"] = v
				}
				if v, ok := prov["api_base"]; ok && v != "" {
					entry["api_base"] = v
				}
				if v, ok := prov["proxy"]; ok && v != "" {
					entry["proxy"] = v
				}
				if v, ok := prov["request_timeout"]; ok && v != nil {
					entry["request_timeout"] = v
				}
				return entry
			},
		},
	}

	// We need access to agents.defaults for user provider/model, but we only have providers map
	// This function is called with just the providers map, so we can't access agents.defaults
	// The caller (migrateV0ToV1) would need to pass this information if needed
	// For now, we skip the user provider/model matching

	var result []any

	for _, migration := range migrations {
		// Find the provider in the providers map
		var provData map[string]any
		found := false
		for _, key := range migration.jsonKeys {
			if v, ok := providers[key]; ok {
				if provMap, ok := v.(map[string]any); ok {
					provData = provMap
					found = true
					break
				}
			}
		}
		if !found {
			continue
		}

		// Extract fields using the extraction function
		entry := migration.extractFn(provData)
		if len(entry) == 0 {
			continue
		}

		// Add model_name and model
		entry["model_name"] = migration.jsonKeys[0]

		// Use the user's model if the provider matches, otherwise use the default
		modelToUse := migration.defModel
		if userProvider != "" && userModel != "" {
			for _, key := range migration.jsonKeys {
				if userProvider == key {
					// Build the model string with protocol prefix if needed
					if !strings.Contains(userModel, "/") {
						modelToUse = migration.protocol + "/" + userModel
					} else {
						modelToUse = userModel
					}
					break
				}
			}
		}
		entry["model"] = modelToUse

		result = append(result, entry)
	}

	return result
}
