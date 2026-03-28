// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/credential"
	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
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

// SensitiveDataCache caches the compiled regex for filtering sensitive data.
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

const (
	notHere = `"[NOT_HERE]"`
)

// SecureStrings is a slice of SecureString
type SecureStrings []*SecureString

// Values returns the decrypted/resolved values
func (s *SecureStrings) Values() []string {
	if s == nil {
		return nil
	}
	keys := make([]string, len(*s))
	for i, k := range *s {
		keys[i] = k.String()
	}
	return unique(keys)
}

func SimpleSecureStrings(val ...string) SecureStrings {
	val = unique(val)
	vv := make(SecureStrings, len(val))
	for i, s := range val {
		vv[i] = NewSecureString(s)
	}
	return vv
}

// unique returns a new slice with duplicate elements removed.
func unique[T comparable](input []T) []T {
	m := make(map[T]struct{})
	var result []T
	for _, v := range input {
		if _, ok := m[v]; !ok {
			m[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func (s SecureStrings) MarshalJSON() ([]byte, error) {
	return []byte(notHere), nil
}

func (s *SecureStrings) UnmarshalJSON(value []byte) error {
	if string(value) == notHere {
		return nil
	}
	var v []*SecureString
	err := json.Unmarshal(value, &v)
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// SecureString the string value that can be decrypted or resolved
//
//nolint:recvcheck
type SecureString struct {
	resolved string // Decrypted/resolved value returned by String()
	raw      string // Persisted raw value (enc://, file://, or plaintext)
}

func callerFromYaml() bool {
	_, file, _, ok := runtime.Caller(2)
	if ok {
		d := filepath.Dir(file)
		// check the caller is from yaml.v
		if !strings.Contains(d, "yaml.v") {
			return true
		}
	}
	return false
}

// IsZero returns true if the SecureString is empty
// if caller not yaml, just return true for prevent marshal this field
func (s SecureString) IsZero() bool {
	if callerFromYaml() {
		return true
	}
	return s.resolved == ""
}

func NewSecureString(value string) *SecureString {
	s := &SecureString{}
	if err := s.fromRaw(value); err != nil {
		logger.Warn(fmt.Sprintf("NewSecureString.fromRaw error: %s", err))
	}
	return s
}

func (s *SecureString) String() string {
	if s == nil {
		return ""
	}
	return s.resolved
}

func (s *SecureString) Set(value string) *SecureString {
	s.resolved = value
	s.raw = ""
	return s
}

func (s SecureString) MarshalJSON() ([]byte, error) {
	return []byte(notHere), nil
}

func (s *SecureString) UnmarshalJSON(value []byte) error {
	if string(value) == notHere {
		return nil
	}
	var v string
	if err := json.Unmarshal(value, &v); err != nil {
		return err
	}
	return s.fromRaw(v)
}

func (s SecureString) MarshalYAML() (any, error) {
	// Preserve raw value if it is already a reference (enc:// or file://)
	if strings.HasPrefix(s.raw, credential.EncScheme) || strings.HasPrefix(s.raw, credential.FileScheme) {
		return s.raw, nil
	}
	// If resolved is a reference format (e.g. set via Set), copy back to raw
	if strings.HasPrefix(s.resolved, credential.EncScheme) || strings.HasPrefix(s.resolved, credential.FileScheme) {
		s.raw = s.resolved
		return s.raw, nil
	}
	// Try to encrypt the resolved value
	if passphrase := credential.PassphraseProvider(); passphrase != "" {
		encrypted, err := credential.Encrypt(passphrase, "", s.resolved)
		if err != nil {
			logger.Errorf("Encrypt error: %v", err)
			return nil, err
		}
		s.raw = encrypted
	} else {
		s.raw = s.resolved
	}
	return s.raw, nil
}

func (s *SecureString) UnmarshalYAML(value *yaml.Node) error {
	return s.fromRaw(value.Value)
}

func (s *SecureString) fromRaw(v string) error {
	s.raw = v
	vv, err := resolveKey(v)
	if err != nil {
		return err
	}
	s.resolved = vv
	return nil
}

var (
	secResolverMu sync.RWMutex
	secResolver   *credential.Resolver
)

func updateResolver(path string) {
	secResolverMu.Lock()
	defer secResolverMu.Unlock()
	secResolver = credential.NewResolver(path)
}

func resolveKey(v string) (string, error) {
	secResolverMu.RLock()
	resolver := secResolver
	secResolverMu.RUnlock()
	if resolver == nil {
		resolver = credential.NewResolver("")
	}
	if strings.HasPrefix(v, "enc://") || strings.HasPrefix(v, "file://") {
		decrypted, err := resolver.Resolve(v)
		if err != nil {
			logger.Errorf("Resolve error: %v", err)
			return "", err
		}
		return decrypted, nil
	}
	return v, nil
}

func (s *SecureString) UnmarshalText(text []byte) error {
	v := string(text)
	return s.fromRaw(v)
}

type SecureModelList []*ModelConfig

func (v *SecureModelList) UnmarshalYAML(value *yaml.Node) error {
	mm := make(map[string]*ModelConfig)
	if err := value.Decode(&mm); err != nil {
		logger.Errorf("Decode error: %v", err)
		return err
	}
	nameList := toNameIndex(*v)
	for i, m := range *v {
		sec := mm[nameList[i]]
		if sec == nil {
			sec = mm[m.ModelName]
		}
		if sec != nil {
			m.APIKeys = sec.APIKeys
		}
	}
	return nil
}

func (v SecureModelList) MarshalYAML() (any, error) {
	type onlySecureData struct {
		APIKeys SecureStrings `yaml:"api_keys,omitempty"`
	}
	mm := make(map[string]onlySecureData)
	nameList := toNameIndex(v)
	for i, m := range v {
		mm[nameList[i]] = onlySecureData{
			APIKeys: m.APIKeys,
		}
	}

	return mm, nil
}
