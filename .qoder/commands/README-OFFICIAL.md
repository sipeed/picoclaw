# OpenSpec 官方配置指南 - PicoClaw 项目

根据 OpenSpec 官方文档整理的正确配置方法。

## 📋 官方文档参考

- **Commands 文档**: `/Users/pengweiye/Documents/codes/OpenSpec/docs/commands.md`
- **Getting Started**: `/Users/pengweiye/Documents/codes/OpenSpec/docs/getting-started.md`
- **OPSX Workflow**: `/Users/pengweiye/Documents/codes/OpenSpec/docs/opsx.md`
- **Supported Tools**: `/Users/pengweiye/Documents/codes/OpenSpec/docs/supported-tools.md`

---

## ✅ 正确的目录结构

### Qoder Agent 配置位置

根据官方文档，Qoder 的命令应该放在：

```
.qoder/commands/opsx/     # ← 注意 opsx/ 子目录
├── new.md               # /opsx:new
├── ff.md                # /opsx:ff  
├── apply.md             # /opsx:apply
├── list.md              # /opsx:list
├── validate.md          # /opsx:validate
├── archive.md           # /opsx:archive
└── show.md              # /opsx:show
```

**而不是：**
```
.qoder/commands/          # ❌ 错误位置
├── opsx-new.md
├── opsx-ff.md
...
```

---

## 🚀 完整的交互流程

### 1. 初始化（如果还没做）

```bash
cd /Users/pengweiye/Documents/codes/picoclaw
openspec init
```

这会：
- ✅ 自动检测 Qoder
- ✅ 在 `.qoder/skills/` 生成技能文件
- ✅ 在 `.qoder/commands/opsx/` 生成命令文件
- ✅ 创建 `openspec/config.yaml`（可选但推荐）

### 2. 开始工作

#### **方式 A: 使用 slash commands（推荐）**

在你的 AI 助手（Qoder）对话中直接使用：

```text
# 1. 探索需求（可选）
/opsx:explore context-dynamic-selection

# 2. 创建变更
/opsx:new context-dynamic-selection-enhancement

# 3. 生成所有规划文档
/opsx:ff

# 4. 实现功能
/opsx:apply

# 5. 验证质量
/opsx:verify

# 6. 归档
/opsx:archive
```

#### **方式 B: 手动指示 AI**

如果 slash commands 不工作，可以直接告诉 AI：

```text
我们将使用 OpenSpec spec-driven 工作流。

请遵循以下步骤：

1. 创建变更目录
   - 运行：openspec new change context-dynamic-selection-enhancement
   
2. 生成规划文档
   - 读取 openspec/changes/context-dynamic-selection-enhancement/proposal.md
   - 按照 proposal 中的 Capabilities 创建 specs
   - 创建设计文档 design.md
   - 创建任务清单 tasks.md

3. 实现功能
   - 从 tasks.md 的第一个任务开始
   - 每完成一个任务就标记为 [x]
   - 参考 specs 确保符合要求

4. 验证和归档
   - 检查所有任务完成
   - 运行测试确保通过
   - 归档到 openspec/changes/archive/
```

---

## 📝 项目配置（强烈推荐）

创建 `openspec/config.yaml` 注入项目上下文：

```yaml
# openspec/config.yaml
schema: spec-driven

context: |
  ## PicoClaw 项目上下文
  Tech stack: Go 1.25.7
  Project type: Ultra-lightweight AI assistant gateway
  Architecture: Event-driven, message bus pattern
  Testing: go test with testify/assert and testify/require
  Code style: Go standard formatting, Godoc comments required
  Concurrency: Use sync.RWMutex for shared state protection
  
  ## 关键组件
  - AgentInstance: Agent 实例管理
  - ContextBuilder: System prompt 构建（带缓存机制）
  - ToolRegistry: 工具注册和执行（支持可见性过滤）
  - SkillsLoader: 技能加载（workspace > global > builtin）
  - SessionManager: 会话管理（按 channel + chatID 隔离）
  
  ## API 约定
  - 公开方法：首字母大写，线程安全
  - 错误处理：返回 error，使用 errors.Wrap
  - 日志：使用 logger.DebugCF/InfoCF/ErrorCF
  - 配置：从 config.json 读取，支持热重载

rules:
  proposal:
    - Must include backward compatibility analysis
    - Identify affected packages and APIs
    - Include performance impact assessment
    
  specs:
    - Use WHEN/THEN format for scenarios
    - Each requirement must have at least one scenario
    - Include concurrency requirements if applicable
    - Specify thread-safety guarantees
    
  design:
    - Explain mutex usage and lock granularity
    - Document cache invalidation strategies
    - Include rollback plan
    - Address backward compatibility
    
  tasks:
    - Tasks must be small enough to complete in one session
    - Order by dependency (what must be done first)
    - Include test writing tasks
    - Mark breaking changes clearly
```

---

## 🎯 让 AI 遵循 Spec 驱动的技巧

### 技巧 1: 在对话开始时设定上下文

```text
在这个对话中，我们将严格遵循 OpenSpec spec-driven 工作流。

当前变更：context-dynamic-selection-enhancement
相关文档：
- proposal.md: Why, What, Capabilities
- specs/: 详细需求和场景
- design.md: 技术决策和权衡
- tasks.md: 实现任务清单

规则：
1. 始终先阅读相关文档再开始编码
2. 每个任务完成后更新 tasks.md
3. 实现必须符合 specs 中的场景
4. 设计决策必须与 design.md 一致
5. 发现文档问题时先更新文档再改代码

现在开始实现 Task 2.1...
```

