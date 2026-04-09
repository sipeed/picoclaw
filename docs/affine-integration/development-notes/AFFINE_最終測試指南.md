# Affine MCP Bridge 最終測試指南

## 📋 概述

本指南說明如何測試 PicoClaw 與 Affine Cloud MCP Bridge 的整合功能。

**完成日期**: 2026-03-05  
**狀態**: 基礎整合完成，3 個工具已實作

---

## 🎯 可用功能

### ✅ 1. 關鍵字搜尋 (keyword_search)
- **狀態**: 完全可用
- **功能**: 搜尋工作區中的文件
- **支援**: 中英文

### ✅ 2. 語意搜尋 (semantic_search)
- **狀態**: 完全可用
- **功能**: 基於意義的搜尋，回傳文件內容
- **優點**: 可作為讀取文件的替代方案

### ⚠️ 3. 讀取文件 (read_document)
- **狀態**: 伺服器端錯誤
- **問題**: Affine 回傳 "An internal error occurred"
- **替代方案**: 使用 semantic_search

---

## 🧪 測試步驟

### 前置準備

1. 在 Codespace 中拉取最新程式碼:
```bash
cd /workspaces/picoclaw
git pull origin main
```

2. 編譯程式:
```bash
go build -o picoclaw ./cmd/picoclaw
```

3. 確認設定檔存在:
```bash
cat ~/.picoclaw/config.json
```

---

## 測試 1: 關鍵字搜尋（英文）

### 使用 curl 直接測試
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "keyword_search",
      "arguments": {
        "query": "the"
      }
    }
  }' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

### 預期結果
```
event: message
data: {"result":{"content":[{"type":"text","text":"{\"docId\":\"eDebZI1h3F\",\"title\":\"簡易教學\",\"createdAt\":\"2025-11-04T03:50:00.592Z\"}"}]},"jsonrpc":"2.0","id":1}
```

### 使用 PicoClaw 測試
```bash
./picoclaw agent -m "Search my Affine workspace for 'the'"
```

---

## 測試 2: 關鍵字搜尋（中文）

### 使用 curl 直接測試
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "keyword_search",
      "arguments": {
        "query": "教學"
      }
    }
  }' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

### 使用 PicoClaw 測試
```bash
./picoclaw agent -m "在 Affine 中搜尋教學"
```

---

## 測試 3: 語意搜尋

### 使用 curl 直接測試
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "semantic_search",
      "arguments": {
        "query": "tutorial"
      }
    }
  }' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

### 預期結果
回傳多份文件，包含完整內容（可用來讀取文件）

### 使用 PicoClaw 測試
```bash
./picoclaw agent -m "Use semantic search to find tutorials in Affine"
```

---

## 測試 4: 讀取文件（已知問題）

### 使用 curl 直接測試
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "read_document",
      "arguments": {
        "docId": "eDebZI1h3F"
      }
    }
  }' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

### 預期結果（錯誤）
```
event: message
data: {"result":{"content":[{"type":"text","text":"An internal error occurred."}],"isError":true},"jsonrpc":"2.0","id":1}
```

### 使用 PicoClaw 測試
```bash
./picoclaw agent -m "Read document eDebZI1h3F from Affine"
```

### 預期錯誤訊息
```
read_document failed: MCP error -32603: An internal error occurred. 
Note: This tool may be unstable on Affine Cloud. Try using search instead to find document content.
```

---

## 測試 5: 列出可用工具

### 使用 curl 測試
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp
```

### 預期結果
只有 3 個工具:
- `keyword_search`
- `semantic_search`
- `read_document`

---

## 📊 測試結果總結

| 功能 | 狀態 | 測試結果 | 備註 |
|------|------|---------|------|
| keyword_search | ✅ | 通過 | 支援中英文 |
| semantic_search | ✅ | 通過 | 回傳完整內容 |
| read_document | ⚠️ | 伺服器錯誤 | 使用 semantic_search 替代 |

---

## 🔧 疑難排解

### 問題 1: HTTP 406 錯誤
**原因**: 缺少 Accept 標頭  
**解決**: 加入 `Accept: application/json, text/event-stream`

### 問題 2: 找不到工具
**原因**: 工具名稱錯誤  
**解決**: 使用正確名稱 `keyword_search`, `semantic_search`, `read_document`

### 問題 3: 無法解析回應
**原因**: SSE 格式  
**解決**: 從 `event: message` 和 `data:` 行提取 JSON

### 問題 4: read_document 失敗
**原因**: Affine 伺服器端問題  
**解決**: 使用 `semantic_search` 取得文件內容

---

## 💡 最佳實踐

### 1. 優先使用 keyword_search
- 速度快
- 結果精確
- 適合已知關鍵字的搜尋

### 2. 需要內容時使用 semantic_search
- 回傳完整文件內容
- 可替代 read_document
- 適合需要讀取文件的場景

### 3. 避免使用 read_document
- 目前有伺服器問題
- semantic_search 是更好的選擇

---

## 🚀 快速測試腳本

建立檔案 `test-affine-all.sh`:

```bash
#!/bin/bash

echo "=== 測試 1: 關鍵字搜尋（英文）==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"keyword_search","arguments":{"query":"the"}}}' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp

echo -e "\n\n=== 測試 2: 語意搜尋 ==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_search","arguments":{"query":"tutorial"}}}' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp

echo -e "\n\n=== 測試 3: 讀取文件（預期失敗）==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Authorization: Bearer ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_document","arguments":{"docId":"eDebZI1h3F"}}}' \
  https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp

echo -e "\n\n=== 測試完成 ==="
```

執行:
```bash
chmod +x test-affine-all.sh
./test-affine-all.sh
```

---

## 📝 測試檢查清單

- [ ] 關鍵字搜尋（英文）可用
- [ ] 關鍵字搜尋（中文）可用
- [ ] 語意搜尋可用
- [ ] 語意搜尋回傳文件內容
- [ ] read_document 顯示友善錯誤訊息
- [ ] 錯誤訊息建議使用替代方案
- [ ] PicoClaw 整合測試通過

---

## 🎓 重要發現

1. **Affine Cloud MCP Bridge ≠ 完整 MCP Server**
   - Cloud Bridge: 3 個工具
   - 完整 Server: 43 個工具（需要 npm 安裝）

2. **semantic_search 是最有用的工具**
   - 不只搜尋，還回傳完整內容
   - 可完全替代 read_document

3. **read_document 目前不可用**
   - 伺服器端問題
   - 已加入友善錯誤訊息
   - 建議使用 semantic_search

---

## ✅ 結論

Affine MCP Bridge 整合已完成，2/3 工具完全可用，1/3 有替代方案。系統已可投入生產使用！

**建議**: 優先使用 keyword_search 和 semantic_search，避免使用 read_document。

---

**測試環境**: GitHub Codespace  
**Affine 版本**: Cloud (app.affine.pro)  
**工作區 ID**: 732dbb91-3973-4b77-adbc-c8d5ec830d6d  
**最後更新**: 2026-03-05
