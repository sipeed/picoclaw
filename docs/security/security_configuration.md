# Security Configuration

## Overview

PicoClaw supports separating sensitive data (API keys, tokens, secrets, passwords) from the main configuration by storing them in a `.security.yml` file. This improves security by:

1. **Separation of concerns**: Configuration settings and secrets are in separate files
2. **Easier sharing**: The main config can be shared without exposing sensitive data
3. **Better version control**: `.security.yml` should be added to `.gitignore`
4. **Flexible deployment**: Different environments can use different security files

## File Structure

```
~/.picoclaw/
├── config.json          # Main configuration (safe to share)
└── .security.yml         # Security data (never share)
```

## How It Works

The security configuration works through **direct field mapping**, NOT through `ref:` string references. The system automatically loads values from `.security.yml` and applies them to the corresponding fields in `config.json`.

### Key Points:

- Values in `.security.yml` are automatically mapped to corresponding fields in the config
- The mapping is based on field names and structure, not on reference strings
- If a value exists in `.security.yml`, it **overrides** the value in `config.json`
- You can omit sensitive fields from `config.json` entirely (recommended)

## Security Configuration Structure

### Complete Example: .security.yml

```yaml
# Model API Keys
# All models MUST use `api_keys` (plural) array format
# Even a single key must be provided as an array with one element
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-your-actual-openai-key-1"
      - "sk-proj-your-actual-openai-key-2"  # Optional: Multiple keys for failover
  claude-sonnet-4.6:
    api_keys:
      - "sk-ant-your-actual-anthropic-key"  # Single key in array format

# Channel Tokens
channels:
  telegram:
    token: "your-telegram-bot-token"
  feishu:
    app_secret: "your-feishu-app-secret"
    encrypt_key: "your-feishu-encrypt-key"
    verification_token: "your-feishu-verification-token"
  discord:
    token: "your-discord-bot-token"
  weixin:
    token: "your-weixin-token"
  qq:
    app_secret: "your-qq-app-secret"
  dingtalk:
    client_secret: "your-dingtalk-client-secret"
  slack:
    bot_token: "your-slack-bot-token"
    app_token: "your-slack-app-token"
  matrix:
    access_token: "your-matrix-access-token"
  line:
    channel_secret: "your-line-channel-secret"
    channel_access_token: "your-line-channel-access-token"
  onebot:
    access_token: "your-onebot-access-token"
  wecom:
    token: "your-wecom-token"
    encoding_aes_key: "your-wecom-encoding-aes-key"
  wecom_app:
    corp_secret: "your-wecom-app-corp-secret"
    token: "your-wecom-app-token"
    encoding_aes_key: "your-wecom-app-encoding-aes-key"
  wecom_aibot:
    secret: "your-wecom-aibot-secret"
    token: "your-wecom-aibot-token"
    encoding_aes_key: "your-wecom-aibot-encoding-aes-key"
  pico:
    token: "your-pico-token"
  irc:
    password: "your-irc-password"
    nickserv_password: "your-irc-nickserv-password"
    sasl_password: "your-irc-sasl-password"

# Channel Settings (nested format for channels that use settings block)
channel_list:
  mqtt:
    settings:
      username: "your-mqtt-username"
      password: "your-mqtt-password"

# Web Tool API Keys
web:
  brave:
    api_keys:
      - "BSAyour-brave-api-key-1"
      - "BSAyour-brave-api-key-2"  # Optional: Multiple keys for failover
  tavily:
    api_keys:
      - "tvly-your-tavily-api-key"  # Single key in array format
  perplexity:
    api_keys:
      - "pplx-your-perplexity-api-key"  # Single key in array format
  glm_search:
    api_key: "your-glm-search-api-key"  # GLMSearch uses single key format (not array)
  baidu_search:
    api_key: "your-baidu-search-api-key"

# Skills Registry Tokens
skills:
  github:
    token: "your-github-token"
  clawhub:
    auth_token: "your-clawhub-auth-token"
```

