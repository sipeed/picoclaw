# Affine Integration for PicoClaw

## Overview

PicoClaw now supports integration with [Affine](https://affine.pro), an open-source, all-in-one workspace that combines note-taking, knowledge management, whiteboarding, and task management. This integration allows your AI agent to read, create, and manage notes in your Affine workspace.

## Features

The Affine tool provides the following capabilities:

- **List Workspaces**: View all available Affine workspaces
- **List Pages**: Browse pages in a workspace with tags and metadata
- **Search**: Full-text search across your workspace
- **Read Pages**: Retrieve complete page content including tags and structure
- **Create Pages**: Create new notes with content and tags
- **Update Pages**: Modify existing pages (title, content, tags)
- **Get Structure**: View workspace organization (categories, tags, page counts)

## Configuration

### 1. Get Your Affine API Credentials

#### For Affine Cloud (app.affine.pro):
1. Log in to [app.affine.pro](https://app.affine.pro)
2. Go to Settings → API Keys
3. Generate a new API key
4. Copy your workspace ID from the URL (e.g., `https://app.affine.pro/workspace/abc123`)

#### For Self-Hosted Affine:
1. Access your Affine instance
2. Navigate to Settings → API Keys
3. Generate an API key
4. Note your GraphQL endpoint (typically `https://your-domain/graphql`)
5. Copy your workspace ID

### 2. Configure PicoClaw

Add the following to your `~/.picoclaw/config.json`:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "your-affine-api-key-here",
      "workspace_id": "your-default-workspace-id",
      "timeout_seconds": 30
    }
  }
}
```

**Configuration Options:**

| Option | Required | Description | Default |
|--------|----------|-------------|---------|
| `enabled` | Yes | Enable/disable Affine integration | `false` |
| `api_url` | Yes | Affine GraphQL API endpoint | - |
| `api_key` | Yes | Your Affine API key | - |
| `workspace_id` | Yes | Default workspace ID to use | - |
| `timeout_seconds` | No | API request timeout | `30` |

### 3. Environment Variables (Optional)

You can also configure using environment variables:

```bash
export PICOCLAW_TOOLS_AFFINE_ENABLED=true
export PICOCLAW_TOOLS_AFFINE_API_URL="https://app.affine.pro/graphql"
export PICOCLAW_TOOLS_AFFINE_API_KEY="your-api-key"
export PICOCLAW_TOOLS_AFFINE_WORKSPACE_ID="your-workspace-id"
export PICOCLAW_TOOLS_AFFINE_TIMEOUT_SECONDS=30
```

## Usage Examples

### List All Workspaces

```
User: Show me my Affine workspaces
Agent: [Uses affine tool with action: list_workspaces]
```

**Agent Response:**
```
Available Workspaces:
- Personal Notes (ID: abc123, Members: 1)
- Team Project (ID: def456, Members: 5)
```

### List Pages in Workspace

```
User: List all pages in my Affine workspace
Agent: [Uses affine tool with action: list_pages]
```

**Agent Response:**
```
Pages in workspace (showing 10):
1. Meeting Notes 2024-02-26 (ID: page-001) [Tags: meetings, work]
   Updated: 2024-02-26T10:30:00Z
2. Project Ideas (ID: page-002) [Tags: brainstorming]
   Updated: 2024-02-25T15:20:00Z
```

### Search for Content

```
User: Search my Affine notes for "API integration"
Agent: [Uses affine tool with action: search, query: "API integration"]
```

**Agent Response:**
```
Search results for 'API integration' (3 found):
1. API Integration Guide (ID: page-123) [documentation, api]
   Complete guide for integrating third-party APIs...
2. Project Roadmap (ID: page-456) [planning]
   Q2 goals include API integration with...
```

### Read a Specific Page

```
User: Read the page with ID page-123
Agent: [Uses affine tool with action: read_page, page_id: "page-123"]
```

**Agent Response:**
```
Title: API Integration Guide
ID: page-123
Tags: documentation, api, development
Updated: 2024-02-26T09:15:00Z

Content:
# API Integration Guide

## Overview
This guide covers best practices for API integration...

## Authentication
Use OAuth 2.0 for secure authentication...
```

### Create a New Page

```
User: Create a new note in Affine titled "Daily Standup 2024-02-26" with tags "meetings" and "standup"
Agent: [Uses affine tool with action: create_page, title: "Daily Standup 2024-02-26", tags: ["meetings", "standup"]]
```

**Agent Response:**
```
Created page 'Daily Standup 2024-02-26' (ID: page-789) with tags [meetings, standup]
```

### Create Page with Content

```
User: Create a note called "Shopping List" with content "- Milk\n- Bread\n- Eggs"
Agent: [Uses affine tool with action: create_page, title: "Shopping List", content: "- Milk\n- Bread\n- Eggs"]
```

### Update an Existing Page

```
User: Update page page-789 and add the content "## Attendees\n- Alice\n- Bob"
Agent: [Uses affine tool with action: update_page, page_id: "page-789", content: "## Attendees\n- Alice\n- Bob"]
```

**Agent Response:**
```
Updated page 'Daily Standup 2024-02-26' (ID: page-789) - changed: content
```

### Update Page Tags

```
User: Add tags "urgent" and "review" to page page-456
Agent: [Uses affine tool with action: update_page, page_id: "page-456", tags: ["urgent", "review"]]
```

### Get Workspace Structure

```
User: Show me the structure of my Affine workspace
Agent: [Uses affine tool with action: get_structure]
```

**Agent Response:**
```
Workspace: Personal Notes (ID: abc123)
Total Pages: 47

