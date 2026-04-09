# 🚀 Codespace Setup - Next Steps

## ✅ Changes Pushed to GitHub

The fix for the build error has been pushed. The issue was that `pkg/agent/instance.go` was trying to call `tools.NewAffineTool()` which doesn't exist anymore.

**What was fixed:**
- Removed the fallback to undefined `NewAffineTool` function
- Now only uses `NewAffineSimpleTool` (the working MCP HTTP client)

---

## 📥 Step 1: Pull Changes in Codespace

Open your Codespace terminal and run:

```bash
# Pull the latest changes
git pull origin main

# Verify the fix
cat pkg/agent/instance.go | grep -A 10 "Register Affine tool"
```

You should see the simplified registration code without `NewAffineTool`.

---

## 🔨 Step 2: Build PicoClaw

```bash
# Set Go toolchain to auto (handles version requirements)
export GOTOOLCHAIN=auto

# Generate embedded files
go generate ./...

# Build the binary
go build -o picoclaw ./cmd/picoclaw

# Verify it works
./picoclaw version
```

**Expected output:**
```
picoclaw version X.X.X
```

---

## ⚙️ Step 3: Configure Affine Integration

Create or edit your config file:

```bash
# Create config directory if it doesn't exist
mkdir -p ~/.picoclaw

# Edit config
nano ~/.picoclaw/config.json
```

Add this configuration (replace with your actual provider config):

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": false,
      "provider": "openai",
      "model_name": "gpt-4",
      "max_tokens": 8192,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "openai": {
      "api_key": "YOUR_OPENAI_API_KEY"
    }
  },
  "tools": {
    "affine": {
      "enabled": true,
      "mcp_endpoint": "https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp",
      "api_key": "ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY",
      "workspace_id": "732dbb91-3973-4b77-adbc-c8d5ec830d6d",
      "timeout_seconds": 30
    }
  }
}
```

**Important:** You need to add your LLM provider configuration (OpenAI, Anthropic, etc.) for PicoClaw to work.

Save and exit (Ctrl+X, then Y, then Enter in nano).

---

## 🧪 Step 4: Test Affine Integration

### Test 1: Search your Affine workspace

```bash
./picoclaw agent -m "Search my Affine notes for 'test'"
```

### Test 2: Natural language query

```bash
./picoclaw agent -m "What documents do I have in Affine about projects?"
```

### Test 3: Read a specific document (if you know the ID)

```bash
./picoclaw agent -m "Read document abc123 from Affine"
```

---

## 🎯 Expected Results

### Successful Search:
```
Found 3 results for 'test':
1. Test Document (ID: abc123)
   This is a test document about...
2. Testing Notes (ID: def456)
   Notes from testing session...
3. Test Plan (ID: ghi789)
   Project test plan...
```

### Successful Read:
```
Document: Test Document

# Test Document

This is the full content of the document...
```

---

## 🐛 Troubleshooting

### Build Error: "telego requires go >= 1.25.5"

```bash
export GOTOOLCHAIN=auto
go build -o picoclaw ./cmd/picoclaw
```

### Build Error: "no matching files found"

```bash
# Run generate first
go generate ./...
# Then build
go build -o picoclaw ./cmd/picoclaw
```

### Runtime Error: "401 Unauthorized"

Check your Affine API key:
- Make sure there are no extra spaces
- Verify the token is still valid in AFFiNE Cloud settings
- Ensure MCP Server is enabled in your workspace

### Runtime Error: "Connection timeout"

- Increase `timeout_seconds` to 60 in config
- Check internet connectivity
- Verify the MCP endpoint URL is correct

### Error: "Provider not configured"

You need to add your LLM provider (OpenAI, Anthropic, etc.) to the config. PicoClaw needs an LLM to process your requests.

---

## 📋 Quick Checklist

- [ ] Pulled latest changes from GitHub
- [ ] Set `GOTOOLCHAIN=auto`
- [ ] Ran `go generate ./...`
- [ ] Built successfully with `go build`
- [ ] Created `~/.picoclaw/config.json`
- [ ] Added LLM provider credentials
- [ ] Added Affine configuration
- [ ] Tested search command
- [ ] Tested read command

---

## 🎉 Success Criteria

You'll know it's working when:
1. Build completes without errors
2. `./picoclaw version` shows version info
3. Search command returns results from your Affine workspace
4. Read command shows document content

---

## 📝 Your Affine Credentials

For reference:
- **MCP Endpoint**: `https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp`
- **API Key**: `ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY`
- **Workspace ID**: `732dbb91-3973-4b77-adbc-c8d5ec830d6d`

---

## 🔍 How It Works

The Affine integration uses:
1. **MCP Protocol**: Model Context Protocol over HTTP
2. **Two Actions**:
   - `search`: Calls `doc-keyword-search` MCP tool
   - `read`: Calls `doc-read` MCP tool
3. **No MCP Server Installation**: Direct HTTP calls to AFFiNE Cloud's MCP endpoint

This is the simplest possible integration - no Node.js, no local MCP server, just HTTP requests!

---

## 📚 Next Steps After Success

Once working, you can:
1. Add more documents to your Affine workspace
2. Test different search queries
3. Integrate Affine into your workflows
4. Create automated tasks that read/search Affine

---

## 💡 Tips

- Use specific search terms for better results
- Document IDs are returned in search results
- The agent can chain operations (search then read)
- Natural language works: "Find my meeting notes from last week"

---

Need help? Check the error messages carefully - they usually indicate what's wrong!
