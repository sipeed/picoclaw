#!/bin/bash

# Affine 功能自動測試腳本
# 使用方式: bash test-affine-features.sh

set -e  # 遇到錯誤就停止

echo "=========================================="
echo "Affine 工具自動測試腳本"
echo "=========================================="
echo ""

# 顏色定義
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 測試結果統計
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 測試函數
run_test() {
    local test_name=$1
    local command=$2
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${YELLOW}測試 $TOTAL_TESTS: $test_name${NC}"
    echo "指令: $command"
    echo ""
    
    if eval "$command"; then
        echo -e "${GREEN}✓ 測試通過${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ 測試失敗${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo ""
    echo "------------------------------------------"
    echo ""
}

# 檢查環境
echo "1. 檢查環境..."
echo ""

if [ ! -f "./picoclaw" ]; then
    echo -e "${YELLOW}找不到 picoclaw 執行檔，開始編譯...${NC}"
    go build -o picoclaw ./cmd/picoclaw
    echo -e "${GREEN}✓ 編譯完成${NC}"
    echo ""
fi

if [ ! -f "$HOME/.picoclaw/config.json" ]; then
    echo -e "${RED}錯誤: 找不到設定檔 ~/.picoclaw/config.json${NC}"
    echo "請先設定 Affine 連線資訊"
    exit 1
fi

echo -e "${GREEN}✓ 環境檢查完成${NC}"
echo ""
echo "=========================================="
echo "開始測試"
echo "=========================================="
echo ""

# 測試 1: 列出文件
run_test "列出所有文件" \
    "./picoclaw agent -m 'List all documents in my Affine workspace'"

# 測試 2: 列出文件（限制數量）
run_test "列出前 5 個文件" \
    "./picoclaw agent -m 'List first 5 documents in Affine'"

# 測試 3: 搜尋功能（已知可用）
run_test "關鍵字搜尋" \
    "./picoclaw agent -m 'Search my Affine workspace for the'"

# 測試 4: 取得文件元資料
run_test "取得文件元資料" \
    "./picoclaw agent -m 'Get metadata for document eDebZI1h3F from Affine'"

# 測試 5: 匯出 Markdown
run_test "匯出文件為 Markdown" \
    "./picoclaw agent -m 'Export document eDebZI1h3F as markdown from Affine'"

# 測試 6: 自然語言測試
run_test "自然語言 - 列出文件" \
    "./picoclaw agent -m 'Show me what documents I have in Affine'"

# 測試 7: 自然語言測試
run_test "自然語言 - 取得資訊" \
    "./picoclaw agent -m 'Tell me about document eDebZI1h3F in Affine'"

# 顯示測試結果
echo ""
echo "=========================================="
echo "測試結果總結"
echo "=========================================="
echo ""
echo "總測試數: $TOTAL_TESTS"
echo -e "${GREEN}通過: $PASSED_TESTS${NC}"
echo -e "${RED}失敗: $FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 所有測試都通過了！${NC}"
    exit 0
else
    echo -e "${RED}⚠️  有 $FAILED_TESTS 個測試失敗${NC}"
    exit 1
fi
