# 向 sipeed/picoclaw 提交 PR 指南

## 当前状态

✅ **已完成**：
- 安全修复已提交到分支 `security-and-prs-merge`（修复 #1525、#1527、#1529）
- 前 5 个 PR 已拉取到本地（pr-110, pr-154, pr-159, pr-181, pr-182）

⚠️ **PR 合并冲突**：前 5 个 PR 与 main 存在较大冲突（尤其是 #110 对 http_provider 的重构），自动合并会破坏代码。建议单独提交安全修复 PR。

---

## 步骤 1：Fork 仓库

1. 打开 https://github.com/sipeed/picoclaw
2. 点击右上角 **Fork** 按钮
3. 等待 Fork 完成，得到 `https://github.com/你的用户名/picoclaw`

---

## 步骤 2：添加 Fork 为远程仓库

```powershell
cd d:\AIbiaoshu\picoclaw_zt

# 将 YOUR_GITHUB_USERNAME 替换为你的 GitHub 用户名
git remote add myfork https://github.com/YOUR_GITHUB_USERNAME/picoclaw.git
```

---

## 步骤 3：推送到你的 Fork

```powershell
# 推送 security-and-prs-merge 分支到你的 fork
git push myfork security-and-prs-merge:security-fixes-1525-1527-1529
```

---

## 步骤 4：在 GitHub 创建 Pull Request

1. 打开 `https://github.com/你的用户名/picoclaw`
2. 会看到 "security-fixes-1525-1527-1529 had recent pushes" 的提示
3. 点击 **Compare & pull request**
4. 填写 PR 信息：

**标题**：
```
fix: address security issues #1525, #1527, #1529
```

**描述**：
```markdown
## Summary
Addresses security issues reported in the picoclaw repository.

## Changes

### #1525 - exec.allow_remote default to false
- Changed default from `true` to `false` to restrict exec tool to local context
- Reduces risk of remote shell execution from non-local sessions

### #1527 - Tighten JSONL session store permissions
- Sessions directory: 0755 → 0700 (owner only)
- JSONL and meta files: 0644 → 0600 (owner read/write only)

### #1529 - Refuse public web mode without allowed_cidrs
- Prevents binding to 0.0.0.0 without explicit CIDR restrictions
- Fails at startup with clear error message when public=true and allowed_cidrs is empty

## Files Modified
- pkg/config/defaults.go
- pkg/config/config_test.go
- pkg/memory/jsonl.go
- web/backend/main.go
- web/backend/api/config_test.go
- web/frontend/src/components/config/form-model.ts
- pkg/migrate/sources/openclaw/openclaw_config_test.go
```

5. 点击 **Create pull request**

---

## 关于前 5 个 PR 的合并

前 5 个 PR（按创建时间排序）：
| # | 标题 | 作者 | 状态 |
|---|------|------|------|
| 110 | feat(provider): add gemini google-generative-ai compatibility | fuhao009 | 与 main 有冲突 |
| 154 | feat: add minimax provider support | Danieldd28 | 待合并 |
| 159 | fix: preserve reasoning_content for thinking models | siciyuan404 | 待合并 |
| 181 | Enable Discord message intents | Paraguanads | 待合并 |
| 182 | feat: add slash command support | Tuzfucius | 待合并 |

PR #110 与当前 main 的 `pkg/providers/http_provider.go` 和 `pkg/config/config.go` 存在结构性冲突，需要人工审查后合并。

如需合并其他 PR，可手动执行：
```powershell
git merge pr-154  # 逐个尝试
# 解决冲突后
git add -A
git commit -m "Merge PR 154"
```

---

## 若 git commit 报错 "unknown option trailer"

你的 Git 可能配置了 commit 模板或别名。使用完整路径绕过：
```powershell
& "C:\Program Files\Git\bin\git.exe" commit -m "your message"
```
