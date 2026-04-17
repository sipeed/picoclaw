// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

const (
	SecurityConfigFile = ".security.yml"
)

// securityPath returns the path to security.yml relative to the config file
func securityPath(configPath string) string {
	configDir := filepath.Dir(configPath)
	return filepath.Join(configDir, SecurityConfigFile)
}

// loadSecurityConfig loads the security configuration from security.yml
// and merges secure field values into the config.
func loadSecurityConfig(cfg *Config, securityPath string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	data, err := os.ReadFile(securityPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read security config: %w", err)
	}

	// Save existing channels and ModelList before unmarshal
	savedChannels := make(ChannelsConfig, len(cfg.Channels))
	for name, bc := range cfg.Channels {
		savedChannels[name] = bc
	}
	// savedModelList := cfg.ModelList

	// Parse YAML into a yaml.Node tree to extract channels node
	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return fmt.Errorf("failed to parse security config: %w", err)
	}

	// Extract channels node (support both 'channels' and 'channel_list' keys)
	var channelsNode *yaml.Node
	if len(rootNode.Content) > 0 {
		content := rootNode.Content[0].Content
		for i := 0; i < len(content); i += 2 {
			if i+1 < len(content) {
				key := content[i].Value
				if key == "channels" || key == "channel_list" {
					channelsNode = content[i+1]
					break
				}
			}
		}
	}

	// Unmarshal non-channel fields from security.yml
	// This will resolve encrypted values for model_list, tools, etc.
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse security config: %w", err)
	}
	if err := applyLegacySkillsSecurityConfig(cfg, data); err != nil {
		return fmt.Errorf("failed to parse legacy skills security config: %w", err)
	}

	// Restore channels from saved, then manually merge from security.yml
	cfg.Channels = make(ChannelsConfig)
	for name, savedBC := range savedChannels {
		cfg.Channels[name] = savedBC
	}

	// If we found a channels node in security.yml, merge it into existing channels
	if channelsNode != nil {
		if err := cfg.Channels.UnmarshalYAML(channelsNode); err != nil {
			return fmt.Errorf("failed to merge channels from security config: %w", err)
		}
	}

	return nil
}

func applyLegacySkillsSecurityConfig(cfg *Config, data []byte) error {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	if len(root.Content) == 0 {
		return nil
	}

	rootMap := root.Content[0]
	if rootMap == nil || rootMap.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(rootMap.Content); i += 2 {
		keyNode := rootMap.Content[i]
		valueNode := rootMap.Content[i+1]
		if keyNode == nil || valueNode == nil || strings.TrimSpace(keyNode.Value) != "skills" {
			continue
		}
		return applyLegacySkillsSecurityNode(cfg, valueNode)
	}

	return nil
}

func applyLegacySkillsSecurityNode(cfg *Config, skillsNode *yaml.Node) error {
	if cfg == nil || skillsNode == nil || skillsNode.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(skillsNode.Content); i += 2 {
		nameNode := skillsNode.Content[i]
		valueNode := skillsNode.Content[i+1]
		if nameNode == nil || valueNode == nil {
			continue
		}

		name := strings.TrimSpace(nameNode.Value)
		if name == "" || name == "registries" {
			continue
		}

		if name == "github" {
			var legacyGitHub SkillsGithubConfig
			if err := valueNode.Decode(&legacyGitHub); err != nil {
				return err
			}
			if cfg.Tools.Skills.Github.Token.String() == "" && legacyGitHub.Token.String() != "" {
				cfg.Tools.Skills.Github.Token = legacyGitHub.Token
			}
		}

		var legacyRegistry SkillRegistryConfig
		if err := valueNode.Decode(&legacyRegistry); err != nil {
			return err
		}
		legacyRegistry.Name = name
		if legacyRegistry.AuthToken.String() == "" {
			if name == "github" && cfg.Tools.Skills.Github.Token.String() != "" {
				legacyRegistry.AuthToken = cfg.Tools.Skills.Github.Token
			} else {
				continue
			}
		}

		registryCfg, ok := cfg.Tools.Skills.Registries.Get(name)
		if !ok {
			registryCfg = SkillRegistryConfig{
				Name:  name,
				Param: map[string]any{},
			}
		}
		if registryCfg.Param == nil {
			registryCfg.Param = map[string]any{}
		}
		if registryCfg.AuthToken.String() == "" {
			registryCfg.AuthToken = legacyRegistry.AuthToken
		}
		if registryCfg.BaseURL == "" && legacyRegistry.BaseURL != "" {
			registryCfg.BaseURL = legacyRegistry.BaseURL
		}
		for key, value := range legacyRegistry.Param {
			if _, exists := registryCfg.Param[key]; !exists {
				registryCfg.Param[key] = value
			}
		}
		cfg.Tools.Skills.Registries.Set(name, registryCfg)
	}

	return nil
}

