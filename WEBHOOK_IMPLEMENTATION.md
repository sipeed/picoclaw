# Webhook Processing Implementation Summary

## ✅ What Was Implemented

A complete **asynchronous webhook processing system** has been added to PicoClaw, allowing you to:

1. Submit long-running tasks via HTTP POST
2. Get an immediate response with a job ID
3. Receive results via webhook callback when processing completes

## 📁 File Structure

### Core Processing Engine (`pkg/webhook/`)
- **`processor.go`** - Job submission, tracking, background processing, webhook delivery
- **`handler.go`** - HTTP authentication and request handling helpers  
- **`example_processor.go`** - Default processor implementation

### API Endpoints (`web/backend/api/`)
- **`webhook.go`** - REST API endpoints integrated with web backend
- **`webhook_test.go`** - Comprehensive test suite

### Integration
- **`router.go`** - Route registration in web backend

### Documentation & Examples (`examples/webhook-processing/`)
- **`README.md`** - User guide and API documentation
- **`INTEGRATION.md`** - Integration guide for custom processors
- **`ARCHITECTURE.md`** - System architecture and design
- **`main.go`** - Standalone example server
- **`test.sh`** - Automated testing script
- **`curl-examples.sh`** - Quick curl command reference

### Official Docs (`docs/`)
- **`webhook-processing.md`** - Official documentation

## 🎯 API Endpoints

### `POST /api/webhook/process`
Submit an async job with webhook callback:

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

### `GET /api/webhook/status?job_id=<id>`
Check job status:

```bash
curl "http://localhost:18800/api/webhook/status?job_id=550e8400-e29b-41d4-a716-446655440000"
```

**Response (200 OK):**
```json
{
  "ID": "550e8400-e29b-41d4-a716-446655440000",
  "Status": "completed",
  "CreatedAt": "2026-04-17T10:00:00Z",
  "CompletedAt": "2026-04-17T10:00:02Z"
}
```

### Webhook Callback
When processing completes, PicoClaw POSTs to your `webhook_url`:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": {
    "processed_data": "result here",
    "processed_at": "2026-04-17T10:00:02Z"
  },
  "timestamp": "2026-04-17T10:00:02Z"
}
```

## 🚀 Quick Start

### 1. Start PicoClaw Web Backend

The webhook endpoints are automatically available when you start the web backend:

```bash
cd web/backend
go build -o picoclaw-web .
./picoclaw-web
```

The endpoints will be available at:
- `http://localhost:18800/api/webhook/process`
- `http://localhost:18800/api/webhook/status`

### 2. Test with webhook.site

```bash
# Visit https://webhook.site and copy your unique URL
WEBHOOK_URL="https://webhook.site/your-unique-id"

# Submit a job
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "'$WEBHOOK_URL'",
    "payload": {"data": "test"}
  }'

# Watch the callback arrive at webhook.site!
```

### 3. Run Automated Tests

```bash
cd examples/webhook-processing
./test.sh
```

## 🏗️ Architecture

```
Client → POST /api/webhook/process → Handler → Processor → Goroutine
                                                               ↓
                                                         Process Job
                                                               ↓
                                                      POST to webhook_url
```

### Key Features

- **Non-blocking**: Returns immediately with job ID
- **Concurrent**: Each job runs in its own goroutine
- **Tracked**: Query status anytime via `/api/webhook/status`
- **Automatic Cleanup**: Old jobs cleaned up periodically
- **Lazy Init**: Processor created on first use
- **Tested**: Comprehensive test suite included

## 🔌 Integration

The webhook processing is **already integrated** with the web backend. No additional setup needed!

### How It Works

1. When you start the web backend, routes are registered in `router.go`
2. On first webhook request, the processor is lazily initialized
3. Jobs run in background goroutines
4. Results are POSTed to webhook URLs automatically
5. Old jobs are cleaned up every 30 minutes

### Custom Processor

To implement custom processing logic, create your own processor function:

```go
import (
    "context"
    "github.com/sipeed/picoclaw/pkg/webhook"
)

func MyProcessor(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
    // Your processing logic here
    data := payload["data"]
    result := processData(data)
    
    return map[string]interface{}{
        "result": result,
    }, nil
}

// Use it
processor := webhook.NewProcessor(MyProcessor)
```

See [`INTEGRATION.md`](examples/webhook-processing/INTEGRATION.md) for details.

## 📝 Use Cases

1. **AI Agent Processing** - Process prompts asynchronously
2. **Document Processing** - Convert, analyze, or summarize documents
3. **External API Integration** - Bridge to Zapier, Make, etc.
4. **Batch Operations** - Process multiple items in background
5. **Scheduled Tasks** - Combine with cron for recurring jobs

## 🧪 Testing

### Unit Tests
```bash
go test github.com/sipeed/picoclaw/web/backend/api -v -run TestWebhook
```

### Integration Test
```bash
cd examples/webhook-processing
./test.sh
```

### Manual Testing
```bash
# Use the curl examples
./curl-examples.sh
```

## 📚 Documentation

- **User Guide**: [`docs/webhook-processing.md`](docs/webhook-processing.md)
- **Examples**: [`examples/webhook-processing/README.md`](examples/webhook-processing/README.md)
- **Integration**: [`examples/webhook-processing/INTEGRATION.md`](examples/webhook-processing/INTEGRATION.md)
- **Architecture**: [`examples/webhook-processing/ARCHITECTURE.md`](examples/webhook-processing/ARCHITECTURE.md)

## ✨ Next Steps

1. **Start the web backend** - Endpoints are ready to use
2. **Test with webhook.site** - Quick validation
3. **Customize processor** - Implement your business logic
4. **Add to CI/CD** - Include tests in your pipeline
5. **Production deployment** - See scaling considerations in docs

## 🔒 Security

- Currently no authentication (relies on web backend auth middleware)
- Add bearer token auth if exposing publicly
- Use HTTPS for webhook callbacks
- Validate webhook URLs to prevent SSRF
- Consider rate limiting for production

## 📈 Scaling

Current implementation is suitable for:
- Development and testing
- Low to medium traffic
- Single server deployment

For production at scale, consider:
- Redis/PostgreSQL for job storage
- Google Cloud Tasks for job queue
- Horizontal scaling across multiple instances
- Webhook retry with exponential backoff

See [`ARCHITECTURE.md`](examples/webhook-processing/ARCHITECTURE.md) for details.

## ❓ FAQ

**Q: Do I need to configure anything?**  
A: No! It's already integrated and ready to use when you start the web backend.

**Q: Is authentication required?**  
A: The endpoints use the same authentication as other `/api/*` endpoints.

**Q: Can I customize the processing logic?**  
A: Yes! See [`INTEGRATION.md`](examples/webhook-processing/INTEGRATION.md) for how to create custom processors.

**Q: What happens if the webhook URL is down?**  
A: Currently no retry. For production, implement retry logic with exponential backoff.

**Q: How long are jobs retained?**  
A: Jobs are cleaned up after 2 hours by default. Configurable in the cleanup function.

**Q: Can I use this in production?**  
A: Yes for moderate traffic. For high-scale production, see scaling considerations in the architecture docs.

## 🎉 Summary

You now have a fully functional webhook processing system integrated into PicoClaw! The endpoints are live as soon as you start the web backend, with no additional configuration needed. Happy processing! 🚀