## Usage

### Step 1: Create .security.yml

Create or copy the security file:
```bash
cp security.example.yml ~/.picoclaw/.security.yml
```

### Step 2: Fill in your actual values

Edit `~/.picoclaw/.security.yml` and replace placeholder values with your actual API keys and tokens.

### Step 3: Set proper permissions

```bash
chmod 600 ~/.picoclaw/.security.yml
```

### Step 4: Simplify config.json (Recommended)

You can now remove sensitive fields from `config.json` since they're loaded from `.security.yml`:

**Before:**
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1",
      "api_keys": ["sk-your-actual-api-key-here"]
    }
  ],
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
    }
  }
}
```

**After:**
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1"
      // api_key is now loaded from .security.yml
    }
  ],
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram"
      // token is now loaded from .security.yml
    }
  }
}
```

### Step 5: Verify

Restart PicoClaw and verify it loads correctly:
```bash
picoclaw --version
```

## Field Mapping Rules

### Models

**In .security.yml:**
```yaml
model_list:
  <model_name>:
    api_keys:
      - "key-1"
      - "key-2"
```

**Mapping:**
- Field `api_keys` (array) maps to the model's API keys
- The `<model_name>` must match the `model_name` field in `config.json`
- Supports indexed names (e.g., "gpt-5.4:0") - the system will also try the base name ("gpt-5.4")

### Channels

Each channel maps its fields directly:

**In .security.yml:**
```yaml
channels:
  telegram:
    token: "value"
  feishu:
    app_secret: "value"
    encrypt_key: "value"
    verification_token: "value"
  discord:
    token: "value"
```

**Mapping:**
- `channels.telegram.token` → `config.channels.telegram.token`
- `channels.feishu.app_secret` → `config.channels.feishu.app_secret`
- etc.

Channels that use a `settings` block (e.g. MQTT) use the `channel_list` key instead:

```yaml
channel_list:
  mqtt:
    settings:
      username: "value"
      password: "value"
```

- `channel_list.mqtt.settings.username` → `config.channel_list.mqtt.settings.username`
- `channel_list.mqtt.settings.password` → `config.channel_list.mqtt.settings.password`

### Web Tools

**Brave, Tavily, Perplexity:**
```yaml
web:
  brave:
    api_keys:
      - "key-1"
      - "key-2"
```
- Use `api_keys` (plural) array format

**GLMSearch:**
```yaml
web:
  glm_search:
    api_key: "single-key-here"
```
- Use `api_key` (singular) single string format

**BaiduSearch:**
```yaml
web:
  baidu_search:
    api_key: "your-key"
```
- Use `api_key` (singular) single string format

### Skills

**In .security.yml:**
```yaml
skills:
  github:
    token: "value"
  clawhub:
    auth_token: "value"
```

## API Key Formats

### Models - Single key

Use array format with one element:
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-key"
```

### Models - Multiple keys (Load Balancing & Failover)

Use array format with multiple elements:
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-key-1"
      - "sk-your-key-2"
      - "sk-your-key-3"
```

**Benefits:**
- **Load balancing**: Requests are distributed across multiple keys
- **Failover**: Automatic switching to another key if one fails
- **Rate limit management**: Distribute usage across multiple keys
- **High availability**: Reduce downtime during API provider issues

### Web Tools (Brave/Tavily/Perplexity) - Single key

```yaml
web:
  brave:
    api_keys:
      - "BSA-your-key"
```

### Web Tools (Brave/Tavily/Perplexity) - Multiple keys

```yaml
web:
  brave:
    api_keys:
      - "BSA-key-1"
      - "BSA-key-2"
```

### Web Tool (GLMSearch/BaiduSearch) - Single key only

```yaml
web:
  glm_search:
    api_key: "your-glm-key"  # Single string (NOT array)
  baidu_search:
    api_key: "your-baidu-key"  # Single string (NOT array)
```