### 技巧 2: 使用明确的检查点

```text
在继续之前，让我们确认：

✅ 已读取 proposal.md，理解为什么要做这个改动
✅ 已读取 specs/tool-visibility-filters/spec.md，了解需求
✅ 已读取 design.md，理解技术决策
✅ 已读取 tasks.md，知道当前任务是 2.1

现在开始实现...
```

### 技巧 3: 要求 AI 自我验证

```text
完成每个任务后，请：

1. 列出你修改的文件
2. 说明如何验证功能正确
3. 指出是否影响向后兼容性
4. 确认是否符合 specs 中的场景
5. 标记 tasks.md 为完成状态
```

---

## 🔧 故障排除

### 问题 1: Slash commands 不工作

**症状**: 输入 `/opsx:new` 没有反应

**解决方案**:

```bash
# 1. 检查命令文件位置
ls -la ~/.qoder/commands/opsx/

# 2. 重新生成命令
openspec update

# 3. 重启 Qoder
# 关闭并重新打开 Qoder 窗口
```

### 问题 2: AI 不遵循 Spec

**症状**: AI 直接开始写代码，不看文档

**解决方案**:

在对话中明确指示：

```text
暂停！我们使用的是 OpenSpec spec-driven 工作流。

在写代码之前，请先：
1. 读取 proposal.md 理解为什么做这个改动
2. 读取 specs/ 了解具体需求
3. 读取 design.md 理解技术决策
4. 读取 tasks.md 知道当前任务

请确认你已经理解了这些文档，然后我们再开始实现。
```

### 问题 3: 文档质量差

**症状**: AI 生成的 proposal/specs/design 很敷衍

**解决方案**:

使用 `openspec instructions` 获取更好的模板：

```bash
# 获取特定 artifact 的指令
openspec instructions --change context-dynamic-selection-enhancement proposal
openspec instructions --change context-dynamic-selection-enhancement specs
openspec instructions --change context-dynamic-selection-enhancement design
```

或者在项目中添加更详细的 `config.yaml`。

---

## 📊 最佳实践

### ✅ 应该做的

1. **总是从 `/opsx:new` 开始重要功能**
   - 确保有完整的规划文档
   - 便于后续维护和回顾

2. **使用 `/opsx:ff` 生成全面的规划文档**
   - 不要跳过规划阶段
   - 花 10 分钟规划可以节省 1 小时编码时间

3. **实现时参考 specs**
   - 确保符合需求规格
   - 每个 scenario 都是一个测试用例

4. **完成任务后立即更新 tasks.md**
   - 保持进度准确
   - 便于追踪和统计

5. **归档前运行 `/opsx:verify`**
   - 确保质量达标
   - 避免遗漏重要文档

### ❌ 不应该做的

1. **不要跳过规划阶段**
   - 这违背了 Spec 驱动的初衷

2. **不要直接开始编码**
   - 即使需求看起来很清晰
   - 先写文档再编码

3. **不要忽略 specs 中的场景**
   - 每个场景都必须实现
   - 这是验收标准

4. **不要修改 tasks.md 的结构**
   - 解析依赖于固定格式
   - 使用 `- [ ]` 复选框格式

5. **不要归档不完整的 changes**
   - 确保所有任务完成
   - 确保测试通过

---

## 🎓 学习资源

### 官方文档优先级

1. **[getting-started.md](file:///Users/pengweiye/Documents/codes/OpenSpec/docs/getting-started.md)** ⭐⭐⭐⭐⭐
   - 必读！完整的工作流程示例
   - 包含 dark mode 的完整案例

2. **[commands.md](file:///Users/pengweiye/Documents/codes/OpenSpec/docs/commands.md)** ⭐⭐⭐⭐⭐
   - 所有 slash commands 的详细说明
   - 包含使用示例和技巧

3. **[opsx.md](file:///Users/pengweiye/Documents/codes/OpenSpec/docs/opsx.md)** ⭐⭐⭐⭐
   - OPSX 工作流的哲学和设计理念
   - 如何自定义工作流

4. **[workflows.md](file:///Users/pengweiye/Documents/codes/OpenSpec/docs/workflows.md)** ⭐⭐⭐⭐
   - 常见工作流模式
   - 何时使用哪个命令

5. **[concepts.md](file:///Users/pengweiye/Documents/codes/OpenSpec/docs/concepts.md)** ⭐⭐⭐
   - 深入理解概念
   - Schema、Artifact、Dependency 等

### 快速上手路径

```
Day 1: 
- 阅读 getting-started.md (15 分钟)
- 运行 openspec init (5 分钟)
- 尝试 /opsx:new test-change (10 分钟)

Day 2:
- 阅读 commands.md (20 分钟)
- 实践 /opsx:ff 和 /opsx:apply (30 分钟)
- 完成第一个完整的 change

Day 3:
- 阅读 workflows.md (15 分钟)
- 尝试不同的工作流模式
- 创建 openspec/config.yaml (10 分钟)
```

---

## 📞 获取帮助

- **Discord**: https://discord.gg/YctCnvvshC
- **GitHub Issues**: https://github.com/Fission-AI/OpenSpec/issues
- **npm**: https://www.npmjs.com/package/@fission-ai/openspec

---

**祝你 Spec 驱动开发愉快！** 🚀
