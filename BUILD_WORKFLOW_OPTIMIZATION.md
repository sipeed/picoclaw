# Build Workflow 優化說明

## ✅ 已完成優化

**日期**: 2026-03-05

---

## 🎯 問題

**之前的行為**:
- 每次 push 到 main 分支都會執行 build workflow
- 即使只是更新文件（.md 檔案）也會觸發建置
- 浪費 GitHub Actions 的執行時間和資源

**範例**:
```bash
# 這些變更也會觸發 build（不必要）
git commit -m "Update README.md"
git commit -m "Add documentation"
git commit -m "Fix typo in CONTRIBUTING.md"
```

---

## ✅ 解決方案

### 新增 paths 過濾器

現在 build workflow 只在以下檔案變更時才執行：

```yaml
on:
  push:
    branches: [ "main" ]
    paths:
      - '**.go'                          # 所有 Go 檔案
      - 'go.mod'                         # Go 模組定義
      - 'go.sum'                         # Go 依賴鎖定
      - 'Makefile'                       # 建置腳本
      - '.github/workflows/build.yml'    # Workflow 本身
      - 'cmd/**'                         # 命令列程式
      - 'pkg/**'                         # 套件程式碼
      - 'workspace/**'                   # 工作區檔案
```

---

## 📊 觸發條件對比

### 之前（會觸發 build）❌

```bash
# 文件變更
git add README.md
git commit -m "Update docs"
# ❌ 會觸發 build（不必要）

# 新增說明文件
git add GUIDE.md
git commit -m "Add guide"
# ❌ 會觸發 build（不必要）

# 修改設定範例
git add config/config.example.json
git commit -m "Update config example"
# ❌ 會觸發 build（不必要）
```

### 現在（會觸發 build）✅

```bash
# Go 程式碼變更
git add pkg/tools/affine_simple.go
git commit -m "Fix Affine tool"
# ✅ 會觸發 build（必要）

# 依賴更新
git add go.mod go.sum
git commit -m "Update dependencies"
# ✅ 會觸發 build（必要）

# Makefile 變更
git add Makefile
git commit -m "Update build script"
# ✅ 會觸發 build（必要）
```

### 現在（不會觸發 build）✅

```bash
# 文件變更
git add README.md
git commit -m "Update docs"
# ✅ 不會觸發 build（節省資源）

# 新增說明文件
git add GUIDE.md
git commit -m "Add guide"
# ✅ 不會觸發 build（節省資源）

# 修改設定範例
git add config/config.example.json
git commit -m "Update config example"
# ✅ 不會觸發 build（節省資源）
```

---

## 🎯 包含的路徑說明

### 1. `**.go` - 所有 Go 檔案
**原因**: 任何 Go 程式碼變更都需要重新建置

**範例**:
- `pkg/tools/affine_simple.go`
- `cmd/picoclaw/main.go`
- `pkg/agent/instance.go`

---

### 2. `go.mod` 和 `go.sum` - Go 依賴
**原因**: 依賴變更可能影響建置

**範例**:
- 新增套件: `go get github.com/example/pkg`
- 更新套件: `go get -u github.com/example/pkg`
- 移除套件: `go mod tidy`

---

### 3. `Makefile` - 建置腳本
**原因**: 建置流程變更需要驗證

**範例**:
- 新增建置目標
- 修改編譯參數
- 更新 LDFLAGS

---

### 4. `.github/workflows/build.yml` - Workflow 本身
**原因**: Workflow 變更需要測試

**範例**:
- 修改建置步驟
- 更新 Go 版本
- 新增建置平台

---

### 5. `cmd/**` - 命令列程式
**原因**: 主程式變更需要重新建置

**範例**:
- `cmd/picoclaw/main.go`
- `cmd/picoclaw/internal/agent/command.go`

---

### 6. `pkg/**` - 套件程式碼
**原因**: 核心邏輯變更需要重新建置

**範例**:
- `pkg/tools/affine_simple.go`
- `pkg/config/config.go`
- `pkg/agent/instance.go`

---

### 7. `workspace/**` - 工作區檔案
**原因**: 這些檔案會被嵌入到二進位檔案中

**範例**:
- `workspace/AGENTS.md`
- `workspace/SOUL.md`
- `workspace/skills/`

---

## 📈 預期效果

### 節省資源

