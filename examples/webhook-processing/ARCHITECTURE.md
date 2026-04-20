# Webhook Processing Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        PicoClaw Gateway                              │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                   Health Server                              │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │   │
│  │  │   /health    │  │   /ready     │  │   /reload    │      │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘      │   │
│  │  ┌──────────────────────────────────────────────────┐      │   │
│  │  │         Webhook Endpoints                         │      │   │
│  │  │  ┌──────────────┐  ┌──────────────┐             │      │   │
│  │  │  │  /webhook/   │  │  /webhook/   │             │      │   │
│  │  │  │   process    │  │   status     │             │      │   │
│  │  │  └──────┬───────┘  └──────────────┘             │      │   │
│  │  └─────────┼──────────────────────────────────────┘      │   │
│  └────────────┼─────────────────────────────────────────────┘   │
│               │                                                    │
│  ┌────────────▼────────────────────────────────────────────────┐ │
│  │              Webhook Handler                                 │ │
│  │  ┌────────────────────────────────────────────────────┐    │ │
│  │  │  - Authentication (Bearer Token)                   │    │ │
│  │  │  - Request Validation                              │    │ │
│  │  │  - JSON Encoding/Decoding                          │    │ │
│  │  └────────────────────────────────────────────────────┘    │ │
│  └────────────┬────────────────────────────────────────────────┘ │
│               │                                                    │
│  ┌────────────▼────────────────────────────────────────────────┐ │
│  │             Webhook Processor                                │ │
│  │  ┌──────────────────────────────────────────────────┐       │ │
│  │  │  Job Management                                  │       │ │
│  │  │  - Job Queue (in-memory map)                     │       │ │
│  │  │  - UUID Generation                               │       │ │
│  │  │  - Status Tracking                               │       │ │
│  │  │  - Cleanup (old jobs)                            │       │ │
│  │  └──────────────────────────────────────────────────┘       │ │
│  │                                                               │ │
│  │  ┌──────────────────────────────────────────────────┐       │ │
│  │  │  Background Processing                           │       │ │
│  │  │  - Goroutine per job                             │       │ │
│  │  │  - Context with timeout                          │       │ │
│  │  │  - Custom processor function                     │       │ │
│  │  └──────────────────────────────────────────────────┘       │ │
│  │                                                               │ │
│  │  ┌──────────────────────────────────────────────────┐       │ │
│  │  │  Webhook Delivery                                │       │ │
│  │  │  - HTTP POST to callback URL                     │       │ │
│  │  │  - JSON payload with results                     │       │ │
│  │  │  - Error handling                                │       │ │
│  │  └──────────────────────────────────────────────────┘       │ │
│  └───────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

## Request Flow

### 1. Submit Job (POST /webhook/process)

```
Client                Handler              Processor              Background
  │                      │                     │                      │
  │─────POST───────────→│                     │                      │
  │  webhook_url         │                     │                      │
  │  payload             │                     │                      │
  │                      │                     │                      │
  │                      │──Auth Check────────→│                      │
  │                      │                     │                      │
  │                      │──Submit Job────────→│                      │
  │                      │                     │                      │
  │                      │                     │──Generate UUID──────→│
  │                      │                     │  Create Job          │
  │                      │                     │  Store in map        │
  │                      │                     │                      │
  │                      │                     │──Launch Goroutine───→│
  │                      │                     │                      │
  │                      │←─Job ID─────────────│                      │
  │                      │   Status            │                      │
  │                      │                     │                      │
  │←────202 Accepted────│                     │                      │
  │  {job_id, status}   │                     │                      │
  │                      │                     │                      │
  │                      │                     │                      │──Process──┐
  │                      │                     │                      │           │
  │                      │                     │                      │           │
  │                      │                     │                      │◄──────────┘
  │                      │                     │                      │
  │                      │                     │                      │──POST to──┐
  │                      │                     │                      │  webhook  │
  │                      │                     │                      │  URL      │
  │                      │                     │                      │◄──────────┘
```

### 2. Check Status (GET /webhook/status)

```
Client                Handler              Processor
  │                      │                     │
  │─────GET───────────→│                     │
  │  ?job_id=xxx         │                     │
  │                      │                     │
  │                      │──Get Job───────────→│
  │                      │                     │
  │                      │                     │──Lookup in map─────┐
  │                      │                     │                     │
  │                      │                     │◄────────────────────┘
  │                      │                     │
  │                      │←─Job Details────────│
  │                      │                     │
  │←────200 OK──────────│                     │
  │  {job status}       │                     │
```

### 3. Webhook Callback

```
Processor                               Client Webhook Endpoint
    │                                            │
    │──Process Job─────────────────────────────┐│
    │                                           ││
    │◄──────────────────────────────────────────┘│
    │                                            │
    │──Build Webhook Payload──────────────────┐ │
    │  {job_id, status, result}               │ │
    │◄─────────────────────────────────────────┘ │
    │                                            │
    │─────POST───────────────────────────────────→│
    │  webhook_url                               │
    │  JSON payload                              │
    │                                            │
    │◄─────200 OK────────────────────────────────│
    │                                            │
    │──Update Job Status──────────────────────┐ │
    │  CompletedAt = now                      │ │
    │◄─────────────────────────────────────────┘ │
```

## Component Relationships