## Model Name Matching

The system supports intelligent model name matching in `.security.yml`:

### Example 1: Exact Match

**config.json:**
```json
{
  "model_name": "gpt-5.4:0"
}
```

**.security.yml (exact match with index):**
```yaml
model_list:
  gpt-5.4:0:
    api_keys: ["key-1"]
```

### Example 2: Base Name Match

**config.json:**
```json
{
  "model_name": "gpt-5.4:0"
}
```

**.security.yml (base name without index):**
```yaml
model_list:
  gpt-5.4:
    api_keys: ["key-1", "key-2"]
```

Both methods work. The base name match allows you to use simpler keys in `.security.yml` even when your config uses indexed model names for load balancing.

## Backward Compatibility

The system maintains full backward compatibility:

1. **Direct values**: You can still use direct values in `config.json` (not recommended for production)
2. **Mixed usage**: You can have some fields in `.security.yml` and others in `config.json`
3. **Optional security file**: If `.security.yml` doesn't exist, the system will only use values from `config.json`
4. **Override behavior**: If a field exists in both files, `.security.yml` value takes precedence

## Environment Variables

You can override any security value using environment variables:

**For models:**
```bash
export PICOCLAW_CHANNELS_TELEGRAM_TOKEN="token-from-env"
```

**For channels:**
```bash
export PICOCLAW_CHANNELS_TELEGRAM_TOKEN="token-from-env"
export PICOCLAW_CHANNELS_FEISHU_APP_SECRET="secret-from-env"
```

**For web tools:**
```bash
export PICOCLAW_TOOLS_WEB_BRAVE_API_KEY="key-from-env"
export PICOCLAW_TOOLS_WEB_BAIDU_API_KEY="baidu-key-from-env"
```

Environment variables have the highest priority and will override both `config.json` and `.security.yml` values.

The pattern is: `PICOCLAW_<SECTION>_<KEY>_<FIELD>` with underscores separating path segments and converted to uppercase.

## Security Best Practices

1. **Never commit `.security.yml`** to version control
2. **Add to .gitignore**: Ensure `.security.yml` is in your `.gitignore` file
3. **Set file permissions**: `chmod 600 ~/.picoclaw/.security.yml`
4. **Use different keys** for different environments (dev, staging, production)
5. **Rotate keys regularly** and update `.security.yml`
6. **Backup securely**: Encrypt backups containing `.security.yml`. Note that config migrations automatically create date-stamped backups (e.g., `config.json.20260330.bak` and `.security.yml.20260330.bak`)
7. **Review access**: Ensure only authorized users have read access to the file

## API

### loadSecurityConfig

```go
func loadSecurityConfig(securityPath string) (*SecurityConfig, error)
```

Loads the security configuration from `.security.yml`. Returns an empty `SecurityConfig` if the file doesn't exist.

### saveSecurityConfig

```go
func saveSecurityConfig(securityPath string, sec *SecurityConfig) error
```

Saves the security configuration to `.security.yml` with `0o600` permissions.

### applySecurityConfig

```go
func applySecurityConfig(cfg *Config, sec *SecurityConfig) error
```

Applies security configuration to the main config by copying values from `.security.yml` to the corresponding fields in the config.

### securityPath

```go
func securityPath(configPath string) string
```

Returns the path to `.security.yml` relative to the config file.

## Example: Complete Configuration

### config.json

```json
{
  "version": 3,
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
      "api_base": "https://api.openai.com/v1"
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_base": "https://api.anthropic.com/v1"
    }
  ],
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": true
      }
    }
  }
}
```

### .security.yml

```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-actual-openai-key-1"
      - "sk-proj-actual-openai-key-2"
  claude-sonnet-4.6:
    api_keys:
      - "sk-ant-actual-anthropic-key"

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
      - "tvly-your-tavily-key"
  glm_search:
    api_key: "your-glm-key"
  baidu_search:
    api_key: "your-baidu-key"
```

