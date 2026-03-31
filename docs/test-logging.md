# 🧪 Picoclaw CLI 日志功能自测指南

## ✅ 构建状态

| 项目 | 状态 |
|------|------|
| **二进制大小** | 77 MB |
| **架构** | ARM64 (aarch64) |
| **构建标签** | `stdjson` (禁用 goolm) |
| **CGO** | 禁用 |

---

## 🚀 快速测试

### 1. 查看帮助

```bash
cd /data/data/com.termux/files/home/picoMind/picoclaw
./picoclaw agent --help
```

**预期输出**：显示新的日志选项
- `--verbose` / `-v`
- `--show-tools`
- `--show-think`

---

### 2. 测试交互式模式（无日志）

```bash
./picoclaw agent
```

**预期行为**：
- 显示欢迎横幅
- 进入交互模式
- 输入消息后直接显示响应（无中间日志）

---

### 3. 测试详细日志模式（推荐）

```bash
./picoclaw agent --verbose
```

**预期日志**（亮色方案，黑色背景清晰可见）：
```
⚙️  Thinking (channel: cli)                    [亮青色]
📡 Calling API: gpt-4 (messages: 5, tools: 3)  [亮紫色]
    📝 User: 你好，请帮我查询天气                 [亮青色]
    🔧 Tools: web_search, web_fetch            [亮黄色]
    ⚙️  MaxTokens: 4096, Temp: 0.70            [亮白色]
✅ API Response (content: 120 chars)           [亮绿色]
    📄 Content: 根据最新数据...                  [亮白色]
    🔧 Tool: web_search(query=北京天气)          [亮黄色]
    🏁 Finish: stop                            [亮青色]
⏱️  Completed in 1.5s                          [亮白色]
```

**颜色方案**：
| 元素 | 颜色 | ANSI 代码 |
|------|------|----------|
| 思考指示 | 亮青色 | `1;36m` |
| API 调用 | 亮紫色 | `1;35m` |
| API 响应 | 亮绿色 | `1;32m` |
| 工具调用 | 亮黄色 | `1;33m` |
| 详细内容 | 亮白色 | `1;37m` |
| 错误信息 | 亮红色 | `1;31m` |
| 响应输出 | 亮蓝色 | `1;34m` |

---

### 4. 测试工具执行日志

```bash
./picoclaw agent --show-tools
```

**预期日志**（如果触发工具调用）：
```
🔧 Tool calling: web_search(query="最新 AI 新闻")
✅ Tool completed: web_search (23ms)
```

---

### 5. 测试思考过程日志

```bash
./picoclaw agent --show-think
```

**预期日志**：
```
⚙️  Thinking (channel: cli)
📡 Calling API: gpt-4 (messages: 5, tools: 3)
✅ API response (content: 120 chars, tool calls: 2)
```

---

### 6. 测试组合模式

```bash
./picoclaw agent --verbose --show-tools --show-think
```

**预期**：显示所有类型的日志，包括：
- 思考指示
- API 请求详情（用户消息、工具列表、参数）
- API 响应详情（内容预览、工具调用、finish reason）
- 工具执行状态
- 性能统计

---

### 7. 测试单条消息模式

```bash
# 无日志
./picoclaw agent -m "你好"

# 带日志
./picoclaw agent -m "查询天气" --verbose
```

---

## 📋 测试检查清单

### 基本功能
- [ ] 帮助信息正确显示新选项
- [ ] 交互模式正常启动
- [ ] 可以正常输入和退出（`exit` 或 Ctrl+C）
- [ ] 响应正常显示

### 日志功能
- [ ] `--verbose` 显示详细日志
- [ ] `--show-tools` 显示工具执行日志
- [ ] `--show-think` 显示思考过程
- [ ] 日志使用彩色输出和 Emoji
- [ ] 日志不干扰最终响应显示

### 性能
- [ ] 日志输出不影响响应速度
- [ ] 无内存泄漏（长时间运行测试）

---

## 🔧 已知限制

### 禁用的功能
1. **Matrix 渠道** - 需要 libolm (CGO 依赖)
   - 影响：无法使用 Matrix 协议连接
   - 解决：需要安装 libolm 库并启用 CGO

### 构建约束
- `CGO_ENABLED=0` - 禁用所有 CGO 依赖
- `-tags stdjson` - 使用标准 JSON 库而非 goolm

---

## 📝 测试记录模板

```markdown
### 测试日期：2026-03-31

**测试环境**：
- OS: Android (Termux)
- CPU: ARM64
- Go: 1.26.1

**测试结果**：
| 测试项 | 状态 | 备注 |
|--------|------|------|
| 帮助显示 | ✅/❌ | |
| 交互模式 | ✅/❌ | |
| Verbose 日志 | ✅/❌ | |
| Tools 日志 | ✅/❌ | |
| Think 日志 | ✅/❌ | |

**问题记录**：
- 

**改进建议**：
- 
```

---

## 🐛 故障排除

### 问题 1: 构建失败 - libolm 错误
```
fatal error: 'olm/olm.h' file not found
```
**解决**：已禁用 Matrix 渠道，使用当前构建即可

### 问题 2: 日志不显示
**检查**：
1. 确认使用了正确的标志（`--verbose` 等）
2. 确认配置了 LLM Provider（需要有效的 API Token）

### 问题 3: 彩色输出异常
**原因**：终端不支持 ANSI 颜色
**解决**：使用支持颜色的终端或重定向到文件

---

## 📊 性能基准

| 模式 | 内存占用 | 启动时间 |
|------|---------|---------|
| 基础模式 | ~15 MB | <1s |
| +Verbose | ~15 MB | <1s |
| +Tools | ~15 MB | <1s |

---

## 🎯 下一步

1. **配置 LLM Provider** - 设置有效的 API Token
2. **实际对话测试** - 验证日志在真实场景的表现
3. **工具调用测试** - 启用 web_search 等工具
4. **长时间运行** - 测试内存稳定性

---

**构建完成，准备就绪！** 🦞
