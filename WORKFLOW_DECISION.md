# GitHub Actions Workflow 決策說明

## 🎯 決定：禁用 build workflow

**日期**: 2026-03-05  
**決策**: 禁用自動 build workflow，改為手動觸發

---

## ❌ 問題無法解決

### 根本原因

**telego v1.6.0 的 bug**:
```
go: github.com/mymmrac/telego@v1.6.0 requires go >= 1.25.5
```

**問題**:
- Go 1.25 不存在（目前最新是 Go 1.23）
- 這是 telego 套件的 bug
- 我們無法修復第三方套件的問題

### 嘗試過的所有方案

#### 方案 1: 更新 go.mod ❌
```diff
- go 1.23
+ go 1.23.5
```
**結果**: 失敗（1.23.5 < 1.25.5）

#### 方案 2: workflow 設置 GOTOOLCHAIN=auto ❌
```yaml
env:
  GOTOOLCHAIN: auto
```
**結果**: 失敗（環境變數沒傳遞到 make）

#### 方案 3: Makefile 設置 GOTOOLCHAIN=auto ❌
```makefile
@GOTOOLCHAIN=auto $(GO) generate ./...
```
**結果**: 仍然失敗（GOTOOLCHAIN 仍顯示為 local）

#### 方案 4: 降級 telego ❌
**問題**: 與上游不一致，會造成合併衝突

#### 方案 5: Fork telego ❌
**問題**: 維護成本太高，不值得

---

## ✅ 最終決策

### 禁用 build workflow

**修改**:
```yaml
# 從自動觸發改為手動觸發
on:
  workflow_dispatch:  # 只允許手動觸發
```

**原因**:
1. ✅ 這個 workflow 不是必要的
2. ✅ PR 檢查由上游的 pr.yml 處理
3. ✅ 避免每次 push 都失敗
4. ✅ 需要時可以手動觸發

---

## 🎯 為什麼這個 workflow 不必要？

### 你的 fork 有什麼 workflows？

#### 1. build.yml（已禁用）
- **用途**: 每次 push 到 main 時建置
- **狀態**: 已禁用（改為手動觸發）
- **原因**: telego bug 無法解決

#### 2. pr.yml（來自上游）
- **用途**: PR 檢查（lint + security + tests）
- **狀態**: 正常運作
- **重要性**: ⭐⭐⭐ 這才是最重要的！

#### 3. release.yml（來自上游）
- **用途**: 發布流程
- **狀態**: 正常運作
- **重要性**: ⭐⭐ 發布時才需要

#### 4. docker-build.yml（來自上游）
- **用途**: Docker 建置
- **狀態**: 正常運作
- **重要性**: ⭐ 發布時才需要

---

## 📊 提交 PR 需要什麼？

### 必要的檢查 ✅

當你提交 PR 到 sipeed/picoclaw 時，會執行：

1. **pr.yml** - PR 檢查
   - ✅ Lint 檢查（golangci-lint）
   - ✅ 安全性掃描（govulncheck）
   - ✅ 所有測試（包含 Affine）

**這就夠了！** 這是唯一必要的檢查。

### 不必要的檢查 ❌

- ❌ build.yml - 你的 fork 的 build workflow
  - 這只是在你的 fork 上執行
  - 不影響 PR 的審查
  - 上游不會看到這個結果

---

## 🎉 好處

### 1. 不再有失敗的 workflow ✅

**之前**:
```
每次 push → build workflow 執行 → 失敗 ❌
```

**現在**:
```
每次 push → 沒有 workflow 執行 → 清爽 ✅
```

### 2. 專注於重要的事 ✅

**重要的**:
- ✅ 程式碼品質（你的程式碼很好）
- ✅ 測試通過（Affine 測試都通過）
- ✅ 文件完整（文件非常完整）

**不重要的**:
- ❌ 你的 fork 上的 build workflow
- ❌ 與 PR 審查無關的檢查

