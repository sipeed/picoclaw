# macOS 开机启动 (launchd) 设计方案

## 概述

添加 `picoclaw install` 命令，自动生成 macOS launchd 的 plist 配置文件并安装到 `~/Library/LaunchAgents/`，实现开机自动启动 `picoclaw gateway`。

## 详细设计

### 1. plist 文件内容

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.picoclaw.gateway</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/picoclaw</string>
        <string>gateway</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

### 2. 命令设计

| 命令 | 描述 |
|------|------|
| `picoclaw install` | 安装 launchd plist（当前用户） |
| `picoclaw install --uninstall` | 卸载 launchd plist |
| `picoclaw install --status` | 查看安装状态 |

### 3. 实现位置

在 `cmd/picoclaw/main.go` 中添加 `install` case。

### 4. 可执行文件路径

- 优先使用 `which picoclaw` 查找已安装路径
- 如果找不到，提示用户需要先 `go install` 或手动复制

## 实现步骤

1. 在 main.go 添加 install case
2. 实现 installCmd() 函数
3. 添加 uninstall 和 status 子命令
