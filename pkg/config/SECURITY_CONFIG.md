# Security Configuration Refactoring

## Overview

This refactoring introduces a `security.yml` file to store all sensitive data (API keys, tokens, secrets, passwords) separately from the main configuration. This improves security by:

1. **Separation of concerns**: Configuration settings and secrets are in separate files
2. **Easier sharing**: The main config can be shared without exposing sensitive data
3. **Better version control**: `security.yml` can be added to `.gitignore`
4. **Flexible deployment**: Different environments can use different security files

## File Structure

```
~/.picoclaw/
├── config.json          # Main configuration (safe to share)
└── security.yml         # Security data (never share)
```

## Usage

### Basic Configuration

In your `config.json`, use `ref:` references to point to values in `security.yml`:

```json
{
  "version": 1,
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1",
      "api_key": "ref:model_list.gpt-5.4.api_key"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "ref:channels.telegram.token"
    }
  }
}
```

### Security Configuration

In your `security.yml`, store the actual values:

```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-actual-api-key-1"
      - "sk-your-actual-api-key-2"  # Optional: Multiple keys for failover
  claude-sonnet-4.6:
    api_keys:
      - "sk-your-actual-anthropic-key"  # Single key in array format

channels:
  telegram:
    token: "your-telegram-bot-token"

web:
  brave:
    api_keys:
      - "BSAyour-brave-api-key-1"
      - "BSAyour-brave-api-key-2"  # Optional: Multiple keys for failover
  tavily:
    api_keys:
      - "tvly-your-tavily-api-key"  # Single key in array format
  glm_search:
    api_key: "your-glm-search-api-key"  # GLMSearch uses single key format
```

## Reference Format

### Model API Keys

Format: `ref:model_list.<model_name>.api_key`

Example: `ref:model_list.gpt-5.4.api_key`

### Channel Tokens/Secrets

Format: `ref:channels.<channel_name>.<field>`

Examples:
- `ref:channels.telegram.token`
- `ref:channels.feishu.app_secret`
- `ref:channels.feishu.encrypt_key`
- `ref:channels.feishu.verification_token`
- `ref:channels.discord.token`
- `ref:channels.qq.app_secret`
- `ref:channels.dingtalk.client_secret`
- `ref:channels.slack.bot_token`
- `ref:channels.slack.app_token`
- `ref:channels.matrix.access_token`
- `ref:channels.line.channel_secret`
- `ref:channels.line.channel_access_token`
- `ref:channels.onebot.access_token`
- `ref:channels.wecom.token`
- `ref:channels.wecom.encoding_aes_key`
- `ref:channels.wecom_app.corp_secret`
- `ref:channels.wecom_app.token`
- `ref:channels.wecom_app.encoding_aes_key`
- `ref:channels.wecom_aibot.token`
- `ref:channels.wecom_aibot.encoding_aes_key`
- `ref:channels.pico.token`
- `ref:channels.irc.password`
- `ref:channels.irc.nickserv_password`
- `ref:channels.irc.sasl_password`

### Web Tool API Keys

Format: `ref:web.<provider>.<field>`

Examples:
- `ref:web.brave.api_key`
- `ref:web.tavily.api_key`
- `ref:web.perplexity.api_key`
- `ref:web.glm_search.api_key`

### Skills Registry Tokens

Format: `ref:skills.<registry>.<field>`

Examples:
- `ref:skills.github.token`
- `ref:skills.clawhub.auth_token`

## Backward Compatibility

The refactoring maintains full backward compatibility:

1. **Direct values**: You can still use direct values in `config.json` (not recommended for production)
2. **Mixed usage**: You can mix `ref:` references and direct values
3. **Optional security file**: If `security.yml` doesn't exist, all references will fail (but direct values still work)

### API Key Formats in security.yml

**Models (gpt-5.4, claude-sonnet-4.6, etc.):**
- Must use `api_keys` (array) format
- Both single and multiple keys use array format

