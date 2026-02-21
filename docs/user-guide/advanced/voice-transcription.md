# Voice Transcription

PicoClaw supports voice message transcription using Groq's Whisper API. This allows users to send voice messages through supported channels and have them automatically transcribed for the AI to process.

## Overview

Voice transcription provides:

- **Automatic transcription** - Voice messages converted to text
- **Multi-language support** - Whisper supports 99+ languages
- **Fast processing** - Groq's infrastructure provides quick transcription
- **Seamless integration** - Transcribed text feeds directly into conversation

## Requirements

- Groq API key configured
- Channel that supports voice messages (Telegram, Discord, etc.)
- Audio format supported by Whisper (MP3, WAV, M4A, etc.)

## Configuration

### Groq Provider Setup

Configure the Groq provider with your API key:

```json
{
  "providers": {
    "groq": {
      "api_key": "gsk_xxx"
    }
  }
}
```

### Environment Variable

Alternatively, use environment variables:

```bash
export PICOCLAW_PROVIDERS_GROQ_API_KEY="gsk_xxx"
```

## How It Works

### Processing Flow

1. User sends voice message to channel (Telegram, Discord, etc.)
2. Channel downloads audio file
3. PicoClaw sends audio to Groq Whisper API
4. Transcription text is received
5. Transcribed text is processed as a regular message

### Architecture

```
Voice Message
     │
     ▼
┌─────────────────┐
│ Channel Handler │
│ (Telegram/etc)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Voice Download  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Groq Whisper    │
│ API             │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Transcribed     │
│ Text Message    │
└─────────────────┘
```

## Supported Formats

Groq Whisper supports common audio formats:

| Format | Extension | Notes |
|--------|-----------|-------|
| MP3 | `.mp3` | Most common |
| WAV | `.wav` | Uncompressed |
| M4A | `.m4a` | Apple format |
| WebM | `.webm` | Web format |
| OGG | `.ogg` | Telegram default |
| FLAC | `.flac` | Lossless |
| MP4 | `.mp4` | Video container (audio extracted) |

## Channel Support

Voice transcription is supported in channels that implement voice message handling:

### Telegram

Telegram voice messages (OGG format) are automatically transcribed:

1. User sends voice message
2. Telegram downloads audio
3. Audio is transcribed
4. Text is processed

### Discord

Discord voice messages are transcribed when available:

1. User uploads voice message
2. Audio file is downloaded
3. Audio is transcribed
4. Text is processed

## API Details

### Endpoint

Voice transcription uses Groq's OpenAI-compatible endpoint:

```
POST https://api.groq.com/openai/v1/audio/transcriptions
```

### Model

PicoClaw uses `whisper-large-v3` for transcription:

- High accuracy
- Multi-language support
- Fast processing on Groq infrastructure

### Response Format

```json
{
  "text": "Hello, this is a transcribed message.",
  "language": "en",
  "duration": 5.2
}
```

## Configuration Options

### Enable/Disable

Voice transcription is enabled when a Groq API key is configured. To disable:

```bash
# Remove or empty the API key
unset PICOCLAW_PROVIDERS_GROQ_API_KEY
```

### Timeout

Voice transcription has a 60-second timeout for API requests. Long audio files may take longer to process.

## Debugging

Enable debug mode to see transcription details:

```bash
picoclaw gateway --debug
```

### Debug Output

```
[voice] Starting transcription: voice_message.ogg
[voice] Audio file details: size_bytes=245760, file_name=voice_message.ogg
[voice] Sending transcription request to Groq API
[voice] Transcription completed: text_length=45, language=en, duration=3.5
```

## Troubleshooting

### Transcription Not Working

1. **Check API key**: Ensure Groq API key is configured
   ```bash
   echo $PICOCLAW_PROVIDERS_GROQ_API_KEY
   ```

2. **Check channel support**: Verify channel handles voice messages

3. **Check audio format**: Ensure format is supported by Whisper

4. **Check logs**: Enable debug mode for detailed logging

### Poor Transcription Quality

1. **Audio quality**: Better quality audio yields better transcription
2. **Background noise**: Noisy audio may reduce accuracy
3. **Language**: Ensure correct language is being spoken
4. **Audio length**: Very short clips may have reduced accuracy

### API Errors

Common API errors:

| Error | Cause | Solution |
|-------|-------|----------|
| 401 Unauthorized | Invalid API key | Check API key |
| 413 Payload Too Large | Audio file too large | Compress or shorten audio |
| 429 Rate Limited | Too many requests | Wait and retry |
| 500 Server Error | Groq service issue | Retry later |

## Privacy Considerations

- Audio files are sent to Groq's servers for transcription
- Files are processed in real-time and not stored by Groq
- Consider privacy implications for sensitive conversations
- Review Groq's privacy policy for details

## Cost Considerations

Groq Whisper pricing:

- Free tier available with rate limits
- Paid tier for higher volume
- Monitor usage to avoid unexpected costs

Check [Groq's pricing page](https://groq.com/pricing) for current rates.

## Example Usage

### Telegram Voice Message

1. Open Telegram chat with bot
2. Hold microphone button and record message
3. Release to send
4. Bot transcribes and responds to text

### Discord Voice Message

1. Open Discord channel with bot
2. Upload or record voice message
3. Bot transcribes and responds to text

## Related Topics

- [Channels](../channels/README.md) - Configure chat platforms
- [Providers](../providers/README.md) - Configure LLM providers
- [Environment Variables](environment-variables.md) - Configuration options
