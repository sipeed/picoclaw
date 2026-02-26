---
description: Create a new OpenSpec change proposal
usage: /opsx:new <change-name>
---

# /opsx:new - Create New Change

Creates a new OpenSpec change directory with the spec-driven schema.

## What This Does

- Creates `openspec/changes/<change-name>/` directory
- Initializes `.openspec.yaml` metadata file
- Sets up the spec-driven workflow schema

## Usage

```
/opsx:new <change-name>
```

Example:
```
/opsx:new context-dynamic-selection-enhancement
```

## Next Steps

After creating the change, use:
- `/opsx:ff` - Fast-forward to generate all planning docs (proposal, specs, design, tasks)
- `/opsx:apply` - Start implementation based on tasks.md
- `/opsx:archive` - Archive completed change

## Related Commands

- `/opsx:list` - List all active changes
- `/opsx:validate` - Validate change completeness
- `/opsx:show` - Display change details
