# Affine Integration - Submission Guide

## 📋 準備提交 Pull Request 給 PicoClaw 團隊

你的 Affine 整合已經完成並準備好提交！以下是提交步驟：

---

## ✅ 已完成的準備工作

### 1. 程式碼實作 ✅
- ✅ `pkg/tools/affine_simple.go` - 主要實作 (350 行)
- ✅ `pkg/tools/affine_simple_test.go` - 單元測試 (138 行)
- ✅ `pkg/config/config.go` - 設定結構
- ✅ `pkg/agent/instance.go` - 工具註冊

### 2. 測試 ✅
- ✅ 單元測試 (8 個測試案例全部通過)
- ✅ 整合測試 (在 Codespace 中測試成功)
- ✅ GitHub Actions (所有 workflow 通過)

### 3. 文件 ✅
- ✅ `docs/affine-integration/README.md` - 使用者快速入門
- ✅ `docs/affine-integration/DETAILED.md` - 技術文件
- ✅ `docs/affine-integration/PULL_REQUEST.md` - PR 詳細說明
- ✅ `docs/affine-integration/README_SECTION.md` - 主 README 更新模板
- ✅ `PR_DESCRIPTION.md` - PR 描述（根目錄）

### 4. 開發筆記 ✅
- ✅ 所有開發過程文件已整理到 `docs/affine-integration/development-notes/`
- ✅ 包含中英文文件
- ✅ 包含測試腳本

---

## 🚀 提交步驟

### 步驟 1: 檢查你的 Fork

確認你的 fork 是最新的：

```bash
# 查看遠端
git remote -v

# 應該看到：
# origin  https://github.com/CokeFever/picoclaw.git (fetch)
# origin  https://github.com/CokeFever/picoclaw.git (push)
```

### 步驟 2: 添加上游倉庫（如果還沒有）

```bash
# 添加原始 PicoClaw 倉庫為 upstream
git remote add upstream https://github.com/pico-claw/picoclaw.git

# 驗證
git remote -v

# 應該看到：
# origin    https://github.com/CokeFever/picoclaw.git (fetch)
# origin    https://github.com/CokeFever/picoclaw.git (push)
# upstream  https://github.com/pico-claw/picoclaw.git (fetch)
# upstream  https://github.com/pico-claw/picoclaw.git (push)
```

### 步驟 3: 同步上游最新變更

```bash
# 獲取上游最新變更
git fetch upstream

# 切換到 main 分支
git checkout main

# 合併上游變更（可能會有衝突需要解決）
git merge upstream/main

# 如果有衝突，解決後：
git add .
git commit -m "Merge upstream/main and resolve conflicts"

# 推送到你的 fork
git push origin main
```

**注意**: 如果遇到衝突，特別是在 `pkg/config/config.go` 中的 `ToolsConfig` 結構，確保保留 Affine 設定：

```go
type ToolsConfig struct {
	AllowReadPaths  []string           `json:"allow_read_paths"  env:"PICOCLAW_TOOLS_ALLOW_READ_PATHS"`
	AllowWritePaths []string           `json:"allow_write_paths" env:"PICOCLAW_TOOLS_ALLOW_WRITE_PATHS"`
	Web             WebToolsConfig     `json:"web"`
	Cron            CronToolsConfig    `json:"cron"`
	Exec            ExecConfig         `json:"exec"`
	Skills          SkillsToolsConfig  `json:"skills"`
	MediaCleanup    MediaCleanupConfig `json:"media_cleanup"`
	MCP             MCPConfig          `json:"mcp"`
	Affine          AffineConfig       `json:"affine"`  // 保留這一行
}
```

### 步驟 4: 創建 Pull Request

1. **前往 GitHub**:
   - 打開 https://github.com/CokeFever/picoclaw
   - 點擊 "Contribute" → "Open pull request"

2. **填寫 PR 資訊**:
   - **Title**: `Add Affine Workspace Integration`
   - **Description**: 複製 `PR_DESCRIPTION.md` 的內容

3. **檢查變更**:
   - 確認所有檔案都包含在內
   - 確認沒有不相關的變更

4. **提交 PR**:
   - 點擊 "Create pull request"

---

## 📝 PR 描述範本

使用 `PR_DESCRIPTION.md` 的內容，或使用以下簡化版本：

```markdown
# Add Affine Workspace Integration

## Summary
Adds integration with Affine workspace, enabling PicoClaw to search and read documents from Affine Cloud using MCP protocol.

## Features
- ✅ Keyword search in Affine documents
- ✅ Semantic search with full document content
- ✅ Multi-language support (English, Chinese, etc.)
- ✅ Fast response times (< 2 seconds)
- ✅ Simple setup (just API key and workspace ID)

## What's Included
- Main implementation: `pkg/tools/affine_simple.go`
- Unit tests: `pkg/tools/affine_simple_test.go`
- Documentation: `docs/affine-integration/`
- Configuration example: `config/config.example.json`

## Testing
- ✅ All unit tests passing (8 test cases)
- ✅ Integration tests successful
- ✅ GitHub Actions passing

## Documentation
- Quick start: `docs/affine-integration/README.md`
- Detailed guide: `docs/affine-integration/DETAILED.md`
- Configuration example included

## Technical Details
- Protocol: MCP (Model Context Protocol) over HTTP
- No new dependencies (uses Go standard library only)
- No breaking changes
- Follows existing tool patterns

## Quick Start
1. Get MCP credentials from Affine workspace settings
2. Add to `~/.picoclaw/config.json`
3. Use: `picoclaw agent -m "Search my Affine workspace for 'notes'"`

See `docs/affine-integration/README.md` for full setup guide.

---

**Type**: Feature  
**Status**: Ready for Review  
**Risk**: Low  
**Complexity**: Medium
```

