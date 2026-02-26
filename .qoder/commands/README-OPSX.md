---
description: 
---
# OpenSpec Slash Commands 配置指南

## 📁 已创建的文件

所有 OpenSpec slash 命令配置文件已创建在 `~/.qoder/commands/` 目录下：

```
~/.qoder/commands/
├── opsx.md              # 主入口 - OpenSpec 总览和快速开始
├── opsx-new.md          # 创建新的 change proposal
├── opsx-ff.md           # Fast-forward 生成所有规划文档
├── opsx-apply.md        # 应用 tasks.md 开始实现
├── opsx-list.md         # 列出所有 active changes
├── opsx-validate.md     # 验证 change 完整性
├── opsx-archive.md      # 归档完成的 change
└── opsx-show.md         # 显示 change 详细信息
```

## 🚀 使用方法

### 1. 在 Qoder 中使用

在你的 Qoder 对话中，直接使用 slash commands：

```
/opsx:new context-dynamic-selection-enhancement
/opsx:ff
/opsx:show context-dynamic-selection-enhancement
/opsx:validate
/opsx:apply
```

### 2. 命令说明

#### **核心工作流命令**

| 命令 | 功能 | 示例 |
|------|------|------|
| `/opsx:new <name>` | 创建新的 change | `/opsx:new feature-auth` |
| `/opsx:ff` | 生成所有规划文档 | `/opsx:ff` |
| `/opsx:apply [name]` | 开始实现任务 | `/opsx:apply feature-auth` |
| `/opsx:archive <name>` | 归档完成的 change | `/opsx:archive feature-auth` |

#### **管理命令**

| 命令 | 功能 | 示例 |
|------|------|------|
| `/opsx:list` | 列出所有 changes | `/opsx:list` |
| `/opsx:show <name>` | 显示详情 | `/opsx:show feature-auth` |
| `/opsx:validate [name]` | 验证完整性 | `/opsx:validate feature-auth` |

## 📋 完整工作流程

```
1. /opsx:new <feature-name>
   ↓ 创建 openspec/changes/<feature-name>/ 目录
   
2. /opsx:ff
   ↓ 自动生成 proposal.md, specs/, design.md, tasks.md
   
3. /opsx:show <feature-name>
   ↓ 审查生成的文档
   
4. /opsx:validate
   ↓ 验证文档质量和完整性
   
5. /opsx:apply
   ↓ 按照 tasks.md 逐项实现功能
   
6. /opsx:archive
   ↓ 完成后归档，合并 specs 到主分支
```

## 🎯 每个命令的详细说明

### `/opsx:new` - 创建 Change

**位置**: `~/.qoder/commands/opsx-new.md`

**功能**:
- 创建 `openspec/changes/<change-name>/` 目录
- 初始化 `.openspec.yaml` 元数据文件
- 设置 spec-driven 工作流 schema

**示例**:
```
/opsx:new context-dynamic-selection-enhancement
```

**输出**:
```
✔ Created change 'context-dynamic-selection-enhancement' at openspec/changes/context-dynamic-selection-enhancement/ (schema: spec-driven)
```

---

### `/opsx:ff` - Fast-Forward

**位置**: `~/.qoder/commands/opsx-ff.md`

