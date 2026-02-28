# Migration Guide: From `providers` to `model_list`

This guide explains how to migrate from the legacy `providers` configuration to the new `model_list` format.

## Why Migrate?

The new `model_list` configuration offers several advantages:

- **Zero-code provider addition**: Add OpenAI-compatible providers with configuration only
- **Load balancing**: Configure multiple endpoints for the same model
- **Protocol-based routing**: Use prefixes like `openai/`, `anthropic/`, etc.
- **Cleaner configuration**: Model-centric instead of vendor-centric

## Timeline

| Version | Status |
|---------|--------|
| v1.x | `model_list` introduced, `providers` deprecated but functional |
| v1.x+1 | Prominent deprecation warnings, migration tool available |
| v2.0 | `providers` configuration removed |

## Before and After

### Before: Legacy `providers` Configuration

```json
{
  "providers": {
    "openai": {
      "api_key": "sk-your-openai-key",
      "api_base": "https://api.openai.com/v1"
    },
    "anthropic": {
      "api_key": "sk-ant-your-key"
    },
    "deepseek": {
      "api_key": "sk-your-deepseek-key"
    }
  },
  "agents": {
    "defaults": {
      "provider": "openai",
      "model": "gpt-5.2"
    }
  }
}
```

### After: New `model_list` Configuration

```json
{
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-5.2",
      "api_key": "sk-your-openai-key",
      "api_base": "https://api.openai.com/v1"
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-your-key"
    },
    {
      "model_name": "deepseek",
      "model": "deepseek/deepseek-chat",
      "api_key": "sk-your-deepseek-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "gpt4"
    }
  }
}
```

## Protocol Prefixes

The `model` field uses a protocol prefix format: `[protocol/]model-identifier`

| Prefix | Description | Example |
|--------|-------------|---------|
| `openai/` | OpenAI API (default) | `openai/gpt-5.2` |
| `anthropic/` | Anthropic API | `anthropic/claude-opus-4` |
| `antigravity/` | Google via Antigravity OAuth | `antigravity/gemini-2.0-flash` |
| `gemini/` | Google Gemini API | `gemini/gemini-2.0-flash-exp` |
| `claude-cli/` | Claude CLI (local) | `claude-cli/claude-sonnet-4.6` |
| `codex-cli/` | Codex CLI (local) | `codex-cli/codex-4` |
| `github-copilot/` | GitHub Copilot | `github-copilot/gpt-4o` |
| `openrouter/` | OpenRouter | `openrouter/anthropic/claude-sonnet-4.6` |
| `groq/` | Groq API | `groq/llama-3.1-70b` |
| `deepseek/` | DeepSeek API | `deepseek/deepseek-chat` |
| `cerebras/` | Cerebras API | `cerebras/llama-3.3-70b` |
| `qwen/` | Alibaba Qwen | `qwen/qwen-max` |
| `zhipu/` | Zhipu AI | `zhipu/glm-4` |
| `nvidia/` | NVIDIA NIM | `nvidia/llama-3.1-nemotron-70b` |
| `ollama/` | Ollama (local) | `ollama/llama3` |
| `vllm/` | vLLM (local) | `vllm/my-model` |
| `moonshot/` | Moonshot AI | `moonshot/moonshot-v1-8k` |
| `shengsuanyun/` | ShengSuanYun | `shengsuanyun/deepseek-v3` |
| `volcengine/` | Volcengine | `volcengine/doubao-pro-32k` |

**Note**: If no prefix is specified, `openai/` is used as the default.

## ModelConfig Fields

| Field | Required | Description |
|-------|----------|-------------|
| `model_name` | Yes | User-facing alias for the model |
| `model` | Yes | Protocol and model identifier (e.g., `openai/gpt-5.2`) |
| `api_base` | No | API endpoint URL |
| `api_key` | No* | API authentication key |
| `proxy` | No | HTTP proxy URL |
| `auth_method` | No | Authentication method: `oauth`, `token` |
| `connect_mode` | No | Connection mode for CLI providers: `stdio`, `grpc` |
| `rpm` | No | Requests per minute limit |
| `max_tokens_field` | No | Field name for max tokens |
| `request_timeout` | No | HTTP request timeout in seconds; `<=0` uses default `120s` |

*`api_key` is required for HTTP-based protocols unless `api_base` points to a local server.

## Load Balancing

Configure multiple endpoints for the same model to distribute load:

```json
{
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-5.2",
      "api_key": "sk-key1",
      "api_base": "https://api1.example.com/v1"
    },
    {
      "model_name": "gpt4",
      "model": "openai/gpt-5.2",
      "api_key": "sk-key2",
      "api_base": "https://api2.example.com/v1"
    },
    {
      "model_name": "gpt4",
      "model": "openai/gpt-5.2",
      "api_key": "sk-key3",
      "api_base": "https://api3.example.com/v1"
    }
  ]
}
```

When you request model `gpt4`, requests will be distributed across all three endpoints using round-robin selection.

## Adding a New OpenAI-Compatible Provider

With `model_list`, adding a new provider requires zero code changes:

```json
{
  "model_list": [
    {
      "model_name": "my-custom-llm",
      "model": "openai/my-model-v1",
      "api_key": "your-api-key",
      "api_base": "https://api.your-provider.com/v1"
    }
  ]
}
```

Just specify `openai/` as the protocol (or omit it for the default), and provide your provider's API base URL.

## Backward Compatibility

During the migration period, your existing `providers` configuration will continue to work:

1. If `model_list` is empty and `providers` has data, the system auto-converts internally
2. A deprecation warning is logged: `"providers config is deprecated, please migrate to model_list"`
3. All existing functionality remains unchanged

## Migration Checklist

- [ ] Identify all providers you're currently using
- [ ] Create `model_list` entries for each provider
- [ ] Use appropriate protocol prefixes
- [ ] Update `agents.defaults.model` to reference the new `model_name`
- [ ] Test that all models work correctly
- [ ] Remove or comment out the old `providers` section

## Troubleshooting

### Model not found error

```
model "xxx" not found in model_list or providers
```

**Solution**: Ensure the `model_name` in `model_list` matches the value in `agents.defaults.model`.

### Unknown protocol error

```
unknown protocol "xxx" in model "xxx/model-name"
```

**Solution**: Use a supported protocol prefix. See the [Protocol Prefixes](#protocol-prefixes) table above.

### Missing API key error

```
api_key or api_base is required for HTTP-based protocol "xxx"
```

**Solution**: Provide `api_key` and/or `api_base` for HTTP-based providers.

## Need Help?

- [GitHub Issues](https://github.com/sipeed/picoclaw/issues)
- [Discussion #122](https://github.com/sipeed/picoclaw/discussions/122): Original proposal
