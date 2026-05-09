# Picoclaw Learning Vault & Tool System Design

**Date**: 2026-05-06  
**Status**: Approved  
**Based on**: Hermes-agent learning mechanisms + Obsidian vault system + OpenClaw plugin architecture

## Overview

Enhance picoclaw's learning capabilities with three memory layers inspired by Hermes-agent's time-based learning, while adopting Obsidian's vault metadata patterns. The system extends existing architecture (JSONL sessions, MEMORY.md, skills) rather than replacing it, preserving picoclaw's <20MB RAM target for 64MB boards.

### Key Design Decisions

1. **Approach**: Layered Enhancement (extend existing systems)
2. **Episodic Memory**: JSONL sessions + Obsidian-style frontmatter
3. **Semantic Memory**: Obsidian-compatible markdown vault (`---` frontmatter, `[[links]]`, `#tags`)
4. **Procedural Memory**: Tool skills as markdown wrappers with metadata + community registry
5. **UI**: Minimal web UI enhancements (vanilla JS, no frameworks, ~25KB gzipped)

---

## Section 1: Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│              Picoclaw Agent Core                     │
│  (pkg/agent/, existing LLM + ToolRegistry)         │
└────────────┬────────────────────────┬───────────────┘
             │                        │
    ┌────────▼────────┐    ┌────────▼──────────┐
    │ Episodic Layer  │    │ Procedural Layer  │
    │ (JSONL +        │    │ (Skills + Tools)  │
    │  Frontmatter)   │    │ + Tool Wrappers   │
    └────────┬────────┘    └────────┬──────────┘
             │                        │
             └────────┬───────────────┘
                      │
             ┌────────▼────────┐
             │ Semantic Layer   │
             │ (MEMORY.md      │
             │  Vault + Tags)  │
             └─────────────────┘
                      │
             ┌────────▼────────┐
             │  Web UI         │
             │  (Enhanced with │
             │   sidebar/tags) │
             └─────────────────┘
```

### Key Principles

- **Extend, don't replace**: JSONL sessions stay as primary storage, gain frontmatter metadata
- **Vault compatibility**: Semantic memory uses Obsidian-compatible markdown
- **Size-first**: All changes must preserve <20MB RAM target; no heavy frameworks
- **Skill-centric tools**: New tools added as skills (markdown wrappers) with enhanced metadata

### Extensions to Existing Systems

1. `pkg/memory/store.go` (JSONL) → Add frontmatter parser, tag extraction
2. `pkg/seahorse/` → Add time-based retrieval with tag filtering
3. `pkg/tools/registry.go` → Tool skills with metadata in `~/.picoclaw/workspace/skills/tools/`
4. `web/backend/` → Vanilla JS sidebar with vault browser, tag cloud, search

---

## Section 2: Episodic Memory Layer (Time-Based Session Storage)

### Current State
Picoclaw uses JSONL in `pkg/memory/store.go` for session storage — append-only, with session shards and compaction.

### Enhancement: Obsidian-Style Frontmatter for JSONL

Each session line becomes:
```jsonl
---
session_id: "abc123"
timestamp: "2026-05-06T14:30:00Z"
model: "claude-sonnet-4"
tags: ["coding", "bug-fix", "golang"]
parent_session: "def456"
related_sessions: ["ghi789"]
status: "completed"
---
{
  "role": "user",
  "content": "Fix the memory leak in seahorse engine"
}
...
```

### Key Additions to `pkg/memory/store.go`

#### 1. Frontmatter Parser (new file `pkg/memory/frontmatter.go`)
- Parse `---` delimited YAML from JSONL lines
- Extract: timestamp, tags, session relationships, model used
- Store metadata in SQLite index for fast queries

#### 2. Time-Based Retrieval (extend `pkg/seahorse/retrieval.go`)
- Search sessions by date range: `memory_search --after 2026-05-01 --tag coding`
- Link sessions via `parent_session_id` (inspired by Hermes-agent)
- Generate "session threads" — chains of related sessions

#### 3. Tag System (new)
- Auto-tag sessions based on tools used (e.g., `tool:exec`, `tool:read`)
- Manual tags via new `session_tag` tool
- Tag cloud in web UI for browsing past sessions

#### 4. Size Optimization
- Frontmatter stored inline in JSONL (no separate DB)
- SQLite index only for metadata queries (optional, can be disabled for 64MB boards)
- Index file stored in `~/.picoclaw/workspace/memory/index.db` (modernc.org/sqlite, already a dependency)

### Integration with Existing Code

- Extend `SessionMessage` struct with `Tags []string` and `Metadata map[string]interface{}`
- Modify `AppendSession()` to accept optional frontmatter
- Add `SearchSessionsByTag()` and `SearchSessionsByDate()` methods
- Keep backward compatibility: old JSONL lines without frontmatter still work

---

## Section 3: Semantic Memory Layer (Obsidian-Compatible Vault)

### Current State
Picoclaw has `MEMORY.md` in `~/.picoclaw/workspace/memory/MEMORY.md` — a single markdown file injected into system prompts.

### Enhancement: Transform into Obsidian-Compatible Vault

#### Vault Structure

```
~/.picoclaw/workspace/vault/
├── MEMORY.md          # Main memory (existing, enhanced)
├── USER.md            # User profile (existing, enhanced)
├── notes/             # Auto-generated knowledge notes
│   ├── 2026-05-06-session-summary.md
│   ├── golang-patterns.md
│   └── tool-usage.md
├── tags/              # Tag index (auto-generated)
│   ├── coding.md
│   ├── bug-fix.md
│   └── _index.md
└── index.md           # Vault root with [[links]] to all notes
```

#### MEMORY.md Format (Enhanced with Frontmatter)

```markdown
---
title: "Agent Memory"
updated: "2026-05-06T15:00:00Z"
tags: ["memory", "core"]
type: "semantic"
---

