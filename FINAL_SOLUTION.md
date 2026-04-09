# ✅ Final Solution: Using AFFiNE MCP Token

## 🎯 The Situation

You have:
- ✅ MCP Server token: `ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY`
- ✅ Workspace ID: `732dbb91-3973-4b77-adbc-c8d5ec830d6d`
- ✅ MCP endpoint: `https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp`

## 🔍 The Discovery

The AFFiNE MCP server uses **stdio protocol** (not HTTP), which means:
- ❌ We can't call it directly via HTTP/GraphQL
- ✅ We need to use the MCP client library
- ✅ Or use the `affine-mcp-server` npm package

## 💡 Best Solution: Use the Official MCP Server

Instead of reimplementing everything, let's use the official `affine-mcp-server` package!

### Option 1: Install MCP Server in Codespace (Recommended)

```bash
# In your Codespace terminal

# 1. Install the MCP server globally
npm install -g affine-mcp-server

# 2. Configure it with your token
affine-mcp login

# When prompted:
# - Affine URL: https://app.affine.pro
# - Auth method: [2] Paste API token
# - Token: ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY
# - Workspace: 732dbb91-3973-4b77-adbc-c8d5ec830d6d

# 3. Test it
affine-mcp status
```

### Option 2: Use PicoClaw with MCP Bridge

We can create a bridge that calls the MCP server:

```bash
# In Codespace, create a simple wrapper script
cat > ~/.picoclaw/affine-mcp-bridge.sh << 'EOF'
#!/bin/bash
# Bridge script to call AFFiNE MCP server

METHOD=$1
PARAMS=$2

# Call the MCP server via stdio
echo "{\"method\":\"$METHOD\",\"params\":$PARAMS}" | affine-mcp
EOF

chmod +x ~/.picoclaw/affine-mcp-bridge.sh
```

## 🚀 Quick Test

Let's test if the MCP server works:

```bash
# Install it
npm install -g affine-mcp-server

# Login with your token
affine-mcp login
# Paste: https://app.affine.pro
# Choose: [2] Paste API token
# Token: ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY

# Test it
affine-mcp status

# If it works, you'll see your workspace info!
```

## 🔧 Alternative: Modify PicoClaw to Use MCP

If you want PicoClaw to directly use MCP, we need to:

1. **Install MCP SDK in Go** (or use the npm package)
2. **Create a bridge** that spawns the MCP server process
3. **Communicate via stdio**

This is more complex but gives better integration.

## 📝 What Should We Do?

### Recommended Path:

**Use the official MCP server alongside PicoClaw:**

1. Install `affine-mcp-server` in your Codespace
2. Configure it with your token
3. Use it directly for AFFiNE operations
4. Keep PicoClaw for other tasks

**Pros:**
- ✅ Works immediately
- ✅ Fully supported by AFFiNE
- ✅ All 43 tools available
- ✅ No code changes needed

**Cons:**
- ❌ Separate tool (not integrated into PicoClaw)
- ❌ Need to switch between tools

### Alternative Path:

**Create an MCP bridge in PicoClaw:**

1. Modify the Affine tool to spawn `affine-mcp` process
2. Communicate via stdio
3. Parse responses

**Pros:**
- ✅ Integrated into PicoClaw
- ✅ Single interface

**Cons:**
- ❌ More complex implementation
- ❌ Need to handle process management
- ❌ Takes more time to build

## 🎯 My Recommendation

**Try the official MCP server first!**

```bash
# In your Codespace:

# 1. Install
npm install -g affine-mcp-server

# 2. Login
affine-mcp login
# URL: https://app.affine.pro
# Method: [2] Paste API token
# Token: ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY

# 3. Test
affine-mcp status

# 4. Try listing docs
# (You'll need to use it with Claude or another MCP client)
```

Then decide if you want me to:
- **A)** Create an MCP bridge in PicoClaw (more work, better integration)
- **B)** Just use the official MCP server separately (works now, less integration)

## 📚 Resources

- **AFFiNE MCP Server**: https://github.com/DAWNCR0W/affine-mcp-server
- **MCP Protocol**: https://modelcontextprotocol.io
- **Your Token**: `ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY`
- **Your Workspace**: `732dbb91-3973-4b77-adbc-c8d5ec830d6d`

---

**What would you like to do?** 🤔

1. Try the official MCP server first (quick test)
2. Build an MCP bridge into PicoClaw (better integration, more work)
3. Something else?

Let me know and I'll help you set it up! 🚀
