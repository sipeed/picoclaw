# OpenSpec 全局测试约束配置指南

## 📋 概述

本指南说明如何将 PicoClaw 项目的测试约束规则应用到**所有新的 OpenSpec 项目**。

## 🎯 已完成的全局配置

### 1. 全局规则存储位置

```bash
~/.config/openspec/
├── config.json                      # OpenSpec CLI 全局配置
├── default-config-template.yaml     # 默认项目配置模板（包含测试约束）
├── rules-index.json                 # 自动应用的规则索引
├── apply-global-rules.sh            # 自动应用脚本
├── GLOBAL_RULES.md                  # 全局规则说明文档
├── rules/                           # 规则文件目录
│   ├── testing-mandatory.md        # 强制测试要求
│   └── README.md                    # 规则使用说明
└── templates/                       # 项目模板目录
    └── picoclaw/                    # PicoClaw 专用模板
        └── config.yaml             # 预配置的测试约束
```

### 2. 核心测试约束内容

已复制的全局规则包括：

**`testing-mandatory.md`** - 强制测试要求：
- ✅ No feature is complete without tests
- ✅ Test-driven development preferred
- ✅ Coverage requirements (>80% line, 100% critical paths)
- ✅ Deterministic, independent, fast tests
- ✅ All tests must pass before committing

**`default-config-template.yaml`** - 项目配置模板：
```yaml
rules:
  tasks:
    - CRITICAL: Every feature task MUST be followed by test task
    - CRITICAL: Implementation NOT complete until tests passing
  verification:
    - MANDATORY: Run /opsx:verify before /opsx:archive
    - All tests MUST pass before archiving
```

---

## 🚀 使用方法

### 方法 A: 手动应用（推荐新手）

每次在新项目中运行 `openspec init` 后：

```bash
# 1. 初始化新项目
cd /path/to/new-project
openspec init --tools qoder

# 2. 应用全局规则
~/.config/openspec/apply-global-rules.sh .

# 3. 验证
ls .qoder/rules/              # 应看到 testing-mandatory.md
cat openspec/config.yaml      # 应看到测试约束规则
```

### 方法 B: 使用别名（自动化）

已将以下配置添加到 `~/.zshrc`：

```bash
# 自动应用全局规则的函数
openspec-init-with-rules() {
    if [ -z "$1" ]; then
        echo "Usage: openspec-init-with-rules <project-path>"
        return 1
    fi
    
    cd "$1" || return 1
    openspec init --tools qoder
    ~/.config/openspec/apply-global-rules.sh "$PWD"
}

alias openspec-init="openspec-init-with-rules"
```

**使用方式：**

```bash
# 简单用法
openspec-init /path/to/new-project

# 或者在当前目录
mkdir my-new-project && cd my-new-project
openspec-init .
```

### 方法 C: 使用模板（特定项目类型）

如果你想为不同类型的项目使用不同的配置：

```bash
# 1. 创建项目类型模板
mkdir -p ~/.config/openspec/templates/frontend
cp /path/to/frontend-config.yaml ~/.config/openspec/templates/frontend/config.yaml

# 2. 初始化时使用模板
openspec init --tools qoder
cp ~/.config/openspec/templates/frontend/config.yaml ./openspec/config.yaml

# 3. 应用全局规则
~/.config/openspec/apply-global-rules.sh .
```

---

## 📁 文件结构说明

### 全局配置文件

#### `~/.config/openspec/default-config-template.yaml`

这是 PicoClaw 项目的配置副本，包含：
- 测试强制约束规则
- PicoClaw 特定的项目上下文
- 开发和测试规范

#### `~/.config/openspec/rules/testing-mandatory.md`

详细的测试要求和流程：
- Testing Requirements (MANDATORY)
- Implementation Workflow
- Code Review Checklist
- Test Quality Standards

#### `~/.config/openspec/rules-index.json`

定义哪些规则应该自动应用：

```json
{
  "autoIncludeRules": ["testing-mandatory"],
  "rulesLocation": "~/.config/openspec/rules/",
  "description": "Global rules automatically included in all new projects"
}
```

### 项目级文件

应用全局规则后，每个新项目会包含：

```
my-project/
├── .qoder/
│   └── rules/
│       └── testing-mandatory.md    ← 从全局复制
├── openspec/
│   └── config.yaml                 ← 从模板复制
└── ...
```

---

## ⚙️ 自定义配置

### 添加新的全局规则

1. **创建规则文件**

```bash
cat > ~/.config/openspec/rules/code-quality.md << 'EOF'
# Code Quality Rules

## Naming Conventions
- Use camelCase for variables
- Use PascalCase for types and classes
- Use kebab-case for file names

## Documentation
- All public APIs must have Godoc comments
- Complex logic must have inline comments
EOF
```