# Agent Memory

## User Preferences
- Prefers Go over Python for backend work
- Uses VS Code as primary editor

## Learned Patterns
- [[golang-patterns]] — Common Go patterns I've learned
- [[tool-usage]] — Which tools work best for specific tasks

## Session History
- [[2026-05-06-session-summary]] — Today's debugging session
```

### New Tools for Vault Management

#### 1. `vault_note_create` — Create a new note with frontmatter
- Parameters: `title`, `content`, `tags[]`, `links[]`
- Stores in `vault/notes/` with date prefix

#### 2. `vault_search` — Search vault (frontmatter + content)
- Uses existing SQLite FTS5 (already in seahorse)
- Search by tag, date, content, outgoing [[links]]

#### 3. `vault_link_add` — Add [[wiki-link]] between notes
- Maintains Obsidian compatibility
- Auto-generates backlinks

#### 4. `vault_tag_list` — List all tags with note counts
- Powers tag cloud in web UI

### Integration with Learning System

- **From Episodic Layer**: After session compaction, auto-create `vault/notes/YYYY-MM-DD-session-summary.md` with key learnings
- **From Tool Usage**: Track which tools work best for tasks, create `tool-usage.md` notes
- **Time-Based Decay**: Old notes get `status: archived` frontmatter (inspired by Hermes' Holographic provider with `temporal_decay_half_life`)

### Size Optimization

- Markdown files: Minimal overhead (already used for skills)
- SQLite FTS index: Optional, disabled by default on 64MB boards
- Tag index: Generated on-demand, not persisted
- Vault browser UI: Vanilla JS, loads note list via API (no client-side markdown parsing)

---

## Section 4: Procedural Memory Layer (Tool Skills & Community System)

### Current State
Picoclaw has skills in `~/.picoclaw/workspace/skills/` (markdown SKILL.md files) and tools registered in `ToolRegistry` via Go code.

### Enhancement: Tool Skills as Markdown Wrappers with Metadata

#### Tool Skill Structure (Extended SKILL.md)

```markdown
---
name: "file-search"
description: "Search files using ripgrep with common patterns"
type: "tool"
tags: ["filesystem", "search", "community"]
version: "1.0.0"
author: "community"
usage_count: 0
last_used: ""
related_tools: ["read", "grep"]
examples:
  - "Search for TODO items: rg 'TODO' --type go"
  - "Find config files: rg 'config' --type yaml"
---

# File Search Tool Wrapper

## When to Use
When you need to search across multiple files quickly.

## Tool Wrapper
This skill wraps the `exec` tool with pre-configured ripgrep commands.

## Common Patterns
- `rg 'pattern' --type <ext>` for language-specific search
- `rg 'pattern' -i` for case-insensitive search

## Learned From
Session [[2026-05-06-debugging-session]] — learned that ripgrep is faster than grep.

