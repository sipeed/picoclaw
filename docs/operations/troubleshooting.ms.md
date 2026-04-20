# Penyelesaian Masalah

## "model ... not found in model_list" atau OpenRouter "free is not a valid model ID"

**Gejala:** Anda akan melihat salah satu daripada mesej berikut:

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter memulangkan 400: `"free is not a valid model ID"`

**Punca:** Medan `model` dalam entri `model_list` anda ialah nilai yang dihantar ke API. Untuk OpenRouter, anda mesti menggunakan ID model **penuh**, bukan bentuk singkatan.

- **Salah:** `"model": "free"` → OpenRouter menerima `free` dan menolaknya.
- **Betul:** `"model": "openrouter/free"` → OpenRouter menerima `openrouter/free` (routing auto free-tier).

**Penyelesaian:** Dalam `~/.picoclaw/config.json` (atau laluan config anda):

1. **agents.defaults.model** mesti sepadan dengan `model_name` dalam `model_list` (contohnya `"openrouter-free"`).
2. Medan **model** bagi entri tersebut mesti merupakan ID model OpenRouter yang sah, contohnya:
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

Dapatkan kunci anda di [OpenRouter Keys](https://openrouter.ai/keys).
