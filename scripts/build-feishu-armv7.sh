#!/bin/bash
# 为ARMv7构建PicoClaw并修复飞书SDK的32位架构问题

set -e

echo "Building PicoClaw for ARMv7 with Feishu support..."

# 找到飞书SDK的安装路径
LARK_PATH="${GOPATH:-$HOME/go}/pkg/mod/github.com/larksuite/oapi-sdk-go/v3@v3.5.3"
FILE="$LARK_PATH/service/drive/v1/api_ext.go"

if [ ! -f "$FILE" ]; then
    echo "Error: Feishu SDK not found at $FILE"
    echo "Please run 'go mod download' first."
    exit 1
fi

# 检查是否已经修复
if grep -q "int(^uint(0) >> 1)" "$FILE"; then
    echo "Feishu SDK already patched."
else
    echo "Patching Feishu SDK for 32-bit architecture support..."
    # 备份原文件
    if [ ! -f "$FILE.bak" ]; then
        cp "$FILE" "$FILE.bak"
    fi
    # 修复 math.MaxInt64 问题
    sed -i 's/math.MaxInt64/int(^uint(0) >> 1)/g' "$FILE"
    echo "Patch applied successfully."
fi

# 构建PicoClaw
BUILD_DIR="build"
BINARY_NAME="picoclaw-linux-arm"

echo "Building $BINARY_NAME..."
mkdir -p "$BUILD_DIR"

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build \
    -v -tags stdjson \
    -ldflags "-s -w" \
    -o "$BUILD_DIR/$BINARY_NAME" \
    ./cmd/picoclaw

echo "Build complete: $BUILD_DIR/$BINARY_NAME"
echo ""
echo "You can now deploy this binary to your ARMv7 device."
echo "Make sure to enable feishu channel in your config.json"