## Pitfalls
- Requires ripgrep installed (apt install ripgrep)
- Binary files can clutter results (use `-g '!.git'`)
```

#### Tool Skill Storage

```
~/.picoclaw/workspace/skills/
├── tools/                    # Tool wrappers (new)
│   ├── file-search/
│   │   └── SKILL.md
│   ├── docker-manager/
│   │   └── SKILL.md
│   └── _index.md           # Auto-generated tool catalog
├── builtin/                 # Existing built-in skills
└── community/               # Community-contributed skills
    └── (cloned from registry)
```

### Community Registry (Git-Based, Lightweight)

#### Registry Structure (hosted on GitHub/GitLab)

```
picoclaw-community-tools/
├── index.json                # Tool catalog
├── tools/
│   ├── file-search/
│   │   ├── SKILL.md
│   │   └── meta.json       # Download stats, ratings
│   └── docker-manager/
│       └── ...
└── README.md
```

#### index.json Format

```json
{
  "tools": [
    {
      "name": "file-search",
      "description": "Search files with ripgrep",
      "tags": ["filesystem", "search"],
      "download_url": "https://github.com/.../file-search.zip",
      "sha256": "abc123...",
      "stars": 42,
      "updated": "2026-05-06"
    }
  ]
}
```

### New Tools for Community System

#### 1. `tool_skill_install` — Install tool skill from registry
- `tool_skill_install --name file-search`
- Downloads SKILL.md to `~/.picoclaw/workspace/skills/tools/`
- Updates usage tracking in frontmatter

#### 2. `tool_skill_search` — Search community registry
- `tool_skill_search --tag filesystem`
- Queries index.json from registry URL

#### 3. `tool_skill_publish` — Share your tool skill (optional)
- Packages SKILL.md + meta.json
- Generates PR instructions for registry

#### 4. `tool_skill_rate` — Track usage/rating (local only)
- Updates `usage_count` and `last_used` in frontmatter
- Time-based: frequently used tools get higher priority in suggestions

### Integration with ToolRegistry

- Tool skills DON'T register new Go tools — they wrap existing ones with metadata
- When agent uses a tool, check for related tool skills
- Inject tool skill examples/patterns into LLM context (like MEMORY.md)
- Skills become "procedural memory" — agent learns which tools work best for tasks

### Size Optimization

- Registry: Simple git repo + index.json (no server needed)
- Tool skills: Markdown files (~2-5KB each)
- No new dependencies — uses existing `web_fetch` tool to download from registry
- Optional: Disable community features with `tools.community: false` in config

---

## Section 5: UI Design (Minimal Web UI with Obsidian-Inspired Elements)

### Current State
Picoclaw has a web-based UI with Go backend (`web/backend/`) serving a static single-page app (`web/backend/dist/`). Features: chat interface, tool management, model/config controls.

### Enhancement: Add Obsidian-Inspired UI Elements Using Vanilla JS/CSS

#### UI Layout (Obsidian-Inspired)

```
┌──────────────────────────────────────────────────────┐
│  [≡] Sidebar Toggle  │  Picoclaw Agent  │  [⚙] [?] │
├──────┬───────────────────────────────────────────────┤
│      │                                              │
│ Vault│  Chat Interface                               │
│ Side │  ┌────────────────────────────────────────┐  │
│ bar  │  │ User: How do I search files?          │  │
│      │  │ Agent: Use the file-search tool...     │  │
│ 📁   │  │ [Tool: exec] [Skill: file-search]    │  │
│ Tags │  └────────────────────────────────────────┘  │
│ 🔗   │                                              │
│ ⚙    │  ┌────────────────────────────────────────┐  │
│      │  │ Tool Skill Suggestions:                 │  │
│      │  │ • file-search (used 5x, 2 days ago)  │  │
│      │  │ • docker-manager (new in registry)      │  │
│      │  └────────────────────────────────────────┘  │
└──────┴───────────────────────────────────────────────┘
```

#### Sidebar Components (Vanilla JS)

##### 1. Vault Browser (`<div id="vault-sidebar">`)
- Tree view of `~/.picoclaw/workspace/vault/` notes
- Show `[[links]]` as expandable nodes
- Click to open note in preview pane (lightbox modal)
- CSS: Use CSS custom properties for theming (Obsidian-style)

##### 2. Tag Cloud (`<div id="tag-cloud">`)
- List all tags from vault + session frontmatter
- Click tag to filter sessions/notes
- Show count badges (e.g., `#coding (42)`)
- Stored in `vault/tags/_index.md` (auto-generated)

