# Complete Setup Steps - Affine Integration

## ✅ What's Ready on Local Machine

All code changes are complete:
- ✅ `pkg/tools/affine_simple.go` - Working MCP HTTP client
- ✅ `pkg/config/config.go` - Added Affine config with MCP endpoint support
- ✅ `pkg/agent/instance.go` - Registers Affine tool
- ✅ `config/config.example.json` - Updated with Affine example
- ✅ `go.mod` - Fixed Go version to 1.23
- ✅ Removed broken `affine.go` file

## 📤 Step 1: Push to GitHub (On Windows Machine)

Open PowerShell or Git Bash:

```bash
# Navigate to your picoclaw directory
cd C:\Users\jackwang\Documents\picoclaw\picoclaw

# Check status
git status

# Add all changes
git add .

# Commit
git commit -m "Add Affine MCP integration - simple HTTP client"

# Push to GitHub
git push origin main
```

**Verify:** Go to GitHub in your browser and check that the files are updated.

---

## 📥 Step 2: Pull in Codespace

In your Codespace terminal:

```bash
# Pull latest changes
git pull origin main

# Verify affine_simple.go exists
ls -la pkg/tools/affine_simple.go

# Verify affine.go is deleted
ls -la pkg/tools/affine.go  # Should say "No such file"
```

---

## 🔨 Step 3: Build in Codespace

```bash
# Generate embedded files
go generate ./...

# Build
make build

# Or if make fails:
go build -o picoclaw ./cmd/picoclaw

# Verify binary exists
ls -la picoclaw
./picoclaw version
```

---

## ⚙️ Step 4: Configure Affine

```bash
# Edit config
code ~/.picoclaw/config.json
```

Add this to the `tools` section:

```json
{
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

**Save** (Ctrl+S)

---

## 🧪 Step 5: Test

```bash
# Test search
./picoclaw agent -m "Search my Affine notes for 'test'"

# Test with a natural query
./picoclaw agent -m "What documents do I have in Affine?"

# If you get a doc ID from search, test read:
./picoclaw agent -m "Read document DOC_ID from Affine"
```

---

## 🎯 Expected Results

### Successful Search Output:
```
Found X results for 'test':
1. Document Title (ID: abc123)
   Snippet of content...
2. Another Document (ID: def456)
   More content...
```

### Successful Read Output:
```
Document: Title

Content of the document...
```

---

## 🐛 Troubleshooting

### Build Fails with "telego requires go >= 1.25.5"

```bash
export GOTOOLCHAIN=auto
go build -o picoclaw ./cmd/picoclaw
```

### "No such file: affine.go"

Good! It should be deleted. Only `affine_simple.go` should exist.

### "401 Unauthorized" when testing

- Check your API key is correct (no extra spaces)
- Verify the MCP endpoint URL is correct
- Make sure MCP server is enabled in AFFiNE Cloud settings

### "Connection timeout"

- Check internet connection
- Verify the MCP endpoint URL is accessible
- Try increasing `timeout_seconds` to 60

---

## 📋 Quick Checklist

- [ ] Push from Windows machine
- [ ] Pull in Codespace
- [ ] Build succeeds
- [ ] Config updated with your token
- [ ] Test search works
- [ ] Test read works (if you have doc IDs)

---

## 🎉 Success!

Once all tests pass, you have:
- ✅ Affine integration working
- ✅ Can search your Affine workspace
- ✅ Can read document content
- ✅ All via natural language commands

**No Node.js or MCP server installation needed!**

---

## 📝 Your Credentials

For reference:
- **MCP Endpoint**: `https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp`
- **API Key**: `ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY`
- **Workspace ID**: `732dbb91-3973-4b77-adbc-c8d5ec830d6d`

Keep these secure! 🔒
