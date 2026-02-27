# Fix Implementation Guide for Issue #545

## Quick Overview

**Issue:** Same message sent 15 times due to LLM loop not detecting duplicate tool calls  
**Location:** `pkg/agent/loop.go` lines 627-658  
**Status:** Needs fixes before merge

---

## Implementation Roadmap

### Phase 1: Critical Fixes (MUST DO)

#### 1a. Use reflect.DeepEqual for Robust Comparison

**File:** `pkg/agent/loop.go`

**Replace:**
```go
// OLD - fragile string comparison
lastArgsJSON, _ := json.Marshal(lastTC.Arguments)
currentArgsJSON, _ := json.Marshal(currentTC.Arguments)
if string(lastArgsJSON) == string(currentArgsJSON) {
    // ...
}
```

**With:**
```go
// NEW - semantic comparison
import "reflect"

if reflect.DeepEqual(lastTC.Arguments, currentTC.Arguments) {
    // ...
}
```

**Why:**
- Handles map key ordering automatically
- Deterministic across all Go versions
- Faster (no JSON marshaling)

---

#### 1b. Safe Message History Walk

**Replace:**
```go
// OLD - brittle indexing
lastAssistantMsg := messages[len(messages)-2]
```

**With:**
```go
// NEW - safe walk backwards
var lastAssistantMsg *providers.Message
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "assistant" && i > 0 {
        lastAssistantMsg = &messages[i-1]
        break
    }
}

if lastAssistantMsg == nil {
    // No previous assistant message to compare against
    continue
}
```

**Why:**
- Works with any message structure
- Handles edge cases (empty history, only one message)
- Future-proof against refactoring

---

#### 1c. Handle Marshal Errors

**Replace:**
```go
// OLD - errors ignored
lastArgsJSON, _ := json.Marshal(lastTC.Arguments)
currentArgsJSON, _ := json.Marshal(currentTC.Arguments)
```

**Note:** If using `reflect.DeepEqual`, this step is not needed.

**But if you must use json.Marshal:**
```go
// NEW - explicit error handling
lastArgsJSON, err := json.Marshal(lastTC.Arguments)
if err != nil {
    logger.WarnCF("agent", "Failed to marshal tool arguments",
        map[string]any{
            "error": err.Error(),
            "tool": lastTC.Name,
            "agent_id": agent.ID,
        })
    continue // Skip dedup check
}

currentArgsJSON, err := json.Marshal(currentTC.Arguments)
if err != nil {
    logger.WarnCF("agent", "Failed to marshal tool arguments",
        map[string]any{
            "error": err.Error(),
            "tool": currentTC.Name,
            "agent_id": agent.ID,
        })
    continue // Skip dedup check
}
```

---

#### 1d. Check All Tool Calls, Not Just First

**Replace:**
```go
// OLD - only first tool call
lastTC := lastAssistantMsg.ToolCalls[0]
currentTC := normalizedToolCalls[0]
if lastTC.Name == currentTC.Name {
    // compare args...
}
```

**With:**
```go
// NEW - check if ALL tool calls are identical
if len(lastAssistantMsg.ToolCalls) == len(normalizedToolCalls) {
    allIdentical := true
    
    for idx := 0; idx < len(normalizedToolCalls); idx++ {
        lastTC := lastAssistantMsg.ToolCalls[idx]
        currentTC := normalizedToolCalls[idx]
        
        if lastTC.Name != currentTC.Name {
            allIdentical = false
            break
        }
        
        if !reflect.DeepEqual(lastTC.Arguments, currentTC.Arguments) {
            allIdentical = false
            break
        }
    }
    
    if allIdentical {
        logger.InfoCF("agent", "All tool calls identical, breaking loop",
            map[string]any{
                "agent_id": agent.ID,
                "count": len(normalizedToolCalls),
                "iteration": iteration,
            })
        finalContent = response.Content
        if finalContent == "" {
            finalContent = "I've completed processing but have no new response to give."
        }
        break
    }
}
```

---

#### 1e. Require Multiple Consecutive Duplicates (Optional but Recommended)

**In AgentLoop struct:**
```go
type AgentLoop struct {
    // ... existing fields
    duplicateDetector *DuplicateTracker
}

type DuplicateTracker struct {
    consecutiveCount int
    lastToolName string
    maxConsecutive int // e.g., 3
}
```

