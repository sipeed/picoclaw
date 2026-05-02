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

## OpenRouter reasoning model leaks thinking into reply

**Symptom:** With an OpenRouter reasoning model (for example `nvidia/nemotron-3-super-120b-a12b:free`), even a strict prompt like `Reply with exactly: PONG` produces a reply whose visible content begins with reasoning preamble such as `We need to follow the instruction ...` before the requested text.

**Cause:** Some OpenRouter reasoning models put their chain-of-thought into `message.content` rather than a separate reasoning channel. Without server-side suppression, that reasoning surfaces as the assistant's visible reply.

**Fix:** Set `extra_body.reasoning.exclude = true` on the model entry so OpenRouter strips reasoning before returning content:

```json
{
  "model_name": "openrouter-nemotron-free",
  "provider": "openrouter",
  "model": "nvidia/nemotron-3-super-120b-a12b:free",
  "api_base": "https://openrouter.ai/api/v1",
  "api_keys": ["sk-or-v1-YOUR_OPENROUTER_KEY"],
  "extra_body": {
    "reasoning": {
      "exclude": true
    }
  }
}
```

PicoClaw ships this as a default `model_list` entry (`openrouter-nemotron-free`); set `api_keys` to use it.

**Tradeoff:** apply this per-model, not globally. Some users want OpenRouter to return reasoning tokens (for example to render thinking separately), so PicoClaw does not auto-inject `reasoning.exclude` for all OpenRouter models.
