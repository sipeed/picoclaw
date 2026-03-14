# Config CLI Usage

Use `picoclaw config` to manage `model_list` and `agents` without editing `config.json` directly.

Default config path: `~/.picoclaw/config.json`. If the file does not exist, some commands will prompt you to run `picoclaw onboard` first.

---

## 1. model_list

### list — List all models

```bash
picoclaw config model_list list
```

Prints the current `model_list` in a table (MODEL_NAME, MODEL, API_BASE, AUTH, etc.). Shows a message when the list is empty or the config file is missing.

### get — Inspect a single model’s config

```bash
# Show all fields for that model (sensitive fields are masked)
picoclaw config model_list get <model_name>

# Show only one key’s value (for scripting; not masked)
picoclaw config model_list get <model_name> <key>
```

Supported keys (snake_case, matching JSON):  
`model_name`, `model`, `api_base`, `api_key`, `proxy`, `auth_method`, `connect_mode`, `workspace`, `token_url`, `client_id`, `client_secret`, `max_tokens_field`, `rpm`.

### set — Set a single field

```bash
picoclaw config model_list set <model_name> <key> <value>
```

Sets the given key for the **first** entry whose `model_name` matches, then writes the config.  
`rpm` is an integer; all other keys are strings. The key must be one of the supported keys above.

Examples:

```bash
picoclaw config model_list set gpt4 api_base https://api.openai.com/v1
picoclaw config model_list set gpt4 rpm 60
```

### add — Add a model

```bash
# Specify everything with flags
picoclaw config model_list add --model-name qwen-turbo --model litellm/qwen-turbo \
  --api-base https://litellm.example.com \
  --token-url https://keycloak.example.com/realms/xxx/protocol/openid-connect/token \
  --client-id ai-bot --client-secret xxx

# Interactive: pass model_name only or omit; in a TTY you’ll be prompted for the rest
picoclaw config model_list add qwen-turbo
picoclaw config model_list add
```

Common flags:  
`--model-name`, `--model`, `--api-base`, `--api-key`, `--proxy`, `--auth-method`, `--max-tokens-field`, `--token-url`, `--client-id`, `--client-secret`.  
For `litellm/...` protocol you must provide `api_base`, `token_url`, `client_id`, and `client_secret`; in a TTY, missing values are prompted interactively.

### remove — Remove model(s)

```bash
picoclaw config model_list remove <model_name>
```

Removes entries by `model_name`. If there are multiple entries with the same name (e.g. round-robin), all are removed by default. Use `--first` to remove only the first match:

```bash
picoclaw config model_list remove loadbalanced-gpt4 --first
```

### update — Update a model

```bash
picoclaw config model_list update <model_name> [flags]
```

Updates the **first** entry matching `model_name`; only the fields corresponding to the given flags are changed.  
Flags are the same as for add: `--model`, `--api-base`, `--api-key`, `--token-url`, `--client-id`, `--client-secret`, `--proxy`, `--auth-method`, `--max-tokens-field`, etc.

Example:

```bash
picoclaw config model_list update qwen-turbo --api-base https://new.example.com
```

---

## 2. agent

### defaults — Global defaults (agents.defaults)

**get**

```bash
# Print all agents.defaults fields
picoclaw config agent defaults get

# Print a single key
picoclaw config agent defaults get model_name
```

Supported keys:  
`workspace`, `restrict_to_workspace`, `provider`, `model_name`, `model`, `model_fallbacks`, `image_model`, `image_model_fallbacks`, `max_tokens`, `temperature`, `max_tool_iterations`.

**set**

```bash
picoclaw config agent defaults set <key> <value>
```

Examples:

```bash
picoclaw config agent defaults set model_name gpt4
picoclaw config agent defaults set max_tokens 8192
picoclaw config agent defaults set restrict_to_workspace true
picoclaw config agent defaults set model_fallbacks "gpt4,claude"
```

`restrict_to_workspace` is true/false; `max_tokens` and `max_tool_iterations` are integers; `temperature` is a float; `model_fallbacks` and `image_model_fallbacks` are comma-separated strings.

### list — List all agents

```bash
picoclaw config agent list
```

Prints `agents.list` in a table (ID, NAME, MODEL, WORKSPACE). Shows a message when the list is empty.

### add — Add an agent

```bash
picoclaw config agent add <id> [--name "Display name"] [--model gpt4] [--workspace ~/ws] [--default]
```

`id` is required; `--name`, `--model`, and `--workspace` are optional. In a TTY, missing values are prompted. `--default` sets this agent as the default.

### remove — Remove an agent

```bash
picoclaw config agent remove <id>
```

Removes the agent with the given id from `agents.list` and saves the config.

### update — Update an agent

```bash
picoclaw config agent update <id> [--name "New name"] [--model gpt4] [--workspace ~/ws] [--default]
```

Only the provided fields are updated; others are left unchanged.

---

## 3. Quick reference

| Purpose              | Command |
|----------------------|--------|
| List models          | `picoclaw config model_list list` |
| Get/set one model     | `picoclaw config model_list get/set <model_name> [key] [value]` |
| Add/remove/update models | `picoclaw config model_list add/remove/update ...` |
| Agent defaults       | `picoclaw config agent defaults get [key]` / `set <key> <value>` |
| List/add/remove/update agents | `picoclaw config agent list/add/remove/update ...` |

For more options and descriptions, run `picoclaw config --help`, `picoclaw config model_list --help`, and `picoclaw config agent --help`.
