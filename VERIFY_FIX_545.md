# Fix Verification for Issue #545: Multiple Duplicate Messages

## Issue Summary
When a subagent completes asynchronously, the main agent was sending the same message 15+ times (matching `max_tool_iterations`), instead of once.

## Root Cause
The LLM iteration loop had no detection for when the same tool was called repeatedly with identical arguments, causing infinite repetition until hitting the iteration limit.

## Fix Applied
Added deduplication logic in `pkg/agent/loop.go` at lines 627-658 in the `runLLMIteration()` function.

## Code Trace - Bug Scenario (BEFORE FIX)

Using data from the issue logs, service health check:
```
max_tool_iterations: 15
Tool: message
Content: "Subagent-3 completed weather check..."
```

### Iteration Flow (Before Fix):
```
Iteration 1:
├─ LLM sees: [System Prompt, User: "health check"]
├─ LLM returns: ToolCall(message, content="Subagent-3 completed...")
├─ Message sent ✓
├─ Tool result added to messages
└─ Continue loop

Iteration 2:
├─ LLM sees: [System, User, Assistant(message call), ToolResult, ...]
├─ LLM returns: ToolCall(message, content="Subagent-3 completed...") ← SAME CONTENT
├─ Message sent (DUPLICATE #1) ✗
├─ Tool result added to messages
└─ Continue loop

Iterations 3-15:
├─ Same pattern repeats...
└─ RESULT: Message sent 15 times ✗✗✗
```

## Code Trace - Bug Scenario (AFTER FIX)

### Iteration Flow (After Fix):
```
Iteration 1:
├─ Check: iteration > 1? NO (iteration == 1)
├─ Skip dedup check
├─ LLM sees: [System Prompt, User: "health check"]
├─ LLM returns: ToolCall(message, content="Subagent-3 completed...")
├─ Message sent ✓
├─ Create: assistantMsg with ToolCalls
├─ messages = [..., assistantMsg, tool_result]
└─ Continue loop

Iteration 2:
├─ Check: iteration > 1? YES ✓
├─ Check: len(messages) >= 2? YES ✓
├─ Get: lastAssistantMsg = messages[len-2] = assistantMsg from iteration 1
├─ Get: lastTC = lastAssistantMsg.ToolCalls[0]
├─ Get: currentTC = LLM's new tool call
├─ Compare: lastTC.Name == currentTC.Name? "message" == "message" ✓
├─ Compare: json.Marshal(lastTC.Arguments) == json.Marshal(currentTC.Arguments)?
│           "Subagent-3..." == "Subagent-3..." ✓ MATCH!
├─ Set: finalContent = response.Content
├─ Break: Exit loop immediately ← FIX APPLIED!
└─ Return: finalContent, iteration=2, nil

RESULT: Message sent ONCE ✓
```

## Verification Checklist

✅ **Import check**: `encoding/json` imported at line 10
✅ **Structure check**: `Message.ToolCalls` field exists (type `[]ToolCall`)
✅ **Field check**: `ToolCall.Name` and `ToolCall.Arguments` fields exist
✅ **Logic check**: Dedup detection before tool execution (correct position)
✅ **Initialization check**: `finalContent` initialized as empty string at line 478
✅ **Return check**: `return finalContent, iteration, nil` at line 758
✅ **Break behavior**: Early return ensures code doesn't reach normal break
✅ **Fallback message**: Default message provided if `response.Content` is empty

## Edge Cases Handled

| Case | Before | After | Status |
|------|--------|-------|--------|
| First iteration | Skipped | Skipped (iteration > 1 check) | ✓ |
| Different tool names | Would loop | Dedup fails, normal flow | ✓ |
| Different arguments | Would loop | Dedup fails, normal flow | ✓ |
| Same tool + args | Repeated 15x | Breaks at iteration 2 | ✓ |
| Empty response | Normal flow | Uses default fallback | ✓ |

## Test Scenario from Issue #545

**Given:**
- Service health check task via subagent
- `max_tool_iterations: 15`
- LLM set to call `message()` tool repeatedly

**Expected (After Fix):**
- Iteration 1: Message tool called → executed
- Iteration 2: Duplicate detected → loop breaks
- User receives: 1 message ✓

**Old behavior (Before Fix):**
- User receives: 15 messages ✗

## Performance Impact

- **Minimal overhead**: One additional JSON marshaling per iteration (only when iteration > 1)
- **Early exit**: Saves 13+ LLM iterations, reducing API calls and latency
- **Memory**: No additional allocations for typical cases

## Logging

The fix adds a single info log when duplicate is detected:
```
[INFO] agent: Detected duplicate tool call, breaking iteration loop {agent_id=main, tool=message, iteration=2}
```

This helps operators understand why iterations stopped early.
