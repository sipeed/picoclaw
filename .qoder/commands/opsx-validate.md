---
description: Validate OpenSpec change completeness
usage: /opsx:validate [change-name]
---

# /opsx:validate - Validate Change

Validates that a change has all required artifacts and they meet quality standards.

## What This Does

- Checks for required artifacts (proposal, specs, design, tasks)
- Validates artifact structure and content
- Verifies task completion status
- Reports missing or incomplete items
- Ensures spec-driven schema compliance

## Usage

```
/opsx:validate [change-name]
```

If change-name is omitted, validates the most recently modified change.

Example:
```
/opsx:validate context-dynamic-selection-enhancement
/opsx:validate --changes api-rate-limiting
```

## Validation Checks

### Required Artifacts
- ✅ proposal.md exists and contains: Why, What Changes, Capabilities, Impact
- ✅ specs/*.md exists with proper requirement/scenario structure
- ✅ design.md exists with Context, Decisions, Risks sections
- ✅ tasks.md exists with properly formatted checkboxes

### Quality Checks
- ✅ All requirements have at least one scenario
- ✅ Scenarios use WHEN/THEN format
- ✅ Tasks are dependency-sorted
- ✅ No breaking changes without migration plan

### Completeness (during implementation)
- ✅ Task completion percentage
- ✅ Unchecked tasks remaining
- ✅ Test coverage for completed tasks

## Example Output

Success:
```
✓ change/context-dynamic-selection-enhancement
  ✓ proposal.md (complete)
  ✓ specs/ (4 capabilities)
  ✓ design.md (complete)
  ✓ tasks.md (23/47 tasks complete)
Totals: 1 passed (1 items)
```

Failure:
```
✗ change/api-rate-limiting
  ✗ proposal.md missing Capabilities section
  ✗ specs/ empty
Totals: 0 passed, 1 failed (1 items)
```

## When to Validate

**Before starting implementation:**
```bash
/opsx:new <name>
/opsx:ff
/opsx:validate    # ← Ensure planning docs are complete
```

**During implementation:**
```bash
/opsx:apply
# ... complete some tasks ...
/opsx:validate    # ← Check progress and quality
```

**Before archiving:**
```bash
# ... all tasks complete ...
/opsx:validate    # ← Final validation required
/opsx:archive
```

## Fixing Validation Errors

If validation fails:
1. Read the error message carefully
2. Use `/openspec instructions --change <name> <artifact>` to regenerate
3. Manually edit to fix structural issues
4. Re-run validation

## Related Commands

- `/opsx:ff` - Generate planning docs
- `/opsx:list` - See all changes
- `/opsx:archive` - Archive after validation passes
