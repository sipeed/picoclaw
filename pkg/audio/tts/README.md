# TTS (Text-to-Speech)

This package handles Text-to-Speech (speech synthesis) capabilities.

## Configuration

PicoClaw uses the unified and secure `ModelList` configuration for TTS. Plain-text API keys are no longer tolerated in the `voice` config block directly. 

To configure a TTS model, define it in your `model_list`, and set it in your `voice` configuration block using the `tts_model_name` field.

### Example `config.json`

```json
{
  "voice": {
    "tts_model_name": "my-tts-model"
  },
  "model_list": [
    {
      "model_name": "my-tts-model",
      "model": "openai/tts-1",
      "api_base": "https://api.openai.com/v1"
    }
  ]
}
```

### Security Configuration

API keys for your TTS model are managed securely with standard `model_list` entries in `.security.yml`:

```yaml
model_list:
  my-tts-model:
    api_keys:
      - "sk-openai-your-key-here"
```

## How It Works

PicoClaw's `DetectTTS` function resolves the TTS Provider efficiently using the secure definitions:

1. **Targeted Selection**: It will resolve the TTS Provider strictly via the `tts_model_name` configured in the `voice` block to pluck the respective model instance, base URL, keys, and proxy details.
2. **Fallback Scanning**: If no explicit `tts_model_name` is set (or missing), PicoClaw will scan the `model_list` for any entry whose model structure explicitly contains the word `tts` and possesses a valid API key.

Most standard TTS routing passes through `OpenAITTSProvider`, which acts universally for OpenAI-compatible audio speech synthesis API formats.
