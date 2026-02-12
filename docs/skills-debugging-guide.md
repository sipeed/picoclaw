# PicoClaw Skills 调试指南

## 问题排查和解决方案

### 问题1: Skills 无法被识别和使用

**症状**：
- Agent 有 21 个 skills，但在处理请求时不使用它们
- 即使明确指定 skill 名称，也提示 skill 不存在

**根本原因**：
Skills 文件不在正确的位置。代码从以下位置加载 skills：
1. `~/.picoclaw/workspace/skills/` (workspace skills - 项目级别)
2. `~/.picoclaw/skills/` (全局 skills)
3. 内置 skills 目录

但用户的 skills 实际存放在 `~/.claude/skills/`。

**解决方案**：
将需要使用的 skills 复制到 workspace：

```bash
# 复制单个 skill
cp -r ~/.claude/skills/sentinel-search ~/.picoclaw/workspace/skills/

# 或者批量复制所有 skills
cp -r ~/.claude/skills/* ~/.picoclaw/workspace/skills/
```

### 问题2: Anthropic API 404 错误

**症状**：
```
Error: LLM call failed: API error: 404 page not found
url=https://meta.nevis.sina.com.cn/wecode/anthropic/chat/completions
```

**根本原因**：
- 原来的代码只有一个通用的 `HTTPProvider`
- 它使用 OpenAI 风格的 API 端点：`/chat/completions`
- 但 Anthropic API 使用不同的端点：`/v1/messages`
- 请求/响应格式也完全不同

**解决方案**：
创建了专门的 `AnthropicProvider` (pkg/providers/anthropic_provider.go)：
- ✅ 使用正确的端点：`/v1/messages`
- ✅ 使用正确的请求头：`x-api-key` 而不是 `Authorization: Bearer`
- ✅ 转换消息格式（system 单独参数、content blocks 等）
- ✅ 正确处理 tool_use 和 tool_result
- ✅ 处理孤立的 tool_result（跳过没有对应 tool_use 的结果）

### 问题3: Tool call 参数传递错误

**症状**：
```
Error: messages.5.content.1.tool_use.name: String should have at least 1 character
Error: messages.5.content.1.tool_use.input: Input should be a valid dictionary
```

**根本原因**：
Agent 在构建 assistant 消息时，将 tool call 信息存储在：
- `tc.Function.Name` - 工具名称
- `tc.Function.Arguments` - JSON 字符串格式的参数

但 AnthropicProvider 最初只读取：
- `tc.Name` - 为空
- `tc.Arguments` - 为空的 map

**解决方案**：
在 AnthropicProvider 中添加了兼容逻辑：

```go
// 提取 name
name := tc.Name
if name == "" && tc.Function != nil {
    name = tc.Function.Name
}

// 提取 arguments
var input map[string]interface{}
if len(tc.Arguments) > 0 {
    input = tc.Arguments
} else if tc.Function != nil && tc.Function.Arguments != "" {
    // 从 JSON 字符串解析
    json.Unmarshal([]byte(tc.Function.Arguments), &input)
}
```

### 问题4: 孤立的 tool_result 导致 API 错误

**症状**：
```
Error: unexpected tool_use_id found in tool_result blocks: toolu_xxx.
Each tool_result block must have a corresponding tool_use block in the previous message.
```

**根本原因**：
- 会话历史可能被截断或清理
- tool_result 保留了，但对应的 tool_use 被删除了
- Anthropic API 严格要求 tool_result 必须紧跟在包含对应 tool_use 的 assistant 消息后

**解决方案**：
添加了 tool_use ID 跟踪机制：

```go
// 跟踪当前有效的 tool_use IDs
validToolUseIDs := make(map[string]bool)

// 在 assistant 消息中记录所有 tool_use IDs
for _, tc := range msg.ToolCalls {
    validToolUseIDs[tc.ID] = true
}

// 检查 tool_result 是否有对应的 tool_use
if msg.Role == "tool" && !validToolUseIDs[msg.ToolCallID] {
    // 跳过孤立的 tool_result
    continue
}
```

## Skills 使用流程

### 1. Skills 加载

Skills 从以下位置按优先级加载：
1. **Workspace skills** (`~/.picoclaw/workspace/skills/`)  - 最高优先级，项目专用
2. **Global skills** (`~/.picoclaw/skills/`) - 中等优先级，用户全局
3. **Builtin skills** - 最低优先级，系统内置

### 2. Skills 在系统提示中的展示

Skills 只显示摘要信息：
```xml
<skills>
  <skill>
    <name>sentinel-search</name>
    <description>Sentinel 安全平台 - 网络资产、服务和 Web 应用搜索工具</description>
    <location>/path/to/SKILL.md</location>
    <source>workspace</source>
  </skill>
</skills>
```

提示 AI："To use a skill, read its SKILL.md file using the read_file tool"

