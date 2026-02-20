# Provider Architecture Refactoring Design

> Issue: #283
> Discussion: #122
> Branch: feat/refactor-provider-by-protocol

## 1. Current Problems

### 1.1 Configuration Structure Issues

**Current State**: Each Provider requires a predefined field in `ProvidersConfig`

```go
type ProvidersConfig struct {
    Anthropic     ProviderConfig `json:"anthropic"`
    OpenAI        ProviderConfig `json:"openai"`
    DeepSeek      ProviderConfig `json:"deepseek"`
    Qwen          ProviderConfig `json:"qwen"`
    Cerebras      ProviderConfig `json:"cerebras"`
    VolcEngine    ProviderConfig `json:"volcengine"`
    // ... every new provider requires changes here
}
```

**Problems**:
- Adding a new Provider requires modifying Go code (struct definition)
- `CreateProvider` function in `http_provider.go` has 200+ lines of switch-case
- Most Providers are OpenAI-compatible, but code is duplicated

### 1.2 Code Bloat Trend

Recent PRs demonstrate this issue:

| PR | Provider | Code Changes |
|----|----------|--------------|
| #365 | Qwen | +17 lines to http_provider.go |
| #333 | Cerebras | +17 lines to http_provider.go |
| #368 | Volcengine | +18 lines to http_provider.go |

Each OpenAI-compatible Provider requires:
1. Modify `config.go` to add configuration field
2. Modify `http_provider.go` to add switch case
3. Update documentation

### 1.3 Agent-Provider Coupling

```json
{
  "agents": {
    "defaults": {
      "provider": "deepseek",  // need to know provider name
      "model": "deepseek-chat"
    }
  }
}
```

Problem: Agent needs to know both `provider` and `model`, adding complexity.

---

## 2. New Approach: model_list

### 2.1 Core Principles