**之前**:
```
10 次 commit = 10 次 build
- 5 次程式碼變更（需要 build）
- 5 次文件變更（不需要 build）
= 10 次 build 執行
```

**之後**:
```
10 次 commit = 5 次 build
- 5 次程式碼變更（需要 build）
- 5 次文件變更（跳過 build）
= 5 次 build 執行（節省 50%）
```

### 更快的反饋

**文件更新**:
- 之前: 等待 5-10 分鐘 build 完成
- 之後: 立即完成（不執行 build）

---

## 🔍 特殊情況

### 情況 1: 同時修改程式碼和文件

```bash
git add pkg/tools/affine_simple.go README.md
git commit -m "Fix bug and update docs"
```

**結果**: ✅ 會觸發 build（因為包含 .go 檔案）

---

### 情況 2: 只修改測試檔案

```bash
git add pkg/tools/affine_simple_test.go
git commit -m "Add more tests"
```

**結果**: ✅ 會觸發 build（測試檔案也是 .go 檔案）

---

### 情況 3: 修改設定範例

```bash
git add config/config.example.json
git commit -m "Update config example"
```

**結果**: ✅ 不會觸發 build（設定範例不影響建置）

---

### 情況 4: 修改 GitHub Actions 其他 workflow

```bash
git add .github/workflows/pr.yml
git commit -m "Update PR workflow"
```

**結果**: ✅ 不會觸發 build（只有 build.yml 變更才觸發）

---

## ⚠️ 注意事項

### 1. PR 仍然會執行完整檢查

**重要**: 這個優化只影響 main 分支的 build workflow

**PR workflow (pr.yml)** 仍然會執行:
- Lint 檢查
- 安全性掃描
- 所有測試

**原因**: PR 需要完整驗證，確保程式碼品質

---

### 2. 如果需要強制執行 build

**方法 1**: 修改任何 Go 檔案
```bash
# 觸碰一個 Go 檔案
touch pkg/tools/affine_simple.go
git add pkg/tools/affine_simple.go
git commit -m "Trigger build"
```

**方法 2**: 在 GitHub 上手動觸發
- 前往 Actions 頁面
- 選擇 build workflow
- 點擊 "Run workflow"

---

### 3. 如果需要新增其他觸發路徑

編輯 `.github/workflows/build.yml`:

```yaml
paths:
  - '**.go'
  - 'go.mod'
  - 'go.sum'
  - 'Makefile'
  - '.github/workflows/build.yml'
  - 'cmd/**'
  - 'pkg/**'
  - 'workspace/**'
  - 'your/new/path/**'  # 新增這一行
```

---

## 📊 統計資訊

### 你的最近 10 次提交

```
1. Add sync completion report              → 文件（不觸發）
2. Update submission guide                 → 文件（不觸發）
3. Merge upstream/main                     → 程式碼（觸發）
4. Fix Affine tests                        → 程式碼（觸發）
5. Add final status summary                → 文件（不觸發）
6. Update Affine documentation             → 文件（不觸發）
7. Improve error handling                  → 程式碼（觸發）
8. Add testing guide                       → 文件（不觸發）
9. Add test scripts                        → 腳本（不觸發）
10. Add Affine features                    → 程式碼（觸發）
```

**結果**: 10 次提交，只有 4 次需要 build（節省 60%）

---

## ✅ 檢查清單

- [x] 新增 paths 過濾器
- [x] 包含所有必要的程式碼路徑
- [x] 排除文件變更
- [x] 測試驗證
- [x] 提交變更
- [x] 推送到 GitHub
- [x] 創建說明文件

---

## 🎉 總結

### 優化效果

1. ✅ **節省資源** - 減少 50-60% 的 build 執行
2. ✅ **更快反饋** - 文件更新立即完成
3. ✅ **保持品質** - PR 仍然執行完整檢查
4. ✅ **靈活控制** - 可以手動觸發 build

### 適用場景

- ✅ 更新文件（README, GUIDE, etc.）
- ✅ 新增說明檔案
- ✅ 修改設定範例
- ✅ 更新 .gitignore
- ✅ 修改其他 workflows

### 仍會觸發 build

- ✅ 修改 Go 程式碼
- ✅ 更新依賴
- ✅ 修改建置腳本
- ✅ 變更工作區檔案

---

**優化完成日期**: 2026-03-05  
**狀態**: ✅ 生效中  
**預期節省**: 50-60% 的 build 執行次數
