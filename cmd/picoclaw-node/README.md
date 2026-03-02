# go_node

Go 实现的 OpenClaw node 节点，与 Android node 架构一致，通过 WebSocket 连接 Gateway，仅实现 `system.run`（执行 shell 命令）。

## 架构

- **GatewaySession**：WebSocket 连接、connect 握手、RPC 协议、`node.invoke.request` 事件处理
- **InvokeDispatcher**：仅支持 `system.run` 命令，执行 shell 并返回 stdout/stderr
- **DeviceIdentity**：Ed25519 设备身份、connect 时 device 签名

## 配置文件

配置通过 JSON 文件管理，支持以下路径（按优先级）：
- `-config` 参数指定的路径
- 当前目录 `config.json`
- `~/.openclaw/go_node.json`

### 配置示例

```json
{
  "gateway": {
    "host": "127.0.0.1",
    "port": 18789,
    "tls": false,
    "token": "your-gateway-token",
    "password": ""
  },
  "node": {
    "displayName": "my-node",
    "nodeId": "my-node"
  },
  "reconnect": {
    "maxRetries": 0,
    "retryIntervalMs": 5000
  },
  "exec": {
    "workDir": "/tmp/go_node_work",
    "allowedCommands": ["ls", "echo", "cat"],
    "allowAllCommands": false
  }
}
```

| 字段 | 说明 | 默认 |
|------|------|------|
| gateway.host | Gateway 地址 | 127.0.0.1 |
| gateway.port | Gateway 端口 | 18789 |
| gateway.tls | 是否使用 WSS | false |
| gateway.token | Gateway 令牌 | - |
| gateway.password | Gateway 密码 | - |
| node.displayName | 节点显示名 | hostname |
| node.nodeId | 节点 ID | 同 displayName |
| reconnect.maxRetries | 最大重连次数（0=无限） | 0 |
| reconnect.retryIntervalMs | 重连间隔（毫秒） | 5000 |
| exec.workDir | 工作目录，命令仅在此目录及子目录下执行，空=不限制 | - |
| exec.allowedCommands | 可执行命令白名单（按 argv[0] 的 basename），空=允许所有 | - |
| exec.allowAllCommands | 为 true 时忽略白名单，允许执行任意命令（含 rawCommand） | false |

**安全说明**：
- `workDir` 非空时：`params.cwd` 必须在 workDir 下（相对路径或绝对路径均校验）
- `allowedCommands` 非空时：仅白名单中的命令可执行，且禁用 rawCommand（只支持 command 数组）
- `allowAllCommands: true` 时： bypass 白名单，允许任意命令和 rawCommand；适用于完全信任的节点环境

> 注意：go_node 使用 client ID `node-host`，与 openclaw node-host 相同，以便通过 gateway 的 client ID 校验。

## 构建

```bash
go build -o go_node .
```

## 运行

```bash
# 生成示例配置
./go_node -init-config
# 或指定路径
./go_node -init-config -config ~/.openclaw/go_node.json

# 使用默认 config.json 运行
./go_node

# 指定配置文件
./go_node -config /path/to/config.json
```

## system.run 参数

与 openclaw node-host 兼容，paramsJSON 示例：

```json
{
  "command": ["echo", "hello"],
  "cwd": "/tmp",
  "env": {"KEY": "value"},
  "timeoutMs": 30000
}
```

或使用 rawCommand（shell -c）：

```json
{
  "rawCommand": "echo hello | wc -c"
}
```