// saveSecurityConfig saves the security configuration to security.yml
func saveSecurityConfig(securityPath string, sec *Config) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(sec)
	if err != nil {
		return fmt.Errorf("failed to marshal security config: %w", err)
	}
	return fileutil.WriteFileAtomic(securityPath, buf.Bytes(), 0o600)
}

// SensitiveDataCache caches the strings.Replacer for filtering sensitive data.
// Computed once on first access via sync.Once.
type SensitiveDataCache struct {
	replacer *strings.Replacer
	once     sync.Once
}

// SensitiveDataReplacer returns the strings.Replacer for filtering sensitive data.
// It is computed once on first access via sync.Once.
func (sec *Config) SensitiveDataReplacer() *strings.Replacer {
	sec.initSensitiveCache()
	return sec.sensitiveCache.replacer
}

// initSensitiveCache initializes the sensitive data cache if not already done.
func (sec *Config) initSensitiveCache() {
	if sec.sensitiveCache == nil {
		sec.sensitiveCache = &SensitiveDataCache{}
	}
	sec.sensitiveCache.once.Do(func() {
		values := sec.collectSensitiveValues()
		if len(values) == 0 {
			sec.sensitiveCache.replacer = strings.NewReplacer()
			return
		}

		// Build old/new pairs for strings.Replacer
		var pairs []string
		for _, v := range values {
			if len(v) > 3 {
				pairs = append(pairs, v, "[FILTERED]")
			}
		}
		if len(pairs) == 0 {
			sec.sensitiveCache.replacer = strings.NewReplacer()
			return
		}
		sec.sensitiveCache.replacer = strings.NewReplacer(pairs...)
	})
}

// collectSensitiveValues collects all sensitive strings from SecurityConfig using reflection.
func (sec *Config) collectSensitiveValues() []string {
	var values []string
	collectSensitive(reflect.ValueOf(sec), &values)
	return values
}

// collectSensitive recursively traverses the value and collects SecureString/SecureStrings values.
func collectSensitive(v reflect.Value, values *[]string) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	t := v.Type()

	// Channel: use CollectSensitiveValues() method
	if t == reflect.TypeOf(Channel{}) {
		if method := v.MethodByName("CollectSensitiveValues"); method.IsValid() {
			results := method.Call(nil)
			if len(results) > 0 {
				if vals, ok := results[0].Interface().([]string); ok {
					*values = append(*values, vals...)
				}
			}
		}
		return
	}

	// SecureString: collect via String() method (defined on *SecureString)
	if t == reflect.TypeOf(SecureString{}) {
		// Create a new pointer to make it addressable for method calls
		ptr := reflect.New(t)
		ptr.Elem().Set(v)
		result := ptr.MethodByName("String").Call(nil)
		if len(result) > 0 {
			if s := result[0].String(); s != "" {
				*values = append(*values, s)
			}
		}
		return
	}

	// SecureStrings ([]*SecureString): iterate and collect each element
	if t == reflect.TypeOf(SecureStrings{}) {
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			for elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
				if elem.IsNil() {
					elem = reflect.Value{}
					break
				}
				elem = elem.Elem()
			}
			if elem.IsValid() && elem.Type() == reflect.TypeOf(SecureString{}) {
				result := elem.Addr().MethodByName("String").Call(nil)
				if len(result) > 0 {
					if s := result[0].String(); s != "" {
						*values = append(*values, s)
					}
				}
			}
		}
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			collectSensitive(v.Field(i), values)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			collectSensitive(v.Index(i), values)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			collectSensitive(v.MapIndex(key), values)
		}
	}
}