Categories:
  - Work (23 pages)
  - Personal (15 pages)
  - Projects (9 pages)

Tags:
  - meetings (12 pages)
  - ideas (8 pages)
  - documentation (6 pages)
  - urgent (3 pages)
```

## Advanced Usage

### Multi-Workspace Operations

You can specify a different workspace for any operation:

```
User: List pages in workspace def456
Agent: [Uses affine tool with action: list_pages, workspace_id: "def456"]
```

### Limit Search Results

```
User: Search for "project" and show only 5 results
Agent: [Uses affine tool with action: search, query: "project", limit: 5]
```

### Complex Page Creation

```
User: Create a comprehensive project plan in Affine
Agent: [Creates page with structured markdown content including headers, lists, and tags]
```

## Integration Patterns

### Automatic Note-Taking

Set up PicoClaw to automatically save important information:

```
User: Remember this: The API key expires on March 1st
Agent: I'll save that to Affine.
[Creates page "Important Reminder" with the information and tag "reminders"]
```

### Meeting Notes

```
User: Start a meeting note for today's standup
Agent: [Creates structured meeting note with date, attendees section, and agenda]
```

### Knowledge Base Search

```
User: What did we decide about the database migration?
Agent: Let me search your Affine notes...
[Searches workspace and provides relevant information from past notes]
```

## Troubleshooting

### Authentication Errors

**Error:** `HTTP 401: Unauthorized`

**Solution:** 
- Verify your API key is correct
- Check if the API key has expired
- Ensure the API key has proper permissions

### Workspace Not Found

**Error:** `Workspace not found`

**Solution:**
- Verify the workspace ID is correct
- Check if you have access to the workspace
- Ensure the workspace hasn't been deleted

### Timeout Errors

**Error:** `context deadline exceeded`

**Solution:**
- Increase `timeout_seconds` in config
- Check your network connection
- Verify the Affine instance is accessible

### GraphQL Errors

**Error:** `graphql error: Field not found`

**Solution:**
- This may indicate the Affine API schema has changed
- Check if you're using a compatible Affine version
- Report the issue on GitHub

## API Compatibility

This integration is designed for:
- **Affine Cloud**: app.affine.pro
- **Self-Hosted Affine**: v0.10.0 and later

**Note:** The GraphQL schema may vary between versions. If you encounter issues, please check the [Affine documentation](https://docs.affine.pro) for your version.

## Security Best Practices

1. **API Key Storage**: Store API keys in config file with restricted permissions (`chmod 600 ~/.picoclaw/config.json`)
2. **Environment Variables**: Use environment variables in production environments
3. **Key Rotation**: Regularly rotate your API keys
4. **Access Control**: Use workspace-specific API keys when possible
5. **Audit Logs**: Monitor API usage through Affine's admin panel

## Limitations

- **Rate Limiting**: Affine may rate-limit API requests. The tool respects these limits.
- **Content Size**: Very large pages may take longer to retrieve
- **Real-time Sync**: Changes made through the API may take a moment to appear in the UI
- **Whiteboard Content**: Currently focuses on text content; whiteboard elements are not fully supported

## Future Enhancements

Planned features for future releases:

- [ ] Whiteboard/canvas operations
- [ ] File attachment handling
- [ ] Real-time collaboration via WebSocket
- [ ] Batch operations for multiple pages
- [ ] Advanced filtering and sorting
- [ ] Comment management
- [ ] Version history access
- [ ] Export/import functionality

## Contributing

Found a bug or have a feature request? Please open an issue on GitHub!

Want to contribute? Check out the [Contributing Guide](../CONTRIBUTING.md).

## Related Documentation

- [Affine Official Documentation](https://docs.affine.pro)
- [Affine GraphQL API](https://docs.affine.pro/api/graphql)
- [PicoClaw Tool Development](./tools_configuration.md)

## Support

- **PicoClaw Discord**: [Join our community](https://discord.gg/V4sAZ9XWpN)
- **Affine Discord**: [Affine Community](https://discord.gg/affine)
- **GitHub Issues**: [Report bugs](https://github.com/sipeed/picoclaw/issues)