**Web Tools (Brave, Tavily, Perplexity):**
- Must use `api_keys` (array) format
- Both single and multiple keys use array format

**Web Tools (GLMSearch):**
- Must use `api_key` (single string) format
- Does NOT support array format

**Channels (Telegram, Discord, etc.):**
- Use single field names (e.g., `token`, `app_secret`)
- Each channel uses its specific field names

### Single Key (Models)

Use array format with one element:
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-key"
```

In `config.json`:
```json
{
  "api_key": "ref:model_list.gpt-5.4.api_key"
}
```

### Single Key (GLMSearch)

Use single string format:
```yaml
web:
  glm_search:
    api_key: "your-glm-key"
```

In `config.json`:
```json
{
  "api_key": "ref:web.glm_search.api_key"
}
```

## Migration Guide

### Step 1: Create security.yml

Copy the example template:
```bash
cp security.example.yml ~/.picoclaw/security.yml
```

### Step 2: Fill in your actual values

Edit `~/.picoclaw/security.yml` and replace placeholder values with your actual API keys and tokens.

### Step 3: Update config.json

Replace sensitive values in `~/.picoclaw/config.json` with `ref:` references:

**Before:**
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "sk-your-actual-api-key-here"
    }
  ]
}
```

**After:**
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "ref:model_list.gpt-5.4.api_key"
    }
  ]
}
```

### Step 4: Verify

Restart PicoClaw and verify it loads correctly:
```bash
picoclaw --version
```

## Security Best Practices

1. **Never commit `security.yml`** to version control
2. **Set file permissions**: `chmod 600 ~/.picoclaw/security.yml`
3. **Use different keys** for different environments (dev, staging, production)
4. **Rotate keys regularly** and update `security.yml`
5. **Backup securely**: Encrypt backups containing `security.yml`

## API

### LoadSecurityConfig

```go
func LoadSecurityConfig(securityPath string) (*SecurityConfig, error)
```

Loads the security configuration from `security.yml`. Returns an empty `SecurityConfig` if the file doesn't exist.

### SaveSecurityConfig

```go
func SaveSecurityConfig(securityPath string, sec *SecurityConfig) error
```

Saves the security configuration to `security.yml` with `0o600` permissions.

### ResolveReference

```go
func (sec *SecurityConfig) ResolveReference(ref string) (string, error)
```

Resolves a reference string (e.g., `"ref:model_list.test.api_key"`) and returns the actual value.

### SecurityPath

```go
func SecurityPath(configPath string) string
```

Returns the path to `security.yml` relative to the config file.

## Example: Complete Configuration

### config.json
```json
{
  "version": 1,
  "agents": {
    "defaults": {
      "workspace": "~/picoclaw-workspace",
      "model_name": "gpt-5.4"
    }
  },
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1",
      "api_key": "ref:model_list.gpt-5.4.api_key"
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_base": "https://api.anthropic.com/v1",
      "api_key": "ref:model_list.claude-sonnet-4.6.api_key"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "ref:channels.telegram.token"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "ref:web.brave.api_key"
      }
    }
  }
}
```

### security.yml
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-actual-openai-key-1"
      - "sk-proj-actual-openai-key-2"
  claude-sonnet-4.6:
    api_keys:
      - "sk-ant-actual-anthropic-key"  # Single key in array format

channels:
  telegram:
    token: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"

web:
  brave:
    api_keys:
      - "BSAactualbravekey-1"
      - "BSAactualbravekey-2"
  tavily:
    api_keys:
      - "tvly-your-tavily-key"  # Single key in array format
  glm_search:
    api_key: "your-glm-key"  # GLMSearch uses single key format
```

## Testing

The refactoring includes comprehensive tests:

```bash
go test ./pkg/config -run TestSecurityConfig
```

## Troubleshooting

### Error: "model security entry not found"

- Ensure the model name in your reference matches exactly in `security.yml`
- Check that the `model_list` section exists in `security.yml`
- For models with indexed names (e.g., "gpt-5.4:0"), ensure the exact name is used or check the base name without index