```
┌──────────────────────────────────────────────────────────────┐
│                        Gateway Layer                          │
│                                                                │
│  pkg/gateway/gateway.go                                       │
│  └─── setupAndStartServices()                                 │
│       └─── Creates and configures webhook processor           │
│                                                                │
└────────────────────┬─────────────────────────────────────────┘
                     │
                     │ integrates
                     ▼
┌──────────────────────────────────────────────────────────────┐
│                      Health Server Layer                      │
│                                                                │
│  pkg/health/server.go                                         │
│  ├─── SetWebhookHandler(handler)                             │
│  ├─── webhookProcessHandler()                                │
│  └─── webhookStatusHandler()                                 │
│                                                                │
└────────────────────┬─────────────────────────────────────────┘
                     │
                     │ uses
                     ▼
┌──────────────────────────────────────────────────────────────┐
│                    Webhook Handler Layer                      │
│                                                                │
│  pkg/webhook/handler.go                                       │
│  ├─── ProcessHandler(w, r)         (HTTP handlers)           │
│  ├─── StatusHandler(w, r)                                    │
│  └─── extractBearerToken()         (Auth helpers)            │
│                                                                │
└────────────────────┬─────────────────────────────────────────┘
                     │
                     │ uses
                     ▼
┌──────────────────────────────────────────────────────────────┐
│                    Processor Core Layer                       │
│                                                                │
│  pkg/webhook/processor.go                                     │
│  ├─── Submit(req)                  (Job submission)           │
│  ├─── GetJob(id)                   (Status query)             │
│  ├─── processJob(job)              (Background processing)    │
│  ├─── callWebhook(url, payload)    (Webhook delivery)        │
│  └─── CleanupOldJobs(maxAge)       (Maintenance)             │
│                                                                │
└────────────────────┬─────────────────────────────────────────┘
                     │
                     │ executes
                     ▼
┌──────────────────────────────────────────────────────────────┐
│                   Custom Processor Function                   │
│                                                                │
│  pkg/webhook/example_processor.go                             │
│  └─── ExampleProcessor(ctx, payload) → (result, error)       │
│                                                                │
│  User can provide custom implementations:                     │
│  └─── func(context.Context, map[string]interface{})          │
│       → (map[string]interface{}, error)                       │
│                                                                │
└──────────────────────────────────────────────────────────────┘
```

## Data Flow

### Job Structure

```go
type Job struct {
    ID          string                     // UUID
    WebhookURL  string                     // Callback URL
    Payload     map[string]interface{}     // Input data
    Status      string                     // "processing"|"completed"|"failed"
    CreatedAt   time.Time                  // Submission timestamp
    CompletedAt *time.Time                 // Completion timestamp (nullable)
}
```

### Request/Response Formats

**Submit Request:**
```json
{
  "webhook_url": "https://example.com/callback",
  "payload": {
    "any": "data",
    "structure": "you want"
  }
}
```

**Submit Response:**
```json
{
  "job_id": "uuid-v4",
  "status": "processing",
  "timestamp": "2026-04-17T10:00:00Z"
}
```

**Webhook Callback (Success):**
```json
{
  "job_id": "uuid-v4",
  "status": "completed",
  "result": {
    "your": "processed data"
  },
  "timestamp": "2026-04-17T10:00:05Z"
}
```

**Webhook Callback (Error):**
```json
{
  "job_id": "uuid-v4",
  "status": "failed",
  "error": "error message here",
  "timestamp": "2026-04-17T10:00:05Z"
}
```

## Concurrency Model

```
Main Goroutine (Gateway)
│
├─── HTTP Server Goroutines (per request)
│    │
│    ├─── POST /webhook/process handler
│    │    └─── Spawns processing goroutine → Background Worker Pool
│    │
│    └─── GET /webhook/status handler
│         └─── Reads from shared job map (mutex-protected)
│
├─── Background Worker Goroutines (one per job)
│    │
│    ├─── Execute processor function
│    │    └─── User-defined processing logic
│    │
│    └─── HTTP POST to webhook URL
│         └─── Deliver results
│
└─── Cleanup Goroutine (periodic)
     └─── Remove old jobs from memory
```

## Scalability Considerations

### Current Implementation (Single Instance)

- In-memory job storage
- Goroutine-based concurrency
- Suitable for:
  - Development/testing
  - Low-to-medium traffic
  - Single gateway instance

### Production Scaling Options

1. **Persistent Storage**
   - Replace in-memory map with Redis/PostgreSQL
   - Enables multi-instance deployment
   - Survives gateway restarts

2. **Message Queue**
   - Use Google Cloud Tasks or Pub/Sub
   - Better retry/backoff handling
   - Horizontal scaling across instances

3. **Distributed Tracing**
   - Add OpenTelemetry spans
   - Track job lifecycle
   - Monitor performance

4. **Load Balancing**
   - Multiple gateway instances
   - Shared job store
   - Sticky sessions not required

## Security Model

```
Request → Authentication → Validation → Processing → Webhook Delivery
   │            │              │             │              │
   │            │              │             │              │
   ▼            ▼              ▼             ▼              ▼
Bearer     Token from      Webhook URL   Context       Timeout
Token      PID file        validation     timeout       handling
```

### Security Features

1. **Authentication**: Bearer token (same as gateway auth)
2. **Input Validation**: JSON schema validation
3. **Timeouts**: Prevent runaway processing
4. **Error Handling**: Safe error messages in responses
5. **HTTPS**: Recommended for webhook callbacks

### Security TODO (Production)

- [ ] Rate limiting per token/IP
- [ ] Webhook URL allowlist/blocklist
- [ ] HMAC signatures for webhook callbacks
- [ ] Webhook retry with exponential backoff
- [ ] Job payload size limits
- [ ] Concurrent job limits per client