Inspired by [LiteLLM](https://docs.litellm.ai/docs/proxy/configs) design:

1. **Model-centric**: Users care about models, not providers
2. **Protocol prefix**: Use `protocol/model_name` format, e.g., `openai/gpt-5.2`, `anthropic/claude-sonnet-4.6`
3. **Configuration-driven**: Adding new Providers only requires config changes, no code changes

### 2.2 New Configuration Structure

```json
{
  "model_list": [
    {
      "model_name": "deepseek-chat",
      "model": "openai/deepseek-chat",
      "api_base": "https://api.deepseek.com/v1",
      "api_key": "sk-xxx"
    },
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_key": "sk-xxx"
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-xxx"
    },
    {
      "model_name": "gemini-3-flash",
      "model": "antigravity/gemini-3-flash",
      "auth_method": "oauth"
    },
    {
      "model_name": "my-company-llm",
      "model": "openai/company-model-v1",
      "api_base": "https://llm.company.com/v1",
      "api_key": "xxx"
    }
  ],

  "agents": {
    "defaults": {
      "model": "deepseek-chat",
      "max_tokens": 8192,
      "temperature": 0.7
    }
  }
}
```

### 2.3 Go Struct Definition

```go
type Config struct {
    ModelList []ModelConfig `json:"model_list"`  // new
    Providers ProvidersConfig `json:"providers"`  // old, deprecated

    Agents   AgentsConfig   `json:"agents"`
    Channels ChannelsConfig `json:"channels"`
    // ...
}

type ModelConfig struct {
    // Required
    ModelName string `json:"model_name"`  // user-facing name (alias)
    Model     string `json:"model"`       // protocol/model, e.g., openai/gpt-5.2

    // Common config
    APIBase   string `json:"api_base,omitempty"`
    APIKey    string `json:"api_key,omitempty"`
    Proxy     string `json:"proxy,omitempty"`

    // Special provider config
    AuthMethod  string `json:"auth_method,omitempty"`   // oauth, token
    ConnectMode string `json:"connect_mode,omitempty"`  // stdio, grpc

    // Optional optimizations
    RPM            int    `json:"rpm,omitempty"`              // rate limit
    MaxTokensField string `json:"max_tokens_field,omitempty"` // max_tokens or max_completion_tokens
}
```

### 2.4 Protocol Recognition

Identify protocol via prefix in `model` field:

| Prefix | Protocol | Description |
|--------|----------|-------------|
| `openai/` | OpenAI-compatible | Most common, includes DeepSeek, Qwen, Groq, etc. |
| `anthropic/` | Anthropic | Claude series specific |
| `antigravity/` | Antigravity | Google Cloud Code Assist |
| `gemini/` | Gemini | Google Gemini native API (if needed) |

---

## 3. Design Rationale

### 3.1 Problems Solved

| Problem | Old Approach | New Approach |
|---------|--------------|--------------|
| Add OpenAI-compatible Provider | Change 3 code locations | Add one config entry |
| Agent specifies model | Need provider + model | Only need model |
| Code duplication | Each Provider duplicates logic | Share protocol implementation |
| Multi-Agent support | Complex | Naturally compatible |

### 3.2 Multi-Agent Compatibility

```json
{
  "model_list": [...],

  "agents": {
    "defaults": {
      "model": "deepseek-chat"
    },
    "coder": {
      "model": "gpt-5.2",
      "system_prompt": "You are a coding assistant..."
    },
    "translator": {
      "model": "claude-sonnet-4.6"
    }
  }
}
```

Each Agent only needs to specify `model` (corresponds to `model_name` in `model_list`).

### 3.3 Industry Comparison

**LiteLLM** (most mature open-source LLM Proxy) uses similar design:

```yaml
model_list:
  - model_name: gpt-4o
    litellm_params:
      model: openai/gpt-5.2
      api_key: xxx
  - model_name: my-custom
    litellm_params:
      model: openai/custom-model
      api_base: https://my-api.com/v1
```

---

## 4. Migration Plan

### 4.1 Phase 1: Compatibility Period (v1.x)

Support both `providers` and `model_list`:

```go
func (c *Config) GetModelConfig(modelName string) (*ModelConfig, error) {
    // Prefer new config
    if len(c.ModelList) > 0 {
        return c.findModelByName(modelName)
    }

    // Backward compatibility with old config
    if !c.Providers.IsEmpty() {
        logger.Warn("'providers' config is deprecated, please migrate to 'model_list'")
        return c.convertFromProviders(modelName)
    }

    return nil, fmt.Errorf("model %s not found", modelName)
}
```

### 4.2 Phase 2: Warning Period (late v1.x)

- Print more prominent warnings at startup
- Provide automatic migration script
- Mark `providers` as deprecated in documentation

### 4.3 Phase 3: Removal Period (v2.0)

- Completely remove `providers` support
- Remove `agents.defaults.provider` field
- Only support `model_list`

### 4.4 Configuration Migration Example

**Old Config**:
```json
{
  "providers": {
    "deepseek": {
      "api_key": "sk-xxx",
      "api_base": "https://api.deepseek.com/v1"
    }
  },
  "agents": {
    "defaults": {
      "provider": "deepseek",
      "model": "deepseek-chat"
    }
  }
}
```

**New Config**:
```json
{
  "model_list": [
    {
      "model_name": "deepseek-chat",
      "model": "openai/deepseek-chat",
      "api_base": "https://api.deepseek.com/v1",
      "api_key": "sk-xxx"
    }
  ],
  "agents": {
    "defaults": {
      "model": "deepseek-chat"
    }
  }
}
```

---

## 5. Implementation Checklist

### 5.1 Configuration Layer

- [ ] Add `ModelConfig` struct
- [ ] Add `Config.ModelList` field
- [ ] Implement `GetModelConfig(modelName)` method
- [ ] Implement old config compatibility conversion
- [ ] Add `model_name` uniqueness validation

### 5.2 Provider Layer

- [ ] Create `pkg/providers/factory/` directory
- [ ] Implement `CreateProviderFromModelConfig()`
- [ ] Refactor `http_provider.go` to `openai/provider.go`
- [ ] Maintain backward compatibility for old `CreateProvider()`

### 5.3 Testing

- [ ] New config unit tests
- [ ] Old config compatibility tests
- [ ] Integration tests

### 5.4 Documentation

- [ ] Update README
- [ ] Update config.example.json
- [ ] Write migration guide

---

## 6. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing configs | Compatibility period keeps old config working |
| User migration cost | Provide automatic migration script |
| Special Provider incompatibility | Keep `auth_method` and other extension fields |

---

## 7. References

- [LiteLLM Config Documentation](https://docs.litellm.ai/docs/proxy/configs)
- [One-API GitHub](https://github.com/songquanpeng/one-api)
- Discussion #122: Refactor Provider Architecture
