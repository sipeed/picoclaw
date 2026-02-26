## ADDED Requirements

### Requirement: Runtime SkillsFilter Modification API
系统必须提供在运行时动态修改 Agent 技能过滤器的能力，无需重启进程。

#### Scenario: Set new skills filter
- **WHEN** developer calls `agent.SetSkillsFilter([]string{"skill1", "skill2"})`
- **THEN** the agent's available skills are immediately updated to only include skill1 and skill2

#### Scenario: Trigger context cache invalidation
- **WHEN** SetSkillsFilter is called with a non-empty filter
- **THEN** ContextBuilder.InvalidateCache() is automatically invoked to rebuild system prompt

#### Scenario: Thread-safe concurrent access
- **WHEN** multiple goroutines call GetSkillsFilter simultaneously
- **THEN** all reads return consistent values without race conditions

#### Scenario: Empty filter clears all restrictions
- **WHEN** SetSkillsFilter is called with empty array or nil
- **THEN** all skills become available (no filtering applied)
