# PicoClaw Postman Collection Guide

## 📦 Import the Collection

### Method 1: Import from File

1. Open Postman
2. Click **Import** button (top left)
3. Select **File** tab
4. Choose `docs/api/picoclaw.postman_collection.json`
5. Click **Import**

### Method 2: Import from Link

1. Open Postman
2. Click **Import** button
3. Select **Link** tab
4. Paste the raw GitHub URL to the collection file
5. Click **Continue** → **Import**

## 🔧 Setup

### Configure Variables

After importing, set these collection variables:

1. Click on the **PicoClaw API** collection
2. Go to the **Variables** tab
3. Set the following:

| Variable | Value | Description |
|----------|-------|-------------|
| `base_url` | `http://localhost:18800` | Launcher backend URL |
| `health_url` | `http://localhost:18790` | Gateway health server URL |
| `bearer_token` | (optional) | Your dashboard token for auth |

### Authentication Options

The collection supports two authentication methods:

**Option 1: Session Cookie (Recommended)**
1. Use **Auth → Login** to authenticate
2. Postman automatically stores the session cookie
3. All subsequent requests will use this cookie

**Option 2: Bearer Token**
1. Set `bearer_token` variable to your dashboard token
2. The collection uses Bearer authentication by default
3. Find your token in `~/.picoclaw/launcher.json` or env var `PICOCLAW_LAUNCHER_TOKEN`

## 🚀 Quick Start

### 1. Test Authentication

**Public Endpoints (No Auth):**
- `Auth → Auth Status` - Check if initialized

**Login:**
- `Auth → Login` - Enter your password
- Or `Auth → Setup Password` if first time

### 2. Test Gateway

- `Gateway → Get Status` - Check if gateway is running
- `Gateway → Start Gateway` - Start the gateway process
- `Gateway → Get Logs` - View gateway logs

### 3. Test Webhook Processing