**功能**:
- 自动生成 proposal.md（Why, What, Capabilities）
- 自动生成 specs/*.md（详细规格说明）
- 自动生成 design.md（技术设计决策）
- 自动生成 tasks.md（实现任务清单）

**示例**:
```
/opsx:ff
```

**生成的结构**:
```
openspec/changes/<name>/
├── proposal.md      ← 自动生成
├── specs/
│   ├── capability-1/spec.md  ← 自动生成
│   └── capability-2/spec.md  ← 自动生成
├── design.md        ← 自动生成
└── tasks.md         ← 自动生成
```

---

### `/opsx:apply` - 应用 Tasks

**位置**: `~/.qoder/commands/opsx-apply.md`

**功能**:
- 读取 tasks.md 文件
- 逐项指导实现
- 更新复选框进度
- 引用 specs 和 design 作为上下文

**示例**:
```
/opsx:apply context-dynamic-selection-enhancement
```

**实现流程**:
1. 读取上下文（proposal, design, specs）
2. 解析未完成的 tasks
3. 从 Task 1.1 开始实现
4. 每完成一项标记为 `- [x]`
5. 继续下一项

---

### `/opsx:list` - 列出 Changes

**位置**: `~/.qoder/commands/opsx-list.md`

**功能**:
- 列出 `openspec/changes/` 下所有目录
- 显示任务完成状态
- 按最后修改时间排序

**示例**:
```
/opsx:list
```

**输出**:
```
Changes:
  context-dynamic-selection-enhancement     23/47 tasks    2 hours ago
  api-rate-limiting                         0/32 tasks     1 day ago
  user-auth-v2                              Complete       1 week ago
```

---

### `/opsx:validate` - 验证

**位置**: `~/.qoder/commands/opsx-validate.md`

**功能**:
- 检查必需 artifacts（proposal, specs, design, tasks）
- 验证 artifact 结构和内容
- 验证任务完成状态
- 报告缺失或不完整的项目

**示例**:
```
/opsx:validate context-dynamic-selection-enhancement
```

**成功输出**:
```
✓ change/context-dynamic-selection-enhancement
  ✓ proposal.md (complete)
  ✓ specs/ (4 capabilities)
  ✓ design.md (complete)
  ✓ tasks.md (23/47 tasks complete)
Totals: 1 passed (1 items)
```

---

### `/opsx:archive` - 归档

**位置**: `~/.qoder/commands/opsx-archive.md`

**功能**:
- 验证所有任务完成
- 移动到 `openspec/changes/archive/`
- 合并 specs 到 `openspec/specs/`
- 保留历史记录

**示例**:
```
/opsx:archive context-dynamic-selection-enhancement
```

**归档后结构**:
```
openspec/changes/
├── active-change-1/          # 仍在进行
└── archive/                  # 已完成的
    └── 2026-02-26-context-dynamic-selection-enhancement/
        ├── proposal.md
        ├── design.md
        ├── specs/
        └── tasks.md (全部勾选)
```

---

### `/opsx:show` - 显示详情

**位置**: `~/.qoder/commands/opsx-show.md`

**功能**:
- 显示完整的 change artifacts
- 可以显示单个 artifact 或整个 change
- 格式化 markdown 输出

**示例**:
```
# 显示整个 change
/opsx:show context-dynamic-selection-enhancement

# 显示特定 artifact
/opsx:show context-dynamic-selection-enhancement/proposal
/opsx:show context-dynamic-selection-enhancement/design
/opsx:show context-dynamic-selection-enhancement/tasks

# 显示 spec
/opsx:show context-dynamic-selection-enhancement/specs/skills-filter-api
```

---

## 🛡️ 最佳实践

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

5. **归档前运行 `/opsx:validate`**
   - 确保质量达标
   - 避免遗漏重要文档

### ❌ 不应该做的

1. **不要跳过规划阶段**
   - 这违背了 Spec 驱动的初衷

2. **不要修改 tasks.md 的结构**
   - 解析依赖于固定格式
   - 使用 `- [ ]` 复选框格式

3. **不要归档不完整的 changes**
   - 确保所有任务完成
   - 确保测试通过

4. **不要忽略验证错误**
   - 及时修复结构问题
   - 保证文档质量

## 📊 当前项目状态

你的 PicoClaw 项目已经有：

✅ **OpenSpec CLI 已安装**: v1.2.0  
✅ **Slash Commands 已配置**: 8 个命令文件  
✅ **第一个 Change 已创建**: `context-dynamic-selection-enhancement`  
✅ **完整文档已生成**: proposal, 4 specs, design, tasks  

**下一步**: 
```bash
/opsx:apply context-dynamic-selection-enhancement
```

开始实现第一个任务：**Task 1.1 - Modify AgentInstance to add skillsFilterMutex**

## 🔧 故障排除

### 命令不工作？

1. **检查文件权限**:
   ```bash
   ls -lh ~/.qoder/commands/opsx*.md
   ```

2. **重启 Qoder**:
   - 关闭并重新打开 Qoder
   - 确保加载了新的 commands

3. **验证语法**:
   - 确保 frontmatter 正确（`---` 包裹）
   - 使用正确的 markdown 格式

### 找不到 Change？

```bash
# 列出所有 changes
/opsx:list

# 查看具体 change
/opsx:show <change-name>

# 验证 change
/opsx:validate --changes <name>
```

## 📚 更多资源

- **OpenSpec 官方文档**: `openspec --help`
- **GitHub**: https://github.com/Fission-AI/OpenSpec
- **npm**: https://www.npmjs.com/package/@fission-ai/openspec
- **Discord**: https://discord.gg/YctCnvvshC

---

**祝你 Spec 驱动开发愉快！** 🚀
