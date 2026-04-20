# PicoClaw API Documentation

Complete API documentation for the PicoClaw launcher backend and gateway.

## 📚 Available Formats

### 1. OpenAPI Specification

**File:** [`openapi.yaml`](openapi.yaml)

Interactive API reference with complete endpoint documentation, schemas, and examples.

**View with:**
```bash
# Redoc (recommended)
npx @redocly/cli preview-docs openapi.yaml

# Swagger UI
npx swagger-ui-watcher openapi.yaml
```

**Features:**
- Complete endpoint documentation
- Request/response schemas
- Authentication guide
- Error handling
- Code examples

### 2. Postman Collection

**File:** [`picoclaw.postman_collection.json`](picoclaw.postman_collection.json)

Ready-to-use Postman collection with all API endpoints.

**Import:**
```
Postman → File → Import → Select picoclaw.postman_collection.json
```

**Features:**
- Pre-configured requests
- Auto variable extraction
- Test scripts
- Multiple examples
- Environment support

**Guides:**
- [Quick Start Guide](WEBHOOK_POSTMAN_QUICKSTART.md) (3 minutes)
- [Full Postman Guide](POSTMAN_GUIDE.md) (comprehensive)

### 3. Markdown Documentation

**Official Docs:**
- [Webhook Processing Guide](../webhook-processing.md)
- [Changelog](../CHANGELOG_WEBHOOK.md)

**Implementation Guides:**
- [Quick Implementation](../../WEBHOOK_IMPLEMENTATION.md)
- [Integration Guide](../../examples/webhook-processing/INTEGRATION.md)
- [Architecture](../../examples/webhook-processing/ARCHITECTURE.md)

## 🚀 Quick Start

### Option 1: Postman (Recommended for Testing)

1. **Import Collection**
   ```
   Postman → Import → picoclaw.postman_collection.json
   ```

2. **Follow Quick Start**
   - Read: [WEBHOOK_POSTMAN_QUICKSTART.md](WEBHOOK_POSTMAN_QUICKSTART.md)
   - Takes 3 minutes
   - Test with webhook.site

### Option 2: OpenAPI (Recommended for Integration)

1. **View Interactive Docs**
   ```bash
   npx @redocly/cli preview-docs openapi.yaml
   ```

2. **Generate Client**
   ```bash
   # Generate SDK for your language
   npx @openapitools/openapi-generator-cli generate \
     -i openapi.yaml \
     -g python \
     -o ./client
   ```

### Option 3: curl (Quick Testing)

```bash
# See examples
cat ../../examples/webhook-processing/curl-examples.sh

# Run automated tests
../../examples/webhook-processing/test.sh
```

## 📖 API Overview

### Base URLs

- **Launcher Backend:** `http://localhost:18800`
- **Gateway Health:** `http://localhost:18790`

### Authentication

Two methods supported:

1. **Session Cookie** (Recommended)
   - Login via `POST /api/auth/login`
   - Cookie set automatically: `picoclaw_launcher_auth`
   - Valid for 7 days

2. **Bearer Token**
   - Header: `Authorization: Bearer <token>`
   - Token from env var or config file

### Endpoint Categories

| Category | Endpoints | Description |
|----------|-----------|-------------|
| **Auth** | 4 | Login, logout, password setup |
| **Config** | 3 | Gateway configuration CRUD |
| **Gateway** | 5 | Process lifecycle, logs |
| **Pico** | 3 | WebSocket chat proxy |
| **Sessions** | 3 | Chat history |
| **OAuth** | 4 | Provider authentication |
| **Models** | 5 | AI model management |
| **Channels** | 2 | Channel configuration |
| **Skills** | 6 | Skill install & search |
| **Tools** | 2 | Tool enable/disable |
| **System** | 4 | Version, autostart, config |
| **Webhook** | 2 | **Async processing** ⭐ |
| **WeChat** | 2 | QR login flows |
| **WeCom** | 2 | WeCom QR login |
| **Health** | 3 | Liveness & readiness |

## 🎯 Featured: Webhook Processing

New asynchronous webhook processing endpoints for background task execution.

### Endpoints

**`POST /api/webhook/process`**
- Submit async job with webhook callback
- Returns immediately with job ID
- Job runs in background
- Result POSTed to webhook URL

**`GET /api/webhook/status`**
- Query job status by ID
- Returns current state and timestamps

### Quick Example

