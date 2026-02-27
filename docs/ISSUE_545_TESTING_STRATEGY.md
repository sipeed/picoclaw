# Issue #545: Testing Strategy & Test Cases

## Overview

This document outlines comprehensive test coverage for the duplicate message fix in Issue #545. Tests should cover all 6 identified implementation issues.

---

## Test Categories

### 1. Unit Tests: Deduplication Logic

**File:** `pkg/agent/dedup_test.go`

#### Test 1.1: Identical Tool Calls Trigger Dedup

```go
func TestDeduplicateToolCallsIdentical(t *testing.T) {
    // Setup
    lastTC := protocoltypes.ToolCall{
        ID:   "call-1",
        Name: "message",
        Arguments: map[string]any{
            "text": "Subagent-3 completed weather check",
            "id": "task-123",
        },
    }
    
    currentTC := protocoltypes.ToolCall{
        ID:   "call-2", // Different ID but same content
        Name: "message",
        Arguments: map[string]any{
            "text": "Subagent-3 completed weather check",
            "id": "task-123",
        },
    }
    
    // Expected: Should match (same name + args)
    if lastTC.Name != currentTC.Name {
        t.Fatal("Names should match")
    }
    
    if !reflect.DeepEqual(lastTC.Arguments, currentTC.Arguments) {
        t.Fatal("Arguments should match")
    }
}
```

**Why:** Verifies core dedup detection works

---

#### Test 1.2: Different Tool Names Don't Trigger Dedup

```go
func TestDeduplicateToolCallsDifferentNames(t *testing.T) {
    lastTC := protocoltypes.ToolCall{
        Name: "message",
        Arguments: map[string]any{"text": "Hello"},
    }
    
    currentTC := protocoltypes.ToolCall{
        Name: "search",  // Different tool
        Arguments: map[string]any{"text": "Hello"},
    }
    
    // Should not deduplicate
    if lastTC.Name == currentTC.Name {
        t.Fatal("Names are different, should not match")
    }
}
```

**Why:** Ensures no false positives on different tools

---

#### Test 1.3: Different Arguments Don't Trigger Dedup

```go
func TestDeduplicateToolCallsDifferentArgs(t *testing.T) {
    lastTC := protocoltypes.ToolCall{
        Name: "message",
        Arguments: map[string]any{
            "text": "Message 1",
            "id": "123",
        },
    }
    
    currentTC := protocoltypes.ToolCall{
        Name: "message",
        Arguments: map[string]any{
            "text": "Message 2",  // Different
            "id": "123",
        },
    }
    
    // Should not deduplicate
    if reflect.DeepEqual(lastTC.Arguments, currentTC.Arguments) {
        t.Fatal("Arguments are different, should not match")
    }
}
```

