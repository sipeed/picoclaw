# 定时任务与 Cron 作业

> 返回 [README](../README.md)

PicoClaw 将定时作业存储在当前工作区中，可以将它们作为提醒、完全自主的 Agent 回合或 shell 命令来运行。

## 调度类型

PicoClaw 目前在 cron 工具中使用三种调度形式：

- `at_seconds`：一次性任务，相对于当前时间。运行后，作业会从存储中删除。
- `every_seconds`：重复间隔，以秒为单位。
- `cron_expr`：重复的 cron 表达式，如 `0 9 * * *`。

CLI 命令 `picoclaw cron add` 目前仅支持重复作业：

- `--every <seconds>`
- `--cron '<expr>'`

目前没有用于一次性 `at` 作业的 CLI 参数。

示例：

```bash
picoclaw cron add --name "Daily summary" --message "Summarize today's logs" --cron "0 18 * * *"
picoclaw cron add --name "Ping" --message "heartbeat" --every 300 --deliver
```

## 执行模式

作业存储时带有消息负载，可以以三种稳定的面向用户的模式执行：

### `deliver: false`

这是 cron 工具的默认值。

当作业触发时，PicoClaw 会将保存的消息作为新的 Agent 回合发送回 Agent 循环。对于可能需要推理、工具或生成回复的定时工作使用此模式。

### `deliver: true`

当作业触发时，PicoClaw 会将保存的消息直接发布到目标渠道和接收者，无需 Agent 处理。

CLI `picoclaw cron add --deliver` 参数使用此模式。

### `command`

当 cron-tool 作业包含 `command` 时，PicoClaw 会通过 `exec` 工具运行该 shell 命令，并将命令输出发布回渠道。

对于命令作业，创建时会将 `deliver` 强制设为 `false`。保存的 `message` 仅作为描述性文本；定时动作是 shell 命令。

当前的 CLI `picoclaw cron add` 命令不暴露 `command` 参数。

## 配置与安全门控

### `tools.cron`

`tools.cron.enabled` 控制面向 Agent 的 `cron` 工具是否注册。默认值：`true`。

如果禁用 `tools.cron`，用户将无法再通过 Agent 工具创建或管理作业。网关仍会启动 `CronService`，但不会安装作业执行回调。因此，定时作业不会实际运行；一次性作业可能会被删除，重复作业可能会被重新调度而不会执行其负载。CLI 仍使用相同的作业存储。

`tools.cron.exec_timeout_minutes` 设置定时命令执行的超时时间。默认值：`5`。设为 `0` 表示无超时。

### `tools.exec`

定时命令作业依赖于 `tools.exec.enabled`。默认值：`true`。

如果 `tools.exec.enabled` 为 `false`：

- 新的命令作业会被 cron 工具拒绝
- 现有命令作业在触发时会发布 `command execution is disabled` 错误

`tools.exec.allow_remote` 仍由 exec 工具强制执行，但 cron 命令调度在创建作业时已经需要内部渠道。实际上，提醒作业可以从远程渠道调度，而定时命令作业仅限于内部渠道。

### `allow_command`

`tools.cron.allow_command` 默认为 `true`。

这不是硬禁用开关。如果将 `allow_command` 设为 `false`，PicoClaw 仍会在调用者明确传递 `command_confirm: true` 时允许命令作业。

命令作业还需要内部渠道。非命令提醒没有此限制。

示例：

```json
{
  "tools": {
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 5,
      "allow_command": true
    },
    "exec": {
      "enabled": true
    }
  }
}
```

## 持久化与位置

Cron 作业存储在：

```text
<workspace>/cron/jobs.json
```

默认工作区是：

```text
~/.picoclaw/workspace
```

如果设置了 `PICOCLAW_HOME`，默认工作区变为：

```text
$PICOCLAW_HOME/workspace
```

网关和 `picoclaw cron` CLI 子命令使用相同的 `cron/jobs.json` 文件。

注意：

- 一次性 `at_seconds` 作业运行后会删除
- 重复作业保留在存储中直到被删除
- 禁用的作业保留在存储中，仍会显示在 `picoclaw cron list` 中
