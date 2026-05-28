# 当前硬件支持现状与串口 Tool 方案

## 现状结论

当前项目已有的硬件相关能力主要分为两条线：

1. 设备事件监控
   - `pkg/devices` 已实现设备事件服务。
   - 当前只有 Linux USB 热插拔事件源 `pkg/devices/sources/usb_linux.go`。
   - 能力定位是“发现和通知”，不是“总线读写控制”。

2. 硬件控制 Tool
   - `pkg/tools/hardware/i2c*.go`：I2C Tool，支持 `detect`、`scan`、`read`、`write`。
   - `pkg/tools/hardware/spi*.go`：SPI Tool，支持 `list`、`transfer`、`read`。
   - 这两类 Tool 当前都只在 Linux 主机上启用，直接依赖 `/dev/i2c-*` 与 `/dev/spidev*`。

因此，项目在“硬件支持能力”上已经具备：

- Linux USB 设备插拔感知
- Linux I2C 总线控制
- Linux SPI 总线控制

但还缺少：

- 串口/UART 控制
- macOS / Windows 下可直接使用的硬件控制 Tool
- 面向统一硬件抽象的跨总线能力模型

## 本次新增

本次新增内建 `serial` Tool，并接入现有 Tool 体系：

- 配置项：`tools.serial.enabled`
- Tool 注册：`pkg/agent/agent_init.go`
- Web 工具页：`/api/tools` 能展示与切换 `serial`
- 前端状态文案：新增 `requires_serial_platform`

## Serial Tool 设计

`serial` 采用无状态调用模型，每次请求都自行打开和关闭端口，避免在 agent 回合之间维护串口会话状态。

支持动作：

- `list`：枚举主机串口
- `read`：从串口读取指定长度字节
- `write`：向串口写入字节或文本

公共参数：

- `port`
- `baud`
- `data_bits`
- `parity`
- `stop_bits`
- `timeout_ms`

当前波特率实现边界：

- Windows 允许配置工具层接受的范围 `50-4000000`
- Linux / macOS 当前仅支持标准 termios 波特率，实际支持到 `230400`
- 因此 `baud` 的跨平台可移植取值应优先使用 `230400` 及以下的常见标准速率

安全约束：

- `write` 必须显式传 `confirm: true`
- 单次读写负载限制为 `4096` 字节
- `port` 只接受白名单串口名：
  - Linux / macOS 仅允许 `/dev/tty*`、`/dev/cu.*` 及对应简写设备名
  - Windows 仅允许 `COM\d+` 或 `\\.\COM\d+`
  - 明确拒绝 `..`、普通文件绝对路径、盘符路径等非串口设备路径，避免路径穿越或误打开任意文件

## 跨平台实现边界

- Linux / macOS：
  - 基于 `golang.org/x/sys/unix` 和 termios 配置串口参数。
  - 当前仅接入标准 termios 波特率映射，最高到 `230400`，尚未扩展 `460800`、`921600`、`1000000`、`2000000` 等更高速率。
  - 通过 `/dev/...` 枚举和访问设备。

- Windows：
  - 基于 `kernel32` 串口 API 配置 `DCB` 和 `COMMTIMEOUTS`。
  - 当前读写仍使用同步 `ReadFile` / `WriteFile`；一旦 syscall 已进入执行，turn context cancellation 不能立即打断，只能等待 `COMMTIMEOUTS` 触发后返回。
  - 通过注册表 `HARDWARE\\DEVICEMAP\\SERIALCOMM` 枚举端口。

- 其他平台：
  - `serial` Tool 显式返回 unsupported，不做静默降级。

## 后续建议

1. 如果需要持续交互式串口会话，建议再增加 session 型 Tool，而不是让 LLM 反复做短连接轮询。
2. 如果后续要支持 CAN、GPIO、PWM，建议抽出统一的硬件 capability 描述层，而不是继续只靠 Tool 名称区分。
3. 若需要生产级稳定性，建议补真实串口回环测试，至少覆盖 Linux PTY 和 Windows COM 模拟场景。
