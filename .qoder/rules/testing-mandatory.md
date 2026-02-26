# PicoClaw Development Rules

## Testing Requirements (MANDATORY)

1. **No feature is complete without tests**
   - Unit tests for all public APIs
   - Integration tests for cross-module interactions
   - E2E tests for user-facing features

2. **Test-Driven Development Preferred**
   - Write test first when possible
   - Red → Green → Refactor cycle

3. **Coverage Requirements**
   - New code MUST have >80% line coverage
   - Critical paths MUST have 100% coverage
   - All error paths MUST be tested

4. **Test Quality Standards**
   - Tests MUST be deterministic (no flaky tests)
   - Tests MUST be independent (no ordering dependencies)
   - Tests MUST be fast (<100ms per unit test)
   - Use table-driven tests for multiple scenarios

5. **Before Committing**
   - Run `go test ./...` - all must pass
   - Run `go test -race ./...` - no race conditions
   - Run `golangci-lint run` - no issues

## Implementation Workflow

```
1. Read task from tasks.md
2. If it's a feature task:
   a. Implement the feature
   b. IMMEDIATELY write corresponding test
   c. Run test - MUST pass
   d. Mark task complete ONLY after test passes
3. If it's a test task:
   a. Write comprehensive tests
   b. Run and verify they pass
   c. Mark complete
```

## Code Review Checklist

- [ ] Tests included for all new functionality?
- [ ] Test scenarios match spec requirements?
- [ ] Edge cases covered?
- [ ] Error handling tested?
- [ ] Concurrent access tested (if applicable)?
- [ ] All tests passing locally?
