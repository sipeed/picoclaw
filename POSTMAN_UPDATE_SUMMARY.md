# Postman Collection Update - Summary

## ✅ What Was Updated

The PicoClaw Postman collection has been updated with complete webhook processing endpoints and comprehensive documentation.

## 📦 Updated Files

### 1. Postman Collection (`docs/api/picoclaw.postman_collection.json`)

**Added New Folder: "Webhook"**

Contains 4 requests:

1. **Submit Processing Job** ⭐
   - POST `/api/webhook/process`
   - Includes auto-save script for `job_id`
   - Multiple payload examples
   - Detailed inline documentation
   - Test script to extract job ID

2. **Get Job Status**
   - GET `/api/webhook/status?job_id={{webhook_job_id}}`
   - Uses saved job ID from previous request
   - Query parameter documentation
   - Response schema examples

3. **Submit Job - Example 1 (Simple)**
   - Minimal payload example
   - Quick test template
   - webhook.site ready

4. **Submit Job - Example 2 (Complex)**
   - Nested data structure
   - Real-world use case
   - Production-ready template

**Added Collection Variable:**
- `webhook_job_id` - Stores job ID from submit request

**Features:**
- ✅ JSON validated
- ✅ Postman v2.1 schema compliant
- ✅ Auto variable extraction via test scripts
- ✅ Inline documentation on all fields
- ✅ Multiple working examples
- ✅ Follows existing collection patterns

### 2. Postman Guide (`docs/api/POSTMAN_GUIDE.md`)

**New comprehensive guide including:**

- 📦 Import instructions (file & link methods)
- 🔧 Setup & configuration
- 🔑 Authentication options (cookie & bearer token)
- 🚀 Quick start workflow
- 📚 Complete webhook examples
- 🔍 Testing workflow diagrams
- 🎯 Advanced features (environments, scripts)
- 🐛 Troubleshooting section
- 💡 Tips & tricks
- 📖 Related documentation links

**Sections:**
1. Import the Collection
2. Setup (Variables & Auth)
3. Quick Start (4 steps)
4. Webhook Examples (3 real-world scenarios)
5. Testing Workflow (complete flow)
6. Advanced Features
7. Request Documentation
8. Security Notes
9. Troubleshooting
10. Tips & Tricks

### 3. Quick Start Guide (`docs/api/WEBHOOK_POSTMAN_QUICKSTART.md`)

**3-minute setup guide:**

- ⚡ Fast setup (4 steps)
- 🎯 What's included
- 📋 Variable reference
- 🚀 Quick commands
- 💡 Pro tips
- 🔄 Testing workflow diagram
- 📊 Status flow chart
- 🎨 Example payloads (3 complexity levels)
- 🐛 Troubleshooting quick fixes
- 📚 Next steps

## 🎯 Key Features

### Auto Variable Management

**Job ID Extraction:**
```javascript
// Automatically runs after "Submit Processing Job"
if (pm.response.code === 202) {
    const response = pm.response.json();
    pm.collectionVariables.set('webhook_job_id', response.job_id);
    console.log('Job ID saved:', response.job_id);
}
```

### Multiple Examples

**Simple:**
```json
{
  "webhook_url": "https://webhook.site/test",
  "payload": {"message": "Hello!"}
}
```

**Complex:**
```json
{
  "webhook_url": "https://your-app.com/callback",
  "payload": {
    "task": "process_document",
    "document": {...},
    "options": {...},
    "metadata": {...}
  }
}
```

### Built-in Documentation

Every request includes:
- Description of what it does
- Expected responses
- Error handling
- Usage examples
- Related endpoints

## 📊 Collection Structure

```
PicoClaw API
├── Auth (4 requests)
├── Config (3 requests)
├── Gateway (5 requests)
├── Pico Channel (3 requests)
├── Sessions (3 requests)
├── OAuth (4 requests)
├── Models (5 requests)
├── Channels (2 requests)
├── Skills (6 requests)
├── Tools (2 requests)
├── System (4 requests)
├── Update (1 request)
├── WeChat (2 requests)
├── WeCom (2 requests)
├── Webhook (4 requests) ← NEW!
└── Gateway Health (3 requests)
```

## 🧪 Testing

### Quick Test Flow

1. **Import Collection**
   ```
   Postman → Import → Select picoclaw.postman_collection.json
   ```

2. **Start Backend**
   ```bash
   cd web/backend && ./picoclaw-web
   ```

3. **Get Webhook URL**
   - Visit https://webhook.site
   - Copy unique URL

