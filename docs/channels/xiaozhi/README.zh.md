# xiaozhi 频道（picoclaw-voice）

`picoclaw-voice` 是实现 xiaozhi-esp32 WebSocket 协议的独立语音网关进程。它将运行 xiaozhi 固件的 ESP32 设备或任意兼容客户端接入 PicoClaw AI 引擎，完成 **Opus 音频 → ASR → LLM → TTS → Opus 音频** 的全双工对话链路。

## 架构概览

```
xiaozhi 客户端                picoclaw-voice               PicoClaw
（ESP32 / UE5 / ...）        WebSocket :8765              config.json
        │                          │                          │
        │  WebSocket (Opus)        │                          │
        │ ─────────────────────>   │                          │
        │                          │  火山引擎 ASR (doubao)   │
        │                          │ ─────────────────────>   │
        │                          │  AgentLoop (LLM)         │
        │                          │ ─────────────────────>   │
        │                          │  火山引擎 TTS (doubao)   │
        │  WebSocket (Opus)        │ ─────────────────────>   │
        │ <─────────────────────   │                          │
```

## 连接

WebSocket 端点：`ws://<host>:8765/xiaozhi/v1/`

连接建立后，**客户端必须首先发送 `hello` 消息**，完成握手后方可收发音频。

## 消息格式

所有文本消息均为 JSON，二进制消息为原始 Opus 帧。

### 客户端 → 服务端

#### hello — 握手

