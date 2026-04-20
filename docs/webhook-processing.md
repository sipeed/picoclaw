# Webhook Processing

PicoClaw Gateway supports asynchronous request processing with webhook callbacks. This allows you to submit long-running tasks via HTTP, receive an immediate response, and get the result delivered to your webhook URL when processing completes.

## Quick Start

### Submit a Processing Job

```bash
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://your-app.com/callback",
    "payload": {
      "data": "your data here"
    }
  }'
```

**Response (202 Accepted):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "timestamp": "2026-04-17T10:00:00Z"
}
```

### Check Job Status

```bash
curl "http://localhost:18800/api/webhook/status?job_id=550e8400-e29b-41d4-a716-446655440000"
```

**Response:**
```json
{
  "ID": "550e8400-e29b-41d4-a716-446655440000",
  "Status": "completed",
  "CreatedAt": "2026-04-17T10:00:00Z",
  "CompletedAt": "2026-04-17T10:00:02Z"
}
```

### Receive Webhook Callback

When processing completes, PicoClaw will POST to your `webhook_url`:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": {
    "processed_data": "result here"
  },
  "timestamp": "2026-04-17T10:00:02Z"
}
```

## Architecture

The webhook processing system provides:

1. **Non-blocking API**: Submit job and get immediate response
2. **Background Processing**: Jobs execute asynchronously in goroutines
3. **Status Tracking**: Check job progress at any time
4. **Webhook Delivery**: Results automatically sent to your callback URL
5. **Error Handling**: Failures reported via webhook with error details

## Implementation Details

### Core Components

- **`pkg/webhook/processor.go`**: Core async processing engine
- **`pkg/webhook/handler.go`**: HTTP request handlers  
- **`pkg/webhook/example_processor.go`**: Example implementation
- **`web/backend/api/webhook.go`**: API endpoints integrated with web backend

### Integration Points

The webhook processor integrates with PicoClaw's existing infrastructure:

- Integrated with web backend API (`/api/webhook/*`)
- Uses the same authentication as other API endpoints
- Leverages existing logging system
- Can integrate with agent loop for AI processing
- Lazy initialization - processor created on first use

### Security

- Follows the same authentication pattern as other API endpoints
- Optional authentication can be added via middleware
- HTTPS recommended for production webhook callbacks
- Rate limiting can be added at the API layer

## Use Cases

### 1. AI Agent Processing

Process prompts through your AI agent asynchronously:

```json
{
  "webhook_url": "https://your-app.com/ai-response",
  "payload": {
    "prompt": "Analyze this dataset and generate insights",
    "channel": "api",
    "chat_id": "user-123"
  }
}
```

### 2. Document Processing

Handle large document transformations:

```json
{
  "webhook_url": "https://your-app.com/document-ready",
  "payload": {
    "document_url": "https://example.com/large.pdf",
    "operations": ["extract_text", "summarize", "translate"]
  }
}
```

### 3. Integration with External Services

Bridge to services like Zapier, Make, or custom webhooks:

```json
{
  "webhook_url": "https://hooks.zapier.com/...",
  "payload": {
    "action": "process_order",
    "order_id": "12345"
  }
}
```

### 4. Scheduled Background Jobs

Combine with cron for recurring tasks with callbacks:

```json
{
  "webhook_url": "https://monitoring.example.com/report",
  "payload": {
    "report_type": "daily_summary",
    "date": "2026-04-17"
  }
}
```

## Configuration

Currently uses built-in defaults. Future configuration options:

```yaml
gateway:
  webhook:
    enabled: true
    max_jobs: 100
    job_retention: 2h
    process_timeout: 5m
    webhook_timeout: 30s
    max_retries: 3
```

## Examples

Complete examples are available in [`examples/webhook-processing/`](../examples/webhook-processing/):

- **`README.md`**: Comprehensive documentation
- **`main.go`**: Standalone example server
- **`test.sh`**: Automated testing script
- **`curl-examples.sh`**: Quick curl command reference
- **`INTEGRATION.md`**: Guide for gateway integration

## Testing

### Using webhook.site

For quick testing without setting up a webhook receiver:

1. Visit [https://webhook.site](https://webhook.site)
2. Copy your unique URL
3. Use it as the `webhook_url` in your request
4. Watch callbacks arrive in real-time

### Running the Example

```bash
# Start the example server
cd examples/webhook-processing
go run main.go

# In another terminal, run tests
./test.sh

# Or use the curl examples
./curl-examples.sh
```

## API Reference

### POST /api/webhook/process

Submit a new processing job.

**Headers:**
- `Authorization: Bearer <token>` (required if auth enabled)
- `Content-Type: application/json`

**Request Body:**
```json
{
  "webhook_url": "string (required)",
  "payload": "object (optional)"
}
```

**Response Codes:**
- `202 Accepted`: Job submitted successfully
- `400 Bad Request`: Invalid request body or missing webhook_url
- `401 Unauthorized`: Invalid or missing auth token

### GET /api/webhook/status

Check the status of a submitted job.

**Query Parameters:**
- `job_id`: UUID of the job (required)

**Response Codes:**
- `200 OK`: Job found, status returned
- `400 Bad Request`: Missing job_id parameter
- `404 Not Found`: Job not found

**Response Body:**
```json
{
  "ID": "string",
  "WebhookURL": "string",
  "Status": "processing|completed|failed",
  "CreatedAt": "timestamp",
  "CompletedAt": "timestamp (nullable)"
}
```

## Production Considerations

For production deployments, consider:

1. **Persistent Storage**: Use Redis or database instead of in-memory storage
2. **Retry Logic**: Add exponential backoff for webhook delivery failures
3. **Rate Limiting**: Prevent abuse with per-client rate limits
4. **Monitoring**: Track processing times, success rates, webhook delivery
5. **Scaling**: Use Cloud Tasks or Pub/Sub for distributed processing
6. **Webhook Verification**: Add HMAC signatures for webhook authenticity
7. **Timeout Handling**: Configure appropriate timeouts for different job types

## Extending

### Custom Processor Functions

Implement your own processing logic:

```go
func MyCustomProcessor(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
    // Your processing logic here
    result := processData(payload)
    
    return map[string]interface{}{
        "result": result,
    }, nil
}

processor := webhook.NewProcessor(MyCustomProcessor)
```

### Integration with Agent Loop

Process jobs through PicoClaw's agent system:

```go
func AgentProcessor(agentLoop *agent.AgentLoop) webhook.ProcessorFunc {
    return func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
        prompt := payload["prompt"].(string)
        response, err := agentLoop.ProcessHeartbeat(ctx, prompt, "webhook", "async")
        return map[string]interface{}{"response": response}, err
    }
}
```

## Troubleshooting

### Webhook Not Called

- Check webhook URL is accessible from gateway
- Verify webhook endpoint accepts POST requests
- Check firewall/network rules
- Review gateway logs for delivery errors

### Jobs Stuck in Processing

- Check processor timeout settings
- Review logs for panics or deadlocks
- Verify context cancellation handling
- Monitor goroutine counts

### Authentication Failures

- Verify token matches PID file token
- Check Authorization header format
- Ensure token is passed correctly in requests

## Related Documentation

- [Gateway Configuration](./configuration.md)
- [Health Endpoints](./health-endpoints.md)
- [Security](./security.md)
- [Integration Guide](../examples/webhook-processing/INTEGRATION.md)
