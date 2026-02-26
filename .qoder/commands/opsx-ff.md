---
description: Fast-forward generate all OpenSpec planning documents
usage: /opsx:ff
aliases: [/opsx:fast-forward]
---

# /opsx:ff - Fast-Forward Planning Docs

Automatically generates all planning artifacts for the current change.

## What This Does

Generates the complete spec-driven workflow documentation:
1. **proposal.md** - Why, What, Capabilities, Impact
2. **specs/*.md** - Detailed specifications (one per capability)
3. **design.md** - Technical design decisions and rationale
4. **tasks.md** - Implementation task checklist

## Usage

```
/opsx:ff
```

This is shorthand for manually creating each artifact in sequence.

## Workflow

```
/opsx:new <change-name>     # Create change directory
/opsx:ff                    # Generate all planning docs ← You are here
/opsx:apply                 # Implement tasks
/opsx:archive               # Archive completed change
```

## Manual Alternative

If you prefer to create artifacts individually:
```
/openspec instructions --change <name> proposal
/openspec instructions --change <name> specs
/openspec instructions --change <name> design
/openspec instructions --change <name> tasks
```

## Output Structure

```
openspec/changes/<change-name>/
├── proposal.md      ← Generated
├── specs/
│   ├── capability-1/spec.md  ← Generated
│   └── capability-2/spec.md  ← Generated
├── design.md        ← Generated
└── tasks.md         ← Generated
```

## Related Commands

- `/opsx:new` - Create new change
- `/opsx:apply` - Start implementation
- `/opsx:validate` - Check completeness
