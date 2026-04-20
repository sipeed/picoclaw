# Webhook Processing with PicoClaw AI Integration

## Overview

The webhook processing feature automatically integrates with PicoClaw's AI agent to process requests intelligently. When you submit a job with a `prompt` field, PicoClaw's AI will process it and return the response.

## How It Works

```
User → POST /api/webhook/process
       {
         "webhook_url": "https://your-app.com/callback",
         "payload": {
           "prompt": "Your question here"
         }
       }
       ↓
Backend creates job → Returns job_id immediately (202)
       ↓
Background processor:
  1. Extracts prompt from payload
  2. Connects to PicoClaw AI via WebSocket
  3. Sends prompt to AI
  4. Collects AI response
  5. POSTs result to webhook_url
       ↓
Your webhook receives:
{
  "job_id": "uuid",
  "status": "completed",
  "result": {
    "data": "<AI response here>",
    "error": null
  },
  "timestamp": "2026-04-17T10:00:05Z"
}
```

## Request Format

### With Prompt (AI Processing)

```json
{
  "webhook_url": "https://your-app.com/callback",
  "payload": {
    "prompt": "Explain quantum computing in simple terms"
  }
}
```

The AI will process your prompt and return an intelligent response in the `result.data` field.

### Without Prompt (Example Processing)

```json
{
  "webhook_url": "https://your-app.com/callback",
  "payload": {
    "data": "some data",
    "other": "fields"
  }
}
```

If no `prompt` field is provided, falls back to example processor (returns dummy data).

## Response Format

### Success Response

```json
{
  "job_id": "27c383ee-9884-452a-bc24-c61507b19f18",
  "status": "completed",
  "result": {
    "data": "Quantum computing uses quantum bits or 'qubits' instead of regular bits...",
    "error": null
  },
  "timestamp": "2026-04-17T08:01:27Z"
}
```

### Error Response

```json
{
  "job_id": "27c383ee-9884-452a-bc24-c61507b19f18",
  "status": "failed",
  "error": "AI processing failed: connection timeout",
  "timestamp": "2026-04-17T08:01:27Z"
}
```

## Requirements

For AI processing to work, ensure:

1. **Gateway Running**: PicoClaw gateway must be running
   ```bash
   # Start gateway if not running
   curl -X POST http://localhost:18800/api/gateway/start
   ```

2. **Pico Channel Enabled**: The Pico channel must be configured
   ```bash
   # Check status
   curl http://localhost:18800/api/pico/token
   ```

3. **Model Configured**: A default AI model must be set
   ```bash
   # Check model
   curl http://localhost:18800/api/config | jq .agents.defaults.model_name
   ```

## Examples

### Example 1: Simple Question

**Request:**
```bash
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/your-id",
    "payload": {
      "prompt": "What is the capital of France?"
    }
  }'
```

**Response to webhook:**
```json
{
  "job_id": "abc-123",
  "status": "completed",
  "result": {
    "data": "The capital of France is Paris.",
    "error": null
  },
  "timestamp": "2026-04-17T10:00:05Z"
}
```

### Example 2: Code Generation

**Request:**
```bash
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://your-app.com/webhook",
    "payload": {
      "prompt": "Write a Python function to calculate fibonacci numbers"
    }
  }'
```

**Response to webhook:**
```json
{
  "job_id": "def-456",
  "status": "completed",
  "result": {
    "data": "def fibonacci(n):\n    if n <= 1:\n        return n\n    return fibonacci(n-1) + fibonacci(n-2)",
    "error": null
  },
  "timestamp": "2026-04-17T10:00:10Z"
}
```

### Example 3: Data Analysis

**Request:**
```bash
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://your-app.com/analysis",
    "payload": {
      "prompt": "Analyze this data: [1, 5, 3, 8, 2, 9] and provide statistics"
    }
  }'
```

**Response to webhook:**
```json
{
  "job_id": "ghi-789",
  "status": "completed",
  "result": {
    "data": "Analysis of [1, 5, 3, 8, 2, 9]:\n- Mean: 4.67\n- Median: 4\n- Range: 8\n- Min: 1\n- Max: 9",
    "error": null
  },
  "timestamp": "2026-04-17T10:00:15Z"
}
```

