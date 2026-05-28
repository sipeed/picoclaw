# `pkg/isolation`

`pkg/isolation` 为 `picoclaw` 启动的子进程提供进程级隔离能力。

它当前不会把 `picoclaw` 主进程自身放进沙箱中运行。

## 生效范围

当前生效范围是子进程启动链路：

- `exec` 工具
- `claude-cli`、`codex-cli` 等 CLI provider
- 进程型 hooks
- MCP `stdio` server

## 一句话理解

- `picoclaw` 主进程仍运行在宿主环境中。
- 所有子进程都应先经过 `pkg/isolation` 的统一启动入口。
- 入口会根据配置和平台，为子进程施加对应隔离。

## 架构

当前实现可以分为四层：

1. 配置层：读取 `config.Config.Isolation`，并通过 `isolation.Configure(cfg)` 注入运行时。
2. 实例目录层：解析 `config.GetHome()`，准备实例目录，并构建运行时用户环境目录。
3. 平台后端层：Linux 使用 `bwrap`；Windows 使用受限 token、低完整性级别和 `Job Object`；其他平台未实现。
4. 统一启动层：`PrepareCommand(cmd)`、`Start(cmd)`、`Run(cmd)`。

所有启动子进程的接入点都应复用这组入口，而不是各自直接调用 `cmd.Start` 或 `cmd.Run`。

## 配置

隔离配置位于：

```json
{
  "isolation": {
    "enabled": false,
    "expose_paths": []
  }
}
```

字段说明：

- `enabled`：是否启用子进程隔离。默认值：`false`。
- `expose_paths`：显式把宿主路径带入隔离环境。仅在 `enabled=true` 时生效。目前只在 Linux 上支持。

示例：

```json
{
  "isolation": {
    "enabled": true,
    "expose_paths": [
      {
        "source": "/opt/toolchains/go",
        "target": "/opt/toolchains/go",
        "mode": "ro"
      },
      {
        "source": "/data/shared-assets",
        "target": "/opt/picoclaw-instance-a/workspace/assets",
        "mode": "rw"
      }
    ]
  }
}
```

`expose_paths` 规则：

- `source`：宿主机路径。
- `target`：隔离环境内的目标路径。
- `mode`：只能是 `ro` 或 `rw`。
- `target` 为空时，默认等于 `source`。
- 同一个 `target` 最终只能保留一条规则。
- 后加载的配置会覆盖先加载的同目标规则。

平台说明：

- Linux 会真实使用 `source -> target` 挂载视图。
- Windows 当前不支持 `expose_paths`。

## 实例根与目录

实例根遵循 `config.GetHome()`：

- 如果设置了 `PICOCLAW_HOME`，使用该值。
- 否则默认使用用户目录下的 `.picoclaw`。

如果 `config.GetHome()` 在隔离开启时最终回退到当前目录 `.`，启动应直接失败。

默认实例目录包括：

- 实例根本身
- `skills`
- `logs`
- `cache`
- `state`
- `runtime-user-env`

`workspace` 优先使用 `cfg.WorkspacePath()` 的结果；未显式配置时才按默认规则派生。

Windows 还会额外准备：

- `runtime-user-env/AppData/Roaming`
- `runtime-user-env/AppData/Local`

## 用户环境重定向

隔离开启后，子进程会收到重定向到实例目录下的独立用户环境。

Linux 注入变量：

- `HOME`
- `TMPDIR`
- `XDG_CONFIG_HOME`
- `XDG_CACHE_HOME`
- `XDG_STATE_HOME`

Windows 注入变量：

- `USERPROFILE`
- `HOME`
- `TEMP`
- `TMP`
- `APPDATA`
- `LOCALAPPDATA`

这些路径都会指向实例根下的 `runtime-user-env`。

## 平台行为

### Linux

Linux 后端当前依赖 `bwrap`（`bubblewrap`）。

能力：

- 最小文件系统视图
- `ipc namespace`
- 子进程用户环境重定向
- `source -> target` 只读或读写挂载

默认映射包括实例根，以及 `/usr`、`/bin`、`/lib`、`/lib64`、`/etc/resolv.conf` 等最小运行时系统路径。

运行时还会按需补充可执行文件本身、其所在目录、生效后的工作目录，以及命令行中的绝对路径参数。

缺少 `bwrap` 时不会自动回退。

安装示例：

- `apt install bubblewrap`
- `dnf install bubblewrap`
- `yum install bubblewrap`
- `pacman -S bubblewrap`
- `apk add bubblewrap`

如果需要临时关闭隔离：

```json
{
  "isolation": {
    "enabled": false
  }
}
```

关闭隔离后，子进程访问或修改更多宿主文件的风险会明显上升。

### Windows

Windows 隔离当前提供的是进程级限制，例如 restricted token、low integrity、job object，以及用户环境目录重定向。

`expose_paths` 目前不支持 Windows。如果配置了该字段，启动应直接失败，而不是假装这些路径已经被暴露进隔离环境。

Windows 后端当前使用：

- 受限 primary token
- 低完整性级别
- `Job Object`
- 子进程用户环境重定向

它当前不会实现真正的 `source -> target` 文件系统重映射。

### macOS 与其他平台

当前尚未实现。

当在未支持的平台上显式开启隔离时，上层运行时应将其视为不支持的配置，而不是假装隔离成功。

## 日志与排障

隔离开启后，PicoClaw 会打印生成后的隔离计划，便于排障。

Linux 日志名：

- `linux isolation mount plan`

Windows 日志名：

- `windows isolation access rules`

如果你怀疑隔离未生效，先检查这些日志里是否出现了不应暴露的宿主路径。

## 与 `restrict_to_workspace` 的关系

- `restrict_to_workspace` 限制的是 agent 默认可访问的路径。
- `pkg/isolation` 限制的是子进程运行时能看到什么文件系统，以及它的用户环境指向哪里。

两者互补，不互相替代。

## 当前限制

- Linux 基于 `bwrap` 实现，而不是纯内建 isolation runtime。
- Linux 当前没有默认启用独立的 `pid namespace`。
- Windows 还没有对所有允许/拒绝路径做完整 ACL 落地。
- macOS 尚未实现。
- 当前隔离的是子进程，不是 `picoclaw` 主进程自身。

## 建议阅读顺序

如果你是第一次看这部分代码，建议按这个顺序阅读：

1. `pkg/config/config.go`
2. `pkg/isolation/runtime.go`
3. `pkg/isolation/platform_linux.go`
4. `pkg/isolation/platform_windows.go`
5. 调用点：
6. `pkg/tools/shell.go`
7. `pkg/providers/*.go`
8. `pkg/agent/hook_process.go`
9. `pkg/mcp/manager.go`

这样能最快建立对配置模型、运行流程和平台边界的整体理解。
