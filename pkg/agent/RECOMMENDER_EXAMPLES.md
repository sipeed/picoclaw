# Skill Recommender Usage Examples

本文件展示了如何使用 PicoClaw 的智能技能推荐功能。

## 概述

SkillRecommender 是一个独立的组件，通过调用 LLM 基于上下文和备选技能集进行智能推理选择。它使用混合方法：

1. **规则预过滤** - 基于通道类型、关键词、历史记录进行初步评分
2. **LLM 推理** - 对多个候选技能进行智能选择
3. **多因子评分** - 整合通道 (40%)、关键词 (30%)、历史 (20%)、近期性 (10%)

## 快速开始

### 1. 启用技能推荐器

```go
// 在 agent 初始化后启用技能推荐器
agent := CreateYourAgent() // 您的 agent 创建逻辑

// 方式 1: 使用默认配置启用
agent.EnableSkillRecommender("", nil)

// 方式 2: 使用自定义权重启用
// 参数：channel, keyword, history, recency
agent.EnableSkillRecommenderWithWeights(0.5, 0.3, 0.15, 0.05)

// 方式 3: 指定使用的模型
agent.EnableSkillRecommender("gpt-4", nil)
```

### 2. 自动推荐（集成到 ContextBuilder）

一旦启用了推荐器，`BuildMessagesWithOptions` 会自动推荐使用：

```go
// 构建消息时会自动触发推荐
messages := contextBuilder.BuildMessagesWithOptions(
    history,
    summary,
    currentMessage,
    media,
    channel,
    chatID,
    agent.ContextBuildOptions{
        Strategy: agent.ContextStrategyFull,
    },
)

// 推荐器会自动：
// 1. 分析用户消息和上下文
// 2. 调用 LLM 进行智能选择
// 3. 只包含推荐的技能到 context 中
// 4. 减少 token 消耗，提高响应质量
```

### 3. 手动调用推荐器

```go
import "github.com/sipeed/picoclaw/pkg/agent"

// 创建推荐器实例
recommender := agent.NewSkillRecommender(
    skillsLoader,  // 技能加载器
    llmProvider,   // LLM 提供商
    "gpt-4",       // 使用的模型
)

// 可选：自定义权重
recommender.SetWeights(
    0.4, // channel weight  - 通道匹配度
    0.3, // keyword weight  - 关键词匹配度
    0.2, // history weight  - 历史使用
    0.1, // recency weight  - 近期使用
)

// 获取推荐
recommendations, err := recommender.RecommendSkillsForContext(
    "telegram",           // 通道类型
    "chat123",            // 聊天 ID
    "帮我创建一个投票",     // 用户消息
    conversationHistory,  // 对话历史
)

if err != nil {
    log.Printf("Recommendation failed: %v", err)
    return
}

// 处理推荐结果
for _, rec := range recommendations {
    fmt.Printf("Skill: %s\n", rec.Name)
    fmt.Printf("  Score: %.1f/100\n", rec.Score)
    fmt.Printf("  Confidence: %.2f\n", rec.Confidence)
    fmt.Printf("  Reason: %s\n", rec.Reason)
    fmt.Println()
}
```

## 使用场景

### 场景 1: 多租户系统（管理员 vs 普通用户）

```go
// 为不同用户角色设置不同的技能过滤器
func SetupAgentForUser(agent *agent.AgentInstance, userRole string) {
    switch userRole {
    case "admin":
        // 管理员：启用推荐器，但允许使用所有技能
        agent.EnableSkillRecommender("", nil)
        agent.SetSkillsFilter(nil) // 无限制
        
    case "regular":
        // 普通用户：只允许使用基础技能 + 智能推荐
        agent.SetSkillsFilter([]string{
            "schedule",
            "reminder",
            "search",
        })
        // 仍然可以使用推荐器在允许的范围内优化
        agent.EnableSkillRecommender("", nil)
        
    case "guest":
        // 访客：仅使用 Lite 模式，不启用推荐器
        // 在调用时使用 Lite 策略
    }
}
```

### 场景 2: 通道特定技能优化

```go
// Telegram 通道 - 推荐器会自动优先推荐 Telegram 相关技能
// 如 sticker、poll、inline 等技能
agent.EnableSkillRecommender("", nil)

// 企业微信通道 - 推荐器会优先推荐审批、会议等企业技能
// 如 approval、meeting、report 等技能
agent.EnableSkillRecommender("", nil)
```

### 场景 3: 动态调整策略

```go
// 根据时间段使用不同的策略
func BuildContextForTimeOfDay(agent *agent.AgentInstance, hour int) agent.ContextBuildOptions {
    if hour >= 9 && hour <= 18 {
        // 工作时间：使用完整上下文 + 推荐器
        return agent.ContextBuildOptions{
            Strategy:       agent.ContextStrategyFull,
            IncludeRuntime: true,
            IncludeMemory:  true,
        }
    } else {
        // 非工作时间：使用 Lite 模式，减少 token 消耗
        return agent.ContextBuildOptions{
            Strategy:       agent.ContextStrategyLite,
            IncludeRuntime: true,
            IncludeMemory:  false,
        }
    }
}
```

## 评分算法说明

推荐器使用加权评分系统：

```
Total Score = (ChannelScore × 0.4) + 
              (KeywordScore × 0.3) + 
              (HistoryScore × 0.2) + 
              (RecencyScore × 0.1)
```

### 各因子说明

1. **Channel Score (40%)**
   - 检查技能名称/描述是否包含通道特定关键词
   - 例如：Telegram → "sticker", "poll"
   - 企业微信 → "approval", "meeting"

