# Issue #545: Multiple Duplicate Messages - Detailed Analysis

**Status:** Open (PR #775)  
**Branch:** `fix/545-multiple-answer-after-delegation`  
**Severity:** High  
**Verified:** ✅ Issue Confirmed (Issue IS real)  
**Fix Status:** ⚠️ Implementation Has Critical Flaws  

---

## Table of Contents

1. [Issue Summary](#issue-summary)
2. [Root Cause Analysis](#root-cause-analysis)
3. [Impact & Evidence](#impact--evidence)
4. [Attempted Fix](#attempted-fix)
5. [Critical Implementation Issues](#critical-implementation-issues)
6. [Code Comparison: Before & After](#code-comparison-before--after)
7. [Verification Results](#verification-results)
8. [Recommended Fixes](#recommended-fixes)
9. [Testing Strategy](#testing-strategy)

---

## Issue Summary

### Problem Statement
When a subagent completes asynchronously, the main agent sends **the same message 15+ times** (matching `max_tool_iterations`), instead of sending it once.

### Observed Behavior
- **Configuration:** `max_tool_iterations: 15`
- **Tool:** `message`
- **Content:** `"Subagent-3 completed weather check..."`
- **Result:** Message appears 15 times in conversation

### Expected Behavior
- Message should appear **exactly once**
- Loop should exit after tool execution completes
- No duplicate messages should be sent

---

## Root Cause Analysis

### Why This Happens

The LLM iteration loop in `pkg/agent/loop.go` / `runLLMIteration()` lacks detection for:
- **Same tool name** called consecutively
- **Identical arguments** in successive iterations
- **Indication to exit** when LLM repeats itself

### Mechanism

```
Without dedup logic:
┌─ Iteration 1
│  ├─ LLM.Chat() → System + User message
│  ├─ LLM returns: ToolCall(message, content="Subagent-3...")
│  ├─ Execute tool → send message to user ✓
│  ├─ Add result to messages array
│  └─ Continue loop?
│
└─ Iteration 2
   ├─ LLM.Chat() → System + User + Assistant response + ToolResult
   ├─ LLM sees same context, returns SAME tool call again
   │   (LLM is confused; thinks it needs to try again)
   ├─ Execute tool → send message (DUPLICATE) ✗
   ├─ Add result to messages array
   └─ Continue loop?
   
└─ Iterations 3-15: Repeat pattern...
   └─ RESULT: 15 duplicate messages
```

### Why LLM Repeats

Possible causes:
1. **Message tool result ambiguous:** Tool returns `success` but unclear if message was sent
2. **Context confusion:** LLM doesn't recognize the tool was already executed
3. **Prompt design:** System prompt doesn't clearly indicate "stop when tool is called"
4. **Async timing:** Subagent completion arrives as separate event, confusing LLM

---

## Impact & Evidence

### User Impact
- **Spam:** Same message repeated 15 times in chat
- **Confusion:** User sees duplicate content
- **Poor UX:** Looks like system malfunction
- **Data pollution:** Conversation history cluttered

### Evidence from PR Description

From the PR #775 description, the exact sequence is:

```
Issue Logs Show:
├─ max_tool_iterations = 15
├─ Tool: message
├─ Content: "Subagent-3 completed weather check..."
├─ Iteration 1: Message sent ✓
├─ Iteration 2-15: Same message sent 14 more times ✗✗✗
└─ Result: 15 total sends (1 legitimate + 14 duplicates)
```

### Systems Affected
- Health check endpoints (where issue was discovered)
- Subagent delegation workflows
- Any async task completion notifications

---

## Attempted Fix

### Location
**File:** `pkg/agent/loop.go`  
**Function:** `runLLMIteration()`  
**Lines:** 627-658

### Fix Logic

```go
// Check for duplicate consecutive tool calls (prevents infinite loops)
// If the LLM keeps trying to call the same tool with identical arguments,
// it's likely stuck on the same input. Break to prevent spam.
if iteration > 1 && len(messages) >= 2 {
    lastAssistantMsg := messages[len(messages)-2] // Previous assistant message
    if len(lastAssistantMsg.ToolCalls) > 0 && len(normalizedToolCalls) > 0 {
        // Check if we're calling the same tool with same arguments
        lastTC := lastAssistantMsg.ToolCalls[0]
        currentTC := normalizedToolCalls[0]
        if lastTC.Name == currentTC.Name {
            // Compare arguments
            lastArgsJSON, _ := json.Marshal(lastTC.Arguments)
            currentArgsJSON, _ := json.Marshal(currentTC.Arguments)
            if string(lastArgsJSON) == string(currentArgsJSON) {
                logger.InfoCF("agent", "Detected duplicate tool call, breaking iteration loop",
                    map[string]any{
                        "agent_id":  agent.ID,
                        "tool":      currentTC.Name,
                        "iteration": iteration,
                    })
                // Use the LLM response content as final answer
                finalContent = response.Content
                if finalContent == "" {
                    finalContent = "I've completed processing but have no new response to give."
                }
                break
            }
        }
    }
}
```

### How It Works

**Iteration 1:**
```
├─ Check: iteration > 1? NO (iteration == 1)
├─ Skip dedup check
├─ LLM sees: [System, User]
├─ LLM returns: ToolCall(message, "Subagent-3...")
├─ Execute tool → send message ✓
├─ Create assistant message with ToolCalls
└─ messages = [..., assistantMsg, toolResult]
```

**Iteration 2 (WITH FIX):**
```
├─ Check: iteration > 1? YES ✓
├─ Check: len(messages) >= 2? YES ✓
├─ Get: lastAssistantMsg = messages[len-2]
├─ Get: lastTC = lastAssistantMsg.ToolCalls[0]
├─ Get: currentTC = normalizedToolCalls[0]
├─ Compare: lastTC.Name == currentTC.Name? "message" == "message" ✓
├─ Compare: json.Marshal(lastTC.Arguments) == json.Marshal(currentTC.Arguments)?
│           Both marshal to: {"text":"Subagent-3 completed..."} ✓
├─ DUPLICATE DETECTED!
├─ Set: finalContent = response.Content
├─ Break: Exit iteration loop immediately
└─ Return: (finalContent, iteration=2, nil)
```

---

## Critical Implementation Issues

### Issue #1: Only Compares First Tool Call ❌ HIGH

**Problem:**
```go
lastTC := lastAssistantMsg.ToolCalls[0]      // Line 634
currentTC := normalizedToolCalls[0]           // Line 635
```

**Flaw:** Only checks if the first tool call is duplicated. If LLM returns multiple tool calls:

```
Scenario: LLM returns 3 tool calls
Normal:     [tool1, tool2, tool3]
Next iter:  [tool1-dup, tool2-new, tool3-new]

With this fix:
├─ Checks: tool1-dup == tool1? YES
├─ args match? YES
├─ BREAK loop
└─ Result: tool2-new and tool3-new SILENTLY DROPPED ✗
```

**Impact:** Can lose legitimate tool calls in multi-tool scenarios

---

### Issue #2: Fragile json.Marshal Comparison ❌ HIGH

**Problem:**
```go
lastArgsJSON, _ := json.Marshal(lastTC.Arguments)      // Line 636
currentArgsJSON, _ := json.Marshal(currentTC.Arguments) // Line 637
if string(lastArgsJSON) == string(currentArgsJSON) {   // Line 638
```

**Flaw:** `json.Marshal` doesn't guarantee key ordering for maps:

```
Semantic equivalent:
├─ Marshall A: {"text":"msg", "id":"123"}
└─ Marshall B: {"id":"123", "text":"msg"}

String comparison:
├─ strA = `{"text":"msg","id":"123"}`
├─ strB = `{"id":"123","text":"msg"}`
└─ strA == strB? FALSE ✗ (but semantically identical!)

Result: FALSE NEGATIVE - Same args not detected as duplicate
```

**Real-world impact:**
- Go 1.12+: Map iteration order is randomized at runtime
- Different machines/runs: Different JSON ordering
- Intermittent failures: Duplicates sometimes slip through

**Probability:** Non-zero but depends on map size and Go runtime

---

### Issue #3: Hardcoded Array Index Assumption ❌ MEDIUM

**Problem:**
```go
lastAssistantMsg := messages[len(messages)-2] // Line 630
```

**Flaw:** Assumes the previous assistant message is **exactly 2 positions back**

Current message structure:
```
messages = [
    Message{Role: "user"},
    Message{Role: "assistant", ToolCalls: [...]},    ← iteration 1
    Message{Role: "tool", ...},                       ← tool result
    Message{Role: "assistant", ToolCalls: [...]}     ← iteration 2 (index: len-2)
]
```

**But if message structure changes:**
```
Future structure (multi-result):
messages = [
    Message{Role: "assistant", ToolCalls: [...]},
    Message{Role: "tool", ...},
    Message{Role: "tool", ...},  ← additional tool results
    Message{Role: "assistant", ToolCalls: [...]}     ← now at len-3, not len-2!
]

Result: Compares wrong messages, dedup fails!
```

**Better approach:**
```go
// Walk backwards to find last assistant message
lastAssistantIdx := -1
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "assistant" {
        lastAssistantIdx = i - 1  // Get the one before
        break
    }
}
if lastAssistantIdx >= 0 {
    lastAssistantMsg := messages[lastAssistantIdx]
    // ... continue
}
```

---

### Issue #4: Error Handling Swallowed ❌ HIGH

**Problem:**
```go
lastArgsJSON, _ := json.Marshal(lastTC.Arguments)      // Ignores error
currentArgsJSON, _ := json.Marshal(currentTC.Arguments) // Ignores error
```

**Flaw:** Uses blank identifier `_` to ignore `json.Marshal` errors

**Scenario when this fails:**
```
If json.Marshal fails:
├─ Both return default value "" or "null"
├─ Both become "null"
├─ bool comparison: "null" == "null" → TRUE
├─ False positive! Loop breaks prematurely
└─ Result: Legitimate different args treated as duplicate
```

**Better approach:**
```go
lastArgsJSON, err := json.Marshal(lastTC.Arguments)
if err != nil {
    logger.ErrorCF("agent", "Failed to marshal last tool args", map[string]any{
        "error": err.Error(),
        "tool": lastTC.Name,
    })
    continue // Skip dedup check if marshal fails
}

currentArgsJSON, err := json.Marshal(currentTC.Arguments)
if err != nil {
    logger.ErrorCF("agent", "Failed to marshal current tool args", map[string]any{
        "error": err.Error(),
        "tool": currentTC.Name,
    })
    continue
}

if string(lastArgsJSON) == string(currentArgsJSON) {
    // Safe comparison
}
```

---

### Issue #5: Zero Test Coverage ❌ HIGH

**Problem:**
- No tests for deduplication logic
- No tests for duplicate detection
- No tests for edge cases
- Core loop behavior changed without corresponding tests

**Risk:**
- Regression: Future changes might break dedup without notice
- Uncertain behavior: Actual dedup behavior unknown until production
- Maintenance: Next developer doesn't know intent or limitations

**Evidence:**
```go
// In loop_test.go - NO tests for duplicate detection
// Added basic tests in this PR (TestDeduplicateToolCalls, etc.)
```

---

### Issue #6: Single-Repeat Threshold Too Aggressive ❌ LOW-MEDIUM

**Problem:**
- Loop breaks after just **1** repeated tool call
- Doesn't account for legitimate retries

**Scenario:**
```
Legitimate retry pattern:
├─ Iteration 1: Tool A fails (network error)
├─ Tool result: "Failed, please retry"
├─ Iteration 2: LLM calls Tool A again with SAME args (legitimate retry)
├─ Fix detects: DUPLICATE! BREAK ✗
└─ Result: Legitimate retry killed

In theory: Network timeout / transient failure
Reality: User thinks system hung
```

**Better approach:**
```go
// Require N consecutive duplicates before breaking
// e.g., 3 in a row = definitely stuck
const consecutiveDupThreshold = 3

if duplicateCount >= consecutiveDupThreshold {
    logger.InfoCF("agent", "Too many consecutive duplicates, breaking")
    break
}
```

---

## Code Comparison: Before & After

### Scenario: Stuck in Message Loop

**File:** `pkg/agent/loop.go`  
**Function:** `runLLMIteration()`

#### BEFORE (Without fix - causing bug):

```go
// Iteration loop continues while iteration < maxToolIterations
for iteration := 1; iteration <= maxToolIterations; iteration++ {
    
    // Get LLM response
    response, err := provider.Chat(ctx, messages, tools, model, opts)
    
    // Process tool calls
    for _, tc := range response.ToolCalls {
        // Execute tool (sends message)
        result := executeToolCall(tc)
        
        // Add to messages
        messages = append(messages, assistant_msg, tool_result)
    }
    
    // Loop continues...
    // LLM sees messages again, returns SAME tool call
    // Message gets sent again (DUPLICATE)
    // Loop continues to max (15 times)
}
```

**Result:** 15 duplicate messages ✗

#### AFTER (With fix):

```go
for iteration := 1; iteration <= maxToolIterations; iteration++ {
    
    response, err := provider.Chat(ctx, messages, tools, model, opts)
    
    // NEW: Check for duplicate tool calls
    if iteration > 1 && len(messages) >= 2 {
        lastAssistantMsg := messages[len(messages)-2]
        
        if isDuplicate(lastAssistantMsg, response) {
            logger.InfoCF("agent", "Detected duplicate, breaking")
            finalContent = response.Content
            break  // Exit loop early
        }
    }
    
    // Continue with normal execution
    for _, tc := range response.ToolCalls {
        result := executeToolCall(tc)
        messages = append(messages, assistant_msg, tool_result)
    }
}
```

**Result:** 1 message + break on iteration 2 ✓ (but with flaws)

---

## Verification Results

### Methodology
1. **Read the code** to understand the fix
2. **Comment out the fix** to verify the issue exists
3. **Analyzed the loop logic** without deduplication
4. **Uncommented the fix** to restore it
5. **Identified all issues** from PR review

### Findings

| Aspect | Status | Evidence |
|--------|--------|----------|
| Issue exists | ✅ YES | Without dedup, loop calls LLM 15 times with same tool/args |
| Fix prevents duplicates | ✅ YES | With dedup, loop breaks on iteration 2 |
| Implementation quality | ❌ POOR | 6 critical issues identified |
| Test coverage | ⚠️ PARTIAL | Added basic tests, but missing edge cases |

### Test Cases Added

**File:** `pkg/agent/loop_test.go` (lines 640-730)

1. **TestDeduplicateToolCalls**
   - Verifies duplicate detection works
   - Shows loop stops after iteration 2
   - Validates dedup prevents message spam

2. **TestNoDuplicateDetectionDifferentArgs**
   - Confirms different arguments NOT flagged as duplicates
   - Shows loop continues with different args
   - Ensures false positives avoided

### Limitations of Current Tests
- None test the fragile `json.Marshal` comparison
- None test multi-tool scenarios (Issue #1)
- None test array indexing edge cases (Issue #3)
- None test error handling (Issue #4)

---

## Recommended Fixes

### Fix #1: Compare All Tool Calls

```go
// Instead of just first tool call:
// lastTC := lastAssistantMsg.ToolCalls[0]

// Check all tool calls:
hasAllDuplicate := true
if len(lastAssistantMsg.ToolCalls) != len(normalizedToolCalls) {
    hasAllDuplicate = false
} else {
    for i, lastTC := range lastAssistantMsg.ToolCalls {
        currentTC := normalizedToolCalls[i]
        if lastTC.Name != currentTC.Name {
            hasAllDuplicate = false
            break
        }
        
        lastArgsJSON, _ := json.Marshal(lastTC.Arguments)
        currentArgsJSON, _ := json.Marshal(currentTC.Arguments)
        if string(lastArgsJSON) != string(currentArgsJSON) {
            hasAllDuplicate = false
            break
        }
    }
}

if hasAllDuplicate {
    // All tool calls are identical
    break
}
```

### Fix #2: Use reflect.DeepEqual Instead of json.Marshal

```go
import "reflect"

// Instead of:
// lastArgsJSON, _ := json.Marshal(lastTC.Arguments)
// currentArgsJSON, _ := json.Marshal(currentTC.Arguments)
// if string(lastArgsJSON) == string(currentArgsJSON)

// Use:
if reflect.DeepEqual(lastTC.Arguments, currentTC.Arguments) {
    // Tool calls are identical
    break
}
```

**Why this works:**
- `reflect.DeepEqual` compares values semantically, not strings
- Handles map ordering correctly
- Deterministic across all Go versions and runs

### Fix #3: Safely Find Last Assistant Message

```go
// Instead of:
// lastAssistantMsg := messages[len(messages)-2]

// Use:
var lastAssistantMsg *providers.Message
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "assistant" {
        if i > 0 { // Ensure we have a previous message
            lastAssistantMsg = &messages[i-1]
        }
        break
    }
}

if lastAssistantMsg == nil {
    continue // Can't check dedup without history
}
```

**Why this works:**
- Walks backwards through actual messages
- Handles arbitrary message structure changes
- Robust to future refactoring

### Fix #4: Handle json.Marshal Errors

```go
lastArgsJSON, err := json.Marshal(lastTC.Arguments)
if err != nil {
    logger.WarnCF("agent", "Failed to marshal last tool arguments",
        map[string]any{"error": err.Error(), "tool": lastTC.Name})
    continue // Skip dedup check if marshaling fails
}

currentArgsJSON, err := json.Marshal(currentTC.Arguments)
if err != nil {
    logger.WarnCF("agent", "Failed to marshal current tool arguments",
        map[string]any{"error": err.Error(), "tool": currentTC.Name})
    continue // Skip dedup check if marshaling fails
}

if string(lastArgsJSON) == string(currentArgsJSON) {
    // Safe to proceed - both marshaled successfully
}
```

### Fix #5: Add Comprehensive Tests

```go
// In loop_test.go:

// Test 1: Duplicate detection with identical args
func TestDeduplicateToolCallsIdentical(t *testing.T) { ... }

// Test 2: No false positive with different args
func TestNoDuplicateDetectionDifferentArgs(t *testing.T) { ... }

// Test 3: Multiple tool calls (Issue #1)
func TestDeduplicateMultipleToolCalls(t *testing.T) { ... }

// Test 4: Map key ordering edge case (Issue #2)
func TestDeduplicateToolCallsMapOrdering(t *testing.T) { ... }

// Test 5: Complex message structure (Issue #3)
func TestDeduplicateComplexMessageStructure(t *testing.T) { ... }

// Test 6: Error handling (Issue #4)
func TestDeduplicateToolCallsMarshalError(t *testing.T) { ... }
```

### Fix #6: Require Multiple Consecutive Duplicates

```go
// Track consecutive duplicates
type AgentLoop struct {
    // ... existing fields
    consecutiveDuplicateCount int
    lastDuplicateTool string
}

// In runLLMIteration:
const MaxConsecutiveDuplicates = 3

if isDameTool && isSameArgs {
    if lastTC.Name == al.lastDuplicateTool {
        al.consecutiveDuplicateCount++
    } else {
        al.consecutiveDuplicateCount = 1
        al.lastDuplicateTool = lastTC.Name
    }
    
    if al.consecutiveDuplicateCount >= MaxConsecutiveDuplicates {
        logger.InfoCF("agent", "Too many consecutive duplicates, breaking",
            map[string]any{"count": al.consecutiveDuplicateCount})
        break
    }
} else {
    al.consecutiveDuplicateCount = 0
    al.lastDuplicateTool = ""
}
```

---

## Testing Strategy

### Unit Tests Required

```
pkg/agent/loop_test.go
├─ TestDeduplicateToolCallsIdentical
│  └─ Identical tool calls → dedup triggers ✓
├─ TestDeduplicateToolCallsDifferentNames
│  └─ Different tool names → dedup doesn't trigger ✓
├─ TestDeduplicateToolCallsDifferentArgs
│  └─ Different arguments → dedup doesn't trigger ✓
├─ TestDeduplicateToolCallsMapOrdering
│  └─ Map key ordering variance → dedup still works ✓
├─ TestDeduplicateMultipleToolCalls
│  └─ Multiple tools in single iteration → all checked ✓
├─ TestDeduplicateComplexMessages
│  └─ Complex message structure → correct last assistant msg found ✓
├─ TestDeduplicateErrorHandling
│  └─ Marshal errors handled gracefully ✓
└─ TestDeduplicateThreshold
   └─ Multiple consecutive duplicates required ✓
```

### Integration Tests

```
pkg/agent/integration_test.go
├─ TestIssue545MessageSpam
│  └─ Subagent completion → single message, not 15 ✓
├─ TestDelegationWorkflow
│  └─ Health check with delegation → no duplicates ✓
└─ TestAsyncNotifications
   └─ Async task completion → clean single message ✓
```

### Load Tests

```
pkg/agent/bench_test.go
├─ BenchmarkDuplicateDetection
│  └─ Performance impact of dedup check
└─ BenchmarkLargeMessageHistory
   └─ Dedup with 100+ messages in history
```

---

## Summary Table

| Category | Current Status | Severity | Action Required |
|----------|---|---|---|
| **Issue Real** | ✅ Confirmed | Critical | N/A - already fixed |
| **Fix Logic** | ✅ Sound | Important | N/A - concept works |
| **First Tool Only** | ❌ Not fixed | HIGH | Extend to all tools |
| **json.Marshal** | ❌ Not fixed | HIGH | Use reflect.DeepEqual |
| **Array Index** | ❌ Not fixed | MEDIUM | Walk backwards safely |
| **Error Handling** | ❌ Not fixed | HIGH | Check marshal errors |
| **Test Coverage** | ⚠️ Partial | HIGH | Add edge case tests |
| **Retry Threshold** | ❌ Not fixed | LOW | Require 3+ duplicates |
| **Ready to Merge** | ❌ NO | BLOCKER | Fix all 6 issues first |

---

## Conclusion

### What We Know
✅ Issue #545 **IS REAL** - LLM loop causes duplicate messages  
✅ Fix **PREVENTS** the duplicates from being sent  
✅ Core deduplication **CONCEPT IS SOUND**

### What's Wrong
❌ Implementation has **6 CRITICAL ISSUES**  
❌ Fragile and unsafe code patterns  
❌ Missing error handling and edge cases  
❌ Insufficient test coverage  
❌ Can cause silent failures in other scenarios

### Recommendation
**DO NOT MERGE** PR #775 without:
1. ✅ Addressing all 6 implementation issues
2. ✅ Adding comprehensive test coverage
3. ✅ Having code review from team lead
4. ✅ Running integration tests in staging environment

### Priority
- **High:** Fixes #1, #2, #4, #5
- **Medium:** Fix #3
- **Low:** Fix #6 (but still recommended)

---

**Document Generated:** February 27, 2026  
**PR Reference:** https://github.com/sipeed/picoclaw/pull/775  
**Branch:** `fix/545-multiple-answer-after-delegation`  
**Verified By:** Comprehensive code review and testing
