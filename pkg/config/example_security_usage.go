// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// This file demonstrates how to use the security configuration feature
// It's not meant to be compiled, just for documentation purposes

/*
Package config

# Example: Using Security Configuration

## 1. Create security.yml

File: ~/.picoclaw/security.yml

```yaml
# Model API Keys
# Note: Use 'api_keys' array for multiple keys (load balancing/failover)
# Single key should be provided as an array with one element
model_list:

	gpt-5.4:
	  api_keys:
	    - "sk-proj-your-actual-openai-key-1"
	    - "sk-proj-your-actual-openai-key-2"  # Failover key
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
# Note: Use 'api_keys' array for multiple keys (load balancing/failover)
# For GLMSearch, use 'api_key' (single string)
web:

	brave:
	  api_keys:
	    - "BSAyour-brave-api-key-1"
	    - "BSAyour-brave-api-key-2"  # Failover key
	tavily:
	  api_keys:
	    - "tvly-your-tavily-api-key"  # Single key in array format
	glm_search:
	  api_key: "your-glm-search-api-key"  # Single key (not array)

```

## 2. Update config.json to use references

File: ~/.picoclaw/config.json

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
		    },
		    "discord": {
		      "enabled": true,
		      "token": "ref:channels.discord.token"
		    }
		  },
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

## 3. Set proper permissions

```bash
chmod 600 ~/.picoclaw/security.yml
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

# Available Reference Paths

## Model API Keys
- ref:model_list.<model_name>.api_key

Examples:
- ref:model_list.gpt-5.4.api_key
- ref:model_list.claude-sonnet-4.6.api_key

**Note:** In .security.yml, use `api_keys` (array) format for models.
Both single and multiple keys should use the array format.

## Channel Tokens/Secrets
- ref:channels.telegram.token
- ref:channels.feishu.app_secret
- ref:channels.feishu.encrypt_key
- ref:channels.feishu.verification_token
- ref:channels.discord.token
- ref:channels.qq.app_secret
- ref:channels.dingtalk.client_secret
- ref:channels.slack.bot_token
- ref:channels.slack.app_token
- ref:channels.matrix.access_token
- ref:channels.line.channel_secret
- ref:channels.line.channel_access_token
- ref:channels.onebot.access_token
- ref:channels.wecom.token
- ref:channels.wecom.encoding_aes_key
- ref:channels.wecom_app.corp_secret
- ref:channels.wecom_app.token
- ref:channels.wecom_app.encoding_aes_key
- ref:channels.wecom_aibot.token
- ref:channels.wecom_aibot.encoding_aes_key
- ref:channels.pico.token
- ref:channels.irc.password
- ref:channels.irc.nickserv_password
- ref:channels.irc.sasl_password

## Web Tool API Keys
- ref:web.brave.api_key
- ref:web.tavily.api_key
- ref:web.perplexity.api_key
- ref:web.glm_search.api_key

**Note:**
- Brave, Tavily, Perplexity: Use `api_keys` (array) format in .security.yml
- GLMSearch: Use `api_key` (single string) format in .security.yml

## Skills Registry Tokens
- ref:skills.github.token
- ref:skills.clawhub.auth_token

# Backward Compatibility

You can still use direct values in config.json if needed:

```json

	{
	  "model_list": [
	    {
	      "model_name": "local-model",
	      "model": "ollama/llama3",
	      "api_base": "http://localhost:11434/v1",
	      "api_key": "ollama"  // Direct value (no reference)
	    }
	  ]
	}

```

You can also mix references and direct values:

```json

	{
	  "model_list": [
	    {
	      "model_name": "cloud-model",
	      "api_key": "ref:model_list.cloud-model.api_key"  // From .security.yml
	    },
	    {
	      "model_name": "local-model",
	      "api_key": "ollama"  // Direct value
	    }
	  ]
	}

```

# Migration from Old Config

## Step 1: Backup your config
```bash
cp ~/.picoclaw/config.json ~/.picoclaw/config.json.backup
```

## Step 2: Copy the example security file
```bash
cp security.example.yml ~/.picoclaw/.security.yml
```

## Step 3: Fill in your API keys
Edit ~/.picoclaw/.security.yml and replace placeholders with your actual keys.

## Step 4: Update config.json references
Replace sensitive values in ~/.picoclaw/config.json with ref: references.

## Step 5: Test
```bash
picoclaw --version
```

If everything works, you can delete the backup:
```bash
rm ~/.picoclaw/config.json.backup
```

# Advanced Features

## Multiple API Keys (Load Balancing & Failover)

You can configure multiple API keys for both models and web tools to enable:
- **Load balancing**: Requests are distributed across multiple keys
- **Failover**: If a key fails, the system automatically switches to another key

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
	      "api_key": "ref:model_list.gpt-5.4.api_key"
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
	        "enabled": true,
	        "api_key": "ref:web.brave.api_key"
	      }
	    }
	  }
	}

```

### Single Key

Use array format with one element:
```yaml
model_list:

	gpt-5.4:
	  api_keys:
	    - "sk-proj-your-key"  # Single key in array format

```

### Multiple Keys (Load Balancing & Failover)

Use array format with multiple elements:
```yaml
model_list:

	gpt-5.4:
	  api_keys:
	    - "sk-proj-key-1"
	    - "sk-proj-key-2"
	    - "sk-proj-key-3"

```

**Important:** All model keys in .security.yml must use the `api_keys` (plural) array format.
The single `api_key` (singular) format is NOT supported for models.

### Model Index Matching

The system supports intelligent model name matching in .security.yml:

**Example 1: Exact Match**
```yaml
# config.json

	{
	  "model_name": "gpt-5.4:0"
	}

# .security.yml (exact match with index)
model_list:

	gpt-5.4:0:
	  api_keys: ["key-1"]

```

**Example 2: Base Name Match**
```yaml
# config.json

	{
	  "model_name": "gpt-5.4:0"
	}

# .security.yml (base name without index)
model_list:

	gpt-5.4:
	  api_keys: ["key-1"]

```

Both methods work. The base name match allows you to use simpler keys in .security.yml
even when your config uses indexed model names for load balancing.

### Security File Permissions

The security file should have restricted permissions:

```bash
chmod 600 ~/.picoclaw/.security.yml
```

This ensures only the owner can read and write the file.

# Security Best Practices

1. Never commit .security.yml to version control
2. Set file permissions: chmod 600 ~/.picoclaw/.security.yml
3. Use different keys for different environments
4. Rotate keys regularly and update .security.yml
5. Encrypt backups containing .security.yml

# Troubleshooting

## Error: "model security entry not found"
- Check that the model name in config.json matches exactly in .security.yml
- Verify the model_list section exists in .security.yml

## Error: "failed to load security config"
- Ensure .security.yml exists in the same directory as config.json
- Check YAML syntax is valid
- Verify file permissions allow reading

## Error: "unknown reference path"
- Verify the reference format is correct
- Check the path structure matches the examples above
- Ensure all required sections exist in .security.yml
*/
package config

// This file is documentation only
