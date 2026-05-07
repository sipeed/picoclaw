// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// This file demonstrates how to use the security configuration feature
// It's not meant to be compiled, just for documentation purposes

/*
Package config

# Example: Using Security Configuration

## Overview

The security configuration feature allows you to separate sensitive data (API keys,
tokens, secrets, passwords) from your main configuration. The system automatically
loads values from `.security.yml` and applies them to the corresponding fields in
your config.

**Key Points:**
- Values from `.security.yml` are automatically mapped to config fields
- No `ref:` syntax is needed - just omit sensitive fields from config.json
- If a field exists in both files, `.security.yml` value takes precedence
- You can mix direct values in config.json with security values

## 1. Create .security.yml

File: ~/.picoclaw/.security.yml

```yaml
# Model API Keys
# All models MUST use 'api_keys' (plural) array format
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
	  token: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
	discord:
	  token: "your-discord-bot-token"

# Web Tool Keys
# Brave, Tavily, Perplexity: Use 'api_keys' array
# GLMSearch, BaiduSearch: Use 'api_key' single string
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
	  api_key: "your-glm-search-api-key"  # Single key (not array)
	baidu_search:
	  api_key: "your-baidu-search-api-key"  # Single key (not array)

```

## 2. Simplify config.json

File: ~/.picoclaw/config.json

Note: Sensitive fields are omitted because they're loaded from .security.yml

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
	      "api_base": "https://api.openai.com/v1"
	      // api_key is automatically loaded from .security.yml
	    },
	    {
	      "model_name": "claude-sonnet-4.6",
	      "model": "anthropic/claude-sonnet-4.6",
	      "api_base": "https://api.anthropic.com/v1"
	      // api_key is automatically loaded from .security.yml
	    }
	  ],
	  "channels": {
	    "telegram": {
	      "enabled": true
	      // token is automatically loaded from .security.yml
	    },
	    "discord": {
	      "enabled": true
	      // token is automatically loaded from .security.yml
	    }
	  },
	  "tools": {
	    "web": {
	      "brave": {
	        "enabled": true
	        // api_key is automatically loaded from .security.yml
	      },
	      "tavily": {
	        "enabled": true
	        // api_key is automatically loaded from .security.yml
	      },
	      "glm_search": {
	        "enabled": true
	        // api_key is automatically loaded from .security.yml
	      },
	      "baidu_search": {
	        "enabled": true
	        // api_key is automatically loaded from .security.yml
	      }
	    }
	  }
	}

```

## 3. Set proper permissions

```bash
chmod 600 ~/.picoclaw/.security.yml
```

## 4. Add to .gitignore

```gitignore
# Security configuration
.security.yml
```

## 5. Verify it works

```bash
picoclaw --version
```

# Supported Fields in .security.yml

## Model API Keys

All models MUST use the `api_keys` (plural) array format in .security.yml.

```yaml
model_list:

	<model_name>:
	  api_keys:
	    - "key-1"
	    - "key-2"  # Optional: Multiple keys for failover

```

Examples:
```yaml
model_list:

	gpt-5.4:
	  api_keys:
	    - "sk-proj-key-1"
	    - "sk-proj-key-2"
	claude-sonnet-4.6:
	  api_keys:
	    - "sk-ant-key"

```

**Important:**
- Always use `api_keys` (plural) for models
- Even a single key must be in an array format
- The model_name in .security.yml must match the model_name in config.json

## Channel Tokens/Secrets

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
	weixin:
	  token: "value"
	qq:
	  app_secret: "value"
	dingtalk:
	  client_secret: "value"
	slack:
	  bot_token: "value"
	  app_token: "value"
	matrix:
	  access_token: "value"
	line:
	  channel_secret: "value"
	  channel_access_token: "value"
	onebot:
	  access_token: "value"
	wecom:
	  token: "value"
	  encoding_aes_key: "value"
	wecom_app:
	  corp_secret: "value"
	  token: "value"
	  encoding_aes_key: "value"
	wecom_aibot:
	  secret: "value"
	  token: "value"
	  encoding_aes_key: "value"
	pico:
	  token: "value"
	irc:
	  password: "value"
	  nickserv_password: "value"
	  sasl_password: "value"

## Web Tool API Keys

**Brave, Tavily, Perplexity:**
```yaml
web:

	brave:
	  api_keys:
	    - "BSA-key-1"
	    - "BSA-key-2"
	tavily:
	  api_keys:
	    - "tvly-key"
	perplexity:
	  api_keys:
	    - "pplx-key"

```
Use `api_keys` (plural) array format.

**GLMSearch, BaiduSearch:**
```yaml
web:

	glm_search:
	  api_key: "your-glm-key"
	baidu_search:
	  api_key: "your-baidu-key"

```
Use `api_key` (singular) single string format.

## Skills Registry Tokens

```yaml
skills:

	github:
	  token: "value"
	clawhub:
	  auth_token: "value"

```

# Backward Compatibility

You can still use direct values in config.json if needed:

```json

	{
	  "model_list": [
	    {
	      "model_name": "local-model",
	      "model": "ollama/llama3",
	      "api_base": "http://localhost:11434/v1",
	      "api_key": "ollama"  // Direct value (works fine)
	    }
	  ]
	}

```

You can also mix security values and direct values:

```json

	{
	  "model_list": [
	    {
	      "model_name": "cloud-model",
	      // api_key loaded from .security.yml
	    },
	    {
	      "model_name": "local-model",
	      "model": "ollama/llama3",
	      "api_base": "http://localhost:11434/v1",
	      "api_key": "ollama"  // Direct value
	    }
	  ]
	}

```

**Priority Order:**
1. Environment variables (highest priority)
2. .security.yml values
3. config.json direct values (lowest priority)

# Migration from Old Config

## Step 1: Backup your config
```bash
cp ~/.picoclaw/config.json ~/.picoclaw/config.json.backup
```

## Step 2: Create .security.yml
```bash
cp security.example.yml ~/.picoclaw/.security.yml
```

## Step 3: Fill in your API keys
Edit ~/.picoclaw/.security.yml and replace placeholders with your actual keys.

## Step 4: Simplify config.json (Recommended)
Remove sensitive fields from ~/.picoclaw/config.json:
- `api_key` fields from model_list entries
- `token` fields from channels
- `api_key` fields from tools.web
- `token`/`auth_token` fields from tools.skills

## Step 5: Set permissions
```bash
chmod 600 ~/.picoclaw/.security.yml
```

## Step 6: Test
```bash
picoclaw --version
```

If everything works, you can delete the backup:
```bash
rm ~/.picoclaw/config.json.backup
```

# Advanced Features

## Multiple API Keys (Load Balancing & Failover)

You can configure multiple API keys for models and web tools to enable:
- **Load balancing**: Requests are distributed across multiple keys
- **Failover**: If a key fails, the system automatically switches to another key
- **Rate limit management**: Distribute usage across multiple keys
- **High availability**: Reduce downtime during API provider issues

### Example: Model with Multiple Keys

**.security.yml:**
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
	      "api_base": "https://api.openai.com/v1"
	    }
	  ]
	}

```

### Example: Web Tool with Multiple Keys

**.security.yml:**
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
	        "enabled": true
	      },
	      "tavily": {
	        "enabled": true
	      },
	      "glm_search": {
	        "enabled": true
	      }
	    }
	  }
	}

```

## Single Key Format

**Models, Brave, Tavily, Perplexity:**
```yaml
model_list:

	gpt-5.4:
	  api_keys:
	    - "sk-proj-your-key"  # Single key in array format

```

**GLMSearch, BaiduSearch:**
```yaml
web:

	glm_search:
	  api_key: "your-glm-key"  # Single key (not array)

```

## Model Name Matching

The system supports intelligent model name matching in .security.yml:

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

Both methods work. The base name match allows you to use simpler keys in .security.yml
even when your config uses indexed model names for load balancing.

## Security File Permissions

The security file should have restricted permissions:

```bash
chmod 600 ~/.picoclaw/.security.yml
```

This ensures only the owner can read and write the file.

# Security Best Practices

1. Never commit .security.yml to version control
2. Add .security.yml to your .gitignore file
3. Set file permissions: chmod 600 ~/.picoclaw/.security.yml
4. Use different keys for different environments (dev, staging, production)
5. Rotate keys regularly and update .security.yml
6. Encrypt backups containing .security.yml
7. Review access regularly

# Environment Variables

You can override any security value using environment variables:

```bash
# Channels
export PICOCLAW_CHANNELS_TELEGRAM_TOKEN="token-from-env"
export PICOCLAW_CHANNELS_DISCORD_TOKEN="discord-token-from-env"

# Web Tools
export PICOCLAW_TOOLS_WEB_BRAVE_API_KEY="brave-key-from-env"
export PICOCLAW_TOOLS_WEB_BAIDU_API_KEY="baidu-key-from-env"

# Skills
export PICOCLAW_TOOLS_SKILLS_GITHUB_TOKEN="github-token-from-env"
```

Environment variables have the highest priority and will override both config.json
and .security.yml values.

# Troubleshooting

## Error: "failed to load security config"
- Ensure .security.yml exists in the same directory as config.json
- Check YAML syntax is valid (use a YAML validator)
- Verify file permissions allow reading

## Error: "model security entry not found"
- Check that the model name in config.json matches exactly in .security.yml
- Verify the model_list section exists in .security.yml
- For indexed names (e.g., "gpt-5.4:0"), check both exact match and base name match
- Ensure the YAML structure is correct (proper indentation)

## Multiple API Keys Not Working
- Ensure you're using `api_keys` (plural) in .security.yml for models and web tools (except GLMSearch/BaiduSearch)
- Check that the array format is correct in YAML (proper indentation with dashes)
- Remember: Models, Brave, Tavily, Perplexity MUST use `api_keys` (array format)
- GLMSearch and BaiduSearch MUST use `api_key` (single string format)

## Keys Not Being Applied
- Check that .security.yml is in the same directory as config.json
- Verify the file permissions allow reading (chmod 600 ~/.picoclaw/.security.yml)
- Ensure the YAML structure matches the expected format
- Check for typos in field names (case-sensitive)
- Verify the model/channel names match exactly (case-sensitive)

## Load Balancing/Failover Issues
- Verify all API keys in the api_keys array are valid
- Check that all keys have the same rate limits and permissions
- Monitor logs to see which keys are being used and failing
- Ensure the api_keys array is properly formatted in YAML
*/
package config

// This file is documentation only