### Error: "failed to load security config"

- Verify `security.yml` exists in the same directory as `config.json`
- Check the YAML syntax is valid (use a YAML validator)
- Ensure file permissions allow reading

### Error: "unknown reference path"

- Verify the reference format is correct
- Check the path structure matches the examples above
- Ensure all required sections exist in `security.yml`

## Advanced Features

### Multiple API Keys (Load Balancing & Failover)

Both models and web tools support multiple API keys for improved reliability:

**Benefits:**
- **Load balancing**: Requests are distributed across multiple keys
- **Failover**: Automatic switching to another key if one fails
- **Rate limit management**: Distribute usage across multiple keys
- **High availability**: Reduce downtime during API provider issues

#### Example: Model with Multiple Keys

**security.yml:**
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-key-1"
      - "sk-proj-key-2"
      - "sk-proj-key-3"
```

**config.json:**
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "ref:model_list.gpt-5.4.api_key"
    }
  ]
}
```

#### Example: Web Tool with Multiple Keys

**security.yml:**
```yaml
web:
  brave:
    api_keys:
      - "BSA-key-1"
      - "BSA-key-2"
  tavily:
    api_keys:
      - "tvly-your-key"  # Single key in array format
  glm_search:
    api_key: "your-glm-key"  # GLMSearch uses single key format
```

**config.json:**
```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "ref:web.brave.api_key"
      },
      "tavily": {
        "enabled": true,
        "api_key": "ref:web.tavily.api_key"
      }
    }
  }
}
```

#### Supported Formats

**Models - Single key:**
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-key"  # Array with one element
```

**Models - Multiple keys:**
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-key-1"
      - "sk-your-key-2"
      - "sk-your-key-3"
```

**Web Tools (Brave/Tavily/Perplexity) - Single key:**
```yaml
web:
  brave:
    api_keys:
      - "BSA-your-key"  # Array with one element
```

**Web Tools (Brave/Tavily/Perplexity) - Multiple keys:**
```yaml
web:
  brave:
    api_keys:
      - "BSA-key-1"
      - "BSA-key-2"
```

**Web Tool (GLMSearch) - Single key only:**
```yaml
web:
  glm_search:
    api_key: "your-glm-key"  # Single string (NOT array)
```

All formats work identically in `config.json` - you always use the same reference format:
```json
{
  "api_key": "ref:model_list.gpt-5.4.api_key"
}
```

### Model Indexing for Load Balancing

When you have multiple models with the same base name but different API keys, you can use indexed names:

**security.yml:**
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-key-1"
      - "sk-proj-key-2"
```

The system will automatically expand this into multiple model entries with fallback support.

### Environment Variables

You can override any security value using environment variables:

**For models:**
```bash
export PICOCLAW_MODEL_LIST_GPT-5.4_API_KEY="sk-from-env"
```

**For channels:**
```bash
export PICOCLAW_CHANNELS_TELEGRAM_TOKEN="token-from-env"
```

**For web tools:**
```bash
export PICOCLAW_WEB_BRAVE_API_KEY="key-from-env"
```

Environment variables follow this pattern: `PICOCLAW_<SECTION>_<KEY1>_<KEY2>_<FIELD>` with dots replaced by underscores and converted to uppercase.

### Multiple API Keys Not Working

- Ensure you're using `api_keys` (plural) in `security.yml` for models and web tools (except GLMSearch)
- Check that the array format is correct in YAML (proper indentation)
- Remember: Models, Brave, Tavily, Perplexity MUST use `api_keys` (array format)
- GLMSearch MUST use `api_key` (single string format)
- The reference in `config.json` is the same regardless of single or multiple keys

### Load Balancing/Failover Issues

- Verify all API keys in the `api_keys` array are valid
- Check that all keys have the same rate limits and permissions
- Monitor logs to see which keys are being used and failing
