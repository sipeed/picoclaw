# PicoClaw Context Window Improvements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-optimized:subagent-driven-development or superpowers-optimized:executing-plans. Steps use checkbox tracking.

**Goal:** Implement 7 context-window improvements to prevent "Context window exceeded" errors.
**Architecture:** Add `ContextOverflowError` type, token-aware fresh tail with `CalculateFreshTailBudget()`, structured summary template, `PruneToolOutputs()` utility, `CompactReason` type, and `TruncateToolOutput()` utility.
**Tech Stack:** Go 1.25, existing PicoClaw packages.
**Assumptions:** 
- Seahorse is active (`!mipsle && !netbsd`).
- Existing `pkg/tokenizer` already estimates at 4.0 chars/token (aligned with design).

---

## File Map

| File | Action | Responsibility |
|------|--------|--------------|
| `pkg/agent/errors.go` | Create | `ContextOverflowError` struct |
| `pkg/agent/pipeline_llm.go` | Modify | Use `ContextOverflowError`; wire `CompactReason` |
| `pkg/agent/context_manager.go` | Modify | Add `CompactReason` type |
| `pkg/agent/pipeline_setup.go` | Modify | Use `CompactReasonProactive` |
| `pkg/agent/context_legacy.go` | Modify | Use `CompactReason` constants |
| `pkg/seahorse/short_constants.go` | Modify | Add tail protection constants; `FreshTailCount` deprecate |
| `pkg/seahorse/short_assembler.go` | Modify | `CalculateFreshTailBudget()`; token-aware fresh tail |
| `pkg/seahorse/short_compaction.go` | Modify | Add `SummaryTemplate`, use in prompts |
| `pkg/utils/tool_pruner.go` | Create | `PruneToolOutputs()` utility |
| `pkg/utils/truncate.go` | Create | `TruncateToolOutput()` utility |
| `pkg/tokenizer/estimator.go` | Modify | Add `Config` struct with `CharsPerToken` |
| `pkg/seahorse/short_assembler_test.go` | Update | Adjust for new fresh tail logic |
| `pkg/agent/context_budget_test.go` | Update | Add `ContextOverflowError` tests |

---

## Task 1: Add `Config` to `pkg/tokenizer/estimator.go`

**Files:**
- Modify: `pkg/tokenizer/estimator.go`

**Does NOT cover:** Token estimation logic change (already at 4.0 chars/token).

- [ ] **Step 1: Add `Config` struct**

```go
// Config allows tuning the chars-per-token heuristic.
type Config struct {
    // CharsPerToken controls the conservative token estimate.
    // Default is 4.0 (matching OpenCode's approach).
    CharsPerToken float64
}

// DefaultConfig returns the default tokenizer configuration.
func DefaultConfig() Config {
    return Config{CharsPerToken: 4.0}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./pkg/tokenizer/`
Expected: PASS

## Task 2: Add `ContextOverflowError` and Update Pipeline

**Files:**
- Create: `pkg/agent/errors.go`
- Modify: `pkg/agent/pipeline_llm.go`
- Modify: `pkg/agent/context_budget_test.go`

**Does NOT cover:** Changing provider error strings or retry logic timing.

- [ ] **Step 2.1: Create `pkg/agent/errors.go`**

```go
package agent

import "fmt"

// ContextOverflowError is a structured error for context window exceeded.
type ContextOverflowError struct {
    Model           string
    ContextWindow   int
    RequestedTokens int
    Reason          string
}

func (e *ContextOverflowError) Error() string {
    return fmt.Sprintf("context window exceeded: model=%s window=%d requested=%d reason=%s",
        e.Model, e.ContextWindow, e.RequestedTokens, e.Reason)
}
```

- [ ] **Step 2.2: Update `pipeline_llm.go` to use `ContextOverflowError`**

In the `isContextError` branch, wrap the context window error:

```go
if isContextError && retry < maxRetries && !ts.opts.NoHistory {
    // ... existing event emission ...
    
    // Emit structured context overflow error
    al.emitEvent(
        runtimeevents.KindAgentError,
        ts.eventMeta("runTurn", "turn.error"),
        ErrorPayload{
            Stage:   "llm",
            Message: (&ContextOverflowError{
                Model:           exec.llmModel,
                ContextWindow:   ts.agent.ContextWindow,
                RequestedTokens: ts.agent.ContextWindow, // best estimate
                Reason:          "context_length_exceeded",
            }).Error(),
        },
    )
    
    // ... existing compact logic ...
}
```

- [ ] **Step 2.3: Verify test compilation**

Run: `go test ./pkg/agent/ -run TestContext -count=1 -v`
Expected: PASS (or no compile errors)

## Task 3: Add `CompactReason` Type and Update References

**Files:**
- Modify: `pkg/agent/context_manager.go`
- Modify: `pkg/agent/pipeline_setup.go`
- Modify: `pkg/agent/pipeline_llm.go`
- Modify: `pkg/agent/context_legacy.go`

**Does NOT cover:** Changing seahorse engine interface.

- [ ] **Step 3.1: Add `CompactReason` to `context_manager.go`**

Replace existing constants with typed version:

```go
// CompactReason distinguishes between proactive and reactive compaction.
type CompactReason string

const (
    CompactReasonProactive CompactReason = "proactive" // Before LLM call
    CompactReasonRetry     CompactReason = "retry"      // After context error
    CompactReasonOverflow  CompactReason = "overflow"   // During streaming
)

// Deprecated: use CompactReasonProactive
const ContextCompressReasonProactive CompactReason = CompactReasonProactive

// Deprecated: use CompactReasonRetry
const ContextCompressReasonRetry CompactReason = CompactReasonRetry
```

- [ ] **Step 3.2: Update `pipeline_setup.go`**

Change:
```go
Reason: ContextCompressReasonProactive,
```
to:
```go
Reason: CompactReasonProactive,
```

- [ ] **Step 3.3: Update `pipeline_llm.go`**

Change:
```go
Reason: ContextCompressReasonRetry,
```
to:
```go
Reason: CompactReasonRetry,
```

- [ ] **Step 3.4: Update `context_legacy.go`**

Update any references to `ContextCompressReasonProactive` / `ContextCompressReasonRetry` to use `CompactReasonProactive` / `CompactReasonRetry`.

- [ ] **Step 3.5: Verify build**

Run: `go build ./pkg/agent/`
Expected: PASS

## Task 4: Implement Token-Aware Fresh Tail Protection

**Files:**
- Modify: `pkg/seahorse/short_constants.go`
- Modify: `pkg/seahorse/short_assembler.go`
- Update: `pkg/seahorse/short_assembler_test.go`

**Does NOT cover:** Changing `compactLeaf` or `compactCondensed` (they still use `FreshTailCount` for message skipping).

- [ ] **Step 4.1: Add tail protection constants to `short_constants.go`**

```go
// Tail protection constants (from OpenCode's proven approach)
const (
    MinPreserveRecentTokens = 2000 // Never preserve less than this many tokens
    MaxPreserveRecentTokens = 8000 // Never preserve more than this many tokens
    PreserveRecentRatio     = 0.25 // 25% of usable context for fresh tail
    DefaultTailTurns        = 2    // Minimum number of turns to preserve
)
```

- [ ] **Step 4.2: Add `CalculateFreshTailBudget` and update `Assemble`**

In `short_assembler.go`, add:

```go
// CalculateFreshTailBudget determines how many tokens to reserve for recent history.
func CalculateFreshTailBudget(contextWindow int) int {
    budget := int(float64(contextWindow) * PreserveRecentRatio)
    if budget < MinPreserveRecentTokens {
        return MinPreserveRecentTokens
    }
    if budget > MaxPreserveRecentTokens {
        return MaxPreserveRecentTokens
    }
    return budget
}
```

Update `Assemble` method:

