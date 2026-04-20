# FreeRide 🦞

FreeRide is a dynamic model rotation and failover system for PicoClaw that leverages OpenRouter's free model pool. It ensures your agent stays alive even if individual free models become rate-limited or go offline.

## Key Features

- **Automatic Discovery**: Scans OpenRouter for the best currently available free models.
- **Dynamic Failover**: Automatically rotates through a pool of models when errors (like 429 Rate Limiting) occur.
- **Intelligent Ranking**: Models are scored and ranked based on context length, capabilities (tools/vision), and provider trust.
- **K3s Ready**: Designed to work seamlessly in Kubernetes environments with secure API key management.
- **Visual Provenance (🦞)**: Responses generated via a fallback model are clearly marked with a "lobster" emoji and the model name, providing transparency about which model handled your request.

## Configuration

FreeRide is implemented as a native PicoClaw tool. For production environments (especially in the **main branch**), ensure you follow the [Security Configuration](../security/security_configuration.md) to manage your API keys safely.

### 1. Enable the Tool
Ensure the `skills` tool is enabled in your `config.json` (FreeRide is bundled with the skills system):
 
```json
{
  "tools": {
    "skills": {
      "enabled": true
    }
  }
}
```

### 2. Set the API Key
FreeRide requires an OpenRouter API key. Even for free models, many providers require a key for identification and higher rate limits.

PicoClaw supports dynamic environment variable resolution using the `env://` scheme.

In **Local Mode** or **Docker**, set the environment variable:
```bash
export OPENROUTER_API_KEY="sk-or-v1-..."
```

Then in your `config.json`, use:
```json
{
  "api_keys": ["env://OPENROUTER_API_KEY"]
}
```
*(Note: `freeride auto` will automatically configure this for you.)*

In **K3s Mode**, add the secret to your cluster (see below).

## Usage

You can interact with FreeRide directly through the agent:

### `freeride auto`
**The most important command.** This command:
1. Fetches the current list of ~28+ free models.
2. Ranks them by quality.
3. Automatically populates your `config.json`'s `model_list`.
4. Adds the top 5 models to your agent's `model_fallbacks` list.
5. Reloads the agent configuration instantly.

### `freeride status`
Shows your current primary model and the active fallback rotation pool.

### `freeride list [limit]`
Displays the current top-ranked free models available on OpenRouter without modifying your configuration.

### `freeride settimeout [seconds]`
Sets the request timeout for all OpenRouter models. Default is 300 seconds (5 minutes). Use this if you need longer timeouts for complex tasks:
```bash
picoclaw freeride settimeout 600  # 10 minutes
```

## K3s Deployment & Secrets

When running PicoClaw on K3s, follow these steps to manage your secrets safely.

### Adding the Secret
If you are creating the secrets for the first time:
```bash
kubectl create secret generic picoclaw-secrets \
  --namespace agi \
  --from-literal=openrouter-api-key="YOUR_KEY_HERE"
```

### Updating Existing Secrets (Safe Patching)
If `picoclaw-secrets` already exists and you want to add the OpenRouter key without losing your Telegram or NVIDIA keys, use **`kubectl patch`**:

```bash
kubectl patch secret picoclaw-secrets \
  --namespace agi \
  --type='json' \
  -p='[{"op": "add", "path": "/data/openrouter-api-key", "value":"'$(echo -n "YOUR_KEY_HERE" | base64 -w0)'"}]'
```

### Deployment Configuration
Ensure your `deployment.yaml` maps the secret to the environment variable:

```yaml
env:
  - name: OPENROUTER_API_KEY
    valueFrom:
      secretKeyRef:
        name: picoclaw-secrets
        key: openrouter-api-key
```

## Cooldown Persistence & Timing ❄️
 
To prevent the agent from "hanging" or retrying known-failed models, PicoClaw uses a two-pronged approach:
 
### 1. Zero-Amnesia Persistence
Model failures (e.g., 429 Rate Limits) are saved to `~/.picoclaw/cooldowns.json`. This ensures that if you restart the agent, it **remembers** which models were saturated and skips them instantly. You no longer have to wait through a series of timeouts every time you restart.
 
### 2. Generous 5 Minute Timeout
The default request timeout for LLM calls is **300 seconds (5 minutes)**. Free models can be slower than paid ones, and complex agentic tasks (multi-step reasoning, file operations, debugging) need time to complete. If a free model truly can't handle the request, it will return an error rather than hanging indefinitely - allowing the agent to fail over to the next fallback.
 
## Troubleshooting

- **404 Errors**: Ensure the model is still available on OpenRouter using `freeride list`. If it's gone, run `freeride auto` to refresh your fallback pool.
- **429 Rate Limiting**: This is common with free models. PicoClaw will automatically try the next model in your `model_fallbacks` list and persist the cooldown to `cooldowns.json`.

---

## Legal & Responsible Use 🛡️

FreeRide is provided for **personal assistance, educational research, and infrastructure failover** purposes only. By using this capability, you acknowledge and agree to the following:

1.  **Terms of Service**: You are responsible for complying with [OpenRouter's Terms of Service](https://openrouter.ai/terms) and the individual "Acceptable Use Policies" of each model provider (e.g., Google, Meta, Mistral).
2.  **No Guarantee of Service**: Free models are provided "as-is" by third parties. They may be withdrawn, rate-limited, or modified at any time without notice. 
3.  **No Reselling**: You should not use FreeRide to build commercial services that "resell" free model access in a way that violates provider licenses (check specific model licenses like Llama 3 Community or Qwen for commercial usage thresholds).
4.  **Rate Limit Respect**: PicoClaw handles failover automatically, but users should not use FreeRide to intentionally overwhelm or evade the fair-use rate limits of providers.

*PicoClaw is an independent tool and is not affiliated with OpenRouter or any specific LLM provider.*