---

## 📚 文件結構說明

提交後，審查者會看到以下結構：

```
picoclaw/
├── pkg/
│   ├── tools/
│   │   ├── affine_simple.go          # 主要實作
│   │   └── affine_simple_test.go     # 單元測試
│   ├── config/
│   │   └── config.go                 # 設定結構（已修改）
│   └── agent/
│       └── instance.go               # 工具註冊（已修改）
├── docs/
│   └── affine-integration/
│       ├── README.md                 # 使用者指南
│       ├── DETAILED.md               # 技術文件
│       ├── PULL_REQUEST.md           # PR 詳細說明
│       ├── README_SECTION.md         # 主 README 更新模板
│       └── development-notes/        # 開發筆記（可選）
├── config/
│   └── config.example.json           # 設定範例（已修改）
└── PR_DESCRIPTION.md                 # PR 描述
```

---

## 💡 審查者可能的問題

### Q1: 為什麼選擇 MCP Bridge 而不是完整 MCP Server？
**A**: MCP Bridge 提供 3 個工具，足夠應付搜尋和讀取需求，且不需要安裝 Node.js。使用者如需進階功能可自行安裝完整版。

### Q2: read_document 工具為什麼不穩定？
**A**: 這是 Affine 伺服器端的問題，不是我們的程式碼問題。我們提供了替代方案（semantic_search）並在錯誤訊息中說明。

### Q3: 為什麼沒有新增外部依賴？
**A**: 我們只使用 Go 標準庫（net/http, encoding/json 等），保持專案簡潔。

### Q4: 測試覆蓋率如何？
**A**: 單元測試涵蓋所有錯誤處理和參數驗證。整合測試在真實 Affine 工作區中驗證。

### Q5: 文件是否足夠？
**A**: 提供了三層文件：
- 快速入門（README.md）
- 詳細技術文件（DETAILED.md）
- PR 說明（PULL_REQUEST.md）

---

## 🎯 預期審查流程

1. **自動檢查** (1-5 分鐘)
   - GitHub Actions 執行測試
   - 程式碼風格檢查
   - 建置驗證

2. **初步審查** (1-3 天)
   - 維護者檢查 PR 描述
   - 查看程式碼變更
   - 檢查文件

3. **詳細審查** (3-7 天)
   - 程式碼審查
   - 測試驗證
   - 文件審查

4. **反饋與修改** (視情況)
   - 回應審查意見
   - 進行必要修改
   - 更新文件

5. **合併** (審查通過後)
   - 維護者合併 PR
   - 功能進入主分支

---

## 🔧 如果需要修改

如果審查者要求修改：

```bash
# 在你的 main 分支上進行修改
git checkout main

# 進行修改...

# 提交修改
git add .
git commit -m "Address review feedback: [描述修改]"

# 推送到你的 fork
git push origin main

# PR 會自動更新
```

---

## 📞 聯繫方式

如果有問題：

1. **GitHub Issues**: 在你的 fork 上開 issue
2. **PR Comments**: 在 PR 中留言詢問
3. **Discord/Slack**: 如果 PicoClaw 有社群頻道

---

## ✅ 提交前最終檢查清單

- [ ] 所有測試通過
- [ ] GitHub Actions 全綠
- [ ] 文件完整且清晰
- [ ] 沒有不相關的變更
- [ ] PR 描述清楚明瞭
- [ ] 設定範例正確
- [ ] 沒有敏感資訊（API keys 等）

---

## 🎉 提交後

提交 PR 後：

1. **監控 PR 狀態**
   - 檢查 GitHub Actions 是否通過
   - 關注審查者的評論

2. **及時回應**
   - 回答問題
   - 進行要求的修改

3. **保持耐心**
   - 開源專案審查需要時間
   - 維護者可能很忙

4. **慶祝貢獻**
   - 你為開源社群做出了貢獻！
   - 這是一個完整且有價值的功能

---

## 📊 統計資訊

你的貢獻：

- **程式碼**: ~500 行（實作 + 測試）
- **文件**: ~2000 行（英文 + 中文）
- **測試**: 8 個單元測試 + 整合測試
- **開發時間**: 約 2-3 天
- **功能**: 完整的 Affine 整合

這是一個高品質的貢獻！

---

## 🌟 下一步

PR 合併後：

1. **更新你的 fork**
   ```bash
   git fetch upstream
   git merge upstream/main
   git push origin main
   ```

2. **分享你的貢獻**
   - 在社群媒體分享
   - 寫部落格文章
   - 告訴朋友

3. **繼續貢獻**
   - 修復 bugs
   - 新增功能
   - 改進文件

---

**祝你提交順利！** 🚀

如果有任何問題，隨時詢問。你已經做了很棒的工作！
