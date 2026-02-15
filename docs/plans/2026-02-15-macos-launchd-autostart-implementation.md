# macOS 开机启动 (launchd) 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在 picoclaw 项目中添加 `picoclaw install` 命令，实现 macOS 开机自动启动 `picoclaw gateway`

**Architecture:** 通过生成 launchd plist 配置文件到 `~/Library/LaunchAgents/` 目录，实现用户级开机自启动

**Tech Stack:** Go, macOS launchd

---

## 实现步骤

### Task 1: 添加 install case 到 main.go

**Files:**
- Modify: `cmd/picoclaw/main.go:126-145`

**Step 1: 在 main.go 的 switch 语句中添加 install case**

在 `case "cron":` 之前添加：

```go
case "install":
    installCmd()
```

**Step 2: 运行测试验证**

Run: `go build ./cmd/picoclaw/`
Expected: 编译成功（因为 installCmd 尚未定义，会有警告）

---

### Task 2: 实现 installCmd 函数

**Files:**
- Modify: `cmd/picoclaw/main.go:199-216`

**Step 1: 添加 installCmd 函数和帮助信息**

在 `printHelp()` 函数后添加：

```go
func installCmd() {
    args := os.Args[2:]
    for _, arg := range args {
        switch arg {
        case "--uninstall":
            uninstallInstall()
            return
        case "--status":
            statusInstall()
            return
        case "--help", "-h":
            installHelp()
            return
        }
    }
    doInstall()
}

func installHelp() {
    fmt.Println("\nInstall commands:")
    fmt.Println("  picoclaw install              Install launchd plist (current user)")
    fmt.Println("  picoclaw install --uninstall Uninstall launchd plist")
    fmt.Println("  picoclaw install --status    Show install status")
}

func doInstall() {
    // 查找 picoclaw 可执行文件路径
    execPath, err := findExecutable()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    // 生成 plist 内容
    plistContent := generatePlist(execPath)

    // 写入文件
    plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "io.picoclaw.gateway.plist")
    if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
        fmt.Printf("Error writing plist: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("✓ Installed launchd plist to %s\n", plistPath)
    fmt.Println("Run 'picoclaw install --status' to check status")
}

func findExecutable() (string, error) {
    // 使用 which 查找
    cmd := exec.Command("which", "picoclaw")
    out, err := cmd.Output()
    if err == nil {
        return strings.TrimSpace(string(out)), nil
    }

    // 尝试常见路径
    commonPaths := []string{
        "/usr/local/bin/picoclaw",
        "/usr/bin/picoclaw",
        filepath.Join(os.Getenv("HOME"), "go", "bin", "picoclaw"),
    }

    for _, p := range commonPaths {
        if _, err := os.Stat(p); err == nil {
            return p, nil
        }
    }

    return "", fmt.Errorf("picoclaw not found in PATH. Please run 'go install' first or ensure picoclaw is in your PATH")
}

func generatePlist(execPath string) string {
    return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.picoclaw.gateway</string>
    <key>ProgramArguments</key>
    <array>
        <string>` + execPath + `</string>
        <string>gateway</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`
}

func uninstallInstall() {
    plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "io.picoclaw.gateway.plist")
    if _, err := os.Stat(plistPath); os.IsNotExist(err) {
        fmt.Println("No launchd plist found.")
        return
    }

    // 先 unload
    exec.Command("launchctl", "unload", plistPath).Run()

    if err := os.Remove(plistPath); err != nil {
        fmt.Printf("Error removing plist: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("✓ Removed %s\n", plistPath)
}

func statusInstall() {
    plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "io.picoclaw.gateway.plist")

    if _, err := os.Stat(plistPath); os.IsNotExist(err) {
        fmt.Println("Status: Not installed")
        fmt.Println("Run 'picoclaw install' to install")
        return
    }

    fmt.Println("Status: Installed")
    fmt.Printf("Path: %s\n", plistPath)

    // 检查是否正在运行
    cmd := exec.Command("launchctl", "list", "io.picoclaw.gateway")
    out, err := cmd.Output()
    if err == nil {
        fmt.Println("\nRunning: Yes")
        fmt.Println(string(out))
    } else {
        fmt.Println("\nRunning: No")
    }
}
```

**Step 2: 添加 import**

确保已导入 `os/exec`：

```go
import (
    // ... 其他 import
    "os/exec"
)
```

**Step 3: 运行测试验证**

Run: `go build ./cmd/picoclaw/`
Expected: 编译成功

**Step 4: 手动测试**

```bash
./picoclaw install --help
./picoclaw install --status
```

---

### Task 3: 更新帮助信息

**Files:**
- Modify: `cmd/picoclaw/main.go:201-215`

**Step 1: 在 printHelp 中添加 install 命令**

在 Commands 列表中添加：

```go
fmt.Println("  install     Install/uninstall launchd service (macOS)")
```

---

### Task 4: 测试完整流程

**Step 1: 构建并测试安装**

```bash
go build -o picoclaw ./cmd/picoclaw/
./picoclaw install
./picoclaw install --status
```

Expected: 成功创建 plist 文件

**Step 2: 测试卸载**

```bash
./picoclaw install --uninstall
./picoclaw install --status
```

Expected: 成功删除 plist 文件

---

### Task 5: 提交代码

```bash
git add cmd/picoclaw/main.go
git commit -m "feat: add macOS launchd autostart support

- Add 'picoclaw install' command
- Support install, uninstall, and status options
- Auto-detect picoclaw executable path"
```
