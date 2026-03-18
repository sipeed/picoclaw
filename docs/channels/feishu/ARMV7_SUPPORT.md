# 飞书渠道 ARMv7 支持

## 问题描述

飞书SDK (`github.com/larksuite/oapi-sdk-go/v3`) 在32位架构（如ARMv7）上存在编译问题。具体问题是：

在 `service/drive/v1/api_ext.go` 文件中，使用了 `math.MaxInt64` 常量：

```go
limit: math.MaxInt64
```

在32位架构上，`int` 类型是32位的，无法容纳 `MaxInt64` (9223372036854775807)，导致编译错误：

```
constant 9223372036854775807 overflows int
```

## 解决方案

### 方案1：手动修复飞书SDK（推荐）

在编译前，手动修复飞书SDK的问题：

```bash
# 找到飞书SDK的安装路径
LARK_PATH="${GOPATH:-$HOME/go}/pkg/mod/github.com/larksuite/oapi-sdk-go/v3@v3.5.3"
FILE="$LARK_PATH/service/drive/v1/api_ext.go"

# 修复 math.MaxInt64 问题
sed -i 's/math.MaxInt64/int(^uint(0) >> 1)/g' "$FILE"
```

### 方案2：使用构建脚本

我们提供了一个构建脚本来简化这个过程：

```bash
make build-linux-arm-with-feishu
```

或者：

```bash
./scripts/build-feishu-armv7.sh
```

### 方案3：使用修复后的Fork（未来）

我们正在向飞书SDK提交PR来修复这个问题。一旦PR被合并，这个问题将自动解决。

跟踪PR：[larksuite/oapi-sdk-go#XXX](https://github.com/larksuite/oapi-sdk-go/pulls)

## 技术细节

### 构建标签

PicoClaw使用Go构建标签来支持不同架构：

- `feishu_64.go`: 支持64位架构（amd64, arm64, riscv64, mips64, ppc64）
- `feishu_32.go`: 支持其他架构（386, mips, 等）

从Issue #1675开始，我们添加了对ARMv7的支持：

- `feishu_64.go`: 现在包含 `arm` 构建标签，支持ARMv7
- `feishu_32.go`: 排除 `arm` 架构

### 为什么PicoClaw可以使用飞书SDK的ARMv7支持？

虽然飞书SDK存在编译问题，但这个问题只影响 `service/drive/v1` 包，而PicoClaw只使用了以下包：

- `github.com/larksuite/oapi-sdk-go/v3` (主客户端)
- `github.com/larksuite/oapi-sdk-go/v3/core`
- `github.com/larksuite/oapi-sdk-go/v3/event/dispatcher`
- `github.com/larksuite/oapi-sdk-go/v3/service/im/v1` (IM服务)
- `github.com/larksuite/oapi-sdk-go/v3/ws` (WebSocket)

这些包在ARMv7上工作正常。问题只出现在 `service/drive/v1` 包中，该包被 `client.go` 导入，但我们不直接使用它。

## 构建指南

### 为ARMv7构建（32位）

```bash
# 方法1：使用Makefile（需要先修复SDK）
make fix-lark-sdk
make build-linux-arm

# 方法2：手动构建
# 1. 先修复飞书SDK
sed -i 's/math.MaxInt64/int(^uint(0) >> 1)/g' \
    ~/go/pkg/mod/github.com/larksuite/oapi-sdk-go/v3@v3.5.3/service/drive/v1/api_ext.go

# 2. 构建
GOOS=linux GOARCH=arm GOARM=7 go build -o picoclaw-linux-arm ./cmd/picoclaw
```

### 为ARM64构建（64位）

```bash
make build-linux-arm64
```

或者：

```bash
GOOS=linux GOARCH=arm64 go build -o picoclaw-linux-arm64 ./cmd/picoclaw
```

## 测试

构建完成后，可以在ARMv7设备上测试飞书渠道：

```bash
./picoclaw-linux-arm gateway
```

确保在 `config.json` 中启用了飞书渠道：

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "your-app-id",
      "app_secret": "your-app-secret",
      "allow_from": []
    }
  }
}
```

## 参考

- [Issue #1675](https://github.com/sipeed/picoclaw/issues/1675)
- [飞书SDK Issue](https://github.com/larksuite/oapi-sdk-go/issues)
