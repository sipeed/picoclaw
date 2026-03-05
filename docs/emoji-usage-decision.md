# Emoji Usage Decision for Issue #578

## Summary

Issue #578 raised concern about emoji inconsistency - specifically that PicoClaw uses the lobster emoji (🦞) instead of the crab emoji (🦀).

## Investigation Results

A comprehensive search of the codebase found:

- **Lobster emoji (🦞)**: Used in **4 files** consistently
- **Crab emoji (🦀)**: **0 instances found**

### Files Using Lobster Emoji (🦞):

1. `pkg/channels/telegram/telegram_commands.go` - Line 59: `"Hello! I am PicoClaw 🦞"`
2. `pkg/agent/context.go` - Line 72: `fmt.Sprintf("# picoclaw 🦞\n\nYou are picoclaw...")`
3. `workspace/IDENTITY.md` - Line 4: `PicoClaw 🦞`
4. `cmd/picoclaw/internal/helpers.go` - Line 12: `const Logo = "🦞"`

## Decision

**Keep the lobster emoji (🦞) and document the decision.**

### Rationale:

1. **Already Consistent**: All 4 usages already use the lobster emoji - there is no inconsistency to fix
2. **Historical Context**: The lobster emoji has been used since the project inception
3. **Brand Recognition**: The lobster emoji is now part of PicoClaw's brand identity
4. **Community Acceptance**: The project has grown significantly with this branding
5. **Minimal Disruption**: Changing the emoji would require updates across documentation, user-facing messages, and potentially confusion for existing users

## Action Taken

No code changes required. This document serves as:
- Evidence of the investigation
- Documentation of the emoji choice
- Reference for future contributors

## Alternative Considered

We considered switching to crab emoji (🦀) to be more literal to the name, but determined:
- The cost of change outweighs the benefit
- "PicoClaw" is a creative name that doesn't need to match a specific crustacean
- The lobster emoji has character and makes the brand memorable

## Related

- Issue #578: https://github.com/sipeed/picoclaw/issues/578