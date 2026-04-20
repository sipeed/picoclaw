# Webhook Processing Feature - Changelog

## Added - Webhook Async Processing (2026-04-17)

### New Features

#### Asynchronous Webhook Processing API

Added complete webhook-based asynchronous processing system to PicoClaw web backend.

**New API Endpoints:**

- `POST /api/webhook/process` - Submit async jobs with webhook callbacks
- `GET /api/webhook/status` - Query job status

**Core Functionality:**

- Submit long-running tasks via HTTP POST
- Immediate response with job ID (202 Accepted)
- Background processing in goroutines
- Automatic webhook delivery when complete
- Job status tracking and queries
- Automatic cleanup of old jobs (2-hour retention)

**Implementation Details:**

- **Package**: `pkg/webhook/` - Core processing engine
  - `processor.go` - Job management, execution, webhook delivery
  - `handler.go` - HTTP authentication helpers
  - `example_processor.go` - Default processor implementation

- **API Integration**: `web/backend/api/webhook.go`
  - REST endpoints integrated with web backend
  - Lazy initialization on first use
  - Periodic cleanup goroutine
  - Comprehensive test coverage

- **Documentation**:
  - `docs/webhook-processing.md` - Official documentation
  - `docs/api/openapi.yaml` - OpenAPI specification updated
  - `examples/webhook-processing/` - Complete examples and guides

**Use Cases:**

- AI agent processing asynchronously
- Document processing and transformations
- External service integration (Zapier, Make, etc.)
- Batch operations
- Scheduled background tasks

**Architecture:**

```
Client → POST /api/webhook/process → Returns 202 with job_id
                                    → Background goroutine processes
                                    → POSTs result to webhook_url
```

**Example Usage:**

```bash
# Submit job
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://your-app.com/callback",
    "payload": {"data": "process this"}
  }'

# Response: {"job_id": "uuid", "status": "processing"}

# Check status
curl "http://localhost:18800/api/webhook/status?job_id=uuid"

# Your webhook receives:
# {"job_id": "uuid", "status": "completed", "result": {...}}
```

**Testing:**

- Unit tests: `web/backend/api/webhook_test.go`
- Integration tests: `examples/webhook-processing/test.sh`
- Example server: `examples/webhook-processing/main.go`

**Security:**

- Follows web backend authentication patterns
- Optional bearer token authentication
- HTTPS recommended for webhook callbacks
- Rate limiting can be added at API layer

**Scalability:**

- In-memory job storage (suitable for moderate traffic)
- Goroutine-based concurrency
- For high-scale production:
  - Use Redis/PostgreSQL for job persistence
  - Implement Cloud Tasks or Pub/Sub
  - Add webhook retry with exponential backoff

### Files Added

**Core Implementation:**
- `pkg/webhook/processor.go`
- `pkg/webhook/handler.go`
- `pkg/webhook/example_processor.go`
- `web/backend/api/webhook.go`
- `web/backend/api/webhook_test.go`

**Documentation:**
- `docs/webhook-processing.md`
- `docs/api/openapi.yaml` (updated)
- `WEBHOOK_IMPLEMENTATION.md`

**Examples:**
- `examples/webhook-processing/README.md`
- `examples/webhook-processing/INTEGRATION.md`
- `examples/webhook-processing/ARCHITECTURE.md`
- `examples/webhook-processing/main.go`
- `examples/webhook-processing/test.sh`
- `examples/webhook-processing/curl-examples.sh`

### Files Modified

- `web/backend/api/router.go` - Added webhook route registration
- `go.mod` - No new dependencies (uses existing `github.com/google/uuid`)

### Breaking Changes

None. This is a purely additive feature.

### Migration Guide

No migration needed. The webhook endpoints are available immediately when the web backend starts.

### Configuration

No configuration required. The feature works out of the box with sensible defaults:

- Job retention: 2 hours
- Cleanup interval: 30 minutes
- Processing timeout: 5 minutes per job
- Webhook timeout: 30 seconds

Future configuration options can be added to `config.yaml`:

```yaml
webhook:
  enabled: true
  max_jobs: 100
  job_retention: 2h
  process_timeout: 5m
  webhook_timeout: 30s
```

### Dependencies

- Existing: `github.com/google/uuid` v1.6.0 (already in go.mod)
- No new external dependencies

### Backward Compatibility

Fully backward compatible. No existing functionality affected.

### Performance Impact

- Minimal overhead when not in use (lazy initialization)
- Each job runs in its own goroutine
- Cleanup runs every 30 minutes in background
- Memory usage: ~1KB per active job

### Known Limitations

1. **In-memory storage** - Jobs lost on server restart
2. **No webhook retries** - Failed webhooks not retried automatically
3. **No job persistence** - Not suitable for critical long-term jobs
4. **No distributed support** - Single-server only

For production at scale, see `ARCHITECTURE.md` for recommendations on using Redis, Cloud Tasks, or Pub/Sub.

### Future Enhancements

Potential improvements for future versions:

- [ ] Persistent job storage (Redis/PostgreSQL)
- [ ] Webhook retry with exponential backoff
- [ ] HMAC signatures for webhook authenticity
- [ ] Job priority levels
- [ ] Rate limiting per client
- [ ] Job scheduling (delayed execution)
- [ ] Batch job submission
- [ ] Job cancellation endpoint
- [ ] Webhook delivery status tracking
- [ ] Metrics and monitoring integration

### Testing

```bash
# Run unit tests
go test github.com/sipeed/picoclaw/web/backend/api -v -run TestWebhook

# Run integration tests
cd examples/webhook-processing
./test.sh

# Test with real webhook receiver
# 1. Visit https://webhook.site
# 2. Copy your unique URL
# 3. Run:
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{"webhook_url": "https://webhook.site/your-id", "payload": {"test": true}}'
```

### References

- [Webhook Processing Documentation](../webhook-processing.md)
- [OpenAPI Specification](openapi.yaml)
- [Integration Guide](../../examples/webhook-processing/INTEGRATION.md)
- [Architecture Design](../../examples/webhook-processing/ARCHITECTURE.md)
- [Quick Start Guide](../../WEBHOOK_IMPLEMENTATION.md)
