# Affine Integration - Quick Start Guide

## 🚀 Get Started in 3 Steps

### Step 1: Get Your Affine API Key

**For Affine Cloud (app.affine.pro):**

1. Go to https://app.affine.pro
2. Click your avatar → Settings
3. Navigate to "API Keys" section
4. Click "Generate New Key"
5. Copy the API key (save it securely!)
6. Copy your workspace ID from the URL: `https://app.affine.pro/workspace/YOUR_WORKSPACE_ID`

**For Self-Hosted Affine:**

1. Access your Affine instance
2. Go to Settings → API Keys
3. Generate a new API key
4. Note your GraphQL endpoint (usually `https://your-domain/graphql`)
5. Copy your workspace ID

### Step 2: Configure PicoClaw

Edit `~/.picoclaw/config.json` and add the Affine section:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "YOUR_API_KEY_HERE",
      "workspace_id": "YOUR_WORKSPACE_ID_HERE",
      "timeout_seconds": 30
    }
  }
}
```

**Or use environment variables:**

```bash
export PICOCLAW_TOOLS_AFFINE_ENABLED=true
export PICOCLAW_TOOLS_AFFINE_API_URL="https://app.affine.pro/graphql"
export PICOCLAW_TOOLS_AFFINE_API_KEY="your-api-key"
export PICOCLAW_TOOLS_AFFINE_WORKSPACE_ID="your-workspace-id"
```

### Step 3: Try It Out!

```bash
# List your workspaces
picoclaw agent -m "Show me my Affine workspaces"

# List pages
picoclaw agent -m "List all pages in my Affine workspace"

# Search your notes
picoclaw agent -m "Search my Affine notes for 'meeting'"

# Create a new note
picoclaw agent -m "Create a note in Affine titled 'Test Note' with content 'Hello from PicoClaw!'"

# Read a page
picoclaw agent -m "Read the page with ID page-123 from Affine"
```

## 📝 Common Use Cases

### 1. Quick Note Taking

```bash
picoclaw agent -m "Create a note called 'Ideas' with these bullet points: - AI integration - Mobile app - API improvements"
```

### 2. Search Your Knowledge Base

```bash
picoclaw agent -m "Search my Affine workspace for information about API authentication"
```

### 3. Update Existing Notes

```bash
picoclaw agent -m "Update my 'Daily Log' page in Affine with today's accomplishments"
```

### 4. Organize with Tags

```bash
picoclaw agent -m "Create a note titled 'Project Alpha Kickoff' with tags 'project', 'meeting', and 'important'"
```

### 5. Get Workspace Overview

```bash
picoclaw agent -m "Show me the structure of my Affine workspace - categories and tags"
```

## 🎯 What Can You Do?

| Action | What It Does | Example |
|--------|--------------|---------|
| **list_workspaces** | See all your workspaces | "Show my Affine workspaces" |
| **list_pages** | Browse pages in workspace | "List my Affine pages" |
| **search** | Find content across notes | "Search for 'API' in Affine" |
| **read_page** | Get full page content | "Read page page-123" |
| **create_page** | Make a new note | "Create note 'Meeting Notes'" |
| **update_page** | Modify existing note | "Update page with new content" |
| **get_structure** | View organization | "Show workspace structure" |

## 🔧 Troubleshooting

### "Authentication failed"
- Check your API key is correct
- Verify the API key hasn't expired
- Make sure you copied the full key

### "Workspace not found"
- Verify your workspace ID
- Check you have access to the workspace
- Try listing workspaces first

### "Connection timeout"
- Check your internet connection
- Verify the API URL is correct
- Try increasing `timeout_seconds` in config

## 💡 Pro Tips

1. **Default Workspace**: Set your most-used workspace as default in config
2. **Tag Everything**: Use tags to organize notes for easier searching
3. **Structured Content**: Use markdown formatting in your notes
4. **Search First**: Before creating, search to avoid duplicates
5. **Batch Operations**: Create multiple notes in one conversation

## 🎨 Example Workflows

### Daily Standup Notes

```bash
picoclaw agent -m "Create a standup note for today with sections for: What I did yesterday, What I'll do today, and Blockers. Tag it with 'standup' and 'daily'"
```

### Meeting Minutes

```bash
picoclaw agent -m "Create meeting minutes for 'Q1 Planning' with attendees Alice and Bob, agenda items, and action items. Tag with 'meeting' and 'planning'"
```

### Knowledge Base Search

```bash
picoclaw agent -m "Search my Affine workspace for all notes about 'database migration' and summarize the key points"
```

### Project Documentation

```bash
picoclaw agent -m "Create a project overview document for 'Project Phoenix' with sections for Goals, Timeline, Team, and Resources"
```

## 📚 Next Steps

- Read the full documentation: `docs/AFFINE_INTEGRATION.md`
- Check implementation details: `AFFINE_IMPLEMENTATION_SUMMARY.md`
- Explore advanced features in the docs
- Join our Discord for support

## 🤝 Need Help?

- **Discord**: https://discord.gg/V4sAZ9XWpN
- **GitHub Issues**: https://github.com/sipeed/picoclaw/issues
- **Documentation**: `docs/AFFINE_INTEGRATION.md`

---

**That's it! You're ready to use Affine with PicoClaw! 🎉**
