#!/bin/bash

# Affine MCP 端點直接測試腳本
# 使用 curl 直接測試 MCP API，不透過 PicoClaw

set -e

echo "=========================================="
echo "Affine MCP 端點直接測試"
echo "=========================================="
echo ""

# 設定
MCP_ENDPOINT="https://app.affine.pro/api/workspaces/732dbb91-3973-4b77-adbc-c8d5ec830d6d/mcp"
API_TOKEN="ut_sdphcGU940Vv5UhGKXy7Rw1WpM2KQjUbyA2bV6bC7nY"
TEST_DOC_ID="eDebZI1h3F"

# 顏色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 測試計數
TOTAL=0
PASSED=0
FAILED=0

# 測試函數
test_mcp_tool() {
    local test_name=$1
    local tool_name=$2
    local arguments=$3
    
    TOTAL=$((TOTAL + 1))
    echo -e "${YELLOW}測試 $TOTAL: $test_name${NC}"
    echo -e "${BLUE}工具: $tool_name${NC}"
    echo "參數: $arguments"
    echo ""
    
    local request_body=$(cat <<EOF
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "$tool_name",
    "arguments": $arguments
  }
}
EOF
)
    
    echo "請求內容:"
    echo "$request_body" | jq '.' 2>/dev/null || echo "$request_body"
    echo ""
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Accept: application/json, text/event-stream" \
        -H "Authorization: Bearer $API_TOKEN" \
        -d "$request_body" \
        "$MCP_ENDPOINT")
    
    echo "回應內容:"
    echo "$response"
    echo ""
    
    # 檢查是否有錯誤
    if echo "$response" | grep -q '"isError":true'; then
        echo -e "${RED}✗ 測試失敗 - API 回傳錯誤${NC}"
        FAILED=$((FAILED + 1))
    elif echo "$response" | grep -q '"error"'; then
        echo -e "${RED}✗ 測試失敗 - JSON-RPC 錯誤${NC}"
        FAILED=$((FAILED + 1))
    else
        echo -e "${GREEN}✓ 測試通過${NC}"
        PASSED=$((PASSED + 1))
    fi
    
    echo ""
    echo "------------------------------------------"
    echo ""
}

# 執行測試

echo "開始測試 MCP 端點..."
echo ""

# 測試 1: list_docs
test_mcp_tool \
    "列出文件" \
    "list_docs" \
    '{"limit": 10, "skip": 0}'

# 測試 2: keyword_search
test_mcp_tool \
    "關鍵字搜尋" \
    "keyword_search" \
    '{"query": "the"}'

# 測試 3: get_doc
test_mcp_tool \
    "取得文件元資料" \
    "get_doc" \
    "{\"docId\": \"$TEST_DOC_ID\"}"

# 測試 4: export_doc_markdown
test_mcp_tool \
    "匯出 Markdown" \
    "export_doc_markdown" \
    "{\"docId\": \"$TEST_DOC_ID\"}"

# 測試 5: semantic_search
test_mcp_tool \
    "語意搜尋" \
    "semantic_search" \
    '{"query": "tutorial"}'

# 測試 6: list_tags
test_mcp_tool \
    "列出標籤" \
    "list_tags" \
    '{}'

# 顯示結果
echo ""
echo "=========================================="
echo "測試結果"
echo "=========================================="
echo ""
echo "總測試數: $TOTAL"
echo -e "${GREEN}通過: $PASSED${NC}"
echo -e "${RED}失敗: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 所有 MCP 端點測試都通過了！${NC}"
    exit 0
else
    echo -e "${RED}⚠️  有 $FAILED 個測試失敗${NC}"
    echo ""
    echo "提示："
    echo "- 檢查 API Token 是否有效"
    echo "- 檢查文件 ID 是否存在"
    echo "- 檢查網路連線"
    exit 1
fi