2. **Keyword Score (30%)**
   - 从用户消息中提取关键词
   - 与技能描述进行匹配
   - 使用 TF-IDF 风格的词频统计

3. **History Score (20%)**
   - 检查技能是否在对话历史中被使用
   - 最近使用的技能获得高分

4. **Recency Score (10%)**
   - 基于时间衰减
   - 越近使用的技能得分越高

### 自定义权重

```go
// 如果您的场景中通道类型更重要
recommender.SetWeights(
    0.6, // channel - 提高通道权重
    0.2, // keyword - 降低关键词权重
    0.15, // history
    0.05, // recency
)

// 如果关键词匹配最关键
recommender.SetWeights(
    0.2, // channel
    0.6, // keyword - 提高关键词权重
    0.15, // history
    0.05, // recency
)
```

## LLM 推理机制

当有多个候选技能（>1）时，推荐器会调用 LLM 进行智能选择：

### Prompt 结构

```
## Context
Channel: telegram
User Message: 帮我创建一个投票

## Available Skills
1. **poll-creator** - Score: 75.0/100
   Description: channel match, keyword match
2. **sticker-manager** - Score: 45.0/100
   Description: channel match
3. **message-scheduler** - Score: 30.0/100
   Description: keyword match

## Task
Based on the context and available skills, recommend the most appropriate skills.
Consider:
- The user's intent from their message
- Channel-specific capabilities
- Historical context

Respond in JSON format:
{
  "recommendations": [
    {
      "skill_name": "skill-name",
      "confidence": 0.9,
      "reason": "why this skill is recommended"
    }
  ]
}
```

### LLM 输出处理

LLM 的推荐会用于调整预评分：
- 高置信度的推荐会boost分数（最多提升 50%）
- 低置信度的推荐保持原分
- 未被 LLM 提及的技能保持原分

## 性能优化建议

### 1. 缓存推荐结果

```go
// 对于相同的 channel+userMessage 组合，可以缓存推荐结果
type RecommendationCache struct {
    mu    sync.RWMutex
    cache map[string][]agent.SkillRecommendation
}

func (c *RecommendationCache) Get(key string) []agent.SkillRecommendation {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.cache[key]
}

func (c *RecommendationCache) Set(key string, recs []agent.SkillRecommendation) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache[key] = recs
}

// 使用
cacheKey := fmt.Sprintf("%s:%s", channel, userMessage)
if cached := cache.Get(cacheKey); cached != nil {
    return cached // 使用缓存
}

// 调用推荐器
recs, _ := recommender.RecommendSkillsForContext(...)
cache.Set(cacheKey, recs)
```

### 2. 设置推荐阈值

```go
// 只使用分数高于阈值的技能
recommendedSkillNames := make([]string, 0)
for _, rec := range recommendations {
    if rec.Score >= 50.0 { // 只使用 50 分以上的技能
        recommendedSkillNames = append(recommendedSkillNames, rec.Name)
    }
}
```

### 3. 限制推荐数量

```go
// 只取前 N 个推荐
maxRecommendations := 5
if len(recommendations) > maxRecommendations {
    recommendations = recommendations[:maxRecommendations]
}
```

## 调试和监控

### 启用调试日志

```go
// 在 config 中启用 debug 模式
config := &config.Config{
    Debug: true,
}

// 查看推荐器的详细日志
// 日志会显示：
// - 预过滤后的技能列表
// - LLM 调用详情
// - 最终推荐的技能和分数
```

### 监控推荐质量

```go
// 记录推荐指标
func LogRecommendationMetrics(recs []agent.SkillRecommendation) {
    totalScore := 0.0
    avgConfidence := 0.0
    
    for _, rec := range recs {
        totalScore += rec.Score
        avgConfidence += rec.Confidence
    }
    
    if len(recs) > 0 {
        log.Printf("Avg Score: %.1f", totalScore/float64(len(recs)))
        log.Printf("Avg Confidence: %.2f", avgConfidence/float64(len(recs)))
    }
}
```

## 常见问题

### Q: 推荐器会影响性能吗？

A: 影响很小：
- 规则预过滤：< 1ms
- LLM 调用：~100-500ms（取决于模型）
- 只在首次调用时发生，后续可使用缓存

### Q: 如何禁用推荐器？

```go
// 方式 1: 不启用推荐器（默认行为）
// 什么都不做即可

// 方式 2: 清除已启用的推荐器
agent.ContextBuilder.SetSkillRecommender(nil)
```

### Q: 推荐器会替换 SkillsFilter 吗？

A: 不会，两者可以共存：
- `SkillsFilter`: 硬性的白名单/黑名单
- `Recommender`: 软性的智能推荐
- 优先级：SkillsFilter > Recommender

### Q: 可以在运行时动态开启/关闭吗？

A: 可以：

```go
// 开启
agent.EnableSkillRecommender("", nil)

// 关闭
agent.ContextBuilder.SetSkillRecommender(nil)

// 重新开启（带不同配置）
agent.EnableSkillRecommenderWithWeights(0.6, 0.3, 0.1, 0.0)
```

## 最佳实践

1. **从小规模开始** - 先在测试环境启用，观察效果
2. **监控 token 使用** - 对比启用前后的 token 消耗
3. **收集用户反馈** - 根据实际使用情况调整权重
4. **定期更新技能描述** - 确保描述准确，便于关键词匹配
5. **使用缓存** - 对于高频场景使用推荐缓存

## 下一步

- 尝试不同的权重配置找到最优解
- 实现推荐结果缓存进一步优化性能
- 添加 A/B 测试对比不同配置的效果
- 收集实际数据持续优化推荐算法
