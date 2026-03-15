# Skills Registries

PicoClaw supports installing skills from multiple registries. This guide covers how to use and create registries.

## Using Registries

### List Available Skills

```bash
picoclaw skills search <query>
```

### Install a Skill

```bash
# Install from a specific registry
picoclaw skills install --registry index:angelhub self-config
picoclaw skills install --registry clawhub github

# Install directly from GitHub hosted SKILL.md
picoclaw skills install owner/repo/skill-name
```

### List Installed Skills

```bash
picoclaw skills list
```

## Supported Registries

### ClawHub Registry

The default registry at [clawhub.ai](https://clawhub.ai). Enable in config:

```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "enabled": true
        }
      }
    }
  }
}
```

### Index Fronted Registry

Install skills from any HTTP endpoint that serves a skills-index.json.

**Basic Configuration:**

```json
{
  "tools": {
    "skills": {
      "registries": {
        "index:myorg": {
          "enabled": true,
          "index_url": "https://example.com/skills-index.json"
        }
      }
    }
  }
}
```

**Configuration with security options:**

```json
{
  "tools": {
    "skills": {
      "registries": {
        "index:angelhub": {
          "enabled": true,
          "index_url": "https://raw.githubusercontent.com/wiki/keithy/angelhub/picoclaw-skills-index.json",
          "extra_header": "X-Custom-Header: value",
          "authorization_header": "Bearer token",
          "agent_header": "picoclaw/1.0",
          "allowed_prefixes": [
            "https://raw.githubusercontent.com/wiki/keithy/angelhub/",
            "https://raw.githubusercontent.com/keithy/angelhub/"
          ]
        }
      }
    }
  }
}
```

**Configuration for local development:**

Use `url_mappings` to redirect downloads to local files or forks, and `symlink_local` to create symlinks instead of copying:

```json
{
  "tools": {
    "skills": {
      "registries": {
        "index:angelhub": {
          "enabled": true,
          "index_url": "https://raw.githubusercontent.com/wiki/keithy/angelhub/picoclaw-skills-index.json",
          "allowed_prefixes": [
            "https://raw.githubusercontent.com/wiki/keithy/angelhub/",
            "https://raw.githubusercontent.com/keithy/angelhub/"
          ],
          "url_mappings": {
            "https://raw.githubusercontent.com/keithy/angelhub/main/": "https://raw.githubusercontent.com/keithy/angelhub/beta/",
            "https://raw.githubusercontent.com/keithy/angelhub/": "file:///home/me/repos/angelhub/"
          },
          "symlink_local": true
        }
      }
    }
  }
}
```

| Option | Description |
|--------|-------------|
| `url_mappings` | Map URL prefixes to redirect downloads. Useful for testing forks (`https://.../main/` → `https://.../beta/`) or local development (`https://...` → `file:///path/to/repo/`) |
| `symlink_local` | When using `file://` URLs in mappings, create a symlink to the local directory instead of copying files. Useful for development |

## Creating Your Own Registry

To create a skill registry:

### 1. Create a Repository

Create a public repository to host your skills.

### 2. Add Skills

Add skills in the `picoclaw/skills/` directory (or `skills/` for ecosystem-agnostic). Each skill needs a `SKILL.md` file:

```
picoclaw/
└── skills/
    ├── self/
    │   ├── self-config/
    │   │   └── SKILL.md
    │   └── self-debug/
    │       └── SKILL.md
    └── weather/
        └── SKILL.md
```

### 3. Create the Index Workflow

Add a workflow to generate the skills index. See [AngelHub's workflow](https://github.com/keithy/angelhub/blob/main/.github/workflows/picoclaw-skills-index.yml) for a complete example.

### 4. Enable in PicoClaw

```json
{
  "tools": {
    "skills": {
      "registries": {
        "index:myorg": {
          "enabled": true,
          "index_url": "https://example.com/skills-index.json"
        }
      }
    }
  }
}
```

## Skill Format

Each skill should have a `SKILL.md` file with frontmatter:

```markdown
---
name: skill-name
description: What the skill does
---

# Skill Name

Your skill documentation here...
```

## Index JSON Format

The `skills-index.json` should look like:

```json
{
  "version": 1,
  "skills": [
    {
      "slug": "my-skill",
      "name": "My Skill",
      "description": "Does something useful",
      "_path": "skills/my-skill",
      "download_url": "https://raw.githubusercontent.com/owner/repo/main/skills/my-skill",
      "files": ["SKILL.md", "scripts/helper.sh"]
    }
  ]
}
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `slug` | Yes | Unique identifier for the skill |
| `name` | No | Display name (defaults to slug) |
| `description` | No | Short description for search results |
| `_path` | No | Derived path to skill folder (for categorization) |
| `download_url` | Yes* | URL to download skill files from |
| `files` | No | List of files to download (if omitted, downloads from download_url directly) |

* A direct download url can reference a ZIP archive.

### Including External Skills

You can include skills from other sources by placing `.json` files in your skills directory:

```
skills/
├── self/
│   ├── self-config/
│   │   └── SKILL.md
│   └── external-skill/
│       └── skills.json    <-- included skills files obtained from any url.*
```

* Hence the `allowed_prefixes` config option.

The JSON file can contain an array of skill objects:

```json
[
  {
    "slug": "external-skill",
    "name": "External Skill",
    "description": "Skill defined in external JSON",
    "download_url": "https://raw.githubusercontent.com/other/repo/main/skill"
  }
]
```