## Testing

Run the security configuration tests:

```bash
go test ./pkg/config -run TestSecurityConfig
```

## Troubleshooting

### Error: "failed to load security config"

- Verify `.security.yml` exists in the same directory as `config.json`
- Check the YAML syntax is valid (use a YAML validator)
- Ensure file permissions allow reading

### Error: "model security entry not found"

- Ensure the model name in `config.json` matches exactly in `.security.yml`
- Check that the `model_list` section exists in `.security.yml`
- For models with indexed names (e.g., "gpt-5.4:0"), ensure the exact name is used or check the base name without index
- Verify the YAML structure is correct (proper indentation)

### Multiple API Keys Not Working

- Ensure you're using `api_keys` (plural) in `.security.yml` for models and web tools (except GLMSearch/BaiduSearch)
- Check that the array format is correct in YAML (proper indentation with dashes)
- Remember: Models, Brave, Tavily, Perplexity MUST use `api_keys` (array format)
- GLMSearch and BaiduSearch MUST use `api_key` (single string format)

### Load Balancing/Failover Issues

- Verify all API keys in the `api_keys` array are valid
- Check that all keys have the same rate limits and permissions
- Monitor logs to see which keys are being used and failing
- Ensure the `api_keys` array is properly formatted in YAML

### Keys Not Being Applied

- Check that `.security.yml` is in the same directory as `config.json`
- Verify the file permissions allow reading (`chmod 600 ~/.picoclaw/.security.yml`)
- Ensure the YAML structure matches the expected format
- Check for typos in field names (case-sensitive)
- Verify the model/channel names match exactly (case-sensitive)

## Migration Guide

### Step 1: Backup your config

The system automatically creates a date-stamped backup before saving a migrated config (e.g., `config.json.20260330.bak` and `.security.yml.20260330.bak`). If you prefer a manual backup:

```bash
cp ~/.picoclaw/config.json ~/.picoclaw/config.json.backup
```

### Step 2: Create .security.yml

```bash
cp security.example.yml ~/.picoclaw/.security.yml
```

### Step 3: Fill in your API keys

Edit `~/.picoclaw/.security.yml` and replace placeholder values with your actual keys.

### Step 4: Remove sensitive fields from config.json

Remove or comment out sensitive fields from `config.json`:
- `api_key` fields from `model_list` entries
- `token` fields from `channels`
- `api_key` fields from `tools.web`
- `token`/`auth_token` fields from `tools.skills`

### Step 5: Set proper permissions

```bash
chmod 600 ~/.picoclaw/.security.yml
```

### Step 6: Test

```bash
picoclaw --version
```

### Step 7: Verify functionality

Test your models and channels to ensure everything works correctly.

### Step 8: Clean up (optional)

If everything works, you can delete the backups:
```bash
rm ~/.picoclaw/config.json.backup
# Also remove auto-generated date-stamped backups if desired:
rm ~/.picoclaw/config.json.20*.bak ~/.picoclaw/.security.yml.20*.bak
```

## Advanced: Encrypted API Keys

PicoClaw supports encrypting API keys in the security file for additional protection.

### Setup

1. Set a passphrase via environment variable:
```bash
export PICOCLAW_CREDENTIAL_PASSPHRASE="your-secure-passphrase"
```

2. When saving config, API keys will be encrypted automatically:
```go
SaveConfig(path, config)
```

### Encrypted Format

Encrypted keys are stored as:
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "enc://encrypted-base64-string"
```

The system automatically decrypts keys at runtime when loading the configuration.

### Benefits

- Additional layer of security
- Keys are encrypted at rest
- Passphrase can be managed separately from the config file

### Important Notes

- Always backup your passphrase securely
- If you lose the passphrase, you'll lose access to encrypted keys
- Use a strong, unique passphrase
- Never commit the passphrase to version control
