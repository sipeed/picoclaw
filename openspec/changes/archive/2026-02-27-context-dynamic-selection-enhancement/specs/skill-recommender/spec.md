## ADDED Requirements

### Requirement: Context-Based Skill Recommendation
系统必须能够根据上下文智能推荐相关技能。

#### Scenario: Recommend skills based on channel type
- **WHEN** user is in Telegram channel
- **AND** RecommendSkillsForContext is called
- **THEN** telegram-specific skills (sticker, poll) are recommended

#### Scenario: Recommend skills based on message content
- **WHEN** user message contains keywords "debug" or "deploy"
- **AND** RecommendSkillsForContext is called
- **THEN** technical skills related to debugging and deployment are recommended

#### Scenario: No recommendations when no matching skills
- **WHEN** no skills match the current context
- **THEN** empty recommendation list is returned (no errors)
