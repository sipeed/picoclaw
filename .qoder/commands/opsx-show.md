---
description: Show details of an OpenSpec change or spec
usage: /opsx:show <name>
aliases: [/opsx:view]
---

# /opsx:show - Display Change Details

Displays detailed information about a specific change or specification.

## What This Does

- Shows complete content of change artifacts
- Displays proposal, design, specs, and tasks
- Provides formatted markdown output
- Can show individual artifacts or entire change

## Usage

### Show Entire Change

```
/opsx:show <change-name>
```

Example:
```
/opsx:show context-dynamic-selection-enhancement
```

Output:
```
## Change: context-dynamic-selection-enhancement
Status: In Progress (23/47 tasks)
Created: 2026-02-26

=== proposal.md ===
[Full content of proposal.md]

=== design.md ===
[Full content of design.md]

=== specs/ ===
- context-strategies/spec.md
- tool-visibility-filters/spec.md
- skills-filter-api/spec.md
- skill-recommender/spec.md

=== tasks.md ===
[Task list with completion status]
```

### Show Specific Artifact

```
/opsx:show <change-name>/<artifact>
```

Examples:
```
/opsx:show context-dynamic-selection-enhancement/proposal
/opsx:show context-dynamic-selection-enhancement/design
/opsx:show context-dynamic-selection-enhancement/tasks
/opsx:show context-dynamic-selection-enhancement/specs/context-strategies
```

### Show Spec

```
/opsx:show spec/<spec-name>
```

Example:
```
/opsx:show spec/agent-context-builder
```

## Use Cases

### Review Before Implementation

```bash
/opsx:list                          # See all changes
/opsx:show <change-name>           # Read full context
/opsx:apply <change-name>          # Start implementation
```

### Check Task Progress

```bash
/opsx:show <change-name>/tasks     # View task checklist
```

### Reference Specific Spec

```bash
/opsx:show <change-name>/specs/skills-filter-api  # Read API spec
```

## Output Formatting

The command displays:
- **Metadata**: Status, creation date, last modified
- **Artifacts**: Full markdown content with proper formatting
- **Progress**: Task completion percentage
- **Dependencies**: Links between artifacts

## Integration with Other Commands

```bash
/opsx:new <name>                    # Create change
/opsx:ff                            # Generate docs
/opsx:show <name>                   # ← Review generated docs
/opsx:validate --changes <name>     # Verify quality
/opsx:apply <name>                  # Implement
```

## Related Commands

- `/opsx:list` - List all changes
- `/opsx:validate` - Validate completeness
- `/opsx:apply` - Start implementation
