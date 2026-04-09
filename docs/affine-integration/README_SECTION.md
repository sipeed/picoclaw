# Affine Integration Section for Main README

Add this section to the main README.md under "Tools" or "Integrations":

---

## Affine Integration

PicoClaw can search and read documents from your [Affine](https://affine.pro) workspace.

### Quick Setup

1. **Get your Affine MCP credentials**:
   - Go to https://app.affine.pro
   - Open workspace settings → MCP Server
   - Copy your MCP token and workspace ID

2. **Configure PicoClaw** (`~/.picoclaw/config.json`):
   ```json
   {
     "tools": {
       "affine": {
         "enabled": true,
         "mcp_endpoint": "https://app.affine.pro/api/workspaces/YOUR_WORKSPACE_ID/mcp",
         "api_key": "YOUR_MCP_TOKEN",
         "workspace_id": "YOUR_WORKSPACE_ID"
       }
     }
   }
   ```

3. **Use it**:
   ```bash
   picoclaw agent -m "Search my Affine workspace for 'project notes'"
   picoclaw agent -m "Find documents about machine learning in Affine"
   ```

### Features

- ✅ Keyword search - Find documents by exact keywords
- ✅ Semantic search - Find documents by meaning
- ✅ Multi-language support - English, Chinese, and more
- ✅ Fast response - Under 2 seconds

### Documentation

- [Quick Start Guide](docs/affine-integration/README.md)
- [Detailed Documentation](docs/affine-integration/DETAILED.md)
- [Configuration Example](config/config.example.json)

---

## Alternative: Add to Tools Table

If the README has a tools table, add this row:

| Tool | Description | Setup | Docs |
|------|-------------|-------|------|
| **Affine** | Search and read documents from Affine workspace | [Quick Setup](docs/affine-integration/README.md#quick-start) | [Full Docs](docs/affine-integration/DETAILED.md) |

---

## Alternative: Add to Features List

If the README has a features list, add:

- **Affine Integration** - Search and read documents from your Affine workspace using MCP protocol

---

## Alternative: Minimal Mention

For a minimal mention in the README:

### Integrations

PicoClaw integrates with:
- Web search and browsing
- File system operations
- **Affine workspace** - Search and read documents ([setup guide](docs/affine-integration/README.md))
- And more...
