package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigExample_DocumentsAdvancedFields(t *testing.T) {
	examplePath := filepath.Join("..", "..", "config", "config.example.json")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read config example: %v", err)
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal config example: %v", err)
	}

	requiredPaths := [][]any{
		{"agents", "defaults", "allow_read_outside_workspace"},
		{"agents", "defaults", "image_model"},
		{"agents", "defaults", "image_model_fallbacks"},
		{"agents", "defaults", "max_media_size"},
		{"agents", "defaults", "routing", "light_model"},
		{"agents", "list", 0, "subagents", "allow_agents"},
		{"bindings", 0, "match", "peer", "kind"},
		{"session", "identity_links"},
		{"channels", "telegram", "typing", "enabled"},
		{"channels", "telegram", "placeholder", "text"},
		{"channels", "qq", "max_message_length"},
		{"channels", "qq", "send_markdown"},
		{"channels", "dingtalk", "group_trigger", "mention_only"},
		{"channels", "slack", "typing", "enabled"},
		{"channels", "line", "typing", "enabled"},
		{"channels", "onebot", "group_trigger", "prefixes"},
		{"channels", "pico", "allow_token_query"},
		{"channels", "pico", "max_connections"},
		{"tools", "web", "proxy"},
		{"tools", "web", "fetch_limit_bytes"},
		{"tools", "exec", "allow_remote"},
		{"tools", "exec", "custom_allow_patterns"},
		{"tools", "exec", "timeout_seconds"},
		{"tools", "media_cleanup", "max_age_minutes"},
		{"tools", "read_file", "max_read_file_size"},
	}

	for _, path := range requiredPaths {
		if _, ok := lookupPath(root, path...); !ok {
			t.Errorf("config example missing documented path %v", path)
		}
	}
}

func lookupPath(root any, parts ...any) (any, bool) {
	current := root
	for _, part := range parts {
		switch key := part.(type) {
		case string:
			obj, ok := current.(map[string]any)
			if !ok {
				return nil, false
			}
			next, ok := obj[key]
			if !ok {
				return nil, false
			}
			current = next
		case int:
			list, ok := current.([]any)
			if !ok || key < 0 || key >= len(list) {
				return nil, false
			}
			current = list[key]
		default:
			return nil, false
		}
	}
	return current, true
}
