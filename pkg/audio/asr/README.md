# ASR (Automatic Speech Recognition)

This package handles speech-to-text for PicoClaw voice input.

If you are new to ASR setup, the simplest mental model is:

1. Add one or more ASR-capable entries to `model_list`.
2. Point `voice.model_name` at the one you want to use.
3. Put the API key in `.security.yml`.

## Quick Recommendation

For most new users, start with one of these:

| Provider | Example model | Why start here |
| --- | --- | --- |
| [Groq](https://console.groq.com/keys) | `groq/whisper-large-v3-turbo` | Fast Whisper-style transcription and a straightforward OpenAI-compatible API. Groq currently advertises a free tier plan for 2000 reqs/day. |
| [ElevenLabs](https://elevenlabs.io/pricing) | `elevenlabs/scribe_v1` | Easy setup and strong speech-to-text quality. ElevenLabs currently advertises a free plan that includes speech-to-text usage. |

Pricing and free-plan limits can change, so check the linked pricing pages before depending on them in production.

## How ASR Configuration Works

PicoClaw does not keep ASR API keys inside the `voice` section.

Instead:

- `voice.model_name` chooses a named entry from `model_list`.
- The matching `model_list` entry describes the actual provider and model.
- `.security.yml` stores the API key for that named model entry.

This is the recommended pattern because it is explicit, reusable, and consistent with the rest of PicoClaw's model configuration.

## Recommended Setup

### Option A: Groq Whisper

`config.json`

```json
{
  "voice": {
    "model_name": "groq-asr",
    "echo_transcription": true
  },
  "model_list": [
    {
      "model_name": "groq-asr",
      "model": "groq/whisper-large-v3-turbo"
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  groq-asr:
    api_keys:
      - "gsk_your_groq_key"
```

Notes:

- You can omit `api_base` and PicoClaw will use Groq's default API base automatically.
- If you set `api_base` manually for Groq Whisper, both of these forms work:
  - `https://api.groq.com/openai/v1`
  - `https://api.groq.com/openai/v1/audio/transcriptions`
- Any OpenAI-compatible Whisper model name containing `whisper` can use the Whisper transcription path, not only `whisper-large-v3-turbo`.

### Option B: ElevenLabs

`config.json`

```json
{
  "voice": {
    "model_name": "elevenlabs-asr",
    "echo_transcription": true
  },
  "model_list": [
    {
      "model_name": "elevenlabs-asr",
      "provider": "elevenlabs",
      "model": "scribe_v1"
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  elevenlabs-asr:
    api_keys:
      - "sk-elevenlabs-your-key"
```

### Option C: OpenAI Whisper

`config.json`

```json
{
  "voice": {
    "model_name": "openai-asr"
  },
  "model_list": [
    {
      "model_name": "openai-asr",
      "model": "openai/whisper-1"
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  openai-asr:
    api_keys:
      - "sk-openai-your-key"
```

## Other ASR-Capable Model Types

PicoClaw currently supports three main ASR routes:

| Route | Example models | Behavior |
| --- | --- | --- |
| ElevenLabs ASR | `provider: elevenlabs`, `model: scribe_v1` | Uses the ElevenLabs transcription API. |
| Whisper endpoint models | `openai/whisper-1`, `groq/whisper-large-v3` | Uses an OpenAI-compatible `/audio/transcriptions` endpoint. |
| Audio-capable chat models **(Under construction)** | `openai/gpt-4o-audio-preview`, `gemini/gemini-2.5-flash` | Sends audio to a multimodal chat model and asks it to transcribe. |

If you are unsure which one to pick, choose Groq Whisper or ElevenLabs first.

## How PicoClaw Chooses a Transcriber

`DetectTranscriber` resolves ASR in this order:

1. **Preferred path**: resolve `voice.model_name` against `model_list`.
2. If that resolved model is:
   - an `elevenlabs` provider model, PicoClaw uses the ElevenLabs transcriber.
   - an OpenAI-compatible Whisper model, PicoClaw uses the Whisper transcriber.
   - an audio-capable chat model, PicoClaw uses `AudioModelTranscriber`.
3. **Fallback path**: if `voice.model_name` is not set, PicoClaw performs a compatibility scan through `model_list` for legacy auto-detected ASR entries.

Fallback scanning exists for backward compatibility. New configurations should set `voice.model_name` explicitly.

## Common Mistakes

- Defining an ASR model in `model_list` but forgetting to set `voice.model_name`.
- Putting the API key in `voice` instead of `.security.yml`.
- Using a non-ASR model and expecting Whisper-style transcription behavior.
- Setting a custom `api_base` that points to the wrong provider endpoint.

## Minimal Checklist

Before testing voice input, make sure:

- `voice.model_name` matches a `model_list[].model_name`.
- The matching `.security.yml` entry contains a valid API key.
- The selected model is actually ASR-capable.
- Voice input is enabled for the channel you are using.
