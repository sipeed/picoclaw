# Webhook Testing with Postman - Quick Start

## ⚡ 3-Minute Setup

### Step 1: Import Collection (30 seconds)

```bash
# In Postman:
File → Import → Select File → Choose picoclaw.postman_collection.json
```

### Step 2: Start Backend (30 seconds)

```bash
cd web/backend
go build && ./picoclaw-web
```

### Step 3: Get Webhook URL (30 seconds)

1. Visit [webhook.site](https://webhook.site)
2. Copy your unique URL (e.g., `https://webhook.site/abc123`)

### Step 4: Test Webhook (90 seconds)

**In Postman:**

1. Navigate to: **PicoClaw API → Webhook → Submit Processing Job**

2. Update the body - Replace `webhook_url`:
   ```json
   {
     "webhook_url": "https://webhook.site/YOUR-ID-HERE",
     "payload": {
       "message": "Hello from PicoClaw!"
     }
   }
   ```

3. Click **Send**

4. You'll get:
   ```json
   {
     "job_id": "550e8400-e29b-41d4-a716-446655440000",
     "status": "processing",
     "timestamp": "2026-04-17T10:00:00Z"
   }
   ```

5. Check webhook.site - You'll see the callback arrive!
   ```json
   {
     "job_id": "550e8400-e29b-41d4-a716-446655440000",
     "status": "completed",
     "result": {
       "processed_data": "...",
       "processed_at": "2026-04-17T10:00:02Z"
     },
     "timestamp": "2026-04-17T10:00:02Z"
   }
   ```

## 🎯 What's Included

### Webhook Folder Contains:

1. **Submit Processing Job** - Main request with auto-save job_id
2. **Get Job Status** - Check job progress
3. **Example 1 (Simple)** - Minimal payload
4. **Example 2 (Complex)** - Nested data structure

### Pre-configured Features:

✅ **Auto Variable Extraction** - `job_id` saved automatically  
✅ **Multiple Examples** - Simple to complex payloads  
✅ **Built-in Tests** - Automatic response validation  
✅ **Inline Docs** - Descriptions on every field  

## 📋 Collection Variables

These are automatically managed:

| Variable | What It Stores | Used By |
|----------|---------------|---------|
| `webhook_job_id` | Last submitted job ID | Get Job Status request |
| `base_url` | Backend URL (localhost:18800) | All requests |

## 🚀 Quick Commands

### Submit Job
```
POST {{base_url}}/api/webhook/process
Body: {webhook_url, payload}
→ Returns: {job_id, status}
```

### Check Status
```
GET {{base_url}}/api/webhook/status?job_id={{webhook_job_id}}
→ Returns: {ID, Status, CreatedAt, CompletedAt}
```

## 💡 Pro Tips

### Tip 1: Auto Job ID
After submitting a job, the `job_id` is automatically saved to `webhook_job_id` variable. Just click **Get Job Status** - it already has the right ID!

### Tip 2: Multiple Webhooks
Want to test multiple jobs? Open multiple tabs in webhook.site and use different URLs for each request.

### Tip 3: Postman Variables
Use dynamic data:
- `{{$randomUUID}}` - Random UUID
- `{{$timestamp}}` - Unix timestamp
- `{{$isoTimestamp}}` - ISO 8601 datetime

Example:
```json
{
  "webhook_url": "https://webhook.site/test",
  "payload": {
    "request_id": "{{$randomUUID}}",
    "timestamp": "{{$isoTimestamp}}"
  }
}
```

### Tip 4: Console Debugging
Enable Postman Console (View → Show Postman Console) to see:
- All requests and responses
- Variable values
- Script execution logs

## 🔄 Testing Workflow

```
┌─────────────────────────────────────────┐
│ 1. Submit Job (Postman)                │
│    → Get job_id                         │
└─────────────┬───────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│ 2. Backend Processes (Background)       │
│    → Job runs in goroutine              │
└─────────────┬───────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│ 3. Webhook Called (webhook.site)        │
│    → See result in dashboard            │
└──────────────────────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│ 4. Check Status (Optional)              │
│    → Verify completion                  │
└──────────────────────────────────────────┘
```

## 📊 Status Flow

```
Submit Job
    ↓
"processing"  ←─────┐
    ↓               │
Processing...       │ Query status anytime
    ↓               │
"completed"   ──────┘
or "failed"
    ↓
Webhook Called
```

## 🎨 Example Payloads

### Minimal
```json
{
  "webhook_url": "https://webhook.site/test",
  "payload": {"test": true}
}
```

### Standard
```json
{
  "webhook_url": "https://your-app.com/webhook",
  "payload": {
    "data": "process this",
    "priority": "high",
    "metadata": {
      "user_id": "123"
    }
  }
}
```

### Complex
```json
{
  "webhook_url": "https://your-app.com/callback",
  "payload": {
    "task": "document_analysis",
    "document": {
      "url": "https://example.com/doc.pdf",
      "pages": [1, 2, 3],
      "format": "pdf"
    },
    "options": {
      "extract_tables": true,
      "extract_images": false,
      "ocr": true,
      "language": "en"
    },
    "metadata": {
      "user_id": "user_12345",
      "request_id": "req_abc123",
      "timestamp": "2026-04-17T10:00:00Z"
    }
  }
}
```

## 🐛 Troubleshooting

### Job Not Found
- Jobs expire after 2 hours
- Make sure `webhook_job_id` variable is set

### Webhook Not Called
- Verify URL is accessible
- Check webhook.site is open
- Review console for errors

### 401 Unauthorized
- Login via **Auth → Login**, or
- Set `bearer_token` variable

### Connection Refused
- Backend not running? Start it:
  ```bash
  cd web/backend && ./picoclaw-web
  ```

## 📚 Next Steps

1. ✅ Basic webhook test working?
2. 📖 Read [Full Postman Guide](POSTMAN_GUIDE.md)
3. 🔧 Try [Custom Processors](../../examples/webhook-processing/INTEGRATION.md)
4. 🚀 Deploy to production

## 🎓 Learn More

- **Full Guide**: [POSTMAN_GUIDE.md](POSTMAN_GUIDE.md)
- **API Spec**: [openapi.yaml](openapi.yaml)
- **Webhook Docs**: [webhook-processing.md](../webhook-processing.md)
- **Examples**: [examples/webhook-processing/](../../examples/webhook-processing/)

---

**Ready to test?** Import the collection and follow the 3-minute setup above! 🚀