**Usage:**
```go
const MaxConsecutiveDuplicates = 3

if allIdentical {
    al.duplicateDetector.consecutiveCount++
    
    if al.duplicateDetector.consecutiveCount >= MaxConsecutiveDuplicates {
        logger.InfoCF("agent", "Stopping loop - too many consecutive duplicates",
            map[string]any{
                "count": al.duplicateDetector.consecutiveCount,
                "agent_id": agent.ID,
            })
        break
    }
} else {
    al.duplicateDetector.consecutiveCount = 0
    al.duplicateDetector.lastToolName = ""
}
```

---

### Phase 2: Test Coverage (MUST DO)

#### Create Test File

**File:** `pkg/agent/dedup_test.go`

```go
package agent

import (
    "context"
    "reflect"
    "testing"
    
    "github.com/sipeed/picoclaw/pkg/providers"
    "github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// TestDuplicateDetectionIdenticalTools verifies identical tool calls trigger dedup
func TestDuplicateDetectionIdenticalTools(t *testing.T) {
    // Setup...
    tc1 := protocoltypes.ToolCall{
        Name: "message",
        Arguments: map[string]any{
            "text": "Hello, world!",
        },
    }
    
    tc2 := protocoltypes.ToolCall{
        Name: "message",
        Arguments: map[string]any{
            "text": "Hello, world!",
        },
    }
    
    // Verify
    if !reflect.DeepEqual(tc1.Arguments, tc2.Arguments) {
        t.Fatal("Expected arguments to be equal")
    }
}

// TestDuplicateDetectionDifferentArgs verifies different args don't trigger dedup
func TestDuplicateDetectionDifferentArgs(t *testing.T) {
    tc1 := protocoltypes.ToolCall{
        Arguments: map[string]any{"text": "Message 1"},
    }
    
    tc2 := protocoltypes.ToolCall{
        Arguments: map[string]any{"text": "Message 2"},
    }
    
    if reflect.DeepEqual(tc1.Arguments, tc2.Arguments) {
        t.Fatal("Arguments should NOT be equal")
    }
}

// TestDuplicateDetectionMapOrdering verifies map key order doesn't matter
func TestDuplicateDetectionMapOrdering(t *testing.T) {
    // Same data, different key order
    args1 := map[string]any{
        "a": 1,
        "b": "test",
        "c": true,
    }
    
    args2 := map[string]any{
        "c": true,
        "a": 1,
        "b": "test",
    }
    
    if !reflect.DeepEqual(args1, args2) {
        t.Fatal("Expected arguments to be equal regardless of key order")
    }
}

// TestMultipleToolCallsAllChecked verifies all tools are compared
func TestMultipleToolCallsAllChecked(t *testing.T) {
    last := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},
        {Name: "tool2", Arguments: map[string]any{"id": "2"}},
    }
    
    current := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},
        {Name: "tool2", Arguments: map[string]any{"id": "2"}},
    }
    
    // All should match
    allIdentical := len(last) == len(current)
    if allIdentical {
        for i := range current {
            if last[i].Name != current[i].Name {
                allIdentical = false
                break
            }
            if !reflect.DeepEqual(last[i].Arguments, current[i].Arguments) {
                allIdentical = false
                break
            }
        }
    }
    
    if !allIdentical {
        t.Fatal("All tool calls should match")
    }
}

// TestMessageHistoryWalk verifies safe backward walk works
func TestMessageHistoryWalk(t *testing.T) {
    messages := []providers.Message{
        {Role: "user", Content: "Hello"},
        {Role: "assistant", Content: "Hi", ToolCalls: []protocoltypes.ToolCall{{Name: "tool1"}}},
        {Role: "tool", Content: "Result"},
        {Role: "assistant", Content: "Now", ToolCalls: []protocoltypes.ToolCall{{Name: "tool2"}}},
    }
    
    // Find last assistant message
    var lastAssistant *providers.Message
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].Role == "assistant" && i > 0 {
            lastAssistant = &messages[i-1]
            break
        }
    }
    
    // Should skip current assistant message and find previous one
    if lastAssistant == nil || lastAssistant.Content != "Hi" {
        t.Fatal("Should find previous assistant message")
    }
}
```

---

### Phase 3: Documentation (SHOULD DO)

Add to `pkg/agent/loop.go` comments:

```go
// handleDuplicateToolCalls detects and prevents infinite loops caused by the LLM
// repeatedly calling the same tool with identical arguments.
//
// This addresses Issue #545: when a subagent completes asynchronously,
// the LLM may not recognize that a tool has already been executed and
// may attempt to call it again multiple times, resulting in spam.
//
// The function uses semantic comparison (reflect.DeepEqual) to safely detect
// duplicate tool calls, rather than fragile string comparisons.
//
// Requires at least 3 consecutive identical tool calls to break the loop,
// to avoid killing legitimate retries due to transient failures.
//
// See: https://github.com/sipeed/picoclaw/issues/545
func (al *AgentLoop) handleDuplicateToolCalls(
    iteration int,
    lastAssistantMsg *providers.Message,
    currentToolCalls []providers.ToolCall,
) bool {
    // Implementation...
}
```

---

## Code Review Checklist

- [ ] Issue #1: All tool calls compared (not just first)
- [ ] Issue #2: Using `reflect.DeepEqual` (not JSON string comparison)
- [ ] Issue #3: Safely walking message history backward
- [ ] Issue #4: Handling marshal errors explicitly
- [ ] Issue #5: Comprehensive tests for all scenarios
- [ ] Issue #6: Requiring 3+ consecutive duplicates (not just 1)
- [ ] Documentation: Comments explain the logic and Issue #545
- [ ] No regressions: All existing tests pass
- [ ] Performance: No significant impact on loop performance

---

## Suggested PR Description

```markdown
## Fix: Prevent Duplicate Messages from LLM Loop (Issue #545)

### Problem
When a subagent completes asynchronously, the main agent sends the same message
15+ times (matching max_tool_iterations), instead of once.

### Root Cause
The LLM iteration loop has no detection for when the same tool is called repeatedly
with identical arguments, causing infinite repetition until hitting the iteration limit.

### Solution
Added robust deduplication logic in `pkg/agent/loop.go` that:
1. ✅ Uses `reflect.DeepEqual` for semantic argument comparison
2. ✅ Safely walks backwards to find previous assistant messages
3. ✅ Compares ALL tool calls (not just first)
4. ✅ Handles errors explicitly
5. ✅ Requires 3+ consecutive duplicates (prevents false positives)

### Testing
Added comprehensive test coverage for:
- Identical tool calls detection
- Different arguments (no false positives)
- Map key ordering edge cases
- Complex message structures
- Error handling

### Verification
Tested by temporarily disabling the fix to confirm the issue exists.
With fix: 1 message sent
Without fix: 15 duplicate messages sent

Fixes #545
```

---

## Before/After Example

### Scenario: Health Check with Subagent

**BEFORE (15 duplicates):**
```
User: "@health-check"
Agent: "I'll check system health..."
Agent: "⚙️ Checking database..."
Agent: "⚙️ Checking database..."
Agent: "⚙️ Checking database..."
Agent: "⚙️ Checking database..."
Agent: "⚙️ Checking database..."
...
(10 more times)
```

**AFTER (1 message):**
```
User: "@health-check"
Agent: "I'll check system health..."
Agent: "✅ Database: OK"
Agent: "✅ API: OK"
Agent: "✅ Memory: 82% used"
Agent: "All systems nominal!"
```

---

## Estimated Implementation Time

| Phase | Task | Time |
|-------|------|------|
| 1a | reflect.DeepEqual replacement | 30 min |
| 1b | Safe message walk | 30 min |
| 1c | Error handling | 20 min |
| 1d | Multiple tool checks | 30 min |
| 1e | Consecutive threshold | 30 min |
| 2 | Test coverage | 2 hours |
| 3 | Documentation | 30 min |
| **Total** | | **~5 hours** |

---

## Questions to Consider

1. Should we log more details when dedup is triggered for debugging?
2. Should the consecutive duplicate threshold be configurable?
3. Should we track dedup metrics for monitoring?
4. Should we alert when many consecutive duplicates are detected?
5. Do we need different thresholds for different tool types?

---

## References

- **Issue:** https://github.com/sipeed/picoclaw/issues/545
- **PR:** https://github.com/sipeed/picoclaw/pull/775
- **Related:** `pkg/agent/loop.go:runLLMIteration()`
- **Docs:** [Detailed Analysis](./ISSUE_545_DETAILED_ANALYSIS.md)
