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
// Returns an empty SecurityConfig if the file doesn't exist
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

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse security config: %w", err)
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

	// SecureString: collect via String() method (defined on *SecureString)
	if t == reflect.TypeOf(SecureString{}) {
		result := v.Addr().MethodByName("String").Call(nil)
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