4. **Test in Postman**
   - Open "Webhook → Submit Processing Job"
   - Update webhook_url
   - Click Send
   - Check webhook.site for callback

### Verification

All verifications passed:
- ✅ Valid JSON format
- ✅ Webhook folder present
- ✅ 4 webhook requests included
- ✅ Variables configured
- ✅ Test scripts working
- ✅ Documentation complete

## 📚 Documentation Files

```
docs/api/
├── picoclaw.postman_collection.json  ← Updated ✅
├── POSTMAN_GUIDE.md                  ← New ✅
├── WEBHOOK_POSTMAN_QUICKSTART.md     ← New ✅
└── openapi.yaml                      ← Already updated ✅
```

## 🎨 Usage Examples

### Example 1: Basic Test

```
1. Postman: Submit Processing Job
   → GET job_id: "abc-123"

2. Backend: Processing...

3. Webhook.site: Receives callback
   {
     "job_id": "abc-123",
     "status": "completed",
     "result": {...}
   }

4. Postman: Get Job Status (optional)
   → Verify completion
```

### Example 2: Multiple Jobs

```
Submit Job 1 → webhook.site/id1
Submit Job 2 → webhook.site/id2
Submit Job 3 → webhook.site/id3

All process in parallel
All callbacks arrive independently
```

### Example 3: Production Flow

```
Submit Job → your-app.com/webhook
            ↓
      Backend processes
            ↓
      POST to your endpoint
            ↓
      Your app handles result
```

## 💡 Pro Tips

### Tip 1: Dynamic Variables
Use Postman's built-in variables:
```json
{
  "webhook_url": "https://webhook.site/test",
  "payload": {
    "request_id": "{{$randomUUID}}",
    "timestamp": "{{$isoTimestamp}}"
  }
}
```

### Tip 2: Multiple Environments
Create environments for different deployments:
- **Dev**: `localhost:18800`
- **Staging**: `staging.yourapp.com`
- **Prod**: `api.yourapp.com`

### Tip 3: Collection Runner
Run all webhook requests at once:
1. Right-click "Webhook" folder
2. Select "Run folder"
3. Watch all tests execute

### Tip 4: Console Debugging
Enable Postman Console to see:
- All HTTP traffic
- Variable values
- Script logs
- Response bodies

## 🔄 Workflow Diagrams

### Submit Job Flow
```
User (Postman)
    ↓
POST /api/webhook/process
    ↓
Backend (202 Accepted)
    ↓
Return {job_id, status: "processing"}
    ↓
Goroutine processes in background
    ↓
POST result to webhook_url
    ↓
User sees callback at webhook.site
```

### Status Check Flow
```
User saved job_id
    ↓
GET /api/webhook/status?job_id=xxx
    ↓
Backend queries job
    ↓
Return {ID, Status, Timestamps}
    ↓
User sees current status
```

## 🐛 Troubleshooting

### Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| 401 Unauthorized | Login via Auth folder or set bearer_token |
| Job not found | Check webhook_job_id variable is set |
| Webhook not called | Verify URL is accessible, check console |
| Connection refused | Start backend: `./picoclaw-web` |
| Invalid JSON | Use Postman's JSON validator |

## 📖 Related Documentation

- [OpenAPI Spec](docs/api/openapi.yaml) - Complete API reference
- [Webhook Docs](docs/webhook-processing.md) - Detailed webhook guide
- [Integration Guide](examples/webhook-processing/INTEGRATION.md) - Custom processors
- [Architecture](examples/webhook-processing/ARCHITECTURE.md) - System design

## ✨ What's Next

### For Users
1. ✅ Import the collection
2. 📖 Read the [Quick Start Guide](docs/api/WEBHOOK_POSTMAN_QUICKSTART.md)
3. 🧪 Test with webhook.site
4. 🚀 Integrate with your app

### For Developers
1. ✅ Review the [Full Guide](docs/api/POSTMAN_GUIDE.md)
2. 🔧 Customize request bodies
3. 📝 Add your own examples
4. 🤝 Share with team

## 🎉 Summary

**Postman collection is complete and ready!**

- ✅ 4 webhook requests added
- ✅ Auto variable extraction
- ✅ Multiple examples included
- ✅ Comprehensive documentation
- ✅ Quick start guide
- ✅ Full testing guide
- ✅ Troubleshooting section
- ✅ Production-ready templates

**Total additions:**
- Requests: 4
- Variables: 1
- Documentation files: 2
- Example payloads: 6+
- Lines of documentation: ~1,200

**Import and start testing:**
```
Postman → Import → docs/api/picoclaw.postman_collection.json
```

Happy testing! 🚀
