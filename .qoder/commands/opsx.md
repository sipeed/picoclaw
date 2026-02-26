---
description: OpenSpec spec-driven development workflow
usage: /opsx:<command> [args]
aliases: [/openspec]
---

# OpenSpec Slash Commands

OpenSpec provides a spec-driven development workflow for AI-assisted coding.

## Available Commands

### Core Workflow

| Command | Description | Example |
|---------|-------------|---------|
| `/opsx:new` | Create new change | `/opsx:new feature-name` |
| `/opsx:ff` | Fast-forward generate all docs | `/opsx:ff` |
| `/opsx:apply` | Implement tasks from tasks.md | `/opsx:apply change-name` |
| `/opsx:archive` | Archive completed change | `/opsx:archive change-name` |

### Management

| Command | Description | Example |
|---------|-------------|---------|
| `/opsx:list` | List all changes | `/opsx:list` |
| `/opsx:show` | Display change details | `/opsx:show change-name` |
| `/opsx:validate` | Validate completeness | `/opsx:validate change-name` |

## Quick Start

```bash
# 1. Create a new change
/opsx:new my-feature

# 2. Generate all planning documents
/opsx:ff

# 3. Review generated docs
/opsx:show my-feature

# 4. Validate quality
/opsx:validate my-feature

# 5. Start implementation
/opsx:apply my-feature

# 6. After completion, archive
/opsx:archive my-feature
```

## Workflow Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐     ┌──────────────┐
│  /opsx:new  │ ──→ │   /opsx:ff   │ ──→ │ /opsx:apply │ ──→ │ /opsx:archive│
│  Create     │     │   Generate   │     │  Implement  │     │  Complete    │
│  Change     │     │   Docs       │     │   Tasks     │     │  & Merge     │
└─────────────┘     └──────────────┘     └─────────────┘     └──────────────┘
```

## File Structure

Changes are stored in:
```
openspec/changes/
├── my-feature/
│   ├── .openspec.yaml      # Metadata
│   ├── proposal.md         # Why, What, Capabilities
│   ├── design.md           # Technical decisions
│   ├── specs/              # Detailed specifications
│   │   ├── capability-1/
│   │   └── capability-2/
│   └── tasks.md            # Implementation checklist
└── archive/                # Completed changes
```

## Best Practices

✅ **Do**:
- Always start with `/opsx:new` for significant features
- Use `/opsx:ff` to generate comprehensive planning docs
- Reference specs during implementation
- Update tasks.md as you complete each task
- Run `/opsx:validate` before archiving

❌ **Don't**:
- Skip the planning phase (defeats the purpose)
- Modify tasks.md structure (breaks parsing)
- Archive incomplete changes
- Ignore validation errors

## Configuration

OpenSpec is configured via:
- Global commands: `~/.qoder/commands/opsx*.md`
- Project config: `openspec/` directory
- CLI tool: `openspec` (installed via npm)

## Learn More

- Full documentation: `openspec --help`
- Supported tools: See OpenSpec README.md
- Schema reference: See openspec/schemas/

## Related Resources

- [OpenSpec GitHub](https://github.com/Fission-AI/OpenSpec)
- [Spec Kit Comparison](https://github.com/Fission-AI/OpenSpec#why-openspec)
- [Workflow Schemas](docs/workflows.md)