**Submit a job:**
1. Go to **Webhook → Submit Processing Job**
2. Replace `webhook_url` with your test URL:
   - Visit [webhook.site](https://webhook.site)
   - Copy your unique URL
   - Paste into the request body
3. Click **Send**
4. The response includes `job_id` (saved automatically to variables)
5. Watch the webhook.site dashboard for the callback!

**Check job status:**
1. Go to **Webhook → Get Job Status**
2. Uses the `webhook_job_id` from previous response
3. Click **Send**
4. See current job status and timestamps

### 4. Explore Other Features

- **Config** - Get/update gateway configuration
- **Models** - Manage AI model configurations
- **Sessions** - View chat history
- **Skills** - Search and install skills
- **OAuth** - Connect AI provider accounts

## 📚 Webhook Examples

### Example 1: Simple Test

```json
{
  "webhook_url": "https://webhook.site/your-unique-id",
  "payload": {
    "message": "Hello, World!"
  }
}
```

**What happens:**
1. Job submitted → Returns `job_id`
2. Processing in background (2 seconds)
3. Webhook receives: `{job_id, status: "completed", result: {...}}`

### Example 2: Complex Payload

```json
{
  "webhook_url": "https://your-app.com/webhook",
  "payload": {
    "task": "process_document",
    "document": {
      "url": "https://example.com/doc.pdf",
      "pages": [1, 2, 3]
    },
    "options": {
      "extract_tables": true,
      "ocr": true
    }
  }
}
```

### Example 3: AI Processing

```json
{
  "webhook_url": "https://your-app.com/ai-callback",
  "payload": {
    "prompt": "Analyze this data and generate insights",
    "context": {
      "user_id": "123",
      "session_id": "abc"
    }
  }
}
```

## 🔍 Testing Workflow

### Complete Webhook Test Flow

1. **Start Backend**
   ```bash
   cd web/backend
   go build && ./picoclaw-web
   ```

2. **Setup Webhook Receiver**
   - Visit [webhook.site](https://webhook.site)
   - Copy your unique URL

3. **In Postman:**
   - Navigate to **Webhook → Submit Processing Job**
   - Update `webhook_url` with your webhook.site URL
   - Click **Send**
   - Note the `job_id` in response

4. **Check Status:**
   - Navigate to **Webhook → Get Job Status**
   - Click **Send** (uses saved `webhook_job_id`)
   - See status: "processing" → "completed"

5. **View Callback:**
   - Check webhook.site dashboard
   - See the callback with results

## 🎯 Advanced Features

### Environment Setup

Create different environments for dev/staging/prod:

1. Click the environment dropdown (top right)
2. Click **+** to create new environment
3. Add variables:
   ```
   base_url: http://localhost:18800    (dev)
   base_url: https://staging.app.com   (staging)
   base_url: https://app.com           (production)
   ```

### Pre-request Scripts

Some requests include automatic variable extraction:

**Submit Processing Job:**
- Automatically saves `job_id` to `webhook_job_id` variable
- Used by **Get Job Status** request

**OAuth Login:**
- Saves `flow_id` to `oauth_flow_id` variable
- Used by **Poll OAuth Flow** request

### Tests Tab

View response tests in the **Tests** tab of each request:
- Validates status codes
- Extracts variables
- Logs useful information

## 📝 Request Documentation

Each request includes:
- **Description** - What the endpoint does
- **Headers** - Required headers
- **Body** - Example request body
- **Query Params** - URL parameters
- **Expected Response** - What you'll receive

Hover over any field for inline documentation.

## 🔒 Security Notes

### Production Use

When using against production:

1. **Use HTTPS** - Always use `https://` URLs
2. **Protect Tokens** - Don't commit bearer tokens
3. **Session Security** - Logout when done
4. **Webhook URLs** - Validate webhook URLs before submitting

### Webhook Security

For production webhooks:
- Use HTTPS endpoints only
- Implement webhook signature verification
- Validate incoming payloads
- Rate limit webhook endpoints

## 🐛 Troubleshooting

### Common Issues

**401 Unauthorized:**
- Set `bearer_token` variable, OR
- Use **Auth → Login** to get session cookie

**404 Not Found:**
- Check `base_url` is correct
- Verify backend is running on port 18800

**Job Not Found (Webhook):**
- Jobs are cleaned up after 2 hours
- Check the `webhook_job_id` variable is set

**Webhook Not Called:**
- Verify webhook URL is accessible
- Check webhook endpoint accepts POST
- Review gateway logs for errors

### Debug Mode

Enable Postman Console:
1. Click **Console** button (bottom left)
2. See all request/response details
3. View extracted variables
4. Check pre-request script logs

## 📖 Related Documentation

- [OpenAPI Specification](openapi.yaml) - Complete API reference
- [Webhook Documentation](../webhook-processing.md) - Detailed webhook guide
- [API Integration Guide](../../examples/webhook-processing/INTEGRATION.md) - Custom implementations

## 🔄 Collection Updates

The Postman collection is versioned with the API:

- **Current Version:** v1
- **Last Updated:** 2026-04-17
- **New in this version:** Webhook processing endpoints

To update:
1. Re-import the collection file
2. Select **Replace** when prompted
3. Your variables and environment settings are preserved

## 💡 Tips & Tricks

### Quick Test All Endpoints

1. Right-click on the **PicoClaw API** collection
2. Select **Run collection**
3. Choose which folders to run
4. Click **Run PicoClaw API**

### Save Responses

1. Send a request
2. Click **Save Response** button
3. Give it a name
4. Access later from **Collections → Responses**

### Share Collection

Export and share with team:
1. Right-click on collection
2. Select **Export**
3. Choose format (v2.1 recommended)
4. Share the JSON file

### Postman Variables Cheat Sheet

- `{{$randomUUID}}` - Generate random UUID
- `{{$timestamp}}` - Current Unix timestamp
- `{{$isoTimestamp}}` - ISO 8601 timestamp
- `{{$randomInt}}` - Random integer
- `{{webhook_job_id}}` - Saved job ID (our variable)

## 🎓 Learn More

- [Postman Learning Center](https://learning.postman.com/)
- [Postman Variables Guide](https://learning.postman.com/docs/sending-requests/variables/)
- [Writing Tests](https://learning.postman.com/docs/writing-scripts/test-scripts/)

---

**Questions?** Check the [main documentation](../../README.md) or [open an issue](https://github.com/sipeed/picoclaw/issues).
