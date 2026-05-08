# Context Management Skill

Use in long or noisy sessions to persist durable state across session boundaries via state.md. Also generates project-map.md when asked to map the project. Triggers on: user explicitly asks to "save state", "compress context", "map this project", "generate project map", "create project map", cross-session handoff needed, or repeated failures indicate context is getting stale.

## Overview

This skill wraps PicoClaw's Seahorse context management system to provide:
- **State persistence** across sessions (state.md)
- **Project mapping** (project-map.md)
- **Context compression** via Seahorse
- **Context snapshots** for long-running sessions

## When to Use

### State Persistence
- User asks to "save state" or "persist context"
- Session is getting long/noisy
- Cross-session handoff needed
- Repeated failures indicate stale context

### Project Mapping
- User asks to "map this project" or "generate project map"
- First time setup for a new project
- Need to understand codebase structure

### Context Compression
- Context window approaching limit
- Need to compress old messages
- Proactive budget management

## How to Use

### Generate Project Map

```bash
# Scan project structure
cd /path/to/project
find . -type f -name "*.go" -o -name "*.ts" -o -name "*.js" | head -50

# Generate project-map.md
# Include:
# - Key directories and their purposes
# - Main entry points
# - Configuration files
# - Important modules/packages
# - Dependencies (go.mod, package.json, etc.)
```

**project-map.md format:**
```markdown
# Project Map: [Project Name]

## Overview
[Brief description]

## Directory Structure
- `cmd/`: [Purpose]
- `pkg/`: [Purpose]
- `internal/`: [Purpose]

## Key Files
- `main.go`: Entry point
- `config.json`: Configuration

## Architecture
[Brief architecture description]

## Generated: [timestamp]
## Git Hash: [latest commit hash]
```

### Save State

```bash
# Create/update state.md with:
# - Current task being worked on
# - Key decisions made
# - What was rejected and why
# - Next steps
```

**state.md format:**
```markdown
# Session State

## Current Task
[What is being worked on]

## Key Decisions
- [Decision and why]

## Rejected Approaches
- [Approach]: [Why rejected]

## Next Steps
- [Step 1]
- [Step 2]

## Last Updated: [timestamp]
```

### Trigger Compression

When context window is nearing limit:
1. Check `isOverContextBudget()` 
2. Call `ContextManager.Compact()` with reason
3. Re-assemble messages via `ContextManager.Assemble()`

## Integration with PicoClaw

This skill uses:
- **Seahorse context manager** (`pkg/seahorse/`) for SQLite-based compression
- **Memory store** (`pkg/agent/memory.go`) for MEMORY.md
- **Context builder** (`pkg/agent/context.go`) for system prompt

## Configuration

Enable in `config.json`:
```json
{
  "agents": {
    "defaults": {
      "context_manager": "seahorse",
      "context_window": 200000,
      "context_safety_buffer": 20000
    }
  }
}
```

## OpenCode Reference

This skill is inspired by OpenCode's `context-management` skill which:
- Dynamically loads .md files when reading related files
- Saves tool output to files when truncated
- Uses structured compaction (Goal/Progress/Decisions/Next Steps)
- Prunes old tool outputs during compaction

PicoClaw's implementation adds:
- Seahorse integration for hierarchical summarization
- project-map.md generation
- state.md persistence
