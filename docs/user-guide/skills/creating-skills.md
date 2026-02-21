# Creating Skills

Learn how to create custom skills to extend PicoClaw's capabilities.

## Skill Structure

A skill is a directory containing a `SKILL.md` file:

```
~/.picoclaw/workspace/skills/
  my-skill/
    SKILL.md      # Required: skill definition
```

## SKILL.md Format

The skill file uses YAML frontmatter for metadata and Markdown for content:

```markdown
---
name: my-skill
description: Brief description of what this skill does
metadata:
  key: value
---

# Skill Title

Instructions for the agent go here.

## Section

More details, commands, and examples...
```

### Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Skill identifier (alphanumeric with hyphens) |
| `description` | Yes | Brief description (max 1024 characters) |
| `metadata` | No | Additional metadata (JSON object) |

### Naming Rules

- Max 64 characters
- Alphanumeric characters and hyphens only
- Must start with a letter or number
- Examples: `github`, `my-skill`, `api-helper`

## Creating Your First Skill

### Step 1: Create the Directory

```bash
mkdir -p ~/.picoclaw/workspace/skills/hello
```

### Step 2: Create SKILL.md

```bash
cat > ~/.picoclaw/workspace/skills/hello/SKILL.md << 'EOF'
---
name: hello
description: A simple greeting skill that demonstrates skill creation.
---

# Hello Skill

This skill demonstrates basic skill structure.

## Usage

When the user asks for a greeting, respond warmly:

"Hello! I'm your PicoClaw assistant. How can I help you today?"

## Example Commands

Show current time:
```bash
date
```

Show system info:
```bash
uname -a
```
EOF
```

### Step 3: Verify Installation

```bash
picoclaw skills list
```

Output:
```
Installed Skills:
------------------
  hello (workspace)
    A simple greeting skill that demonstrates skill creation.
```

## Skill Content Best Practices

### 1. Clear Instructions

Write clear, actionable instructions for the agent:

```markdown
# Database Backup Skill

## Purpose

Create backups of PostgreSQL databases.

## Commands

Full backup:
```bash
pg_dump -h localhost -U postgres -d mydb > backup_$(date +%Y%m%d).sql
```

Compressed backup:
```bash
pg_dump -h localhost -U postgres -d mydb | gzip > backup_$(date +%Y%m%d).sql.gz
```

## Important Notes

- Always verify the backup was created
- Store backups in a secure location
- Include timestamp in filename
```

### 2. Include Examples

Provide concrete examples for common use cases:

```markdown
## Examples

Backup specific tables:
```bash
pg_dump -h localhost -U postgres -d mydb -t users -t orders > partial_backup.sql
```

Restore from backup:
```bash
psql -h localhost -U postgres -d mydb < backup_20240115.sql
```
```

### 3. Document Dependencies

List required tools and how to install them:

```markdown
## Requirements

- `curl` - HTTP client
- `jq` - JSON processor

Install on macOS:
```bash
brew install curl jq
```

Install on Ubuntu:
```bash
sudo apt install curl jq
```
```

### 4. Add Error Handling

Include troubleshooting information:

```markdown
## Troubleshooting

### Connection Failed

If you see "Connection refused":
1. Check if PostgreSQL is running: `pg_isready`
2. Verify connection parameters
3. Check firewall settings

### Permission Denied

If you see "Permission denied":
1. Check user has backup privileges
2. Verify .pgpass file is configured
```

## Advanced Skill Features

### API Integration

Create skills that work with APIs:

```markdown
---
name: openai-api
description: Interact with OpenAI API for various AI tasks.
---

# OpenAI API Skill

## Setup

Set your API key:
```bash
export OPENAI_API_KEY="sk-..."
```

## Chat Completion

```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }' | jq .
```

## Image Generation

```bash
curl https://api.openai.com/v1/images/generations \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A sunset over mountains",
    "size": "1024x1024"
  }' | jq .
```
```

### Workflow Skills

Create skills for multi-step workflows:

```markdown
---
name: deploy
description: Deploy application to production with safety checks.
---

# Deployment Skill

## Pre-deployment Checklist

Before deploying, verify:
1. All tests pass: `npm test`
2. Build succeeds: `npm run build`
3. No security vulnerabilities: `npm audit`

## Deployment Steps

1. Create deployment branch:
```bash
git checkout -b deploy/$(date +%Y%m%d-%H%M)
```

2. Build and test:
```bash
npm ci
npm run build
npm test
```

3. Deploy:
```bash
npm run deploy:production
```

4. Verify deployment:
```bash
curl -s https://api.example.com/health | jq .
```

## Rollback

If issues occur:
```bash
npm run deploy:rollback
```
```

### Tool Requiring Skills

Document external tool requirements:

```markdown
---
name: docker
description: Manage Docker containers and images.
metadata:
  requires:
    bins: ["docker"]
    install:
      - id: brew
        kind: brew
        formula: docker
        bins: ["docker"]
      - id: apt
        kind: apt
        package: docker.io
        bins: ["docker"]
---

# Docker Skill

## Requirements

Docker must be installed. Install with:

macOS:
```bash
brew install --cask docker
```

Ubuntu:
```bash
sudo apt install docker.io
sudo usermod -aG docker $USER
```

## Common Commands

List containers:
```bash
docker ps -a
```

View logs:
```bash
docker logs <container>
```

Execute in container:
```bash
docker exec -it <container> /bin/sh
```
```

## Testing Your Skill

### 1. List and Verify

```bash
picoclaw skills list
```

Your skill should appear with the correct description.

### 2. Show Content

```bash
picoclaw skills show my-skill
```

Verify the content is displayed correctly.

### 3. Use in Conversation

Start a conversation and test the skill:

```bash
picoclaw agent -m "Use my-skill to help me with something"
```

## Sharing Skills

### Publishing to GitHub

1. Create a GitHub repository:
```
my-picoclaw-skills/
  my-skill/
    SKILL.md
  skills.json
```

2. Create `skills.json` registry:
```json
[
  {
    "name": "my-skill",
    "repository": "username/my-picoclaw-skills/my-skill",
    "description": "Description of my skill",
    "author": "your-username",
    "tags": ["automation", "productivity"]
  }
]
```

3. Users can install with:
```bash
picoclaw skills install username/my-picoclaw-skills/my-skill
```

### Contributing to Official Registry

Submit a pull request to the official skills repository:
https://github.com/sipeed/picoclaw-skills

## Skill Templates

### Basic Template

```markdown
---
name: skill-name
description: One-line description of the skill.
---

# Skill Name

Brief introduction to the skill.

## Commands

### Command 1

```bash
command-here
```

### Command 2

```bash
another-command
```

## Examples

Show practical usage examples.

## Troubleshooting

Common issues and solutions.
```

### API Template

```markdown
---
name: api-name
description: Interact with API-NAME service.
---

# API Name Skill

## Authentication

Set API key:
```bash
export API_KEY="your-key-here"
```

## Endpoints

### GET /resource

```bash
curl -H "Authorization: Bearer $API_KEY" \
  https://api.example.com/resource
```

### POST /resource

```bash
curl -X POST \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}' \
  https://api.example.com/resource
```
```

## Best Practices Summary

1. **Clear naming** - Use descriptive, lowercase names with hyphens
2. **Good descriptions** - Write concise descriptions that explain the skill's purpose
3. **Include examples** - Show how to use commands and features
4. **Document dependencies** - List required tools and installation methods
5. **Add troubleshooting** - Help users solve common problems
6. **Test thoroughly** - Verify your skill works before sharing
7. **Keep it focused** - One skill, one purpose

## See Also

- [Skills Overview](README.md)
- [Using Skills](using-skills.md)
- [Installing Skills](installing-skills.md)
- [Builtin Skills](builtin-skills.md)
- [CLI Reference: skills](../cli/skills.md)
