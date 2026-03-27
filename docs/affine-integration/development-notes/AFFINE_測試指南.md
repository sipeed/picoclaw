# Affine 工具測試指南

## 📋 新增功能（2026-02-26）

### 階段 1.2: 基礎文件管理

已新增以下功能：
1. ✅ `list_docs` - 列出所有文件
2. ✅ `get_doc` - 取得文件元資料
3. ✅ `export_markdown` - 匯出文件為 Markdown

---

## 🧪 測試步驟

### 準備工作（在 Codespace 中）

```bash
# 1. 拉取最新程式碼
git pull origin main

# 2. 編譯
go build -o picoclaw ./cmd/picoclaw

# 3. 確認設定檔存在
cat ~/.picoclaw/config.json
```

---

## 測試 1: 列出所有文件

### 指令
```bash
./picoclaw agent -m "List all documents in my Affine workspace"
```

### 預期結果
- 顯示工作區中的所有文件
- 包含文件標題、ID、建立時間
- 如果有標籤，也會顯示

### 測試變化
```bash
# 限制數量
./picoclaw agent -m "List first 5 documents in Affine"

# 使用分頁
./picoclaw agent -m "List documents in Affine, skip first 10"
```

---

## 測試 2: 取得文件元資料

### 指令
```bash
# 使用已知的文件 ID
./picoclaw agent -m "Get metadata for document eDebZI1h3F from Affine"
```

### 預期結果
- 顯示文件標題
- 顯示文件 ID
- 顯示建立和更新時間
- 顯示標籤（如果有）
- 顯示公開/私密狀態

---

## 測試 3: 匯出文件為 Markdown

### 指令
```bash
./picoclaw agent -m "Export document eDebZI1h3F from Affine as markdown"
```

### 預期結果
- 顯示文件的 Markdown 格式內容
- 保留原始格式（標題、列表、連結等）

---

## 測試 4: 組合測試

### 測試流程
```bash
# 1. 先列出所有文件
./picoclaw agent -m "List all documents in Affine"

# 2. 從結果中選一個文件 ID，取得元資料
./picoclaw agent -m "Get metadata for document [DOC_ID] from Affine"

# 3. 匯出該文件為 Markdown
./picoclaw agent -m "Export document [DOC_ID] as markdown from Affine"
```

---

## 測試 5: 自然語言測試

測試 AI 是否能理解自然語言並選擇正確的動作：

```bash
# 應該使用 list_docs
./picoclaw agent -m "Show me what documents I have in Affine"

# 應該使用 get_doc
./picoclaw agent -m "Tell me about the document 簡易教學 in Affine"

# 應該使用 export_markdown
./picoclaw agent -m "Give me the markdown version of document eDebZI1h3F"
```

---

## 🐛 除錯指令

### 使用 curl 直接測試 MCP 端點

#### 測試 list_docs
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_docs","arguments":{"limit":10,"skip":0}}}' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

#### 測試 get_doc
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_doc","arguments":{"docId":"eDebZI1h3F"}}}' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

#### 測試 export_doc_markdown
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"export_doc_markdown","arguments":{"docId":"eDebZI1h3F"}}}' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

---

## 📊 測試記錄表

| 測試項目 | 狀態 | 備註 |
|---------|------|------|
| list_docs - 基本功能 | ⏳ 待測試 | |
| list_docs - 限制數量 | ⏳ 待測試 | |
| list_docs - 分頁 | ⏳ 待測試 | |
| get_doc - 基本功能 | ⏳ 待測試 | |
| get_doc - 顯示標籤 | ⏳ 待測試 | |
| export_markdown - 基本功能 | ⏳ 待測試 | |
| 自然語言理解 | ⏳ 待測試 | |
| 錯誤處理 | ⏳ 待測試 | |

---

## ⚠️ 已知問題

### 1. read_document 仍有問題
- 伺服器回傳內部錯誤
- 建議使用 `export_markdown` 作為替代方案

### 2. 回應格式可能不一致
- 有些端點回傳單一物件
- 有些端點回傳陣列
- 程式碼已處理兩種情況

---

## 🎯 成功標準

### list_docs
- ✅ 能列出至少一個文件
- ✅ 顯示文件標題和 ID
- ✅ 支援 limit 和 skip 參數
- ✅ 正確處理空結果

### get_doc
- ✅ 能取得文件元資料
- ✅ 顯示所有可用欄位
- ✅ 正確處理不存在的文件 ID

### export_markdown
- ✅ 能匯出文件內容
- ✅ 保留 Markdown 格式
- ✅ 正確處理匯出失敗

---

## 📝 測試後續步驟

### 如果測試成功
1. 更新 `AFFINE_整合總結.md` 的實作狀態
2. 記錄測試結果
3. 繼續實作階段 1.3（標籤功能）

### 如果測試失敗
1. 使用 curl 直接測試 MCP 端點
2. 檢查回應格式
3. 調整解析邏輯
4. 重新測試

---

## 🚀 下一步

完成這些測試後，我們將實作：

### 階段 1.3: 標籤功能
- `list_tags` - 列出所有標籤
- `list_docs_by_tag` - 依標籤搜尋文件
- `create_tag` - 建立新標籤
- `add_tag_to_doc` - 為文件加標籤
- `remove_tag_from_doc` - 移除文件標籤

---

**測試日期**: 2026-02-26  
**測試環境**: GitHub Codespace  
**測試者**: 使用者
