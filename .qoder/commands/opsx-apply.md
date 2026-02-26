---
description: Apply OpenSpec tasks and start implementation
usage: /opsx:apply [change-name]
---

# /opsx:apply - Implement Tasks

Starts implementation based on the tasks.md checklist.

## What This Does

- Reads `openspec/changes/<change-name>/tasks.md`
- Guides implementation task by task
- Tracks progress by updating checkboxes
- References specs and design for context

## Usage

```
/opsx:apply [change-name]
```

If change-name is omitted, uses the most recent active change.

Example:
```
/opsx:apply context-dynamic-selection-enhancement
```

## Implementation Flow

1. **Read Context**: Load proposal.md, design.md, and specs/*.md
2. **Parse Tasks**: Extract unchecked items from tasks.md
3. **Prioritize**: Start with Task 1.1 (first uncompleted)
4. **Implement**: Complete one task at a time
5. **Update**: Mark as `- [x]` when done
6. **Repeat**: Continue to next task

## Task Structure

Tasks are organized in phases:
```markdown
## 1. Infrastructure Setup
- [ ] 1.1 Modify AgentInstance to add mutex
- [ ] 1.2 Implement SetSkillsFilter method

## 2. Tool Visibility Filters
- [ ] 2.1 Define ToolVisibilityContext struct
- [ ] 2.2 Define ToolVisibilityFilter type
```

## Best Practices

✅ **Do**:
- Complete tasks in order (they're dependency-sorted)
- Update tasks.md immediately after completing each task
- Run tests after each phase
- Reference specs for requirements

❌ **Don't**:
- Skip tasks (breaks dependency chain)
- Batch update multiple tasks (lose granularity)
- Modify tasks.md structure (parsing depends on format)

## Progress Tracking

Check progress anytime:
```
/opsx:list
```

Shows completion percentage: `23/47 tasks`

## Related Commands

- `/opsx:ff` - Generate planning docs before applying
- `/opsx:validate` - Verify implementation completeness
- `/opsx:archive` - Archive after all tasks complete