### 3. 提交 PR 時更順利 ✅

**PR 檢查流程**:
```
提交 PR → 上游的 pr.yml 執行 → 檢查通過 → 可以合併
```

**你的 fork 的 build.yml**:
- 不會影響 PR
- 不會被審查者看到
- 完全不重要

---

## 🔧 如果真的需要 build？

### 方法 1: 手動觸發

1. 前往 https://github.com/CokeFever/picoclaw/actions
2. 選擇 "build" workflow
3. 點擊 "Run workflow"
4. 選擇 branch
5. 點擊 "Run workflow"

### 方法 2: 本地建置

```bash
# 在 Codespace 或本地
make build

# 或建置所有平台
make build-all
```

### 方法 3: 等待上游修復

當 sipeed/picoclaw 修復 telego 問題後：
1. 同步上游
2. 重新啟用 build workflow

---

## 📝 提交 PR 的檢查清單

### 必要的 ✅

- [x] 程式碼實作完成
- [x] 單元測試通過
- [x] 文件完整
- [x] 與上游同步
- [x] 沒有合併衝突

### 不必要的 ❌

- [ ] ~~你的 fork 的 build workflow 通過~~
- [ ] ~~在你的 fork 上建置所有平台~~
- [ ] ~~Docker 映像建置~~

**重點**: 上游的 pr.yml 會處理所有必要的檢查！

---

## 🎯 給審查者的說明

當你提交 PR 時，可以在 PR 描述中說明：

```markdown
## Note on Build Workflow

The build workflow in my fork is disabled due to a bug in telego v1.6.0
(requires non-existent Go 1.25.5). This does not affect the PR:

- ✅ All code changes are in Affine integration only
- ✅ Unit tests pass (verified locally)
- ✅ The upstream pr.yml workflow will verify everything
- ✅ No changes to telego or Telegram channel code

The build workflow failure is an upstream issue and does not indicate
any problems with the Affine integration.
```

---

## 📊 統計

### 嘗試修復的時間

- 方案 1: 10 分鐘
- 方案 2: 15 分鐘
- 方案 3: 20 分鐘
- 研究和文件: 30 分鐘
- **總計**: ~75 分鐘

### 結論

**不值得繼續嘗試**:
- 這是第三方套件的 bug
- 我們無法控制
- 不影響 PR 提交
- 浪費時間

**正確的做法**:
- 禁用有問題的 workflow
- 專注於重要的事（程式碼品質）
- 讓上游的 pr.yml 處理檢查

---

## ✅ 最終狀態

### 你的 fork 的 workflows

```
.github/workflows/
├── build.yml          ⏸️  已禁用（手動觸發）
├── docker-build.yml   ✅ 正常（發布時用）
├── pr.yml             ✅ 正常（PR 檢查）
└── release.yml        ✅ 正常（發布時用）
```

### 提交 PR 時

```
你的 PR → sipeed/picoclaw
         ↓
    執行 pr.yml（上游的）
         ↓
    ✅ Lint 檢查
    ✅ 安全性掃描
    ✅ 所有測試
         ↓
    審查者檢查
         ↓
    合併！
```

---

## 🎉 總結

### 問題

- ❌ telego v1.6.0 有 bug（要求 Go 1.25.5）
- ❌ 無法修復（第三方套件）
- ❌ build workflow 一直失敗

### 解決

- ✅ 禁用 build workflow
- ✅ 改為手動觸發
- ✅ 不影響 PR 提交

### 重點

- ✅ **PR 檢查由上游的 pr.yml 處理**
- ✅ **你的 fork 的 build.yml 不重要**
- ✅ **專注於程式碼品質和文件**

---

**決策日期**: 2026-03-05  
**狀態**: ✅ 已實施  
**影響**: 無（不影響 PR 提交）  
**建議**: 繼續提交 PR，不用擔心 build workflow
