# Troubleshooting

## "model ... not found in model_list" or OpenRouter "free is not a valid model ID"

**Symptom:** You see either:

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter returns 400: `"free is not a valid model ID"`

**Cause:** The `model` field in your `model_list` entry is what gets sent to the API. For OpenRouter you must use the **full** model ID, not a shorthand.

- **Wrong:** `"model": "free"` → OpenRouter receives `free` and rejects it.
- **Right:** `"model": "openrouter/free"` → OpenRouter receives `openrouter/free` (auto free-tier routing).

**Fix:** In `~/.picoclaw/config.json` (or your config path):

1. **agents.defaults.model** must match a `model_name` in `model_list` (e.g. `"openrouter-free"`).
2. That entry’s **model** must be a valid OpenRouter model ID, for example:
   - `"openrouter/free"` – auto free-tier
   - `"google/gemini-2.0-flash-exp:free"`
   - `"meta-llama/llama-3.1-8b-instruct:free"`

Example snippet:

```json
{
  "agents": {
    "defaults": {
      "model": "openrouter-free"
    }
  },
  "model_list": [
    {
      "model_name": "openrouter-free",
      "model": "openrouter/free",
      "api_key": "sk-or-v1-YOUR_OPENROUTER_KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  ]
}
```

Get your key at [OpenRouter Keys](https://openrouter.ai/keys).