Replace:
```go
// Split into evictable prefix and protected fresh tail
tailStart := len(resolved) - FreshTailCount
if tailStart < 0 {
    tailStart = 0
}
evictable := resolved[:tailStart]
freshTail := resolved[tailStart:]
```

With token-aware logic:
```go
// Calculate fresh tail based on token budget (not fixed message count)
freshTailBudget := CalculateFreshTailBudget(input.Budget)
var freshTail []resolvedItem
var evictable []resolvedItem

accum := 0
for i := len(resolved) - 1; i >= 0; i-- {
    if accum+resolved[i].tokenCount <= freshTailBudget && len(freshTail) < len(resolved) {
        freshTail = append(freshTail, resolved[i])
        accum += resolved[i].tokenCount
    } else {
        evictable = append(evictable, resolved[i])
    }
}
// Reverse to restore chronological order
for i, j := 0, len(freshTail)-1; i < j; i, j = i+1, j-1 {
    freshTail[i], freshTail[j] = freshTail[j], freshTail[i]
}
for i, j := 0, len(evictable)-1; i < j; i, j = i+1, j-1 {
    evictable[i], evictable[j] = evictable[j], evictable[i]
}
```

- [ ] **Step 4.3: Update tests**

Update `short_assembler_test.go` to assert on token budgets instead of fixed message counts.

- [ ] **Step 4.4: Verify build and tests**

Run: `go test ./pkg/seahorse/ -run TestAssemble -count=1 -v`
Expected: PASS

## Task 5: Add Structured Summary Template

**Files:**
- Modify: `pkg/seahorse/short_compaction.go`

**Does NOT cover:** Changing leaf summary generation prompts directly.

- [ ] **Step 5.1: Add `SummaryTemplate` constant**

```go
// SummaryTemplate is a structured format for seahorse summaries.
// It ensures important context is preserved across compaction.
const SummaryTemplate = `# Summary of Previous Conversation

## Goal
[Current objective]

## Constraints
[Key limitations or requirements]

## Progress
- [List of completed items]

## Key Decisions
- [Decision 1 with rationale]
- [Decision 2 with rationale]

## Next Steps
- [Planned action]

## Critical Context
[Any other context needed to continue]`
```

- [ ] **Step 5.2: Update `generateLeafSummary` to use template**

Add a toggle flag and apply template in the prompt when enabled:

```go
// UseStructuredSummaries controls whether to use the structured template.
// Can be toggled via seahorse config.
var UseStructuredSummaries = false
```

Update `buildLeafSummaryPrompt` to include template instructions when enabled.

- [ ] **Step 5.3: Verify build**

Run: `go build ./pkg/seahorse/`
Expected: PASS

## Task 6: Add Tool Output Pruning Utility

**Files:**
- Create: `pkg/utils/tool_pruner.go`
- Create: `pkg/utils/tool_pruner_test.go`

- [ ] **Step 6.1: Create `pkg/utils/tool_pruner.go`**

```go
package utils

import "github.com/sipeed/picoclaw/pkg/providers"

// PruneToolOutputs walks backward through messages and replaces
// old tool outputs with stubs to save context window.
func PruneToolOutputs(messages []providers.Message, protectedTurns int, pruneThreshold int) []providers.Message {
    if len(messages) == 0 || protectedTurns <= 0 {
        return messages
    }
    
    result := make([]providers.Message, len(messages))
    copy(result, messages)

    // Determine how many messages to protect from the end
    protectedEnd := len(result) - protectedTurns
    if protectedEnd < 0 {
        protectedEnd = 0
    }

    prunedCount := 0
    for i := protectedEnd - 1; i >= 0; i-- {
        if result[i].Role == "tool" {
            result[i].Content = "[Tool output pruned — use expand for details]"
            prunedCount++
        }
    }

    if prunedCount < pruneThreshold {
        // Not enough to bother; return original
        return messages
    }

    return result
}
```

- [ ] **Step 6.2: Write tests**

