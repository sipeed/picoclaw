# Sensitive Data Filtering

PicoClaw can filter sensitive values (API keys, tokens, secrets, passwords) from tool call results before they are sent to the LLM. This prevents the LLM from seeing its own credentials, which could otherwise leak through tool output or cause confusing behavior.

---

## Overview

When the LLM uses a tool that returns its own credentials (e.g., a tool that echoes the API key being used), those values are automatically replaced with `[FILTERED]` in the message sent to the LLM.

Sensitive values are collected from [`.security.yml`](./credential_encryption.md) — the centralized storage for all sensitive configuration (API keys, tokens, secrets stored alongside `config.json`). This includes:

- Model API keys
- Channel tokens (Telegram, Discord, Slack, Matrix, etc.)
- Web tool API keys (Brave, Tavily, Perplexity, etc.)
- Skills tokens (GitHub, ClawHub)

---

## Configuration

Sensitive data filtering is configured in the `tools` section of `config.json`:

| Config | Type | Default | Description |
|--------|------|---------|-------------|
| `filter_sensitive_data` | bool | `true` | Enable/disable filtering. When `false`, no filtering is performed. |
| `filter_min_length` | int | `8` | Minimum content length to trigger filtering. Short content is skipped for performance. |

```json
{
  "tools": {
    "filter_sensitive_data": true,
    "filter_min_length": 8
  }
}
```

### Environment Variable

| Variable | Description |
|----------|-------------|
| `PICOCLAW_TOOLS_FILTER_SENSITIVE_DATA` | Set to `true` or `false` to override the config value |

---

## How It Works

1. **On startup**: All sensitive values are collected from `.security.yml` using reflection and compiled into a `strings.Replacer` (O(n+m) performance, computed once).

2. **Per tool result**: Before sending any tool result content to the LLM:
   - If `filter_sensitive_data` is `false`, content is passed through unchanged
   - If content length < `filter_min_length`, content is passed through unchanged (fast path)
   - Otherwise, all sensitive values are replaced with `[FILTERED]`

3. **Replacement**: Uses `strings.Replacer` for efficient O(n+m) string substitution, where n = content length and m = total sensitive value length.

---

## Example

Given the following `.security.yml`:

```yaml
model_list:
  my-model:
    api_keys:
      - sk-secret-key-12345

channels:
  telegram:
    token: "123456:ABC-DEF"
```

And a tool result containing:

```
The model is using API key sk-secret-key-12345 and Telegram bot 123456:ABC-DEF
```

The LLM will receive:

```
The model is using API key [FILTERED] and Telegram bot [FILTERED]
```

---

## Performance

- **Fast path**: Content shorter than `filter_min_length` (default 8) is returned unchanged without any string scanning
- **Efficient replacement**: Uses `strings.Replacer` with O(n+m) complexity instead of regex
- **Lazy initialization**: The replacement map is built once on first access via `sync.Once`

---

## Security Considerations

- **Credential exposure prevention**: Without filtering, tools that echo credentials could cause the LLM to see its own API keys, potentially leading to confusion or credential leakage in logs
- **Defense in depth**: Filtering complements (but does not replace) credential encryption — both features should be used together
- **No false positives**: Only values explicitly stored in `.security.yml` are filtered; the LLM's general knowledge is unaffected

---

## Related

- [Credential Encryption](./credential_encryption.md) — encrypting API keys in config
- [Tools Configuration](../reference/tools_configuration.md)
