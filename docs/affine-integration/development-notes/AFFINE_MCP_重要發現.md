# Affine MCP 重要發現

## 🔍 關鍵發現（2026-03-05）

### Affine Cloud MCP Bridge 只提供 3 個工具！

經過實際測試，我們發現 **Affine Cloud 的 MCP Bridge** 和 **完整的 Affine MCP Server** 是不同的東西：

---

## 📊 可用工具對比

### Affine Cloud MCP Bridge（你目前使用的）
**端點**: `https://app.affine.pro/api/workspaces/{workspaceId}/mcp`

**可用工具**: 僅 3 個
1. ✅ `keyword_search` - 關鍵字搜尋
2. ✅ `semantic_search` - 語意搜尋  
3. ✅ `read_document` - 讀取文件

**特點**:
- 不需要安裝任何東西
- 直接透過 HTTPS 存取
- 功能有限，只能搜尋和讀取
- 無法列出文件、建立文件、編輯文件

---

### 完整 Affine MCP Server（需要安裝）
**安裝**: `npm i -g affine-mcp-server`

**可用工具**: 43 個
包括：
- 工作區管理（5 個）
- 文件管理（23 個）
- 資料庫功能（2 個）
- 留言功能（5 個）
- 版本歷史（1 個）
- 使用者與權杖（7 個）
- 通知功能（2 個）
- Blob 儲存（3 個）

**特點**:
- 需要安裝 Node.js 和 NPM 套件
- 透過 stdio 通訊（不是 HTTP）
- 功能完整，可以建立、編輯、刪除文件
- 支援 WebSocket 即時編輯

---

## 🎯 實際測試結果

### 測試 1: 列出可用工具
```bash
curl -X POST \
  -H "Authorization: Bearer {token}" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
  {mcp_endpoint}
```

**結果**: 只回傳 3 個工具
```json
{
  "tools": [
    {"name": "read_document"},
    {"name": "semantic_search"},
    {"name": "keyword_search"}
  ]
}
```

### 測試 2: 嘗試使用 list_docs
```bash
curl -X POST \
  -d '{"method":"tools/call","params":{"name":"list_docs"}}' \
  {mcp_endpoint}
```

**結果**: ❌ 錯誤
```json
{
  "error": "Tool list_docs not found"
}
```

---

## 💡 這對我們的影響

### 目前可以做的 ✅
1. **關鍵字搜尋** - 搜尋文件內容
2. **語意搜尋** - 基於意義的搜尋
3. **讀取文件** - 取得文件內容（但目前有伺服器錯誤）

### 目前無法做的 ❌
1. 列出所有文件
2. 取得文件元資料
3. 匯出 Markdown
4. 建立新文件
5. 編輯文件
6. 刪除文件
7. 管理標籤
8. 管理留言
9. 查看版本歷史
10. 其他 40+ 個功能

---

## 🔧 解決方案選項

### 選項 1: 繼續使用 MCP Bridge（目前方案）
**優點**:
- 不需要安裝任何東西
- 設定簡單
- 搜尋功能已經可用

**缺點**:
- 功能非常有限
- 無法列出或管理文件
- `read_document` 目前有問題

**適合**: 只需要搜尋功能的場景

---

### 選項 2: 安裝完整 MCP Server
**優點**:
- 43 個完整功能
- 可以建立、編輯、刪除文件
- 支援所有進階功能

**缺點**:
- 需要安裝 Node.js 和 NPM
- 需要改用 stdio 通訊（不是 HTTP）
- 整合複雜度較高

**適合**: 需要完整文件管理功能

**安裝步驟**:
```bash
# 1. 安裝
npm i -g affine-mcp-server

# 2. 登入
affine-mcp login

# 3. 整合到 PicoClaw
# 需要修改程式碼，使用 stdio 而不是 HTTP
```

---

### 選項 3: 混合方案
**方案**: 
- 使用 MCP Bridge 進行搜尋（HTTP）
- 使用完整 MCP Server 進行文件管理（stdio）

**優點**:
- 搜尋功能簡單快速
- 需要時可以使用進階功能

**缺點**:
- 需要維護兩套整合
- 複雜度最高

---

## 📝 建議

### 短期（目前）
✅ **繼續使用 MCP Bridge**
- 搜尋功能已經可用且穩定
- 足夠應付基本需求
- 不需要額外安裝

### 中期（如果需要更多功能）
🔄 **評估是否需要完整 MCP Server**
- 如果需要列出文件 → 考慮安裝
- 如果需要建立/編輯文件 → 必須安裝
- 如果只是搜尋 → 不需要

### 長期（完整整合）
🚀 **實作完整 MCP Server 整合**
- 研究 stdio 通訊方式
- 實作 Go 到 Node.js 的橋接
- 獲得所有 43 個功能

---

## 🎯 目前狀態總結

### 已實作並可用 ✅
- `keyword_search` - 關鍵字搜尋（已測試，可用）
- `semantic_search` - 語意搜尋（已實作，待測試）

### 已實作但有問題 ⚠️
- `read_document` - 讀取文件（伺服器回傳內部錯誤）

### 已移除（不存在的功能）❌
- `list_docs` - 列出文件
- `get_doc` - 取得元資料
- `export_doc_markdown` - 匯出 Markdown

---

## 📚 參考資料

### Affine MCP Server 官方文件
- GitHub: https://github.com/DAWNCR0W/affine-mcp-server
- 完整工具列表: 43 個工具
- 安裝方式: `npm i -g affine-mcp-server`

### Affine Cloud MCP Bridge
- 端點: `https://app.affine.pro/api/workspaces/{id}/mcp`
- 工具列表: 3 個工具
- 存取方式: HTTP + Bearer Token

---

## 🔄 下一步行動

1. ✅ **已完成**: 修正程式碼，移除不存在的功能
2. ⏳ **待測試**: `semantic_search` 功能
3. ⏳ **待修復**: `read_document` 的伺服器錯誤
4. 🤔 **待決定**: 是否需要安裝完整 MCP Server

---

**發現日期**: 2026-03-05  
**測試環境**: GitHub Codespace  
**Affine 版本**: Cloud (app.affine.pro)  
**工作區 ID**: 732dbb91-3973-4b77-adbc-c8d5ec830d6d