```go
package utils

import (
    "testing"
    "github.com/sipeed/picoclaw/pkg/providers"
)

func TestPruneToolOutputs_Basic(t *testing.T) {
    msgs := []providers.Message{
        {Role: "user", Content: "hello"},
        {Role: "tool", Content: "very long output..."},
        {Role: "user", Content: "again"},
        {Role: "tool", Content: "another long output..."},
    }
    result := PruneToolOutputs(msgs, 2, 1)
    if result[1].Content == msgs[1].Content {
        t.Error("expected tool output to be pruned")
    }
}
```

- [ ] **Step 6.3: Verify tests**

Run: `go test ./pkg/utils/ -run TestPruneToolOutputs -count=1 -v`
Expected: PASS

## Task 7: Add Tool Output Truncation with Disk Saving

**Files:**
- Create: `pkg/utils/truncate.go`
- Create: `pkg/utils/truncate_test.go`

- [ ] **Step 7.1: Create `pkg/utils/truncate.go`**

```go
package utils

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// TruncatedResult holds the result of truncating a tool output.
type TruncatedResult struct {
    Preview string // Short preview for the LLM
    FullPath string // Path to the full output on disk
    WasTruncated bool
    Hint string // Instruction for the model
}

// TruncateToolOutput truncates tool output when it exceeds limits
// and saves full output to disk with a hint.
func TruncateToolOutput(output string, maxLines int, maxBytes int, toolName string) (TruncatedResult, error) {
    lines := strings.Split(output, "\n")
    if len(lines) <= maxLines && len(output) <= maxBytes {
        return TruncatedResult{Preview: output, WasTruncated: false}, nil
    }

    // Save full output to disk
    dir := os.TempDir()
    filename := fmt.Sprintf("picoclaw_tool_%s_%d.txt", toolName, os.Getpid())
    fullPath := filepath.Join(dir, filename)
    if err := os.WriteFile(fullPath, []byte(output), 0644); err != nil {
        return TruncatedResult{}, fmt.Errorf("save full output: %w", err)
    }

    // Build preview
    previewLines := lines
    if len(lines) > maxLines {
        previewLines = lines[:maxLines]
    }
    preview := strings.Join(previewLines, "\n")
    if len(preview) > maxBytes {
        preview = preview[:maxBytes]
    }

    return TruncatedResult{
        Preview: preview,
        FullPath: fullPath,
        WasTruncated: true,
        Hint: fmt.Sprintf("Tool output truncated. Full output saved to %s. Use read_file or grep to access."),
    }, nil
}
```

- [ ] **Step 7.2: Write tests**

```go
package utils

import (
    "strings"
    "testing"
)

func TestTruncateToolOutput_NotTruncated(t *testing.T) {
    output := "line1\nline2"
    result, err := TruncateToolOutput(output, 10, 1000, "test")
    if err != nil {
        t.Fatal(err)
    }
    if result.WasTruncated {
        t.Error("expected not truncated")
    }
}

func TestTruncateToolOutput_Truncated(t *testing.T) {
    output := strings.Repeat("line\n", 100)
    result, err := TruncateToolOutput(output, 5, 1000, "test")
    if err != nil {
        t.Fatal(err)
    }
    if !result.WasTruncated {
        t.Error("expected truncated")
    }
    if result.FullPath == "" {
        t.Error("expected full path set")
    }
}
```

- [ ] **Step 7.3: Verify tests**

Run: `go test ./pkg/utils/ -run TestTruncate -count=1 -v`
Expected: PASS

## Task 8: Final Integration and Verification

**Files:** All modified files.

- [ ] **Step 8.1: Run full test suite**

Run: `make test`
Expected: PASS

- [ ] **Step 8.2: Run linter**

Run: `make lint`
Expected: PASS

- [ ] **Step 8.3: Run build**

Run: `make build`
Expected: PASS

---

## Execution Options

**Plan complete and saved.** Two execution options:

1. **Subagent-Driven** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — Execute tasks in this session using `executing-plans`, with checkpoints.

**Which approach?**