### 3. Skills 使用流程

当用户请求使用某个 skill 时：
1. AI 使用 `read_file` 工具读取 `SKILL.md`
2. 理解 skill 的 API 文档和使用方法
3. 使用 `exec` 工具调用相应的命令/API
4. 处理结果并返回给用户

## 测试 Skills

### 测试 sentinel-search skill

```bash
# 1. 确保 skill 在正确位置
ls ~/.picoclaw/workspace/skills/sentinel-search/SKILL.md

# 2. 明确指定使用 sentinel 平台
./picoclaw agent -m "使用 sentinel 平台搜索 nginx 服务，查询语句是 fp.nmap.product=nginx"

# 3. 观察日志，应该看到：
#    - list_dir: 列出 skills 目录
#    - read_file: 读取 SKILL.md
#    - exec: 执行 curl 命令调用 API
```

### 日志分析

成功的 skill 使用日志：
```
[INFO] agent: LLM requested tool calls {tools=[list_dir], count=1, iteration=1}
[INFO] agent: Tool call: list_dir({"path":"/Users/xingyue/.picoclaw/workspace/skills"})
[INFO] agent: LLM requested tool calls {tools=[read_file], count=1, iteration=2}
[INFO] agent: Tool call: read_file({"path":"...sentinel-search/SKILL.md"})
[INFO] agent: LLM requested tool calls {tools=[exec], count=1, iteration=3}
[INFO] agent: Tool call: exec({"command":"curl -X POST http://..."})
```

## 调试技巧

### 1. 启用 Debug 日志

```bash
# 启用所有 debug 日志
export PICOCLAW_LOG_LEVEL=debug

# 只启用 llm 分类的 debug 日志
export PICOCLAW_LOG_CATEGORY_LLM=debug
```

### 2. 检查 Skills 是否加载

```bash
# 查看 agent 初始化日志
./picoclaw agent -m "test" 2>&1 | grep "Agent initialized"
# 应该看到: skills_total=22, skills_available=22
```

### 3. 验证 API 连接

```bash
# 手动测试 Sentinel API
curl -X POST http://172.16.10.239:31223/api/search \
  -H "Content-Type: application/json" \
  -d '{"query":"fp.nmap.product=nginx","page":1,"page_size":5}'
```

### 4. 检查网络连接

如果 skill 涉及网络请求，确保：
- ✅ VPN 已连接（如果需要）
- ✅ 防火墙允许访问
- ✅ 目标服务正常运行

## 创建自定义 Skills

### Skill 目录结构

```
~/.picoclaw/workspace/skills/my-skill/
├── SKILL.md          # Skill 文档（必需）
└── .claude/          # 可选的元数据
    └── config.json
```

### SKILL.md 格式

```markdown
---
name: my-skill
description: 简短描述（会显示在 skills 列表中）
version: 1.0.0
author: your-name
tags:
  - tag1
  - tag2
triggers:
  - 触发词1
  - 触发词2
---

# Skill 名称

详细的使用说明和 API 文档...

## 使用示例

\```bash
# 示例命令
curl ...
\```
```

### Best Practices

1. **清晰的文档** - 提供完整的 API 文档和示例
2. **错误处理** - 说明常见错误和解决方法
3. **网络要求** - 明确说明网络依赖和访问限制
4. **参数说明** - 详细描述所有参数和选项

## 相关文件

- `pkg/skills/loader.go` - Skills 加载器
- `pkg/agent/context.go` - Skills 在上下文中的集成
- `pkg/providers/anthropic_provider.go` - Anthropic API 适配器
- `docs/llm-debug-guide.md` - LLM 调试指南

## 未来改进

### 1. Skills 自动同步

考虑添加配置选项，自动从 `~/.claude/skills/` 同步到 workspace：

```yaml
skills:
  auto_sync: true
  source: ~/.claude/skills
```

### 2. Skills 作为工具

考虑将常用 skills 直接注册为工具，而不需要每次都 read_file：

```go
// 将 skill 转换为 tool definition
func (s *Skill) AsToolDefinition() providers.ToolDefinition {
    // ...
}
```

### 3. Skills 模板

提供 skill 创建模板：

```bash
picoclaw skill new my-skill --template api-wrapper
```

## 总结

经过调试，现在 PicoClaw 的 skills 系统可以正常工作：

1. ✅ Skills 从正确的位置加载
2. ✅ Anthropic API 正确处理消息和工具调用
3. ✅ Tool calls 的 name 和 arguments 正确提取
4. ✅ 孤立的 tool_result 被过滤
5. ✅ 详细的调试日志帮助排查问题

Skills 使用流程清晰简单：
1. 将 skill 放到 workspace/skills/
2. 明确告诉 AI 使用哪个 skill
3. AI 读取 SKILL.md 并执行相应操作