```bash
# Submit job
curl -X POST http://localhost:18800/api/webhook/process \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/your-id",
    "payload": {"data": "test"}
  }'

# Response: {"job_id": "uuid", "status": "processing"}

# Check status
curl "http://localhost:18800/api/webhook/status?job_id=<uuid>"

# Your webhook receives the result automatically!
```

### Documentation

- **Quick Start:** [WEBHOOK_POSTMAN_QUICKSTART.md](WEBHOOK_POSTMAN_QUICKSTART.md)
- **Full Guide:** [webhook-processing.md](../webhook-processing.md)
- **Examples:** [examples/webhook-processing/](../../examples/webhook-processing/)

## 📝 Common Workflows

### 1. First Time Setup

```
1. POST /api/auth/setup
   → Set password

2. POST /api/auth/login
   → Get session cookie

3. GET /api/config
   → View configuration

4. POST /api/gateway/start
   → Start gateway
```

### 2. Webhook Processing

```
1. POST /api/webhook/process
   → Submit job with webhook URL
   → Get job_id

2. (Optional) GET /api/webhook/status
   → Check progress

3. (Automatic) Webhook receives result
   → Your endpoint gets POST with result
```

### 3. Model Configuration

```
1. GET /api/oauth/providers
   → Check provider status

2. POST /api/oauth/login
   → Connect provider

3. POST /api/models
   → Add model config

4. POST /api/models/default
   → Set default model
```

## 🔧 Development

### Generate API Client

```bash
# Python
openapi-generator-cli generate -i openapi.yaml -g python

# TypeScript
openapi-generator-cli generate -i openapi.yaml -g typescript-fetch

# Go
openapi-generator-cli generate -i openapi.yaml -g go
```

### Validate OpenAPI

```bash
# Validate spec
npx @redocly/cli lint openapi.yaml

# Bundle for distribution
npx @redocly/cli bundle openapi.yaml -o openapi.bundle.yaml
```

### Test with Postman

```bash
# Run collection with Newman
newman run picoclaw.postman_collection.json \
  --environment dev.postman_environment.json
```

## 📊 API Status

| Feature | Status | Version |
|---------|--------|---------|
| OpenAPI Spec | ✅ Complete | 3.0.3 |
| Postman Collection | ✅ Complete | v2.1 |
| Webhook Processing | ✅ Complete | 1.0.0 |
| Documentation | ✅ Complete | 1.0.0 |

## 🐛 Troubleshooting

### Common Issues

**401 Unauthorized**
- Use `POST /api/auth/login` to get session
- Or set `Authorization: Bearer <token>` header

**404 Not Found**
- Check base URL is correct
- Verify endpoint path matches spec

**Webhook not called**
- Verify webhook URL is accessible
- Check for firewall/network issues
- Review gateway logs

### Getting Help

1. Check the [Troubleshooting Guide](../webhook-processing.md#troubleshooting)
2. Review [Postman Guide](POSTMAN_GUIDE.md)
3. See [Examples](../../examples/webhook-processing/)
4. Open an issue on GitHub

## 📖 Related Documentation

### User Documentation
- [Main README](../../README.md)
- [Configuration Guide](../configuration.md)
- [Webhook Guide](../webhook-processing.md)

### Developer Documentation
- [Integration Guide](../../examples/webhook-processing/INTEGRATION.md)
- [Architecture](../../examples/webhook-processing/ARCHITECTURE.md)
- [Contributing](../../CONTRIBUTING.md)

### API Tools
- [OpenAPI Spec](openapi.yaml)
- [Postman Collection](picoclaw.postman_collection.json)
- [Postman Guide](POSTMAN_GUIDE.md)

## 🔄 Updates

**Latest:** 2026-04-17
- ✅ Added webhook processing endpoints
- ✅ Updated Postman collection
- ✅ Enhanced OpenAPI spec
- ✅ New documentation guides

See [CHANGELOG](../CHANGELOG_WEBHOOK.md) for details.

## 🤝 Contributing

Found an issue or want to improve the docs?

1. Check existing [issues](https://github.com/sipeed/picoclaw/issues)
2. Open a new issue or PR
3. Follow [Contributing Guidelines](../../CONTRIBUTING.md)

## 📄 License

See [LICENSE](../../LICENSE) file.

---

**Questions?** Check the documentation above or [open an issue](https://github.com/sipeed/picoclaw/issues).
