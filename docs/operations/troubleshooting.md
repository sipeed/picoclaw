# Troubleshooting

## "model ... not found in model_list" or OpenRouter "free is not a valid model ID"

**Symptom:** You see either:

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter returns 400: `"free is not a valid model ID"`

**Cause:** PicoClaw now resolves provider/model in two steps:

- If `provider` is set, the `model` field is sent to that provider unchanged.
- If `provider` is omitted, PicoClaw infers the provider from the first `/` segment and sends everything after that first `/` as the runtime model ID.

For OpenRouter free-tier routing, the preferred config is explicit `provider`.

- **Wrong:** `"model": "free"` → no OpenRouter provider is selected, so `free` is not a valid OpenRouter model route.
- **Right:** `"provider": "openrouter", "model": "free"` → OpenRouter receives `free`.
- **Also supported:** `"model": "openrouter/free"` → provider resolves to `openrouter`, runtime model ID resolves to `free`.

**Fix:** In `~/.picoclaw/config.json` (or your config path):

1. **agents.defaults.model_name** must match a `model_name` in `model_list` (e.g. `"openrouter-free"`).
2. That entry should preferably set **provider** to `openrouter`, and **model** should be a valid OpenRouter model ID, for example:
   - `"free"` – auto free-tier
   - `"google/gemini-2.0-flash-exp:free"`
   - `"meta-llama/llama-3.1-8b-instruct:free"`

Example snippet:

```json
{
  "agents": {
    "defaults": {
      "model_name": "openrouter-free"
    }
  },
  "model_list": [
    {
      "model_name": "openrouter-free",
      "provider": "openrouter",
      "model": "free",
      "api_keys": ["sk-or-v1-YOUR_OPENROUTER_KEY"],
      "api_base": "https://openrouter.ai/api/v1"
    }
  ]
}
```

Get your key at [OpenRouter Keys](https://openrouter.ai/keys).
