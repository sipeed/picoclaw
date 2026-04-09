# Affine Integration for PicoClaw

This integration allows PicoClaw to search and read documents from [Affine](https://affine.pro) workspaces using the Affine Cloud MCP (Model Context Protocol) Bridge.

## Features

- ✅ **Keyword Search** - Search documents by exact keywords
- ✅ **Semantic Search** - Find documents by meaning and context
- ✅ **Document Reading** - Retrieve document content (via semantic search)
- ✅ **Multi-language Support** - Works with English, Chinese, and other languages
- ✅ **Fast Response** - Typical response time under 2 seconds

## Quick Start

### 1. Get Your Affine MCP Credentials

1. Go to your Affine workspace at https://app.affine.pro
2. Click on workspace settings (gear icon)
3. Find "MCP Server" section
4. Copy your:
   - MCP Token (starts with `ut_`)
   - Workspace ID (UUID format)

### 2. Configure PicoClaw

Add to your `~/.picoclaw/config.json`:

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
# Search for documents
picoclaw agent -m "Search my Affine workspace for 'project notes'"

# Find documents by meaning
picoclaw agent -m "Find documents about machine learning in Affine"

# Read document content
picoclaw agent -m "Show me the content of tutorial documents in Affine"
```

## Architecture

This integration uses the **Affine Cloud MCP Bridge**, which provides 3 tools via HTTP:

1. `keyword_search` - Exact keyword matching
2. `semantic_search` - Meaning-based search with full content
3. `read_document` - Direct document reading (currently unstable)

### Why MCP Bridge?

- **No Installation Required** - Works directly via HTTPS
- **Simple Setup** - Just API key and workspace ID
- **Cloud-Based** - No local MCP server needed

### Alternative: Full MCP Server

For advanced features (document creation, editing, 43 total tools), you can install the full MCP server:

```bash
npm i -g affine-mcp-server
```

See [Full MCP Server Guide](./full-mcp-server.md) for details.

## Implementation Details

### File Structure

```
pkg/tools/
  ├── affine_simple.go       # Main implementation
  └── affine_simple_test.go  # Unit tests

pkg/config/
  └── config.go              # Configuration structure

pkg/agent/
  └── instance.go            # Tool registration
```

### How It Works

1. **HTTP Client** - Uses standard Go `net/http` package
2. **MCP Protocol** - JSON-RPC 2.0 over Server-Sent Events (SSE)
3. **Authentication** - Bearer token in Authorization header
4. **Response Parsing** - Handles both JSON and SSE formats

### Code Example

```go
tool := NewAffineSimpleTool(AffineSimpleToolOptions{
    MCPEndpoint:    "https://app.affine.pro/api/workspaces/xxx/mcp",
    APIKey:         "ut_xxx",
    WorkspaceID:    "xxx",
    TimeoutSeconds: 30,
})

result := tool.Execute(ctx, map[string]any{
    "action": "search",
    "query":  "tutorial",
})
```

## Testing

### Unit Tests

```bash
go test ./pkg/tools -v -run TestAffineSimpleTool
```

### Integration Tests

```bash
# Build
make build

# Test search
./picoclaw agent -m "Search Affine for 'test'"

# Test semantic search
./picoclaw agent -m "Find documents about testing in Affine"
```

### Manual API Testing

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "keyword_search",
      "arguments": {"query": "test"}
    }
  }' \
  https://app.affine.pro/api/workspaces/YOUR_WORKSPACE_ID/mcp
```

## Troubleshooting

### HTTP 406 Error
**Problem**: "Not Acceptable: Client must accept both application/json and text/event-stream"

**Solution**: Ensure Accept header includes both content types:
```
Accept: application/json, text/event-stream
```

### Tool Not Found Error
**Problem**: "Tool list_docs not found"

**Solution**: Affine Cloud MCP Bridge only provides 3 tools. Use `keyword_search`, `semantic_search`, or `read_document`.

### read_document Returns Error
**Problem**: "An internal error occurred"

**Solution**: Use `semantic_search` instead - it returns full document content and is more reliable.

## Best Practices

1. **Use keyword_search** for exact keyword matching (fastest)
2. **Use semantic_search** when you need document content
3. **Avoid read_document** (use semantic_search instead)
4. **Set reasonable timeout** (30 seconds recommended)
5. **Handle errors gracefully** (network issues, API limits)

## Limitations

### Current Limitations
- Cannot list all documents (use search instead)
- Cannot create or edit documents
- Cannot manage tags or comments
- read_document tool is unstable

### Why These Limitations?
The Affine Cloud MCP Bridge provides only 3 tools for security and simplicity. For full functionality, install the complete MCP server (43 tools).

## Performance

- **Keyword Search**: ~700ms average
- **Semantic Search**: ~1000ms average
- **Concurrent Requests**: Supported
- **Rate Limiting**: Follows Affine Cloud limits

## Security

- ✅ HTTPS encryption
- ✅ Bearer token authentication
- ✅ No credentials in code
- ✅ Config file permissions (0600 recommended)
- ✅ Timeout protection

## Contributing

### Adding New Features

1. Check if the feature is available in MCP Bridge (only 3 tools)
2. If not, consider full MCP server integration
3. Add tests for new functionality
4. Update documentation

### Testing Changes

```bash
# Run tests
go test ./pkg/tools -v

# Build and test
make build
./picoclaw agent -m "Test your changes"
```

## References

- [Affine Official Site](https://affine.pro)
- [Affine MCP Server (GitHub)](https://github.com/DAWNCR0W/affine-mcp-server)
- [Model Context Protocol](https://modelcontextprotocol.io)
- [PicoClaw Documentation](../../README.md)

## Support

For issues or questions:
1. Check [Troubleshooting](#troubleshooting) section
2. Review [detailed documentation](./DETAILED.md)
3. Open an issue on GitHub

## License

This integration follows the same license as PicoClaw.

---

**Status**: ✅ Production Ready  
**Version**: 1.0.0  
**Last Updated**: March 5, 2026  
**Maintainer**: Community Contribution
