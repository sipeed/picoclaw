## Context

PicoClaw 当前的工具和技能加载机制存在以下问题：

1. **SkillsFilter 只读** - AgentInstance.SkillsFilter 在初始化后无法修改
2. **工具全量注册** - 所有工具对所有用户可见，无法按权限过滤
3. **技能摘要全量推送** - system prompt 包含所有技能，浪费 token
4. **缺少智能推荐** - LLM 需要自己判断使用哪个技能

当前架构中：
- `AgentInstance` 持有 `Tools *ToolRegistry` 和 `ContextBuilder *ContextBuilder`
- `ToolRegistry` 使用 `sync.RWMutex` 保护工具映射
- `ContextBuilder` 已有缓存机制（issue #607），支持 InvalidateCache()
- `SkillsLoader` 支持按名称加载技能 (`LoadSkillsForContext`)

**约束条件：**
- 必须保持向后兼容
- 不能破坏现有的 ToolRegistry 核心架构
- 改动涉及多个核心模块 (agent, tools, skills)

## Goals / Non-Goals

**Goals:**
- ✅ 提供运行时动态修改 SkillsFilter 的 API
- ✅ 实现工具可见性过滤器机制
- ✅ 支持技能摘要按需加载
- ✅ 提供多种 Context 构建策略
- ✅ 所有变更向后兼容

**Non-Goals:**
- ❌ 不改变 ToolRegistry 的核心数据结构
- ❌ 不改变 SkillsLoader 的文件系统加载机制
- ❌ 不涉及 LLM 侧的工具调用逻辑修改
- ❌ 不需要数据库或持久化存储变更

## Decisions

### Decision 1: SkillsFilter 动态修改采用 Mutex 保护

**选择:** 在 `AgentInstance` 中添加 `skillsFilterMutex sync.RWMutex`

**理由:**
- 与现有 `Tools *ToolRegistry` 的并发控制模式一致
- RWMutex 允许多个 goroutine 同时读取，写操作独占
- Go 标准库，无额外依赖

**替代方案:**
- 使用 `atomic.Value` - 不适合 slice 类型，需要频繁转换
- 使用 channel 传递修改请求 - 增加复杂性，不必要

### Decision 2: 工具过滤器使用函数类型而非配置结构

**选择:** `type ToolVisibilityFilter func(ctx ToolVisibilityContext) bool`

**理由:**
- 最大灵活性，开发者可以实现任意复杂的过滤逻辑
- 符合 Go 的函数式编程习惯
- 易于组合和测试

**替代方案:**
- 使用 JSON 配置的规则引擎 - 过于复杂，性能开销大
- 使用固定的过滤条件枚举 - 限制了扩展性

### Decision 3: Context 策略使用 iota 枚举而非字符串

**选择:** 
```go
type ContextStrategy int
const (
    ContextStrategyFull ContextStrategy = iota
    ContextStrategyLite
    ContextStrategyCustom
)
```

**理由:**
- 类型安全，编译器可以检查
- 性能优于字符串比较
- 符合 Go 标准库的枚举模式

### Decision 4: 技能推荐作为独立组件而非 SkillsLoader 的方法

**选择:** 创建新的 `SkillRecommender` 结构体

**理由:**
- 单一职责原则 - 推荐逻辑与文件加载逻辑分离
- 便于未来扩展（可以使用 ML 模型而不影响 Loader）
- 易于单元测试

## Risks / Trade-offs

### Risk 1: 过滤器函数的性能开销

**风险:** 每次获取工具定义都要执行过滤函数，可能影响性能

**缓解:**
- 过滤器应该保持简单，避免复杂计算
- 考虑添加缓存层（如果性能成为瓶颈）
- 通过基准测试验证性能影响 < 10%

### Risk 2: 过滤器配置错误导致工具不可见

**风险:** 开发者配置错误的过滤条件，导致关键工具对用户不可见

**缓解:**
- 添加调试日志，记录被过滤掉的工具
- 提供 `GetAllDefinitions()` 用于调试（绕过过滤）
- 在开发环境默认禁用所有过滤器

### Risk 3: Cache 失效导致的竞争条件

**风险:** SetSkillsFilter 触发 InvalidateCache，但 concurrent 请求可能看到不一致的状态

**缓解:**
- 确保 InvalidateCache 和 BuildSystemPrompt 使用相同的 mutex
- 在 agent 级别处理请求，而不是全局共享
- 添加集成测试验证并发场景

### Trade-off: 灵活性 vs 复杂性

**权衡:** 提供了高度灵活的过滤器机制，但增加了代码复杂性

**接受理由:**
- 目标用户是企业级应用，需要细粒度控制
- 通过良好的文档和示例降低使用门槛
- 保持向后兼容，简单场景可以忽略新特性

## Migration Plan

### Phase 1: 基础设施 (Week 1)
1. 修改 `AgentInstance` 添加 mutex 保护和 `SetSkillsFilter()` 方法
2. 修改 `ToolRegistry` 添加 `RegisterWithFilter()` 和 `GetDefinitionsForContext()`
3. 修改 `ContextBuilder` 添加 `BuildMessagesWithOptions()`
4. 所有修改保持向后兼容

### Phase 2: 增强功能 (Week 2)
1. 实现 `SkillRecommender` 基础版本
2. 实现 `ContextStrategyLite` 和 `ContextStrategyCustom`
3. 添加单元测试覆盖所有新 API

### Phase 3: 集成测试 (Week 3)
1. E2E 测试验证多租户场景
2. 性能测试对比 token 消耗
3. 更新文档和使用示例

**Rollback Strategy:**
- 所有新功能都是可选的，不使用就不会生效
- 如果出现严重 bug，可以回退到只使用旧 API
- 不需要数据库迁移，回滚只需代码回退

## Open Questions

1. **是否需要在 config.json 中声明过滤器规则？**
   - 当前设计：硬编码在 Go 代码中
   - 未来可能：支持从配置文件加载
   
2. **技能推荐的权重如何调优？**
   - 当前设计：固定权重（channel 40%, keyword 30%, history 20%, recency 10%）
   - 未来可能：基于使用数据自动优化

3. **是否需要可视化工具管理过滤器？**
   - 暂不实现，等待用户反馈
