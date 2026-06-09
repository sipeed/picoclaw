# 貢獻指南 PicoClaw

感謝您對 PicoClaw 的關注！本專案是一個社群驅動的開源專案，目標是打造輕量靈活、人人可用的個人 AI 助手。我們歡迎各種形式的貢獻：錯誤修復、新功能、文件、翻譯與測試。

PicoClaw 本身在很大程度上是借助 AI 輔助開發的——我們擁抱這種方式，並圍繞它建立了貢獻流程。

## 目錄

- [行為準則](#行為準則)
- [貢獻方式](#貢獻方式)
- [開始貢獻](#開始貢獻)
- [開發環境設定](#開發環境設定)
- [修改程式碼](#修改程式碼)
- [AI 輔助貢獻](#ai-輔助貢獻)
- [Pull Request 流程](#pull-request-流程)
- [分支策略](#分支策略)
- [程式碼審查](#程式碼審查)
- [溝通管道](#溝通管道)

---

## 行為準則

我們致力於維護一個友善、相互尊重的社群環境。請保持善意與建設性的態度，並以善意解讀他人。任何形式的騷擾或歧視均不被接受。

---

## 貢獻方式

- **錯誤回報** — 使用錯誤回報範本開立 issue。
- **功能建議** — 使用功能請求範本開立 issue；建議在開始實作前先討論。
- **程式碼貢獻** — 修復錯誤或實作新功能，請參閱下方工作流程。
- **文件改進** — 改善 README、文件、行內注釋或翻譯。
- **測試與驗證** — 在新硬體、新頻道或新 LLM 提供者上執行 PicoClaw 並回報結果。

對於較大的新功能，請先開立 issue 討論設計方案，再動手撰寫程式碼。這能避免無效投入，也確保與專案方向保持一致。

對於文件貢獻，請優先遵循 [`docs/README.md`](docs/README.md) 中的目錄結構與命名慣例。新增或移動 Markdown 檔案後，請執行 `make lint-docs` 以提早發現常見的一致性問題。

---

## 開始貢獻

1. 在 GitHub 上 **Fork** 本儲存庫。
2. 將您的 fork **Clone** 到本地端：
   ```bash
   git clone https://github.com/<your-username>/picoclaw.git
   cd picoclaw
   ```
3. 加入上游遠端：
   ```bash
   git remote add upstream https://github.com/sipeed/picoclaw.git
   ```

---

## 開發環境設定

### 前置需求

- Go 1.25 或更新版本
- `make`

### 建置

```bash
make build       # Build binary (runs go generate first)
make generate    # Run go generate only
make check       # Full pre-commit check: deps + fmt + vet + test + docs consistency checks
```

### 執行測試

```bash
make test                                    # Run all tests
make integration-test                        # Run Docker-backed integration suites
go test -run TestName -v ./pkg/session/      # Run a single test
go test -bench=. -benchmem -run='^$' ./...  # Run benchmarks
```

Docker 整合測試套件會從 [`integration/suites/`](integration/suites/) 自動發現。套件結構與 CI 慣例請參閱 [`integration/README.md`](integration/README.md)。

### 程式碼風格

```bash
make fmt   # Format code
make vet   # Static analysis
make lint  # Full linter run
make lint-docs  # Check common documentation layout and naming conventions
```

所有 CI 檢查通過後，PR 才能被合併。推送程式碼前請先在本地端執行 `make check`，包含 `make lint-docs` 的文件一致性檢查，以提早發現問題。

---

## 修改程式碼

### 分支管理

請始終從 `main` 切出分支，並在 PR 中以 `main` 為目標分支。不要直接推送到 `main` 或任何 `release/*` 分支：

```bash
git checkout main
git pull upstream main
git checkout -b your-feature-branch
```

請使用具描述性的分支名稱，例如：`fix/telegram-timeout`、`feat/ollama-provider`、`docs/contributing-guide`。

### Commit 規範

- 使用英文撰寫清晰、簡潔的 commit 訊息。
- 使用祈使語氣：寫 "Add retry logic"，而非 "Added retry logic"。
- 有關聯 issue 時請引用：`Fix session leak (#123)`。
- 保持 commit 專注，每個 commit 只做一件事為佳。
- 對於小幅清理或錯字修正，請在開立 PR 前將其合併為單一 commit。
- 請參照 [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) 規範撰寫。

### 保持與上游同步

開立 PR 前，請將您的分支 rebase 到上游 `main`：

```bash
git fetch upstream
git rebase upstream/main
```

---

## AI 輔助貢獻

PicoClaw 在很大程度上借助 AI 輔助開發，我們完全擁抱這種開發方式。但貢獻者必須清楚了解自己在使用 AI 工具時所承擔的責任。

### 必須揭露

每個 PR 都必須透過 PR 範本中的 **🤖 AI Code Generation** 區段揭露 AI 參與情況，共分三個等級：

| 等級 | 說明 |
|---|---|
| 🤖 Fully AI-generated | AI 撰寫程式碼；貢獻者負責審查與驗證 |
| 🛠️ Mostly AI-generated | AI 產生草稿；貢獻者進行了大幅修改 |
| 👨‍💻 Mostly Human-written | 貢獻者主導；AI 提供建議或完全未使用 AI |

誠實揭露是基本要求。三種等級皆可接受，任何等級都不帶負面評價——重要的是貢獻的品質。

### 您需為提交的內容負責

使用 AI 產生程式碼，並不會減輕您身為貢獻者的責任。在開立包含 AI 產生程式碼的 PR 之前，您必須：

- **逐行閱讀並理解**產生的程式碼。
- **在真實環境中測試**（請參閱 PR 範本中的測試環境區段）。
- **檢查安全性問題** — AI 模型可能產生存在安全漏洞的程式碼（例如：路徑遍歷、注入攻擊、憑證外洩）。請仔細審查。
- **驗證正確性** — AI 產生的邏輯可能聽起來合理，但實際上是錯誤的。請驗證行為，而非僅確認語法。

若明顯可看出貢獻者未閱讀或測試 AI 產生的程式碼，該 PR 將被直接關閉，不予審查。

### AI 產生程式碼的品質標準

AI 產生的貢獻與人工撰寫的程式碼遵循**相同的品質要求**：

- 必須通過所有 CI 檢查（`make check`）。
- 必須符合 Go 慣用寫法，並與現有程式碼庫的風格保持一致。
- 不得引入不必要的抽象、無效程式碼或過度設計。
- 須在適當之處包含或更新測試。

### 安全審查

AI 產生的程式碼需要額外仔細的安全審查。請特別注意以下方面：

- 檔案路徑處理與沙箱逃逸（參見 commit `244eb0b` 中的真實案例）
- channel 處理器與 tool 實作中的外部輸入驗證
- 憑證或金鑰的處理
- 指令執行（`exec.Command`、shell 呼叫等）

若您不確定某段 AI 產生的程式碼是否安全，請在 PR 中說明——審查者將協助判斷。

---

## Pull Request 流程

### 開立 PR 前的確認事項

- [ ] 在本地端執行 `make check` 並確認通過。
- [ ] 完整填寫 PR 範本，包括 AI 揭露區段。
- [ ] 在 PR 說明中關聯相關 issue。
- [ ] 保持 PR 專注，避免將不相關的修改混在一起。

### PR 範本各區段說明

PR 範本要求填寫：

- **Description** — 此變更做了什麼，以及為什麼要做？
- **Type of Change** — 錯誤修復、新功能、文件或重構。
- **AI Code Generation** — AI 參與情況揭露（必填）。
- **Related Issue** — 此 PR 所對應的 issue 連結。
- **Technical Context** — 參考連結與設計理由（純文件類 PR 可略過）。
- **Test Environment** — 用於測試的硬體、作業系統、模型／提供者及頻道。
- **Evidence** — 選填的日誌或截圖，用以佐證變更有效。
- **Checklist** — 自我審查確認。

### PR 規模

請盡量提交小而易於審查的 PR。涉及 5 個檔案共 200 行修改的 PR，遠比涉及 30 個檔案共 2000 行修改的 PR 容易審查。若您的功能較大，請考慮將其拆分為一系列邏輯完整的小型 PR。

---

## 分支策略

### 長期分支

- **`main`** — 活躍開發分支。所有功能 PR 均以 `main` 為目標分支。此分支受保護：禁止直接推送，合併前至少需一位維護者核准。
- **`release/x.y`** — 穩定發布分支，在某版本準備發布時從 `main` 切出。這些分支的保護等級高於 `main`。

### 合併至 `main` 的條件

PR 必須同時滿足以下所有條件，才能被合併：

1. **CI 全部通過** — 所有 GitHub Actions 工作流程（lint、test、build）均為綠燈。
2. **審查者核准** — 至少一位維護者已核准該 PR。
3. **無未解決的審查意見** — 所有審查討論執行緒均已關閉。
4. **PR 範本填寫完整** — 包括 AI 揭露與測試環境資訊。

### 誰可以合併

只有維護者才能合併 PR。貢獻者不能合併自己的 PR，即使擁有寫入權限也不例外。

### 合併策略

為保持 `main` 歷史紀錄清晰易讀，我們對大多數 PR 使用 **squash merge**。每個合併的 PR 會成為單一 commit 並標示 PR 編號，例如：

```
feat: Add Ollama provider support (#491)
```

若一個 PR 包含多個獨立、結構清晰且能呈現完整脈絡的 commit，維護者可視情況使用一般合併。

### Release 分支

當某個版本準備就緒時，維護者會從 `main` 切出 `release/x.y` 分支。此後：

- **新功能不會被回移（backport）。** Release 分支切出後，不再接收任何新功能。
- **安全性修復與重大錯誤修復會被 cherry-pick 進來。** 若 `main` 上的某個修復屬於安全漏洞、資料遺失或崩潰等問題，維護者會將相關 commit cherry-pick 至受影響的 `release/x.y` 分支，並發布修補版本。

若您認為 `main` 上的某個修復應該回移至某個 release 分支，請在 PR 說明中註記，或另行開立 issue 說明。最終決定由維護者做出。

Release 分支的保護等級高於 `main`，任何情況下均不允許直接推送。

---

## 程式碼審查

### 對貢獻者的建議

- 請在合理時間內回覆審查意見。若需要更多時間，請告知。
- 根據意見更新 PR 後，請簡要說明改動內容（例如：「已按建議改用 `sync.RWMutex`」）。
- 若不同意某條意見，請以尊重的態度說明您的理由——審查者也可能有判斷失誤的時候。
- 審查開始後請勿 force push——這會使審查者難以追蹤變更。請改用額外的 commit，維護者在合併時會進行 squash。

### 對審查者的建議

審查重點：

1. **正確性** — 程式碼是否實現了所述的功能？是否存在邊界情況？
2. **安全性** — 尤其針對 AI 產生的程式碼、tool 實作與 channel 處理器。
3. **架構** — 實作方式是否與現有設計一致？
4. **簡潔性** — 是否有更簡單的方案？是否引入了不必要的複雜度？
5. **測試** — 修改是否有測試涵蓋？現有測試是否仍然有意義？

請給予具建設性且具體的意見。「如果兩個 goroutine 同時呼叫此函式可能會有競態條件——建議在這裡加一個 mutex」遠比「這裡看起來有問題」更有幫助。

### 審查者名單

PR 提交後，您可參考下表聯繫對應的審查者。

|Function| Reviewer|
|---     |---      |
|Provider|@yinwm   |
|Channel |@yinwm/@alexhoshina   |
|Agent   |@lxowalle/@Zhaoyikaiii|
|Tools   |@lxowalle|
|SKill   ||
|MCP     ||
|Optimization|@lxowalle|
|Security||
|AI CI   |@imguoguo|
|UX      ||
|Document||

---

## 溝通管道

- **GitHub Issues** — 錯誤回報、功能建議、設計討論。
- **GitHub Discussions** — 一般性問題、想法交流、社群對話。
- **Pull Request 留言** — 與特定程式碼相關的意見回饋。
- **Wechat&Discord** — 當您有至少一個已合併的 PR 後，我們將邀請您加入。

有疑問時，請先開立 issue 再動手撰寫程式碼。這幾乎零成本，卻能避免大量無效投入。

---

## 關於本專案的 AI 驅動起源

PicoClaw 的架構在人工監督下，經由 AI 輔助完成了大量設計與實作工作。若您發現某處看起來奇怪或過度設計，這可能是該過程留下的痕跡——歡迎開立 issue 討論。

我們相信，負責任地使用 AI 輔助開發能產生優秀的成果。我們同樣相信，人類必須對自己提交的內容負責。這兩點並不矛盾。

感謝您的貢獻！
