# 动态速率限制

PicoClaw 通过在每次发送请求之前对每个模型实施可配置的请求速率限制来防止 LLM 提供商 API 的 429 错误。与被动式冷却/回退系统（在收到 429 *之后*才激活）不同，速率限制是**主动式的**：它将出口 QPS 保持在提供商的免费层或计划限制内。

## 工作原理

### 令牌桶算法

每个限速模型都有一个令牌桶：

- **容量** = `rpm`（突发大小等于每分钟限制）
- **补充速率** = `rpm / 60` 每秒令牌数
- 每次 LLM 调用消耗一个令牌；如果桶为空，则调用阻塞直到令牌补充或请求上下文被取消

### 调用链集成

```
AgentLoop.callLLM()
  └─ FallbackChain.Execute()         ← 遍历候选者
       ├─ CooldownTracker.IsAvailable()   ← 如果处于 429 后冷却期则跳过
       ├─ RateLimiterRegistry.Wait()      ← 新增：阻塞直到令牌可用
       └─ provider.Chat()                 ← 实际的 LLM HTTP 调用
```

速率限制器在冷却检查**之后**和提供商调用**之前**运行，因此：
- 已处于冷却期的候选者被完全跳过（不消耗令牌）
- 可用的候选者被限制到配置的 RPM

相同的检查也适用于 `ExecuteImage`。

### 线程安全

`RateLimiterRegistry` 是并发安全的。每个限速器的令牌桶使用细粒度互斥锁，因此并发 goroutine 各自独立获取令牌。

## 配置

在 `model_list` 中的任何模型上设置 `rpm`：

```yaml
model_list:
  - model_name: gpt-4o-free
    model: openai/gpt-4o
    api_base: https://api.openai.com/v1
    rpm: 3          # 每分钟最多 3 个请求
    api_keys:
      - sk-...

  - model_name: claude-haiku
    model: anthropic/claude-haiku-4-5
    rpm: 60         # 60 rpm（Anthropic 免费层）
    api_keys:
      - sk-ant-...

  - model_name: local-llm
    model: openai/llama3
    api_base: http://localhost:11434/v1
    # 不设置 rpm → 无限制
```

| 字段 | 类型 | 默认值 | 描述 |
|---|---|---|---|
| `rpm` | `int` | `0` | 每分钟请求数。`0` 表示无限制。 |

### 与回退的交互

当模型配置了回退时，每个候选者独立限速：

```yaml
model_list:
  - model_name: gpt4-with-fallback
    model: openai/gpt-4o
    rpm: 5
    fallbacks:
      - gpt-4o-mini   # 必须也在 model_list 中；适用其自己的 rpm
```

如果当前候选者的桶为空且有更多可用候选者，PicoClaw 立即跳过本地饱和的候选者并尝试下一个回退。只有最后一个候选者等待令牌补充。如果在等待最后一个候选者时达到上下文截止时间，则传播等待错误。

对于解析为相同底层提供商/模型的 `model_list` 别名，速率限制按稳定的配置标识（例如 `model_name`）进行键控，而不是按解析后的运行时模型字符串。这为多密钥和基于别名的配置保留不同的 RPM 设置。

### 突发行为

桶开始时是**满的**（突发 = RPM）。对于 `rpm: 3`，前 3 个请求立即发出；后续请求间隔约 20 秒。

要为严格的 API 减少突发性，请设置较低的 `rpm` 并依赖稳态补充。

## 变更的文件

| 文件 | 变更内容 |
|---|---|
| `pkg/providers/ratelimiter.go` | `RateLimiter`（令牌桶）+ `RateLimiterRegistry` |
| `pkg/providers/ratelimiter_test.go` | 限速器和注册表的单元测试 |
| `pkg/providers/fallback.go` | `FallbackCandidate.RPM` 字段；`FallbackChain.rl`；`Execute`/`ExecuteImage` 中的 `Wait()` 调用 |
| `pkg/agent/model_resolution.go` | 从 `model_list` 解析候选者，保留稳定的配置标识并将 `RPM` 传播到 `FallbackCandidate` |
| `pkg/agent/loop.go` | 构建 `RateLimiterRegistry`，注册所有代理的候选者，传递给 `NewFallbackChain` |