## Automatic Fallback

If the AI processor cannot initialize (gateway not running, Pico channel not configured), the system automatically falls back to the example processor:

```json
{
  "job_id": "fallback-123",
  "status": "completed",
  "result": {
    "processed_data": { "your": "payload" },
    "processed_at": "2026-04-17T10:00:00Z",
    "message": "Processing completed successfully"
  },
  "timestamp": "2026-04-17T10:00:02Z"
}
```

Check logs to see which processor is active:
```bash
# View backend logs
tail -f ~/.picoclaw/logs/launcher.log | grep webhook
```

You'll see either:
- `Initializing webhook processor with PicoClaw AI` (AI enabled)
- `Pico WebSocket not available, using example processor` (fallback)

## Testing

### Test with webhook.site

1. Visit https://webhook.site and copy your URL

2. Submit a job with AI prompt:
```bash
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/YOUR-ID",
    "payload": {
      "prompt": "Tell me a programming joke"
    }
  }'
```

3. Watch webhook.site for the AI's response!

### Test AI Availability

```bash
# Check if AI is available
curl http://localhost:18800/api/gateway/status

# Check Pico channel
curl http://localhost:18800/api/pico/token

# Submit test prompt
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/test",
    "payload": {
      "prompt": "Say hello"
    }
  }'
```

## Troubleshooting

### AI Not Responding

**Symptom:** Webhook receives example data instead of AI response

**Solutions:**
1. Check gateway is running:
   ```bash
   curl http://localhost:18800/api/gateway/status
   ```

2. Check Pico channel:
   ```bash
   curl http://localhost:18800/api/pico/token
   # Should return a token, not empty
   ```

3. Check logs:
   ```bash
   tail -f ~/.picoclaw/logs/launcher.log | grep webhook
   ```

### Timeout Errors

**Symptom:** `"error": "AI processing failed: context deadline exceeded"`

**Solutions:**
- Complex prompts may take longer
- Default timeout is 5 minutes
- Check gateway logs for actual AI response time

### Connection Refused

**Symptom:** `"error": "failed to connect to PicoClaw"`

**Solutions:**
1. Ensure gateway is running on port 18790
2. Ensure backend is running on port 18800
3. Check firewall settings

## Advanced Usage

### Custom Context

Pass additional context to the AI:

```json
{
  "webhook_url": "https://your-app.com/webhook",
  "payload": {
    "prompt": "Based on the following data: [user context here], answer: [your question]"
  }
}
```

### Multiple Requests

Process multiple prompts in parallel:

```bash
# Submit job 1
curl -X POST http://localhost:18800/api/webhook/process \
  -d '{"webhook_url": "https://webhook.site/id1", "payload": {"prompt": "Question 1"}}'

# Submit job 2  
curl -X POST http://localhost:18800/api/webhook/process \
  -d '{"webhook_url": "https://webhook.site/id2", "payload": {"prompt": "Question 2"}}'

# Both process in parallel!
```

### Webhook Chaining

Chain webhooks together:

```javascript
// Your webhook endpoint
app.post('/webhook', async (req, res) => {
  const { job_id, result } = req.body;
  
  // Process AI response
  const aiResponse = result.data;
  
  // Submit follow-up question
  await fetch('http://localhost:18800/api/webhook/process', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      webhook_url: 'https://your-app.com/webhook2',
      payload: {
        prompt: `Follow up on: ${aiResponse}`
      }
    })
  });
  
  res.sendStatus(200);
});
```

## Performance

- **Response Time**: 2-30 seconds depending on prompt complexity
- **Concurrent Jobs**: Unlimited (each runs in own goroutine)
- **Rate Limiting**: Respects model's RPM limits
- **Timeout**: 5 minutes per job (configurable)

## Security

- WebSocket connections use token authentication
- Tokens stored securely in config
- localhost-only connections (backend → gateway)
- HTTPS recommended for webhook callbacks

## Related Documentation

- [Webhook Processing Guide](webhook-processing.md)
- [Postman Quick Start](api/WEBHOOK_POSTMAN_QUICKSTART.md)
- [Integration Guide](../examples/webhook-processing/INTEGRATION.md)
