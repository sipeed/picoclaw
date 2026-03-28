# ASR (Automatic Speech Recognition)

This package handles Automatic Speech Recognition (speech-to-text) capabilities.

## Configuration

PicoClaw uses the unified and secure `ModelList` configuration for ASR. Instead of plain-text API keys in the `voice` configuration, you should define your ASR providers in the global `model_list` and reference them by name in the `voice` configuration section.

To configure an ASR model, set the `model_name` under the `voice` configuration to match a defined model in your `model_list`.

### Example `config.json`

```json
{
  "voice": {
    "model_name": "my-asr-model",
    "echo_transcription": true
  },
  "model_list": [
    {
      "model_name": "my-asr-model",
      "model": "openai/whisper-1",
      "api_base": "https://api.openai.com/v1"
    },
    {
      "model_name": "elevenlabs-asr",
      "model": "elevenlabs/scribe_v1"
    }
  ]
}
```

### Security Configuration

API keys for the ASR model should be supplied in your `.security.yml` file matching the respective `model_name`:

```yaml
model_list:
  my-asr-model:
    api_keys:
      - "sk-openai-your-key-here"
  elevenlabs-asr:
    api_keys:
      - "sk-elevenlabs-your-key"
```

## How It Works

PicoClaw's `DetectTranscriber` function will attempt to detect the appropriate Transcriber in the following order:

1. **Targeted Selection**: Standard matching via `cfg.Voice.ModelName`.
    - If the protocol matches `elevenlabs/`, the ElevenLabs transcriber is initiated.
    - If the protocol supports general OpenAI-compatible audio transcription endpoints (e.g., `openai`, `azure`, `groq`, `deepseek`), `AudioModelTranscriber` is leveraged.
2. **Fallback Scanning**: If no `model_name` is selected, it scans `model_list` specifically looking for `elevenlabs/` protocol models or `groq/` provider formats (e.g. for Whisper fallback). 
