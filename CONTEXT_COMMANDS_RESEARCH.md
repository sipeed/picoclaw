# PicoClaw 上下文管理功能调研报告

## 📊 OpenClaw 原有功能 vs PicoClaw 当前状态

### OpenClaw 核心命令（PicoClaw 缺失）

| 命令 | 功能 | PicoClaw 状态 |
|------|------|---------------|
| `/status` | 查看会话状态（模型、Token使用量、成本） | ❌ 未实现 |
| `/compact` | 手动压缩上下文（摘要化历史） | ❌ 未实现 |
| `/new` 或 `/reset` | 开始新会话（重置session） | ❌ 未实现 |
| `/usage` | 查看Token消耗详情 | ❌ 未实现 |
| `/context list` | 查看已加载的上下文 | ❌ 未实现 |

---

## 🔍 OpenClaw /status 命令详解

### 功能
显示当前会话的详细状态信息

### 输出内容
| 信息 | 说明 |
|------|------|
| 当前模型 | 正在使用的AI模型 |
| 上下文使用量 | 已消耗的Token数量 |
| 最后响应Token | 上一轮对话的Token |
| 预估成本 | 本次会话费用（仅API Key用户可见） |
| Gateway状态 | 网关是否繁忙 |

### 使用场景
- 感觉对话变慢时检查 `/status`
- 上下文超过50%时考虑 `/new`
- 定期检查成本，避免超支

---

## 🔍 OpenClaw /compact 命令详解

### 功能
将历史对话压缩成摘要，保留最近消息完整

### 命令格式
```
/compact                    # 默认压缩
/compact 保留代码讨论        # 带指令的压缩
```

### 工作原理
1. 将旧的对话内容摘要化
2. 保留最近的消息完整
3. 摘要存储在transcript中
4. 显著减少Token消耗

### 使用场景
- 长任务开始前
- 上下文接近上限
- 保留重要上下文但省Token

### /compact vs /new 对比

| 命令 | 效果 | 适用场景 |
|------|------|---------|
| `/new` | 完全重置，清空历史 | 切换任务 |
| `/compact` | 压缩历史，保留摘要 | 继续当前任务但省Token |

---

## 📁 PicoClaw 现有上下文管理

### 已实现的功能

#### 1. 自动压缩 (Auto-compaction)
- **触发条件**: 上下文超过阈值
- **配置项**: `summarize_message_threshold`, `summarize_token_percent`
- **实现位置**: `pkg/agent/context_legacy.go`

#### 2. 强制压缩 (Force Compression)
- 当上下文超限时，丢弃最旧的50%消息
- 保留完整的对话轮次（Turn）
- 创建压缩说明摘要

#### 3. 上下文管理器
PicoClaw 支持两种上下文管理器：
- `legacy` (默认) - 基于摘要的压缩
- `seahorse` - 高级向量检索

---

## 🎯 PicoClaw 需要补充的命令

### 1. /status 命令
**功能**: 显示当前会话状态

```go
// 实现建议
func statusCommand() *Definition {
    return &Definition{
        Name:        "status",
        Aliases:     []string{"s"},
        Description: "Show session status (model, tokens, cost)",
        Handler: func(req *Request) error {
            // 获取当前session信息
            session := getSession(req.SessionKey)
            return req.Replyf(
                "📊 Session Status\n\n"+
                "Model: %s\n"+
                "Context: %d / %d tokens (%.1f%%)\n"+
                "Messages: %d\n"+
                "Compactions: %d",
                session.Model,
                session.UsedTokens,
                session.MaxTokens,
                session.UsagePercent,
                session.MessageCount,
                session.CompactionCount,
            )
        },
    }
}
```

### 2. /compact 命令
**功能**: 手动触发上下文压缩

```go
// 实现建议
func compactCommand() *Definition {
    return &Definition{
        Name:        "compact",
        Aliases:     []string{"c"},
        Description: "Compact session context (summarize history)",
        Handler: func(req *Request) error {
            // 调用现有的Compact功能
            agent := getAgent(req.AgentID)
            agent.CompactContext(req.SessionKey, req.Text)
            return req.Reply("✅ Context compacted successfully")
        },
    }
}
```

### 3. /new 命令
**功能**: 开始新的会话

```go
// 实现建议
func newCommand() *Definition {
    return &Definition{
        Name:        "new",
        Aliases:     []string{"reset"},
        Description: "Start a new session (reset conversation)",
        Handler: func(req *Request) error {
            // 创建新的session
            agent := getAgent(req.AgentID)
            agent.NewSession(req.SessionKey)
            return req.Reply("🆕 New session started. Previous context has been archived.")
        },
    }
}
```

---

## 📂 相关代码文件

### PicoClaw 核心文件
- `pkg/agent/context_manager.go` - 上下文管理器接口
- `pkg/agent/context_legacy.go` - 现有压缩实现
- `pkg/session/manager.go` - 会话管理
- `pkg/commands/builtin.go` - 内置命令列表
- `pkg/commands/cmd_*.go` - 各命令实现

### Telegram 集成
- `pkg/channels/telegram/telegram.go` - Telegram通道
- `pkg/channels/telegram/commands.go` - Telegram命令处理（需新增）

---

## 🛠️ 实现计划

### 第一阶段：基础命令
1. 实现 `/status` 命令
2. 实现 `/new` 命令
3. 实现 `/reset` 命令（别名）

### 第二阶段：压缩命令
4. 实现 `/compact` 命令
5. 添加压缩计数追踪
6. 优化压缩摘要质量

### 第三阶段：增强功能
7. 实现 `/usage` 命令
8. 实现 `/context` 命令
9. 添加成本计算

---

## 📝 配置项建议

```json
{
  "agents": {
    "defaults": {
      "summarize_message_threshold": 20,
      "summarize_token_percent": 75
    }
  },
  "commands": {
    "status": {
      "show_cost": true,
      "show_context_percent": true
    },
    "compact": {
      "default_summary": "会话摘要"
    }
  }
}
```

---

## 🔗 参考资源

- [OpenClaw Usage Guide](https://openclaw-ai.online/usage/)
- [OpenClaw Session Management Deep Dive](https://github.com/openclaw/openclaw/blob/main/docs/reference/session-management-compaction.md)
- [Compaction Documentation](https://openclaw.dog/docs/concepts/compaction/)

---

## 📌 下一步行动

1. 在 `pkg/commands/` 目录下创建新命令文件
2. 在 `builtin.go` 中注册新命令
3. 确保 Telegram 通道正确路由这些命令
4. 添加测试用例
5. 更新文档

---

*调研完成时间: 2026-04-12*
