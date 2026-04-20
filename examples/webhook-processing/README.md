# Webhook Processing Example

This example demonstrates how to use the webhook processing feature in PicoClaw Gateway to handle asynchronous requests with callback webhooks.

## Overview

The webhook processor allows you to:
1. Accept a processing request via HTTP POST
2. Return immediately with a job ID (202 Accepted)
3. Process the request asynchronously in the background
4. Send the result to a webhook URL when complete

## Architecture

```
Client                Gateway               Webhook URL
  |                      |                       |
  |---POST /webhook/---->|                       |
  |    process           |                       |
  |                      |                       |
  |<--202 Accepted-------|                       |
  |    {job_id}          |                       |
  |                      |                       |
  |                      |---Processing--------->|
  |                      |                       |
  |                      |---POST Result-------->|
  |                      |                       |
```

## API Endpoints

### Submit Processing Job

**Endpoint:** `POST /webhook/process`

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "webhook_url": "https://your-app.com/callback",
  "payload": {
    "data": "your data here",
    "any_field": "any value"
  }
}
```

**Response:** `202 Accepted`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "timestamp": "2026-04-17T10:00:00Z"
}
```

### Check Job Status

**Endpoint:** `GET /webhook/status?job_id=<job_id>`

**Response:** `200 OK`
```json
{
  "ID": "550e8400-e29b-41d4-a716-446655440000",
  "WebhookURL": "https://your-app.com/callback",
  "Status": "completed",
  "CreatedAt": "2026-04-17T10:00:00Z",
  "CompletedAt": "2026-04-17T10:00:02Z"
}
```

### Webhook Callback

When processing completes, the gateway will POST to your `webhook_url`:

**Success Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": {
    "processed_data": "your processed result",
    "processed_at": "2026-04-17T10:00:02Z"
  },
  "timestamp": "2026-04-17T10:00:02Z"
}
```

**Error Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "failed",
  "error": "processing error message",
  "timestamp": "2026-04-17T10:00:02Z"
}
```

## Usage Example

### 1. Submit a Processing Job

```bash
curl -X POST http://localhost:18800/webhook/process \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/unique-id",
    "payload": {
      "data": "process this",
      "priority": "high"
    }
  }'
```

### 2. Check Job Status (Optional)

```bash
curl "http://localhost:18800/webhook/status?job_id=550e8400-e29b-41d4-a716-446655440000"
```

### 3. Receive Webhook Callback

Your webhook endpoint will receive a POST request with the processing result.

## Custom Processor Function

To implement your own processing logic:

```go
package main

import (
    "context"
    "github.com/sipeed/picoclaw/pkg/webhook"
)

// CustomProcessor implements your business logic
func CustomProcessor(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
    // Extract input
    data := payload["data"]
    
    // Do your processing here
    result := processYourData(data)
    
    // Return result
    return map[string]interface{}{
        "result": result,
        "success": true,
    }, nil
}

// In your gateway setup:
processor := webhook.NewProcessor(CustomProcessor)
handler := webhook.NewHandler(processor, authToken)
healthServer.SetWebhookHandler(handler)
```

## Integration with Gateway

The webhook processing is integrated with the PicoClaw gateway health server. To enable it:

1. The processor is automatically initialized with the gateway
2. Endpoints are available on the same port as health endpoints
3. Uses the same auth token as other protected endpoints

## Use Cases

- **AI/ML Inference:** Process AI model predictions asynchronously
- **Image/Video Processing:** Handle media transformations
- **Long-Running Tasks:** Any operation that takes more than a few seconds
- **External API Integration:** Call third-party APIs without blocking
- **Batch Operations:** Process multiple items in the background

## Configuration

The webhook processor uses the same configuration as the gateway:

- Port: Configured via `gateway.port` in config
- Auth: Uses the PID file token for authentication
- Timeout: Default 5 minutes per job
- Cleanup: Old jobs are retained in memory (can be configured)

## Testing with webhook.site

For quick testing, use [webhook.site](https://webhook.site):

1. Go to https://webhook.site and copy your unique URL
2. Use that URL as your `webhook_url` in the request
3. Watch the results arrive in real-time on the webhook.site dashboard
