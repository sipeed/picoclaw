# Add Affine Workspace Integration

## 🎯 What This PR Does

Adds integration with [Affine](https://affine.pro) workspace, enabling PicoClaw to search and read documents from Affine Cloud.

## ✨ Features

- **Keyword Search** - Find documents by exact keywords
- **Semantic Search** - Find documents by meaning (includes full content)
- **Multi-language** - Works with English, Chinese, and more
- **Fast** - Response time under 2 seconds
- **Simple Setup** - Just API key and workspace ID, no installation

## 📦 What's Included

### Code
- `pkg/tools/affine_simple.go` - Main implementation (350 lines)
- `pkg/tools/affine_simple_test.go` - Unit tests (138 lines)
- `pkg/config/config.go` - Configuration structure
- `pkg/agent/instance.go` - Tool registration

### Documentation
- `docs/affine-integration/README.md` - Quick start guide
- `docs/affine-integration/DETAILED.md` - Technical documentation
- `docs/affine-integration/PULL_REQUEST.md` - This PR details
- `config/config.example.json` - Configuration example

## 🧪 Testing

### Unit Tests ✅
```bash
go test ./pkg/tools -v -run TestAffineSimpleTool
```
All tests passing (8 test cases)

### Integration Tests ✅
Tested with real Affine workspace:
- Keyword search (English): ✅ 697ms
- Keyword search (Chinese): ✅ 1777ms
- Semantic search: ✅ 1000ms

### CI/CD ✅
All GitHub Actions workflows passing

## 🚀 Quick Start

### 1. Get Credentials
1. Go to https://app.affine.pro
2. Open workspace settings
3. Find "MCP Server" section
4. Copy MCP token and workspace ID

### 2. Configure
Add to `~/.picoclaw/config.json`:
```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "mcp_endpoint": "https://app.affine.pro/api/workspaces/YOUR_WORKSPACE_ID/mcp",
      "api_key": "YOUR_MCP_TOKEN",
      "workspace_id": "YOUR_WORKSPACE_ID",
      "timeout_seconds": 30
    }
  }
}
```

### 3. Use It
```bash
picoclaw agent -m "Search my Affine workspace for 'project notes'"
```

## 🏗️ Technical Details

### Architecture
- **Protocol**: MCP (Model Context Protocol) over HTTP
- **Format**: JSON-RPC 2.0 with Server-Sent Events
- **Authentication**: Bearer token
- **No Dependencies**: Uses only Go standard library

### Why MCP Bridge?
We use Affine Cloud MCP Bridge (3 tools) instead of full MCP server (43 tools) because:
- ✅ No installation required
- ✅ Simple HTTP-based communication
- ✅ Sufficient for search and read use cases
- ✅ Easy to maintain

Users who need advanced features can install the full MCP server separately.

## 📊 Performance

- Response time: 700ms - 2000ms
- Memory usage: < 1MB per request
- Supports concurrent requests
- Follows Affine Cloud rate limits

## 🔒 Security

- ✅ HTTPS encryption
- ✅ Bearer token authentication
- ✅ No credentials in code
- ✅ Timeout protection
- ✅ No sensitive data in logs

## ⚠️ Known Limitations

1. **read_document tool is unstable** - Server returns internal error
   - Workaround: Use semantic_search (returns full content)
   
2. **Read-only access** - Cannot create/edit documents
   - Reason: MCP Bridge limitation
   - Alternative: Install full MCP server for write access

## 💡 Future Enhancements

- [ ] Caching layer for performance
- [ ] Retry logic for transient failures
- [ ] Support for full MCP server (43 tools)
- [ ] Document creation and editing

## 📝 Checklist

- [x] Code follows project style
- [x] Unit tests added and passing
- [x] Integration tests performed
- [x] Documentation complete
- [x] Configuration example provided
- [x] No breaking changes
- [x] No new dependencies
- [x] CI/CD passing

## 🙏 Review Notes

This is a complete, tested, and documented feature ready for production use. No breaking changes, no new dependencies, follows existing tool patterns.

**Questions for reviewers:**
1. Is the documentation clear enough?
2. Should we add more test cases?
3. Any concerns about the MCP Bridge approach?

---

**Type**: Feature  
**Status**: Ready for Review  
**Risk**: Low  
**Complexity**: Medium
