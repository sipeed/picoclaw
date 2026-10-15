# QQ C2C Channel · 设计文档

> 目标：为 PicoClaw 新增一个**专用于一对一聊天**的 QQ 通道（`qq_c2c`），
> **不依赖 `tencent-connect/botgo`**，依据 [QQ 机器人 API v2](https://bot.q.qq.com/wiki/develop/api-v2/dev-prepare/getting-started.html) 从零实现，
> 做到自主可控、可靠、稳定，并尽可能榨干单聊 API 的交互能力。
>
> 起于 2026-10-16 05:13（无人值守窗口 05:13–09:00）。

---

## 0. 先记三条平台事实（决定设计边界）

| 事实 | 影响 |
|---|---|
| **主动消息可用**（官方文档 2026-09 更新） | 同一接口 `POST /v2/users/{openid}/messages`：**不带 `msg_id`/`event_id` 即为主动消息**。用户可在 QQ 客户端关闭「允许主动发送」开关（关闭后报 `40054013 用户拒收消息`；机器人无权限报 `40034105`） |
| **主动消息频控（单聊）** | 认证机器人 10/qps；单关系 20/qpm；**每日上限 1000 条/用户** |
| **互动召回消息** `is_wakeup=true` | 用户对话后 30 天内可召回，每周期 1 条（当天 / 1-3 天 / 3-7 天 / 7-30 天，共 4 个周期） |
| **被动回复**（携带 `msg_id`） | 单聊窗口 **60 分钟**，每条最多回 **4** 次 |
| **单聊能力矩阵** | 文本 收发✅ / **Markdown 发✅ 收❌** / 图片·视频·语音·文件 收发✅ / 结构化卡片 发❌ 收✅ / 引用 收✅ |

> ✅ 定调：**这是一个兼具「应答」与「主动触达」的通道**。
> 两者共用同一个接口，区别只是**是否携带 `msg_id`**：
> - 有 `msg_id` → 被动回复（受 60 分钟 / 4 次限制）
> - 无 `msg_id` → **主动消息**（受 qps/qpm/每日 1000 条限制，且用户开关必须开着）
> - `is_wakeup=true` → 互动召回（每周期 1 条）
>
> 通道已跟踪用户的开关状态（`C2C_MSG_RECEIVE` 开启 / `C2C_MSG_REJECT` 关闭），
> 并提供 `SendActive()` / `SendWakeup()` 两个入口。

---

## 1. API 事实（实现依据）

### 1.1 鉴权（Access Token，Token 方式已废弃）

```
POST https://bots.qq.com/app/getAppAccessToken
{ "appId": <AppID>, "clientSecret": <AppSecret> }
→ { "access_token": "...", "expires_in": 7200 }        # 秒
```

- 之后所有开放接口带：`Authorization: QQBot <access_token>`
- **需自管刷新**（提前 ~60s 刷新；401/11244 时立即重取）

### 1.2 WebSocket 接入

```
GET https://api.sgroup.qq.com/websocket   (Authorization: QQBot <token>)
→ { "url": "wss://api.sgroup.qq.com/websocket/..." }
```

- 连上后平台下发 `op=10 Hello`（含 `heartbeat_interval`，本案实测 ~41s）
- 发 `op=2 Identify { token: "QQBot <access>", intents, shard:[0,1] }`
- 按 interval 发 `op=1 Heartbeat`（可带最新 `s` 序列号）
- 断线用 `op=6 Resume` 续；重连要有退避
- 事件：`op=0`，`t` = 事件名（我们关心 `C2C_MESSAGE_CREATE`、`INTERACTION_CREATE`、`FRIEND_ADD`、`C2C_MSG_RECEIVE`）

### 1.3 发消息（被动回复）

```
POST /v2/users/{user_openid}/messages
{
  "msg_type": 0|2|6|7,          # 0 文本 / 2 Markdown / 6 输入状态 / 7 富媒体
  "content": "...",             # msg_type=0
  "markdown": { "content": "..." },   # msg_type=2（与 content 互斥）
  "keyboard": { ... },          # 按钮
  "msg_id": "<事件 d.id>",      # 被动回复必带（与 event_id 二选一）
  "msg_seq": 1                  # 同一 msg_id+msg_seq 重复会失败 → 递增
}
→ { "id": "...", "timestamp": "...", "ext_info": { "ref_idx": "REFIDX_..." } }
```

- **`msg_seq` 必须递增**（同一 `msg_id` 下）→ 天然适配"多条回复/流式"
- `msg_id` 来自入站事件 `d.id`；窗口 60 分钟、上限 4 条

### 1.4 流式输出（本通道的重点能力）

```
POST /v2/users/{user_openid}/stream_messages
{
  "input_mode": "replace",        # replace: content_raw 为全量正文（须以上次已下发前缀开头）
  "input_state": 1|10,            # 1 生成中 / 10 结束
  "index": 0,1,2...,              # 分片序号递增
  "content_type": "markdown",     # text | markdown
  "content_raw": "...",
  "msg_id": "...", "msg_seq": 1,
  "stream_msg_id": "..."          # 首片由服务端返回，续片必带
}
→ { "id": "<stream_msg_id>", "timestamp": "...", "remain_msg_len": N }
```

- 首片返回 `id` = `stream_msg_id`；后续分片带上它，`index` 递增
- 结束片：`input_state=10`
- **限速 50 QPS** → 客户端做**节流**（建议 ≥300ms 合并一次增量）

### 1.5 按钮 / 结构化交互

- 发消息带 `keyboard`：
  ```json
  { "keyboard": { "content": { "rows": [ { "buttons": [
      { "id": "b1",
        "render_data": { "label": "确认", "visited_label": "已确认", "style": 1 },
        "action": { "type": 2, "permission": { "type": 2 }, "data": "/confirm",
                    "reply": false, "enter": true, "unsupport_tips": "请升级QQ" } } ] } ] } } }
  ```
  - `action.type`: `0` 跳转 / `1` 回调 / `2` **指令**（自动在输入框插入 `@bot data`）
  - 单聊**自定义按钮已开放，无需申请模板**
  - 行≤5、每行≤5 个按钮
- 用户点击 → `INTERACTION_CREATE`（`intents` 含 `1<<26`）→ 机器人须在 **3 秒**内
  `PUT /interactions/{interaction_id}` 响应

### 1.6 指令面板 / 文本交互（Markdown 内嵌标签）

- 回车指令（点击即发送）：`<qqbot-cmd-enter text="xxx" />`
- 参数指令（插入输入框）：`<qqbot-cmd-input text="xxx" show="xxx" reference="false" />`
- `text` 需 **urlencode**，≤100 字符；**仅 Markdown 消息支持**，**单聊可用**
- → 这就是"**指令面板**"的落地方式：把 PicoClaw 的内置能力（skills/命令）渲染成一组 `<qqbot-cmd-enter>` 标签 ✓

---

## 2. 架构设计

```
pkg/channels/qqc2c/
├─ channel.go      // Channel 实现：Start/Stop/Send/StartTyping/ReasoningChannelID、事件分发
├─ api.go          // 纯 HTTP 客户端：getAppAccessToken / PostC2CMessage / StreamC2CMessage / PutInteraction
├─ token.go        // Access Token 缓存 + 提前刷新 + 401 重试
├─ ws.go           // 自研 WebSocket 客户端：Hello/Identify/Heartbeat/Resume/重连退避/分片重组
├─ dto.go          // 事件与消息体结构体（自己定义，不引 botgo）
├─ keyboard.go     // 按钮/指令面板构造器（结构化提问 → buttons；内置能力 → cmd-enter 面板）
├─ stream.go       // 流式会话：增量合并 + 节流 + input_state 收尾
├─ config.go       // QQ C2C 配置结构（注册到 channel_list.qq_c2c）
└─ *_test.go       // 单测（token 刷新、分片合并、msg_seq 递增、键盘 JSON、URL 编码）
```

**关键设计点**

1. **零 botgo**：只依赖标准库 + `github.com/gorilla/websocket`（已是仓库直接依赖 ✓）
2. **一份通道，两种收发**：
   - 入站：WS 事件 → `bus.InboundMessage`（带 `msg_id` 供回复）
   - 出站：`Send()` → 被动回复（自动带 `msg_id` + 递增 `msg_seq`）
3. **流式**：`AgentLoop` 已有 streaming 钩子 → 每个增量喂给 `stream.go`（≥300ms 合并）→ 结束发 `input_state=10`
4. **结构化提问**：通道提供 `AskWithButtons(question, options[])` → 发 keyboard → 等 `INTERACTION_CREATE`（带超时）→ 回调
5. **指令面板**：`RenderCommandPanel()` → 由 skills/命令清单生成 Markdown + `<qqbot-cmd-enter>` 标签
6. **健壮性**：token 过期自愈、WS 断线重连（退避 + Resume）、发送失败按平台码分类（`msg_seq` 冲突 → 递增重试；窗口超时 → 明确报错不重试）
7. **与旧通道并存**：新类型名 `qq_c2c`，配置段独立；旧 `qq` 通道不动（可并存，也可只启用新的）

---

## 3. 配置设计（含 launcher UI）

```jsonc
"channel_list": {
  "qq_c2c": {
    "enabled": true,
    "type": "qq_c2c",
    "allow_from": ["<user_openid>"],
    "settings": {
      "app_id": "...",
      "app_secret": "...",          // 走 .security.yml 或 env 注入
      "markdown": true,             // 默认 Markdown 发送
      "streaming": true,            // 流式输出
      "stream_interval_ms": 300,    // 增量合并节流
      "command_panel": true,        // 指令面板
      "reply_window_minutes": 60,   // 平台限制，用于本地判断能否被动回复
      "max_replies_per_message": 4,
      "keyboard_max_rows": 5,
      "keyboard_max_per_row": 5
    }
  }
}
```

**launcher 侧**（`web/frontend`）：新增「QQ C2C Channel」配置卡片 —— AppID / AppSecret / 开关（Markdown·流式·指令面板）/ 白名单 / 节流毫秒。
（按兴逸指示：**等后端 PR 被接受后再动前端**；本轮先留好 settings 结构，前端后续补。）

---

## 4. 验收条件（逐条自证）

- [ ] **A1** 新通道 `qq_c2c` 可在 config 独立启用；与旧 `qq` 并存不冲突
- [ ] **A2** `go list -deps ./... | grep -c tencent-connect/botgo` 对**本包**为 0（自主可控）
- [ ] **A3** 能取到 Access Token，并在过期前自动刷新（单测 + 实机）
- [ ] **A4** WS 握手成功（Hello→Identify→Heartbeat），日志可见；断线能自动重连（可用 kill 网络验证）
- [ ] **A5** 收到单聊消息 → 生成回复 → 用户 QQ 可见（**实机与兴逸私聊验证**）
- [ ] **A6** 流式输出：回复以多片增量呈现（实机观察 + `remain_msg_len` 递减）
- [ ] **A7** Markdown 消息发送成功（`msg_type=2`）
- [ ] **A8** 按钮：发出带 `keyboard` 的消息；点击后收到 `INTERACTION_CREATE` 并 3 秒内 `PUT` 响应
- [ ] **A9** 指令面板：Markdown 内含 `<qqbot-cmd-enter>` 标签，点击后能触发内置命令
- [ ] **A10** 编译通过：`CGO_ENABLED=0 GOOS=linux GOARCH=arm64 -tags goolm,stdjson`
- [ ] **A11** 部署到路由器后：通道健康（无 panic / 无反复重连 / 内存稳定），旧功能不退化
- [ ] **A12** 单测覆盖：token 刷新、分片合并、msg_seq 递增、keyboard JSON、urlencode
- [ ] **A13** 两个上游各提 PR（含设计说明与验证证据）
- [ ] **A14** 出《无人值守工作报告》

---

## 5. 里程碑（按窗口推进）

| # | 阶段 | 产出 | 状态 |
|---|---|---|---|
| M1 | 设计 | 本文件 | ✅ |
| M2 | 骨架 | 包结构 + 配置注册 + 编译通过（空实现） | ⬜ |
| M3 | 鉴权 | token.go + api.go（取 token 成功） | ⬜ |
| M4 | 收发 | ws.go 自研客户端 + channel.go 收发打通 | ⬜ |
| M5 | 高级 | stream.go 流式 + keyboard.go 按钮/指令面板 | ⬜ |
| M6 | 测试 | 单测 + 实机联调 | ⬜ |
| M7 | 部署 | 路由器替换 + Vi 剩余安装 | ⬜ |
| M8 | 提交 | 两个上游 PR | ⬜ |
| M9 | 报告 | 《无人值守工作报告》 | ⬜ |

> **风险**：平台"主动推送已取消"意味着**不能指望通道主动发消息** —— 若兴逸期望"定时提醒推送"，
> 需改为"被动回复 + 互动召回（`is_wakeup`，4 周期各 1 条）"策略，并写入报告请其确认。