```json
{
  "type": "hello",
  "version": 3,
  "transport": "websocket",
  "device_id": "aa:bb:cc:dd:ee:ff",
  "audio_params": {
    "format": "opus",
    "sample_rate": 16000,
    "channels": 1,
    "frame_duration": 60
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `version` | int | 是 | 协议版本，当前为 `3` |
| `transport` | string | 是 | 传输方式，固定 `"websocket"` |
| `device_id` | string | 否 | 设备唯一标识（MAC 地址或自定义 ID）；留空时服务端自动分配 |
| `audio_params.format` | string | 是 | 音频格式，固定 `"opus"` |
| `audio_params.sample_rate` | int | 是 | 采样率，固定 `16000` |
| `audio_params.channels` | int | 是 | 声道数，固定 `1` |
| `audio_params.frame_duration` | int | 是 | 帧时长（ms），固定 `60` |

---

#### listen — VAD 控制

```json
{
  "type": "listen",
  "state": "start",
  "session_id": "uuid-v4",
  "memory_id": "user-key"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `state` | string | 是 | `"start"` 开始录音 / `"end"` 停止录音（VAD 完成）/ `"stop"` 取消当前轮次 |
| `session_id` | string | 否 | 本轮对话的唯一 ID（turn_id）；`start` 时建议由客户端生成 UUID，服务端兜底自动生成 |
| `memory_id` | string | 否 | 指定 LLM 记忆 key；留空时依次回退：`PICOCLAW_VOICE_OWNER_ID` → `device_id` |

状态转换：

```
(空闲) ── start ──> (录音中) ── end ──> (推理中) ── (完成) ──> (空闲)
                        └── stop ──> (空闲，丢弃当前音频)
```

---

#### abort — 打断

```json
{
  "type": "abort",
  "reason": "wake_word_detected"
}
```

立即中止当前 TTS 播放和推理，服务端发送 `tts.state=abort` 并回到空闲状态。

---

#### 二进制帧 — 上行音频

`listen.start` 之后持续发送的 Opus 压缩帧，每帧 60 ms（960 samples @ 16 kHz）。

---

### 服务端 → 客户端

#### hello — 握手响应

```json
{
  "type": "hello",
  "version": 3,
  "transport": "websocket",
  "session_id": "uuid-v4",
  "audio_params": {
    "format": "opus",
    "sample_rate": 16000,
    "channels": 1,
    "frame_duration": 60
  }
}
```

| 字段 | 说明 |
|------|------|
| `session_id` | 本次连接的会话 ID |

---

#### stt — ASR 识别结果

```json
{ "type": "stt", "text": "你好世界", "state": "recognizing" }
{ "type": "stt", "text": "你好世界。", "state": "stop" }
```

| `state` 值 | 说明 |
|------------|------|
| `"recognizing"` | 流式中间结果（可能更新） |
| `"stop"` | 最终识别结果 |

---

#### llm — LLM 推理内容

```json
{ "type": "llm", "text": "你好！有什么可以帮你的？", "emotion": "neutral" }
```

| 字段 | 说明 |
|------|------|
| `text` | 本句 LLM 输出文本（断句后逐句发送） |
| `emotion` | 当前情绪标签，固定 `"neutral"`（预留扩展） |

---

#### tts — TTS 状态控制

```json
{ "type": "tts", "state": "start" }
{ "type": "tts", "state": "sentence_start", "text": "你好！有什么可以帮你的？" }
{ "type": "tts", "state": "sentence_end" }
{ "type": "tts", "state": "stop" }
```

| `state` 值 | 说明 |
|------------|------|
| `"start"` | 开始整轮 TTS 播放 |
| `"sentence_start"` | 本句开始，附带文本 `text` 字段 |
| `"sentence_end"` | 本句结束 |
| `"stop"` | 整轮 TTS 播放结束 |
| `"abort"` | TTS 被打断（客户端发 `abort` 或会话异常） |

---

#### 二进制帧 — 下行音频

TTS 合成后的 Opus 帧，编码参数与握手一致（16 kHz、单声道、60 ms）。

---

## 完整会话时序

```
Client                          Server
  │                               │
  │   WS connect                  │
  │ ─────────────────────────>    │
  │   hello (device_id, ...)      │
  │ ─────────────────────────>    │
  │                               │   hello (session_id)
  │ <─────────────────────────    │
  │                               │
  │   listen (state=start)        │
  │ ─────────────────────────>    │
  │   [binary Opus frames]        │
  │ ─────────────────────────>    │
  │   listen (state=end)          │
  │ ─────────────────────────>    │
  │                               │   stt (recognizing)
  │ <─────────────────────────    │
  │                               │   stt (stop, final text)
  │ <─────────────────────────    │
  │                               │   llm (text=...)
  │ <─────────────────────────    │
  │                               │   tts (start)
  │ <─────────────────────────    │
  │                               │   tts (sentence_start)
  │ <─────────────────────────    │
  │                               │   [binary Opus frames]
  │ <─────────────────────────    │
  │                               │   tts (sentence_end)
  │ <─────────────────────────    │
  │                               │   tts (stop)
  │ <─────────────────────────    │
```

## 音频规格

| 参数 | 值 |
|------|----|
| 编码 | Opus (libopus) |
| 采样率 | 16000 Hz |
| 声道 | 1（单声道） |
| 帧时长 | 60 ms（960 samples/帧） |
| 最大帧长 | 4000 bytes |
| 应用模式 | VoIP (`OPUS_APPLICATION_VOIP`) |

## 会话与记忆管理

**device_id 注册**：同一 `device_id` 新连接到来时，旧会话立即被关闭（last-write-wins），确保不存在幽灵会话。

**记忆 key 优先级**（高→低）：
1. `PICOCLAW_VOICE_OWNER_ID` 环境变量（强制全局共享，适合单用户部署）
2. 客户端 `listen.memory_id` 字段
3. 客户端 `hello.device_id`
4. 服务端自动生成的连接 ID

**turn_id**：每轮对话的唯一标识，由客户端在 `listen.start` 时通过 `session_id` 字段传入；客户端未提供时服务端自动生成。

## 与 xiaozhi-esp32-server 的关系

本模块实现了与 [xiaozhi-esp32-server](https://github.com/xinnan-tech/xiaozhi-esp32-server) 兼容的 v3 协议子集，替换其 LLM/ASR/TTS 后端为 PicoClaw 引擎与火山引擎服务。

未实现的 xiaozhi 协议扩展：OTA 更新、配网相关消息。
