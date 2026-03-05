# How to Get AFFiNE API Key/Token

## ⚠️ Important Note

**AFFiNE Cloud (app.affine.pro) may not have a public API key system yet!**

AFFiNE is primarily designed as a collaborative workspace app, and the GraphQL API is mainly for self-hosted instances. Let me explain your options:

---

## Option 1: Self-Hosted AFFiNE (Recommended for API Access)

If you want full API access, you should self-host AFFiNE:

### Quick Self-Host Setup

```bash
# Using Docker (easiest)
docker run -d \
  --name affine \
  -p 3000:3000 \
  -v affine-data:/app/data \
  ghcr.io/toeverything/affine:stable

# Access at: http://localhost:3000
```

### Get API Token from Self-Hosted Instance

1. **Access your instance**: `http://localhost:3000`
2. **Create an account** or log in
3. **Get your token**:
   - Open browser DevTools (F12)
   - Go to Application → Local Storage
   - Look for authentication token
   - Or check Network tab for GraphQL requests to see the Authorization header

---

## Option 2: Use AFFiNE MCP Server (Alternative Approach)

Instead of direct API access, you can use the existing AFFiNE MCP server:

### What is MCP Server?

The Model Context Protocol (MCP) server provides a standardized way to interact with AFFiNE. There's already an MCP server for AFFiNE: `dawncr0w/affine-mcp-server`

### Setup MCP Server

```bash
# Install the MCP server
npm install -g @dawncr0w/affine-mcp-server

# Or use npx
npx @dawncr0w/affine-mcp-server
```

### Configure PicoClaw to Use MCP

This would require modifying the integration to use MCP instead of direct GraphQL.

---

## Option 3: Browser Session Token (For Testing)

If you just want to test with AFFiNE Cloud:

### Step 1: Log in to AFFiNE Cloud

1. Go to https://app.affine.pro
2. Log in to your account

### Step 2: Extract Session Token

1. **Open DevTools** (F12 or Right-click → Inspect)
2. **Go to Network tab**
3. **Refresh the page**
4. **Find a GraphQL request**:
   - Look for requests to `/graphql`
   - Click on one
   - Go to "Headers" section
   - Find "Authorization" header
   - Copy the token (looks like: `Bearer eyJhbGc...`)

### Step 3: Use Token in Config

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "eyJhbGc...",  // Paste token here (without "Bearer ")
      "workspace_id": "your-workspace-id",
      "timeout_seconds": 30
    }
  }
}
```

**⚠️ Warning**: Session tokens expire! You'll need to refresh them periodically.

---

## Option 4: Wait for Official API (Future)

AFFiNE is actively developing their API. Check:
- https://github.com/toeverything/AFFiNE/issues
- https://docs.affine.pro
- Their Discord: https://discord.gg/affine

---

## 🎯 Recommended Approach for Now

Since AFFiNE Cloud doesn't have a public API key system yet, here's what I recommend:

### For Testing (Quick & Easy)

**Use the browser session token method (Option 3)**:

1. Log in to app.affine.pro
2. Open DevTools → Network tab
3. Find a GraphQL request
4. Copy the Authorization token
5. Use it in your config (without "Bearer " prefix)

**Pros**: Works immediately
**Cons**: Token expires (need to refresh every few days/weeks)

### For Production (Better)

**Self-host AFFiNE (Option 1)**:

```bash
# Quick Docker setup
docker run -d -p 3000:3000 ghcr.io/toeverything/affine:stable
```

Then use the self-hosted instance's API.

**Pros**: Full control, stable tokens
**Cons**: Need to run your own server

---

## 📝 Updated Instructions for Your Codespace

Since you're in the Codespace now, let's use the session token method:

### Step-by-Step:

1. **Open a new browser tab** (keep Codespace open)

2. **Go to** https://app.affine.pro and log in

3. **Open DevTools** (F12)

4. **Go to Network tab**

5. **Refresh the page** (Ctrl+R or Cmd+R)

6. **Find a GraphQL request**:
   - Look for requests with "graphql" in the name
   - Click on one
   - Click "Headers" tab
   - Scroll to "Request Headers"
   - Find "Authorization: Bearer eyJhbGc..."

7. **Copy the token** (the part after "Bearer ")

8. **Back in Codespace**, edit config:
   ```bash
   code ~/.picoclaw/config.json
   ```

9. **Paste the token**:
   ```json
   {
     "tools": {
       "affine": {
         "enabled": true,
         "api_url": "https://app.affine.pro/graphql",
         "api_key": "PASTE_TOKEN_HERE",
         "workspace_id": "YOUR_WORKSPACE_ID",
         "timeout_seconds": 30
       }
     }
   }
   ```

10. **Get workspace ID**:
    - Look at your browser URL: `https://app.affine.pro/workspace/abc123`
    - Copy the ID: `abc123`
    - Paste it in the config

11. **Save** (Ctrl+S)

12. **Test**:
    ```bash
    ./picoclaw agent -m "List my Affine workspaces"
    ```

---

## 🔍 Visual Guide: Finding the Token

```
Browser DevTools (F12)
├── Network Tab
│   ├── Refresh page (Ctrl+R)
│   ├── Find "graphql" request
│   ├── Click on it
│   └── Headers section
│       └── Request Headers
│           └── Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
│                                      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
│                                      Copy this part (without "Bearer ")
```

---

## ❓ FAQ

### Q: Do I need to enable MCP server in settings?

**A**: No! The MCP server is a separate tool. For our integration, you just need the authentication token.

### Q: Where is the API Keys section in AFFiNE settings?

**A**: AFFiNE Cloud doesn't have a public API Keys section yet. Use the session token method above.

### Q: How long does the token last?

**A**: Session tokens typically last days to weeks. If it expires, just get a new one using the same method.

### Q: Can I use this in production?

**A**: For production, self-host AFFiNE or wait for official API support. Session tokens are best for testing.

### Q: What if I get "401 Unauthorized"?

**A**: Your token expired. Get a new one from the browser DevTools.

---

## 🎉 Next Steps

Once you have your token:

1. Add it to `~/.picoclaw/config.json`
2. Add your workspace ID
3. Test: `./picoclaw agent -m "List my Affine workspaces"`

If it works, you're all set! 🚀

If you get errors, check:
- Token is copied correctly (no extra spaces)
- Workspace ID is correct
- You're logged in to app.affine.pro

---

**Need help?** Let me know what error you're seeing!
