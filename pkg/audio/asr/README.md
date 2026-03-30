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

1. **Targeted Selection**: Resolve `cfg.Voice.ModelName` against `model_list`, then create the transcriber from that resolved model entry.
    - This means aliases such as `my-asr-model` are the primary ASR contract.
    - If the resolved model uses `elevenlabs/...`, the ElevenLabs transcriber is initiated.
    - If the resolved model uses an OpenAI-compatible Whisper model name such as `openai/whisper-1` or `groq/whisper-large-v3`, the Whisper transcriber is initiated.
    - If the resolved model uses an OpenAI-compatible audio-capable provider (for example `openai`, `azure`, `gemini`, `deepseek`), `AudioModelTranscriber` is leveraged.
2. **Fallback Scanning**: If no `model_name` is selected, PicoClaw performs a compatibility scan through `model_list` for legacy auto-detected ASR providers such as `elevenlabs/...` entries and OpenAI-compatible Whisper models.

Fallback scanning exists for compatibility, but the recommended configuration is to set `voice.model_name` to a named `model_list` entry such as `my-asr-model`.
