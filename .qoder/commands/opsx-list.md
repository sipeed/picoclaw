---
description: List all active OpenSpec changes
usage: /opsx:list
aliases: [/opsx:ls]
---

# /opsx:list - List Changes

Displays all active OpenSpec changes with their completion status.

## What This Does

- Lists all directories in `openspec/changes/`
- Shows task completion status for each change
- Indicates which changes are active vs completed
- Sorts by most recently modified

## Usage

```
/opsx:list
```

Example output:
```
Changes:
  context-dynamic-selection-enhancement     23/47 tasks    2 hours ago
  api-rate-limiting                         0/32 tasks     1 day ago
  user-auth-v2                              Complete       1 week ago
```

## Output Format

Each change shows:
- **Name**: Directory name (kebab-case)
- **Progress**: Completed/Total tasks or "Complete"
- **Age**: Time since last modification

## Filtering

Show only specific states:
```
/opsx:list --active      # Only changes with pending tasks
/opsx:list --complete    # Only completed changes
```

## Integration with Other Commands

```bash
/opsx:list                          # See all changes
/opsx:show <change-name>           # View details of one change
/opsx:apply <change-name>          # Start implementing
/opsx:validate --changes <name>    # Check completeness
```

## File Locations

All changes stored in:
```
openspec/changes/
├── change-1/
│   ├── .openspec.yaml
│   ├── proposal.md
│   ├── design.md
│   ├── specs/
│   └── tasks.md
└── change-2/
    └── ...
```

## Related Commands

- `/opsx:new` - Create new change
- `/opsx:show` - Display change details
- `/opsx:archive` - Archive completed change