2. **更新规则索引**

```bash
# 编辑 rules-index.json，添加新规则到 autoIncludeRules
vim ~/.config/openspec/rules-index.json
```

3. **验证**

```bash
# 在新项目中应用
openspec-init /tmp/test-project
ls /tmp/test-project/.qoder/rules/
# 应看到 code-quality.md
```

### 修改现有规则

直接编辑规则文件：

```bash
vim ~/.config/openspec/rules/testing-mandatory.md
```

**注意：** 修改后，需要在现有项目中手动更新：

```bash
# 在已有项目中
~/.config/openspec/apply-global-rules.sh .
```

### 创建项目特定覆盖

如果某个项目需要特殊配置：

```bash
# 项目根目录
cat > openspec/config.local.yaml << 'EOF'
# Project-specific overrides
rules:
  tasks:
    # Override global rule
    - Custom rule for this project only
EOF
```

---

## 🔍 验证和故障排查

### 验证全局规则已应用

```bash
# 检查规则文件
ls -la .qoder/rules/

# 检查配置
cat openspec/config.yaml | grep -A 5 "rules:"

# 检查 AI 是否理解规则
/opsx:propose "Add new feature"
# AI 应该在 proposal 中包含测试需求
```

### 常见问题

**Q: 规则文件没有复制到 `.qoder/rules/`**

A: 检查以下几点：
```bash
# 1. 全局规则是否存在
ls ~/.config/openspec/rules/*.md

# 2. 手动复制
cp ~/.config/openspec/rules/*.md .qoder/rules/

# 3. 检查权限
chmod 644 .qoder/rules/*.md
```

**Q: 配置没有被应用**

A: 确保 YAML 语法正确：
```bash
# 验证 YAML 语法
python3 -c "import yaml; yaml.safe_load(open('openspec/config.yaml'))"

# 或重新复制模板
cp ~/.config/openspec/default-config-template.yaml openspec/config.yaml
```

**Q: AI 不遵守测试约束**

A: 尝试以下方法：
1. 在对话中明确提醒："Remember to follow the testing rules in .qoder/rules/"
2. 重新运行 `/opsx:apply` 让 AI 重新读取规则
3. 检查 `.qoder/rules/testing-mandatory.md` 内容是否清晰

---

## 📊 工作流程示例

### 完整的新项目初始化

```bash
# 1. 创建项目目录
mkdir my-awesome-project && cd my-awesome-project

# 2. 使用别名初始化（自动应用全局规则）
openspec-init .

# 输出示例：
# ✔ Setup complete for Qoder
# 🔧 Applying OpenSpec global rules to /path/to/project...
# 📋 Copying global rules...
# ✅ Copied 1 rule files
# ✨ Global rules applied successfully!

# 3. 开始第一个变更
/opsx:propose "Add user authentication"

# 4. AI 会自动遵循测试约束：
# - Proposal 中包含测试需求分析
# - Specs 中每个场景对应测试用例
# - Design 中有测试策略
# - Tasks 中每个功能后有测试任务

# 5. 实现过程中
/opsx:apply add-user-authentication
# AI 会：实现功能 → 立即写测试 → 跑测试 → 标记完成

# 6. 完成后验证
/opsx:verify add-user-authentication
# 检查测试覆盖率和通过率

# 7. 归档
/opsx:archive add-user-authentication
```

---

## 🎓 最佳实践

### ✅ 推荐做法

1. **始终使用全局规则** - 保证所有项目的一致性
2. **定期更新规则** - 根据项目经验优化约束
3. **分享规则改进** - 将有效的规则贡献回团队
4. **结合 CI/CD** - 在流水线中强制执行相同规则

### ❌ 避免的做法

1. **不要跳过规则应用** - 即使项目很小
2. **不要随意降低标准** - 测试覆盖率不能妥协
3. **不要在多个项目中使用冲突的规则** - 保持一致性

---

## 📚 相关资源

- [OpenSpec 官方文档](https://github.com/Fission-AI/OpenSpec/tree/main/docs)
- [项目配置指南](https://github.com/Fission-AI/OpenSpec/blob/main/docs/customization.md)
- [Slash Commands 参考](https://github.com/Fission-AI/OpenSpec/blob/main/docs/commands.md)
- PicoClaw 项目配置示例：`/Users/pengweiye/Documents/codes/picoclaw/openspec/config.yaml`

---

## 🤝 贡献规则

如果你有好的规则想法，可以：

1. 在团队内分享 `~/.config/openspec/` 配置
2. 提交 PR 到团队的规则模板仓库
3. 记录在团队 Wiki 中的开发规范章节

---

**最后更新**: 2026-02-27  
**维护者**: PicoClaw Team