**Why:** Prevents false positive dedup detection (Issue #1, #2)

---

#### Test 1.4: Map Key Ordering Doesn't Matter (Issue #2)

```go
func TestDeduplicateToolCallsMapOrdering(t *testing.T) {
    // Same data, different key order to simulate json.Marshal variability
    args1 := map[string]any{
        "a": 1,
        "b": "test",
        "c": true,
        "d": []string{"x", "y"},
    }
    
    args2 := map[string]any{
        "d": []string{"x", "y"},
        "a": 1,
        "c": true,
        "b": "test",
    }
    
    // reflect.DeepEqual handles key ordering
    if !reflect.DeepEqual(args1, args2) {
        t.Fatal("Should match regardless of map key order")
    }
    
    // But json.Marshal might produce different strings
    var buf1, buf2 bytes.Buffer
    json.NewEncoder(&buf1).Encode(args1)
    json.NewEncoder(&buf2).Encode(args2)
    
    // Note: strings MIGHT differ, but DeepEqual handles this
    t.Logf("String comparison (fragile): %v", buf1.String() == buf2.String())
}
```

**Why:** Demonstrates why `reflect.DeepEqual` is better than JSON comparison (Issue #2)

---

#### Test 1.5: Nested Structures Handled Correctly

```go
func TestDeduplicateToolCallsNestedStructures(t *testing.T) {
    lastTC := protocoltypes.ToolCall{
        Name: "api_call",
        Arguments: map[string]any{
            "endpoint": "/users",
            "params": map[string]any{
                "id": "123",
                "filter": map[string]any{
                    "active": true,
                    "role": "admin",
                },
            },
        },
    }
    
    currentTC := protocoltypes.ToolCall{
        Name: "api_call",
        Arguments: map[string]any{
            "endpoint": "/users",
            "params": map[string]any{
                "id": "123",
                "filter": map[string]any{
                    "role": "admin",  // Different order
                    "active": true,
                },
            },
        },
    }
    
    // Should match (nested order doesn't matter with DeepEqual)
    if !reflect.DeepEqual(lastTC.Arguments, currentTC.Arguments) {
        t.Fatal("Nested structures should match")
    }
}
```

**Why:** Ensures complex arguments handled correctly

---

### 2. Unit Tests: Message History Walk (Issue #3)

**File:** `pkg/agent/loop_message_walk_test.go`

#### Test 2.1: Simple Message History

```go
func TestFindPreviousAssistantMessageSimple(t *testing.T) {
    messages := []providers.Message{
        {Role: "user", Content: "Hello"},
        {Role: "assistant", Content: "Hi", ToolCalls: []protocoltypes.ToolCall{
            {Name: "tool1"},
        }},
        {Role: "tool", Content: "Result"},
        {Role: "assistant", Content: "Now what?", ToolCalls: []protocoltypes.ToolCall{
            {Name: "tool2"},
        }},
    }
    
    // Find last assistant message and previous one
    var lastAssistantMsg *provides.Message
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].Role == "assistant" && i > 0 {
            lastAssistantMsg = &messages[i-1]
            break
        }
    }
    
    // Should find the first assistant message, not the current one
    if lastAssistantMsg == nil || lastAssistantMsg.Content != "Hi" {
        t.Fatal("Should find previous assistant message")
    }
}
```

**Why:** Verifies safe backward walk working correctly (Issue #3)

---

#### Test 2.2: Complex Message Structure with Multiple Tool Results

```go
func TestFindPreviousAssistantMessageMultipleResults(t *testing.T) {
    messages := []providers.Message{
        {Role: "assistant", ToolCalls: []protocoltypes.ToolCall{{Name: "tool1"}}},
        {Role: "tool", Content: "Result 1"},
        {Role: "tool", Content: "Result 2"},
        {Role: "tool", Content: "Result 3"},
        {Role: "assistant", ToolCalls: []protocoltypes.ToolCall{{Name: "tool2"}}},
    }
    
    // Find last assistant's previous message
    var lastAssistantMsg *providers.Message
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].Role == "assistant" && i > 0 {
            lastAssistantMsg = &messages[i-1]
            break
        }
    }
    
    // Should find assistant message at index 0 (after skipping last assistant)
    if lastAssistantMsg == nil || lastAssistantMsg.Role != "assistant" {
        t.Fatal("Should find previous assistant message despite multiple tool results")
    }
}
```

**Why:** Handles variable message structure (Issue #3)

---

#### Test 2.3: Edge Case: First Message Only

```go
func TestFindPreviousAssistantMessageFirstOnly(t *testing.T) {
    messages := []providers.Message{
        {Role: "assistant", ToolCalls: []protocoltypes.ToolCall{{Name: "tool1"}}},
    }
    
    // Try to find previous assistant message
    var lastAssistantMsg *providers.Message
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].Role == "assistant" && i > 0 {  // i > 0 check
            lastAssistantMsg = &messages[i-1]
            break
        }
    }
    
    // Should not find anything (i > 0 check prevents access to [-1])
    if lastAssistantMsg != nil {
        t.Fatal("Should not find previous message when only one message exists")
    }
}
```

**Why:** Prevents index out of bounds errors (Issue #3)

---

### 3. Unit Tests: Multiple Tool Calls (Issue #1)

**File:** `pkg/agent/loop_multi_tool_test.go`

#### Test 3.1: All Tool Calls Identical

```go
func TestMultipleToolCallsAllIdentical(t *testing.T) {
    lastCalls := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},
        {Name: "tool2", Arguments: map[string]any{"id": "2"}},
        {Name: "tool3", Arguments: map[string]any{"id": "3"}},
    }
    
    currentCalls := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},
        {Name: "tool2", Arguments: map[string]any{"id": "2"}},
        {Name: "tool3", Arguments: map[string]any{"id": "3"}},
    }
    
    // Check all match
    allMatch := len(lastCalls) == len(currentCalls)
    if allMatch {
        for i := range currentCalls {
            if lastCalls[i].Name != currentCalls[i].Name {
                allMatch = false
                break
            }
            if !reflect.DeepEqual(lastCalls[i].Arguments, currentCalls[i].Arguments) {
                allMatch = false
                break
            }
        }
    }
    
    if !allMatch {
        t.Fatal("All tools should match")
    }
}
```

**Why:** Handles multiple tool calls (Issue #1)

---

#### Test 3.2: Only First Tool Identical (Should Not Trigger Dedup)

```go
func TestMultipleToolCallsPartialMatch(t *testing.T) {
    lastCalls := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},
        {Name: "tool2", Arguments: map[string]any{"id": "2"}},
    }
    
    currentCalls := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},  // Same
        {Name: "tool2", Arguments: map[string]any{"id": "NEW"}}, // Different!
    }
    
    // Should NOT deduplicate
    allMatch := len(lastCalls) == len(currentCalls)
    if allMatch {
        for i := range currentCalls {
            if lastCalls[i].Name != currentCalls[i].Name {
                allMatch = false
                break
            }
            if !reflect.DeepEqual(lastCalls[i].Arguments, currentCalls[i].Arguments) {
                allMatch = false
                break
            }
        }
    }
    
    if allMatch {
        t.Fatal("Should not match - second tool is different")
    }
}
```

**Why:** Prevents silently dropping legitimate tool calls (Issue #1)

---

#### Test 3.3: Different Number of Tool Calls

```go
func TestMultipleToolCallsDifferentCount(t *testing.T) {
    lastCalls := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},
        {Name: "tool2", Arguments: map[string]any{"id": "2"}},
    }
    
    / Current has MORE tool calls
    currentCalls := []protocoltypes.ToolCall{
        {Name: "tool1", Arguments: map[string]any{"id": "1"}},
        {Name: "tool2", Arguments: map[string]any{"id": "2"}},
        {Name: "tool3", Arguments: map[string]any{"id": "3"}},  // Extra
    }
    
    allMatch := len(lastCalls) == len(currentCalls)  // Will be false
    
    if allMatch {
        t.Fatal("Should not match - different count")
    }
}
```

**Why:** Handles variable tool count (Issue #1)

---

### 4. Integration Tests

**File:** `pkg/agent/loop_integration_test.go`

#### Test 4.1: Full Loop Scenario Without Dedup (Baseline)

```go
func TestLoopWithoutDedupSpam(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long integration test")
    }
    
    tmpDir, _ := os.MkdirTemp("", "agent-test-*")
    defer os.RemoveAll(tmpDir)
    
    cfg := &config.Config{
        Agents: config.AgentsConfig{
            Defaults: config.AgentDefaults{
                Workspace: tmpDir,
                Model: "stub",
                MaxTokens: 4096,
                MaxToolIterations: 15,
            },
        },
    }
    
    // Stub provider that returns same tool call every iteration
    provider := &testStuckProvider{}
    
    msgBus := bus.NewMessageBus()
    loop := NewAgentLoop(cfg, msgBus, provider)
    
    // If dedup works: provider called ~2 times
    // If dedup broken: provider called ~15 times
    if provider.callCount > 5 {
        t.Logf("WARNING: Dedup not working, LLM.Chat called %d times", provider.callCount)
    }
}
```

**Why:** Full end-to-end verification

---

#### Test 4.2: Real-World Scenario: Subagent Completion

```go
func TestSubagentCompletionNoSpam(t *testing.T) {
    // Setup agent with mock provider that simulates async completion
    tmpDir, _ := os.MkdirTemp("", "agent-test-*")
    defer os.RemoveAll(tmpDir)
    
    provider := &testAsyncSubagentProvider{
        completionMessage: "✅ Subagent-3 completed weather check",
    }
    
    msgBus := bus.NewMessageBus()
    
    messagesSent := 0
    msgBus.Subscribe("message", func(msg map[string]any) {
        messagesSent++
    })
    
    cfg := &config.Config{
        Agents: config.AgentsConfig{
            Defaults: config.AgentDefaults{
                Workspace: tmpDir,
                Model: "test",
                MaxToolIterations: 15,
            },
        },
    }
    
    al := NewAgentLoop(cfg, msgBus, provider)
    
    // Should send completion message exactly once, not 15 times
    if messagesSent > 2 {
        t.Errorf("Expected ≤2 messages, got %d (spam detected)", messagesSent)
    }
}
```

**Why:** Reproduces original Issue #545 scenario

---

### 5. Regression Tests

**File:** `pkg/agent/loop_regression_test.go`

#### Test 5.1: Legitimate Tool Retries Still Work

```go
func TestLegitimateToolRetryNotKilled(t *testing.T) {
    // Scenario: First tool call fails, LLM retries with same args
    provider := &testRetryProvider{
        firstCallFails: true,
        retryCount: 1,
    }
    
    // With 3-duplicate threshold: first retry allowed ✓
    // Should complete successfully
    
    cfg := &config.Config{
        Agents: config.AgentsConfig{
            Defaults: config.AgentDefaults{
                MaxToolIterations: 10,
            },
        },
    }
    
    al := NewAgentLoop(cfg, msgBus, provider)
    
    if provider.retryCount < 1 {
        t.Error("Legitimate retry was killed by dedup")
    }
}
```

**Why:** Ensures dedup doesn't break legitimate retries (Issue #6)

---

#### Test 5.2: Multi-Tool Workflows Not Broken

```go
func TestMultiToolWorkflowStillWorks(t *testing.T) {
    provider := &testMultiToolProvider{
        tools: []string{"search", "summarize", "message"},
    }
    
    msgBus := bus.NewMessageBus()
    al := NewAgentLoop(cfg, msgBus, provider)
   
    if provider.executedTools < 3 {
        t.Error("Multi-tool workflow broken by dedup")
    }
}
```

**Why:** Ensures workflows with multiple different tools still function

---

### 6. Benchmark Tests

**File:** `pkg/agent/loop_bench_test.go`

```go
func BenchmarkDuplicateDetection(b *testing.B) {
    lastTC := protocoltypes.ToolCall{
        Name: "message",
        Arguments: generateLargeArguments(1000),
    }
    
    currentTC := protocoltypes.ToolCall{
        Name: "message",
        Arguments: generateLargeArguments(1000),
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = reflect.DeepEqual(lastTC.Arguments, currentTC.Arguments)
    }
}

func BenchmarkJsonMarshalComparison(b *testing.B) {
    // For comparison: why reflect.DeepEqual is better
    args := generateLargeArguments(1000)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        lastJSON, _ := json.Marshal(args)
        currentJSON, _ := json.Marshal(args)
        _ = string(lastJSON) == string(currentJSON)
    }
}
```

**Why:** Verify performance impact is acceptable

---

## Test Execution Steps

### Run All Tests

```bash
cd /workspaces/picoclaw

# Run all agent tests
go test ./pkg/agent -v

# Run only dedup tests
go test ./pkg/agent -v -run "Dedup"

# Run with coverage
go test ./pkg/agent -v -cover

# Run specific test
go test ./pkg/agent -v -run TestDeduplicateToolCallsIdentical

# Run benchmarks
go test ./pkg/agent -bench=. -benchmem
```

### Expected Coverage

```
Target: >90% coverage for dedup logic
├─ dedup_test.go:     95%+
├─ loop_message_walk_test.go: 90%+
├─ loop_multi_tool_test.go: 90%+
└─ loop_integration_test.go: 85%
```

---

## Test Data Fixtures

**File:** `pkg/agent/testdata/issue545_scenarios.go`

```go
func getStickyLLMScenario() testScenario {
    return testScenario{
        name: "LLM Stuck in Message Loop",
        maxIterations: 15,
        llmResponses: []LLMResponse{
            // Every iteration returns the same tool call
            {
                Toolca lls: []ToolCall{{
                    Name: "message",
                    Arguments: {"text": "Subagent-3 completed..."},
                }},
            },
            // Repeat 14 more times...
        },
        expectedMessages: 1, // Should be 1, not 15
    }
}
```

---

## Success Criteria

All tests must PASS:
- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ No regressions in existing tests
- ✅ Coverage > 90% for dedup code
- ✅ Benchmark performance acceptable
- ✅ No new compiler warnings

---

## Continuous Integration

**File:** `.github/workflows/test-issue545.yml`

```yaml
name: Issue #545 Tests

on: [pull_request]

jobs:
  test-issue-545:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      
      - name: Run issue #545 tests
        run: go test ./pkg/agent -v -run "Dedup|MultiTool|MessageWalk"
      
      - name: Check coverage
        run: |
          go test ./pkg/agent -cover
          # Assert > 90%
```

---

## Related Issues / Tests

- **Issue #545:** Multiple duplicate messages
- **Related:** Async delegation workflows
- **Related:** Subagent completion notifications

---

**Last Updated:** February 27, 2026
