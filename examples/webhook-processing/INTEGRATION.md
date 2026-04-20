# Integrating Webhook Processing into PicoClaw Gateway

This guide shows how to integrate the webhook processing feature into the main PicoClaw gateway.

## Step 1: Update Gateway Setup

Modify `pkg/gateway/gateway.go` to initialize the webhook processor:

```go
import (
    "github.com/sipeed/picoclaw/pkg/webhook"
)

// In setupAndStartServices function, after creating HealthServer:

// Setup webhook processor
webhookProcessor := webhook.CreateDefaultProcessor()
webhookHandler := webhook.NewHandler(webhookProcessor, authToken)
runningServices.HealthServer.SetWebhookHandler(webhookHandler)

// Start cleanup goroutine
go func() {
    ticker := time.NewTicker(30 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            webhookProcessor.CleanupOldJobs(2 * time.Hour)
            logger.Debug("Cleaned up old webhook jobs")
        case <-ctx.Done():
            ticker.Stop()
            return
        }
    }
}()
```

## Step 2: Custom Processor for Your Use Case

Create a custom processor that integrates with your agent loop:

```go
// In pkg/gateway/webhook_processor.go

package gateway

import (
    "context"
    "github.com/sipeed/picoclaw/pkg/agent"
    "github.com/sipeed/picoclaw/pkg/webhook"
)

func CreateAgentProcessor(agentLoop *agent.AgentLoop) *webhook.Processor {
    processorFn := func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
        // Extract prompt from payload
        prompt, ok := payload["prompt"].(string)
        if !ok {
            return nil, fmt.Errorf("missing 'prompt' field")
        }
        
        // Get channel and chat ID
        channel := "webhook"
        chatID := "async"
        if ch, ok := payload["channel"].(string); ok {
            channel = ch
        }
        if cid, ok := payload["chat_id"].(string); ok {
            chatID = cid
        }
        
        // Process through agent loop
        response, err := agentLoop.ProcessHeartbeat(ctx, prompt, channel, chatID)
        if err != nil {
            return nil, fmt.Errorf("agent processing failed: %w", err)
        }
        
        return map[string]interface{}{
            "response": response,
            "channel": channel,
            "chat_id": chatID,
        }, nil
    }
    
    return webhook.NewProcessor(processorFn)
}
```

Then use it in gateway setup:

```go
// In setupAndStartServices:
webhookProcessor := CreateAgentProcessor(agentLoop)
webhookHandler := webhook.NewHandler(webhookProcessor, authToken)
runningServices.HealthServer.SetWebhookHandler(webhookHandler)
```

## Step 3: Update Gateway Startup Message

Add webhook endpoint info to the startup message:

```go
// In gateway.go, after printing health endpoints:

fmt.Printf("✓ Webhook endpoints available:\n")
fmt.Printf("  POST   http://%s/webhook/process - Submit async job\n", healthAddr)
fmt.Printf("  GET    http://%s/webhook/status  - Check job status\n", healthAddr)
```

## Step 4: Add Configuration Options

Add webhook settings to `pkg/config/config.go`:

```go
type WebhookConfig struct {
    Enabled         bool          `yaml:"enabled" json:"enabled"`
    MaxJobs         int           `yaml:"max_jobs" json:"max_jobs"`
    JobRetention    time.Duration `yaml:"job_retention" json:"job_retention"`
    ProcessTimeout  time.Duration `yaml:"process_timeout" json:"process_timeout"`
}

type GatewayConfig struct {
    // ... existing fields ...
    Webhook WebhookConfig `yaml:"webhook" json:"webhook"`
}
```

Default config in `config/config.yaml`:

```yaml
gateway:
  # ... existing config ...
  webhook:
    enabled: true
    max_jobs: 100
    job_retention: 2h
    process_timeout: 5m
```

## Step 5: Use Cases

### AI Agent Processing

Process prompts through your AI agent asynchronously:

```bash
curl -X POST http://localhost:18800/webhook/process \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://your-app.com/ai-callback",
    "payload": {
      "prompt": "Analyze this data and generate a report",
      "context": {
        "user_id": "123",
        "session_id": "abc"
      }
    }
  }'
```

### External Tool Processing

Integrate with external tools that need async responses:

```bash
curl -X POST http://localhost:18800/webhook/process \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://zapier.com/hooks/catch/...",
    "payload": {
      "action": "generate_summary",
      "document_url": "https://example.com/doc.pdf"
    }
  }'
```

### Scheduled Tasks with Webhooks

Combine with cron for scheduled jobs that report back:

```json
{
  "schedule": "0 9 * * *",
  "command": "curl -X POST http://localhost:18800/webhook/process ...",
  "description": "Daily report generation with webhook callback"
}
```

## Architecture Benefits

1. **Non-blocking**: Gateway remains responsive during long operations
2. **Scalable**: Can handle many concurrent processing jobs
3. **Reliable**: Jobs tracked with status, can check progress
4. **Flexible**: Easy to customize processor for different use cases
5. **Simple**: No external queue service needed for basic async processing

## Production Considerations

For production deployments:

1. **Persistent Storage**: Store job state in Redis/database instead of memory
2. **Retry Logic**: Add exponential backoff for webhook delivery failures
3. **Rate Limiting**: Add rate limits per client/token
4. **Monitoring**: Add metrics for job processing times, failure rates
5. **Queue System**: Consider Cloud Tasks or Pub/Sub for horizontal scaling

## Next Steps

1. Implement the integration in `pkg/gateway/gateway.go`
2. Test with the example scripts
3. Customize the processor for your specific use case
4. Add monitoring and logging
5. Deploy and test with real webhook receivers
