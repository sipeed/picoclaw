## ADDED Requirements

### Requirement: Multiple Context Building Strategies
系统必须支持多种上下文构建策略，适应不同的使用场景。

#### Scenario: Full strategy (default)
- **WHEN** BuildMessagesWithOptions is called with `Strategy: ContextStrategyFull`
- **THEN** complete context is built including identity, bootstrap, skills, tools, memory, and runtime info

#### Scenario: Lite strategy for minimal context
- **WHEN** BuildMessagesWithOptions is called with `Strategy: ContextStrategyLite`
- **THEN** only core identity, current time, and session info are included (no skills/tools details)

#### Scenario: Custom strategy with explicit includes
- **WHEN** BuildMessagesWithOptions is called with `Strategy: ContextStrategyCustom` and `IncludeSkills: ["skill1"]`
- **THEN** only skill1 is included in the context, all other skills are excluded
