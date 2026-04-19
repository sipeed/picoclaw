# FreeRide 🦞

FreeRide is a dynamic model rotation and failover system for PicoClaw that leverages OpenRouter's free model pool. It ensures your agent stays alive even if individual free models become rate-limited or go offline.

## Key Features

- **Automatic Discovery**: Scans OpenRouter for the best currently available free models.
- **Dynamic Failover**: Automatically rotates through a pool of models when errors (like 429 Rate Limiting) occur.
- **Intelligent Ranking**: Models are scored and ranked based on context length, capabilities (tools/vision), and provider trust.
- **K3s Ready**: Designed to work seamlessly in Kubernetes environments with secure API key management.

## Configuration

FreeRide is implemented as a native PicoClaw tool.

### 1. Enable the Tool
Ensure the `freeride` tool is enabled and whitelisted in your `config.json`:

```json
{
  "tools": {
    "whitelist": ["freeride", ...],
    "whitelist_enabled": true,
    "security_policy": {
      "enabled": true,
      "config": {
        "allowed_tools": {
          "freeride": true
        }
      }
    }
  }
}
```

### 2. Set the API Key
FreeRide requires an OpenRouter API key. Even for free models, many providers require a key for identification and higher rate limits.

In **Local Mode**, set the environment variable:
```bash
export OPENROUTER_API_KEY="sk-or-v1-..."
```

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

## Troubleshooting

- **404 Errors**: Ensure the model is still available on OpenRouter using `freeride list`. If it's gone, run `freeride auto` to refresh your fallback pool.
- **429 Rate Limiting**: This is common with free models. PicoClaw will automatically try the next model in your `model_fallbacks` list.
- **Security Blocks**: Ensure `freeride` is added to your `security_policy` allowed tools map.
