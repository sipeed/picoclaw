package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/sipeed/picoclaw/pkg/credential"
)

func TestLoadSecurityValue(t *testing.T) {
	type valueStruct struct {
		Url     string        `json:"url,omitempty"      yaml:"-"`
		Token   *SecureString `json:"token,omitempty"    yaml:"token,omitempty"    env:"PICO_TOKEN"`
		ApiKeys SecureStrings `json:"api_keys,omitempty" yaml:"api_keys,omitempty" env:"PICO_API_KEYS"`
	}

	type testStruct struct {
		Pico *valueStruct `json:"pico,omitempty" yaml:"pico,omitempty"`
	}

	v1 := &testStruct{
		Pico: &valueStruct{
			Url:     "https://example.com",
			Token:   NewSecureString("token1"),
			ApiKeys: SecureStrings{NewSecureString("api-key1"), NewSecureString("api-key2")},
		},
	}
	bytes, err := yaml.Marshal(v1)
	assert.NoError(t, err)
	jsonBytes, err := json.Marshal(v1)
	assert.NoError(t, err)
	const want = `pico:
    token: token1
    api_keys:
        - api-key1
        - api-key2
`
	const jsonPost = `{"pico":{"url":"https://example.com","token":"token0"}}`
	v0 := &testStruct{}
	err = json.Unmarshal([]byte(jsonPost), v0)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", v0.Pico.Url)
	assert.Equal(t, "token0", v0.Pico.Token.String())

	const jsonWant = `{"pico":{"url":"https://example.com","token":"[NOT_HERE]","api_keys":"[NOT_HERE]"}}`
	assert.Equal(t, want, string(bytes))
	assert.Equal(t, jsonWant, string(jsonBytes))

	v2 := &testStruct{}
	err = json.Unmarshal(jsonBytes, v2)
	assert.NoError(t, err)
	err = yaml.Unmarshal(bytes, v2)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", v2.Pico.Url)
	if v2.Pico.Token != nil {
		assert.Equal(t, "token1", v2.Pico.Token.String())
		assert.Equal(t, "token1", v2.Pico.Token.raw)
	}

	v2.Pico.Token = NewSecureString("token1")
	v2.Pico.Token.raw = "abc"
	err = yaml.Unmarshal(bytes, v2)
	assert.NoError(t, err)
	assert.Equal(t, "token1", v2.Pico.Token.raw)

	os.Setenv("PICO_TOKEN", "token_env")
	err = env.Parse(v2)
	assert.NoError(t, err)
	assert.NotNil(t, v2.Pico.Token)
	assert.Equal(t, "token1", v2.Pico.Token.String())

	v3 := &testStruct{Pico: &valueStruct{}}
	err = env.Parse(v3)
	assert.NoError(t, err)
	if v3.Pico.Token != nil {
		assert.Equal(t, "token_env", v3.Pico.Token.String())
	}

	type toolsStruct struct {
		Pico valueStruct `json:"pico,omitempty" yaml:"pico,omitempty"`
	}

	type testStruct2 struct {
		Tools toolsStruct `json:"tools,omitempty" yaml:",inline"`
	}

	v4 := &testStruct2{
		Tools: toolsStruct{
			Pico: valueStruct{
				Url:     "https://example.com",
				Token:   NewSecureString("token1"),
				ApiKeys: SecureStrings{NewSecureString("api-key1"), NewSecureString("api-key2")},
			},
		},
	}
	bytes, err = yaml.Marshal(v4)
	assert.NoError(t, err)
	assert.Equal(t, want, string(bytes))
	jsonBytes, err = json.Marshal(v4)
	assert.NoError(t, err)
	assert.Equal(
		t,
		`{"tools":{"pico":{"url":"https://example.com","token":"[NOT_HERE]","api_keys":"[NOT_HERE]"}}}`,
		string(jsonBytes),
	)

	v5 := &testStruct2{}
	err = json.Unmarshal(jsonBytes, v5)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", v5.Tools.Pico.Url)
	err = yaml.Unmarshal(bytes, v5)
	assert.NoError(t, err)
	assert.NotNil(t, v5.Tools.Pico.Token)
	assert.Equal(t, "token1", v5.Tools.Pico.Token.raw)

	dir := t.TempDir()
	sshKeyPath := filepath.Join(dir, "picoclaw_ed25519.key")
	if err = os.WriteFile(sshKeyPath, []byte("fake-ssh-key-material\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	const passphrase = "test-passphrase-32bytes-long-ok!"

	t.Setenv(credential.SSHKeyPathEnvVar, sshKeyPath)

	t.Setenv(credential.PassphraseEnvVar, passphrase)

	v5.Tools.Pico.Token.Set("newtoken1")
	v5.Tools.Pico.ApiKeys[0].Set("newapi-key1")
	bytes, err = yaml.Marshal(v5)
	assert.NoError(t, err)
	t.Logf("yaml: %s", string(bytes))

	v6 := &testStruct2{}
	err = yaml.Unmarshal(bytes, v6)
	assert.NoError(t, err)
	assert.NotNil(t, v6.Tools.Pico.Token)
	assert.Equal(t, "newtoken1", v6.Tools.Pico.Token.String())
}

func TestSkillRegistryConfigDecodeParam(t *testing.T) {
	registry := SkillRegistryConfig{
		Name: "github",
		Param: map[string]any{
			"proxy": "http://127.0.0.1:7890",
		},
	}

	var private struct {
		Proxy string `json:"proxy"`
	}
	err := registry.DecodeParam(&private)
	assert.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:7890", private.Proxy)
}

func TestSkillRegistryConfigJSONFlattensParam(t *testing.T) {
	registry := SkillRegistryConfig{
		Name:    "github",
		Enabled: true,
		BaseURL: "https://github.com",
		Param: map[string]any{
			"proxy": "http://127.0.0.1:7890",
		},
	}

	data, err := json.Marshal(registry)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"proxy":"http://127.0.0.1:7890"`)
	assert.NotContains(t, string(data), `"param"`)

	var loaded SkillRegistryConfig
	err = json.Unmarshal(data, &loaded)
	assert.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:7890", loaded.Param["proxy"])
}

func TestSkillRegistryConfigJSONIgnoresShadowSecretFields(t *testing.T) {
	var registry SkillRegistryConfig
	err := json.Unmarshal([]byte(`{
		"enabled": true,
		"base_url": "https://github.com",
		"_auth_token": "shadow-secret",
		"proxy": "http://127.0.0.1:7890"
	}`), &registry)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com", registry.BaseURL)
	assert.Equal(t, "http://127.0.0.1:7890", registry.Param["proxy"])
	_, exists := registry.Param["_auth_token"]
	assert.False(t, exists)

	registry.Param["_auth_token"] = "should-not-round-trip"
	data, err := json.Marshal(registry)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "_auth_token")
	assert.Contains(t, string(data), `"proxy":"http://127.0.0.1:7890"`)

	yamlData, err := yaml.Marshal(registry)
	assert.NoError(t, err)
	assert.NotContains(t, string(yamlData), "_auth_token")
	assert.Contains(t, string(yamlData), "proxy: http://127.0.0.1:7890")
}

func TestSkillRegistryConfigYAMLIgnoresShadowSecretFields(t *testing.T) {
	var registry SkillRegistryConfig
	err := yaml.Unmarshal([]byte(`
enabled: true
base_url: https://github.com
_auth_token: shadow-secret
proxy: http://127.0.0.1:7890
`), &registry)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com", registry.BaseURL)
	assert.Equal(t, "http://127.0.0.1:7890", registry.Param["proxy"])
	_, exists := registry.Param["_auth_token"]
	assert.False(t, exists)
}

func TestSkillsRegistriesConfigMarshalYAMLIncludesRegistryToken(t *testing.T) {
	registries := SkillsRegistriesConfig{
		&SkillRegistryConfig{
			Name:      "github",
			AuthToken: *NewSecureString("registry-auth-token"),
		},
	}

	data, err := yaml.Marshal(registries)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "github:")
	assert.Contains(t, string(data), "auth_token: registry-auth-token")

	loaded := SkillsRegistriesConfig{
		&SkillRegistryConfig{Name: "github"},
	}
	err = yaml.Unmarshal(data, &loaded)
	assert.NoError(t, err)
	github, ok := loaded.Get("github")
	assert.True(t, ok)
	assert.Equal(t, "registry-auth-token", github.AuthToken.String())
}

func TestSkillsRegistriesConfigUnmarshalYAMLBuildsEntriesFromEmptySlice(t *testing.T) {
	var registries SkillsRegistriesConfig
	err := yaml.Unmarshal([]byte(`github:
  enabled: true
  base_url: https://ghe.example.com/git
  proxy: http://127.0.0.1:7890
`), &registries)
	assert.NoError(t, err)

	github, ok := registries.Get("github")
	assert.True(t, ok)
	assert.True(t, github.Enabled)
	assert.Equal(t, "https://ghe.example.com/git", github.BaseURL)
	assert.Equal(t, "http://127.0.0.1:7890", github.Param["proxy"])
}

func TestSkillsRegistriesConfigMarshalJSONPreservesObjectShape(t *testing.T) {
	registries := SkillsRegistriesConfig{
		&SkillRegistryConfig{
			Name:    "github",
			Enabled: true,
			BaseURL: "https://ghe.example.com/git",
			Param: map[string]any{
				"proxy": "http://127.0.0.1:7890",
			},
		},
		&SkillRegistryConfig{
			Name:    "clawhub",
			Enabled: true,
			BaseURL: "https://clawhub.ai",
		},
	}

	data, err := json.Marshal(registries)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"github":{`)
	assert.Contains(t, string(data), `"clawhub":{`)
	assert.NotContains(t, string(data), `[{`)
	assert.NotContains(t, string(data), `"name":"github"`)
	assert.NotContains(t, string(data), `"name":"clawhub"`)

	var decoded map[string]json.RawMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Contains(t, decoded, "github")
	assert.Contains(t, decoded, "clawhub")

	var roundTripped SkillsRegistriesConfig
	err = json.Unmarshal(data, &roundTripped)
	assert.NoError(t, err)

	github, ok := roundTripped.Get("github")
	assert.True(t, ok)
	assert.Equal(t, "https://ghe.example.com/git", github.BaseURL)
	assert.Equal(t, "http://127.0.0.1:7890", github.Param["proxy"])

	clawhub, ok := roundTripped.Get("clawhub")
	assert.True(t, ok)
	assert.Equal(t, "https://clawhub.ai", clawhub.BaseURL)
}

func TestSkillsRegistriesConfigUnmarshalJSONPreservesDefaultRegistries(t *testing.T) {
	registries := DefaultConfig().Tools.Skills.Registries

	err := json.Unmarshal([]byte(`{
		"clawhub": {
			"base_url": "https://clawhub.example.com"
		}
	}`), &registries)
	assert.NoError(t, err)

	clawhub, ok := registries.Get("clawhub")
	assert.True(t, ok)
	assert.True(t, clawhub.Enabled)
	assert.Equal(t, "https://clawhub.example.com", clawhub.BaseURL)

	github, ok := registries.Get("github")
	assert.True(t, ok)
	assert.True(t, github.Enabled)
	assert.Equal(t, "https://github.com", github.BaseURL)
	assert.Empty(t, github.Param)
}

func TestSkillsRegistriesConfigUnmarshalJSONListPreservesDefaultRegistries(t *testing.T) {
	registries := DefaultConfig().Tools.Skills.Registries

	err := json.Unmarshal([]byte(`[
		{
			"name": "clawhub",
			"base_url": "https://clawhub.example.com"
		}
	]`), &registries)
	assert.NoError(t, err)

	clawhub, ok := registries.Get("clawhub")
	assert.True(t, ok)
	assert.True(t, clawhub.Enabled)
	assert.Equal(t, "https://clawhub.example.com", clawhub.BaseURL)

	github, ok := registries.Get("github")
	assert.True(t, ok)
	assert.True(t, github.Enabled)
	assert.Equal(t, "https://github.com", github.BaseURL)
	assert.Empty(t, github.Param)
}

func TestSkillsRegistriesConfigUnmarshalYAMLAppendsNewRegistryToExistingSlice(t *testing.T) {
	registries := DefaultConfig().Tools.Skills.Registries

	err := yaml.Unmarshal([]byte(`custom:
  base_url: https://skills.example.com
  auth_token: custom-token
`), &registries)
	assert.NoError(t, err)

	custom, ok := registries.Get("custom")
	assert.True(t, ok)
	assert.Equal(t, "https://skills.example.com", custom.BaseURL)
	assert.Equal(t, "custom-token", custom.AuthToken.String())

	github, ok := registries.Get("github")
	assert.True(t, ok)
	assert.Equal(t, "https://github.com", github.BaseURL)
}

func TestSkillsRegistriesConfigUnmarshalYAMLOverridesDefaultRegistryFields(t *testing.T) {
	registries := DefaultConfig().Tools.Skills.Registries

	err := yaml.Unmarshal([]byte(`github:
  enabled: false
  base_url: https://ghe.example.com/git
  proxy: http://127.0.0.1:7890
`), &registries)
	assert.NoError(t, err)

	github, ok := registries.Get("github")
	assert.True(t, ok)
	assert.False(t, github.Enabled)
	assert.Equal(t, "https://ghe.example.com/git", github.BaseURL)
	assert.Equal(t, "http://127.0.0.1:7890", github.Param["proxy"])
}

func TestSkillsRegistriesConfigUnmarshalYAMLRetainsDefaultsForOmittedFields(t *testing.T) {
	registries := DefaultConfig().Tools.Skills.Registries

	err := yaml.Unmarshal([]byte(`github:
  auth_token: registry-token
`), &registries)
	assert.NoError(t, err)

	github, ok := registries.Get("github")
	assert.True(t, ok)
	assert.True(t, github.Enabled)
	assert.Equal(t, "https://github.com", github.BaseURL)
	assert.Equal(t, "registry-token", github.AuthToken.String())
	assert.Empty(t, github.Param)
}