##### 3. Session History (`<div id="session-history">`)
- Chronological list of past sessions (from JSONL + frontmatter)
- Group by date (Today, Yesterday, Last Week...)
- Show tags as colored pills
- Click to load session context

##### 4. Tool Skills Panel (`<div id="tool-skills">`)
- List installed tool skills from `~/.picoclaw/workspace/skills/tools/`
- Show `usage_count` from frontmatter
- "Browse Community" button → modal with registry search
- Install via `tool_skill_install` (AJAX call to backend)

### New Backend API Endpoints (Go, extend `web/backend/main.go`)

```
GET  /api/vault/list           → List vault notes (tree structure)
GET  /api/vault/note?path=     → Get note content + frontmatter
POST /api/vault/note           → Create/update note
GET  /api/vault/tags           → List all tags with counts
GET  /api/sessions/search      → Search sessions by tag/date
GET  /api/tools/skills        → List tool skills
POST /api/tools/skills/install → Install from registry
GET  /api/tools/registry      → Fetch registry index.json
```

### Vanilla JS Implementation (No Frameworks)

#### File Structure (extend existing `web/backend/dist/`)
```
web/backend/dist/
├── index.html
├── app.js                    # Existing app logic
├── vault.js                  # NEW: Vault sidebar logic
├── tags.js                   # NEW: Tag cloud logic
├── sessions.js               # NEW: Session history logic
├── tools-skills.js           # NEW: Tool skills panel
├── styles/
│   ├── main.css
│   └── vault.css            # NEW: Vault/sidebar styles
└── assets/
    └── icons/               # Lucide-style SVG icons (inline)
```

#### Key Patterns (Obsidian-Inspired)
- **CSS Custom Properties**: Define `--background-primary`, `--text-normal`, `--interactive-accent` for theming
- **Modal Dialogs**: Reuse existing patterns (see `15eb2e62 fix: integrate PermissionPrompt dialog`)
- **Icons**: Inline SVG (no icon font download), 24x24 viewBox, 2px stroke (Lucide guidelines)
- **No Build Step**: Plain `.js` files, no webpack/vite (keep it simple)

### Size Optimization

| Component | Size Estimate | Optimization |
|------------|---------------|---------------|
| vault.js + tags.js + sessions.js + tools-skills.js | ~15KB gzipped | Minify only, no bundler |
| vault.css | ~8KB gzipped | Minimal selectors, reuse existing styles |
| SVG icons (inline) | ~2KB total | Only icons actually used |
| **Total added** | **~25KB gzipped** | Well within 2MB budget |

### Local-First Benefits (Obsidian Lesson)

- No loading states for cloud sync (all local)
- Instant UI responses (no API delays for vault browsing)
- Works offline (vault stored in `~/.picoclaw/workspace/`)
- Simpler code (no conflict resolution, no presence indicators)

---

## Implementation Notes

### Dependencies
- No new Go dependencies required
- Uses existing: `modernc.org/sqlite`, `encoding/json`, `golang.org/x/exp`
- Frontmatter parsing: Use `gopkg.in/yaml.v3` (already in go.mod or add)

### Config Changes
New config options in picoclaw.json:
```json
{
  "memory": {
    "vault_enabled": true,
    "session_index_enabled": false,
    "community_tools_enabled": true
  }
}
```

### Backward Compatibility
- Old JSONL sessions without frontmatter: Still work, no tags
- Old MEMORY.md without frontmatter: Treated as body-only, no metadata
- Skills without tool metadata: Still load, no usage tracking

### Testing Strategy
1. Unit tests for frontmatter parser
2. Integration tests for vault API endpoints
3. UI tests for sidebar components (manual, screenshot-based)
4. Performance tests: Ensure <20MB RAM with all features enabled

---

## Summary

This design enhances picoclaw with:
- **Time-based learning** via Hermes-agent-inspired three-layer memory
- **Obsidian compatibility** via markdown vault with frontmatter, tags, and wiki-links
- **Community tool sharing** via lightweight git-based registry and skill wrappers
- **Minimal UI** with ~25KB of vanilla JS enhancements

All while preserving picoclaw's core constraint: **<20MB RAM for 64MB boards**.
