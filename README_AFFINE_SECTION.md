# README Section for Affine Integration

Add this section to your main README.md after the "Chat Apps" section:

---

## 📝 Affine Integration

Connect PicoClaw to your [Affine](https://affine.pro) workspace for AI-powered note-taking and knowledge management.

### Quick Setup

1. **Get API Key**: Log in to [app.affine.pro](https://app.affine.pro) → Settings → API Keys
2. **Configure**: Add to `~/.picoclaw/config.json`:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "YOUR_API_KEY",
      "workspace_id": "YOUR_WORKSPACE_ID"
    }
  }
}
```

3. **Use It**:

```bash
# List your notes
picoclaw agent -m "Show my Affine pages"

# Search your knowledge base
picoclaw agent -m "Search my Affine notes for 'API integration'"

# Create a new note
picoclaw agent -m "Create a note in Affine titled 'Meeting Notes' with tags 'work' and 'meetings'"

# Read and summarize
picoclaw agent -m "Read my 'Project Plan' from Affine and summarize it"
```

### Features

- 📋 **List & Browse**: View all pages with tags and metadata
- 🔍 **Search**: Full-text search across your workspace
- 📖 **Read**: Retrieve complete page content
- ✍️ **Create**: Make new notes with content and tags
- ✏️ **Update**: Modify existing pages
- 🏗️ **Structure**: View workspace organization

### Use Cases

- **Automatic Note-Taking**: "Remember this in Affine: API key expires March 1st"
- **Meeting Minutes**: "Create meeting notes for today's standup"
- **Knowledge Search**: "What did we decide about the database migration?"
- **Project Documentation**: "Update my project plan with today's progress"

See [Affine Integration Guide](docs/AFFINE_INTEGRATION.md) for detailed documentation.

---

## Alternative Shorter Version

If you prefer a more concise section:

---

## 📝 Affine Integration

Connect to [Affine](https://affine.pro) for AI-powered note-taking:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "YOUR_API_KEY",
      "workspace_id": "YOUR_WORKSPACE_ID"
    }
  }
}
```

**Features**: List pages, search notes, create/update content, manage tags

**Example**: `picoclaw agent -m "Create a note in Affine titled 'Meeting Notes'"`

See [docs/AFFINE_INTEGRATION.md](docs/AFFINE_INTEGRATION.md) for details.

---

## Table of Contents Update

Also update the table of contents to include:

```markdown
- [Chat Apps](#chat-apps)
- [Affine Integration](#affine-integration)  <!-- Add this line -->
- [Configuration](#configuration)
```

## Features Section Update

In the features list, you can add:

```markdown
🦾 **Demonstration**

### 🛠️ Standard Assistant Workflows

- 🧩 Full-Stack Engineer
- 🗂️ Logging & Planning Management
- 🔎 Web Search & Learning
- 📝 Affine Knowledge Base Integration  <!-- Add this -->
```
