---
description: Archive completed OpenSpec change and update main specs
usage: /opsx:archive <change-name>
---

# /opsx:archive - Archive Completed Change

Archives a completed change and merges its specifications into the main codebase.

## What This Does

- Validates all tasks are complete
- Moves change to `openspec/changes/archive/`
- Merges approved specs into `openspec/specs/`
- Updates main specification documents
- Preserves historical record

## Usage

```
/opsx:archive <change-name>
```

Example:
```
/opsx:archive context-dynamic-selection-enhancement
```

## Pre-Archive Checklist

Before archiving, ensure:
- ✅ All tasks in tasks.md are marked complete (`- [x]`)
- ✅ All tests pass (`go test ./...`)
- ✅ Code is committed to version control
- ✅ Documentation is updated
- ✅ `/opsx:validate --changes <name>` passes

## Archive Process

1. **Validation**: Verify all tasks complete
2. **Review**: Final check of implementation
3. **Move**: Transfer to archive directory
4. **Merge**: Integrate specs into main specs
5. **Update**: Modify openspec/specs/index.md
6. **Timestamp**: Add completion date

## Directory Structure After Archive

```
openspec/changes/
├── active-change-1/          # Still active
├── active-change-2/          # Still active
└── archive/                  # ← Completed changes moved here
    ├── 2026-02-26-context-dynamic-selection-enhancement/
    │   ├── proposal.md
    │   ├── design.md
    │   ├── specs/
    │   └── tasks.md (all checked)
    └── 2026-02-20-api-rate-limiting/
        └── ...
```

## Spec Migration

New capabilities from the change are merged into main specs:
```
openspec/specs/
├── agent-context-builder/    # From archived change
│   └── spec.md
├── tool-registry/            # From archived change
│   └── spec.md
└── index.md                  # Updated with new specs
```

## Rollback

If you need to restore an archived change:
```bash
mv openspec/changes/archive/<date>-<name> openspec/changes/<name>
```

Note: You may need to re-validate after rollback.

## Related Commands

- `/opsx:validate` - Verify completeness before archiving
- `/opsx:list` - See all changes including archived
- `/opsx:show` - View archived change details
