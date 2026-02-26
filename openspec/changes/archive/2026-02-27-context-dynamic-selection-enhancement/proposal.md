## Why

PicoClaw 当前的工具和技能加载机制存在以下问题：

1. **SkillsFilter 只读，不支持运行时动态修改** - Agent 初始化后无法调整可用技能列表，需要重启进程才能修改配置
2. **工具注册全量，不支持按 context 过滤** - 所有工具对所有通道可见，无法根据通道类型、用户权限限制工具，浪费 token
3. **技能摘要全量推送，不支持按需筛选** - 即使配置了 SkillsFilter，system prompt 仍然包含所有技能的摘要
4. **缺少基于上下文的智能推荐机制** - LLM 需要自己判断该用哪个工具、该看哪个技能

本需求旨在实现基于上下文的动态工具和技能选择机制，支持运行时动态过滤和智能推荐，提升系统的灵活性和资源利用效率。

## What Changes

- ✅ **新增运行时 SkillsFilter 动态修改 API** - 提供 `SetSkillsFilter()` 方法在运行时修改技能过滤器
- ✅ **新增工具可见性过滤器** - 支持注册工具时设置可见性规则，根据 channel、chatID、用户权限等条件过滤
- ✅ **新增技能摘要按需加载** - BuildSystemPrompt 时根据 SkillsFilter 只加载匹配的技能
- ✅ **新增 Context 构建策略** - 支持 Full/Lite/Custom 三种上下文构建策略
- ✅ **增强 ContextBuilder 缓存机制** - 支持显式缓存失效触发重建
- ⚠️ **BREAKING**: 无 - 所有变更都是向后兼容的，现有代码继续使用旧 API

## Capabilities

### New Capabilities
- `context-strategies`: 支持多种 Context 构建策略 (Full/Lite/Custom)
- `tool-visibility-filters`: 工具可见性过滤器，支持基于条件的动态过滤
- `skills-filter-api`: 运行时动态修改 SkillsFilter 的 API
- `skill-recommender`: 基于上下文的智能技能推荐

### Modified Capabilities
- `agent-context-builder`: 增强 ContextBuilder 支持按需加载技能和工具过滤
- `tool-registry`: 增强 ToolRegistry 支持可见性过滤器和按 context 获取定义

## Impact

**Affected Code:**
- `pkg/agent/context.go` - ContextBuilder 增强
- `pkg/agent/instance.go` - AgentInstance 添加 SetSkillsFilter 方法
- `pkg/tools/registry.go` - ToolRegistry 添加可见性过滤器
- `pkg/skills/loader.go` - SkillsLoader 添加按需加载方法

**APIs:**
- 新增: `AgentInstance.SetSkillsFilter()`, `ToolRegistry.RegisterWithFilter()`, `ToolRegistry.GetDefinitionsForContext()`
- 新增: `ContextBuilder.BuildMessagesWithOptions()`, `SkillsLoader.BuildSkillsSummaryFiltered()`

**Dependencies:**
- 无外部依赖变更

**Systems:**
- 影响所有通道的工具定义推送逻辑
- 影响 system prompt 构建的 token 消耗
