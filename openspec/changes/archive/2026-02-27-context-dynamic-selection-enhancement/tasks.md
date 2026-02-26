## 1. Infrastructure Setup

- [x] 1.1 Modify `AgentInstance` to add `skillsFilterMutex sync.RWMutex` field
- [x] 1.2 Implement `SetSkillsFilter(filters []string)` method in AgentInstance
- [x] 1.3 Implement `GetSkillsFilter() []string` method in AgentInstance
- [x] 1.4 Update `ContextBuilder.InvalidateCache()` to be publicly accessible
- [x] 1.5 Add integration: call InvalidateCache() in SetSkillsFilter()

## 2. Tool Visibility Filters

- [x] 2.1 Define `ToolVisibilityContext` struct with Channel, ChatID, UserID, UserRoles, Args, Timestamp
- [x] 2.2 Define `ToolVisibilityFilter` function type
- [x] 2.3 Add `visibilityFilters map[string]ToolVisibilityFilter` to ToolRegistry
- [x] 2.4 Implement `RegisterWithFilter(tool Tool, filter ToolVisibilityFilter)` method
- [x] 2.5 Implement `GetDefinitionsForContext(ctx ToolVisibilityContext)` method
- [x] 2.6 Modify legacy `Register()` to maintain backward compatibility (no filtering)
- [x] 2.7 Update `AgentLoop.runLLMIteration()` to use `GetDefinitionsForContext()` instead of `GetDefinitions()`

## 3. Skills On-Demand Loading

- [x] 3.1 Implement `BuildSkillsSummaryFiltered(skillNames []string)` in SkillsLoader
- [x] 3.2 Implement `BuildSkillsSummaryForNames(skillNames []string)` in SkillsLoader
- [x] 3.3 Modify `ContextBuilder.BuildSystemPrompt()` to check SkillsFilter and call filtered version
- [x] 3.4 Add logic: if SkillsFilter is empty, load all skills; otherwise load only filtered skills

## 4. Context Building Strategies

- [x] 4.1 Define `ContextStrategy` int type and constants (Full, Lite, Custom)
- [x] 4.2 Define `ContextBuildOptions` struct with Strategy, IncludeTools, ExcludeTools, IncludeSkills, etc.
- [x] 4.3 Implement `buildLiteContext()` private method in ContextBuilder
- [x] 4.4 Implement `buildCustomContext(opts ContextBuildOptions)` private method in ContextBuilder
- [x] 4.5 Implement public `BuildMessagesWithOptions(history, summary, message, media, channel, chatID, opts)` method
- [x] 4.6 Update existing `BuildMessages()` to call `BuildMessagesWithOptions()` with default Full strategy

## 5. Skill Recommender (Optional - P1)

- [x] 5.1 Create `SkillRecommender` struct with dependencies (SkillsLoader, optional history store)
- [x] 5.2 Implement `RecommendSkillsForContext(channel, chatID, userMessage string, history []Message)` method
- [x] 5.3 Implement channel-based recommendation rules (Telegram → sticker/poll skills, etc.)
- [x] 5.4 Implement keyword-based matching (debug/deploy → technical skills)
- [x] 5.5 Implement scoring algorithm with weights: channel 40%, keyword 30%, history 20%, recency 10%
- [x] 5.6 Integrate recommender into ContextBuilder as optional enhancement

## 6. Testing

- [x] 6.1 Write unit tests for `AgentInstance.SetSkillsFilter()` and concurrent access
- [x] 6.2 Write unit tests for `ToolRegistry.RegisterWithFilter()` and `GetDefinitionsForContext()`
- [x] 6.3 Write unit tests for `SkillsLoader.BuildSkillsSummaryFiltered()`
- [x] 6.4 Write unit tests for `ContextBuilder.BuildMessagesWithOptions()` with all strategies
- [x] 6.5 Write integration test: multi-tenant scenario (admin vs regular user tool visibility)
- [x] 6.6 Write integration test: channel-specific skill loading
- [x] 6.7 Write performance benchmark: token usage comparison (Full vs Lite vs Custom)
- [x] 6.8 Verify all existing tests still pass (backward compatibility)

## 7. Documentation

- [x] 7.1 Update ARCHITECTURE.md with new components and data flow
- [x] 7.2 Add usage examples to pkg/agent README or code comments
- [x] 7.3 Add configuration example to config/config.example.json (optional SkillsFilter config)
- [x] 7.4 Create MIGRATION.md guide for users upgrading from old versions
- [x] 7.5 Update Godoc comments for all new and modified public APIs

## 8. Validation and Cleanup

- [x] 8.1 Run `openspec validate --changes context-dynamic-selection-enhancement` to verify completeness
- [x] 8.2 Run full test suite: `make test` or `go test ./...`
- [x] 8.3 Run linter: `golangci-lint run`
- [x] 8.4 Check for breaking changes using API diff tool
- [x] 8.5 Clean up temporary files and debug logging
- [x] 8.6 Final code review and refactoring
